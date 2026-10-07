// Package promptcache is a response cache keyed on the request body (the
// "prompt") rather than the URL, so it can cache POST responses. Streaming
// responses are passed through; bodies larger than MaxBodyBytes are not cached.
package promptcache

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
)

type Config struct {
	base.BaseConfig

	Expiration   time.Duration
	MaxSize      int
	MaxBodyBytes int64
	Methods      []string
	// VaryHeaders are request headers folded into the cache key (e.g. a model
	// selector). Never include auth headers here.
	VaryHeaders  []string
	KeyGenerator func(c *context.Context, body []byte) string
}

func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}

func (c *Config) Validate() error {
	if c.Expiration <= 0 {
		c.Expiration = 5 * time.Minute
	}
	if c.MaxSize <= 0 {
		c.MaxSize = 1000
	}
	if c.MaxSize > 100000 {
		return base.NewConfigError("MaxSize", "cannot exceed 100,000 entries")
	}
	if c.MaxBodyBytes <= 0 {
		c.MaxBodyBytes = 1 << 20
	}
	if len(c.Methods) == 0 {
		c.Methods = []string{http.MethodPost}
	}
	return nil
}

type entry struct {
	status    int
	headers   http.Header
	body      []byte
	expiresAt time.Time
}

type store struct {
	mu      sync.RWMutex
	entries map[string]entry
	maxSize int
}

func (s *store) get(key string) (entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entries[key]
	if !ok || time.Now().After(e.expiresAt) {
		return entry{}, false
	}
	return e, true
}

func (s *store) put(key string, e entry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, replacing := s.entries[key]
	if !replacing && len(s.entries) >= s.maxSize {
		now := time.Now()
		for k, v := range s.entries { // evict an expired entry if possible
			if now.After(v.expiresAt) {
				delete(s.entries, k)
			}
		}
		if len(s.entries) >= s.maxSize {
			for k := range s.entries { // still full: drop one arbitrary entry
				delete(s.entries, k)
				break
			}
		}
	}
	s.entries[key] = e
}

type captureWriter struct {
	http.ResponseWriter
	status   int
	body     *bytes.Buffer
	limit    int64
	overflow bool // response exceeded limit; must not be cached truncated
}

func (w *captureWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *captureWriter) Write(b []byte) (int, error) {
	if !w.overflow {
		remaining := w.limit - int64(w.body.Len())
		if int64(len(b)) <= remaining {
			w.body.Write(b)
		} else {
			if remaining > 0 {
				w.body.Write(b[:remaining])
			}
			w.overflow = true
		}
	}
	return w.ResponseWriter.Write(b)
}

func (w *captureWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *captureWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func New(config ...Config) func(next context.HandlerFunc) context.HandlerFunc {
	cfg := Config{}
	if len(config) > 0 {
		cfg = config[0]
	}
	if err := cfg.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "PromptCache")
			}
		}
	}
	s := &store{entries: make(map[string]entry), maxSize: cfg.MaxSize}

	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if cfg.Skip != nil && cfg.Skip(c) {
				next(c)
				return
			}
			if !methodAllowed(cfg.Methods, c.Request.Method) {
				next(c)
				return
			}
			// With the default key (body only), an authenticated request must not
			// have its response served to a different user. Custom KeyGenerators
			// take responsibility for folding identity into the key.
			if cfg.KeyGenerator == nil && requestIsPrivate(c) {
				next(c)
				return
			}

			body, err := readAndRestore(c, cfg.MaxBodyBytes)
			if err != nil {
				next(c)
				return
			}
			key := cfg.key(c, body)

			if e, ok := s.get(key); ok {
				for k, v := range e.headers {
					c.Writer.Header()[k] = v
				}
				c.Writer.Header().Set("X-Cache", "HIT")
				c.Writer.WriteHeader(e.status)
				_, _ = c.Writer.Write(e.body)
				return
			}

			cw := &captureWriter{
				ResponseWriter: c.Writer,
				status:         http.StatusOK,
				body:           bytes.NewBuffer(nil),
				limit:          cfg.MaxBodyBytes,
			}
			cw.Header().Set("X-Cache", "MISS")
			c.Writer = cw
			next(c)
			c.Writer = cw.ResponseWriter

			if cacheable(cw) {
				s.put(key, entry{
					status:    cw.status,
					headers:   cloneHeader(cw.Header()),
					body:      append([]byte(nil), cw.body.Bytes()...),
					expiresAt: time.Now().Add(cfg.Expiration),
				})
			}
		}
	}
}

func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New()
}

func (cfg Config) key(c *context.Context, body []byte) string {
	if cfg.KeyGenerator != nil {
		return cfg.KeyGenerator(c, body)
	}
	h := sha256.New()
	h.Write([]byte(c.Request.Method))
	h.Write([]byte{0})
	h.Write([]byte(c.Request.URL.Path))
	h.Write([]byte{0})
	for _, hdr := range cfg.VaryHeaders {
		h.Write([]byte(hdr))
		h.Write([]byte{'='})
		h.Write([]byte(c.GetHeader(hdr)))
		h.Write([]byte{0})
	}
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

var errBodyTooLarge = errors.New("promptcache: body exceeds MaxBodyBytes")

// readAndRestore reads up to max bytes for keying, then restores the request body
// so the handler still sees it in full. When the body is larger than max it is
// left un-keyed (errBodyTooLarge) but the full stream is preserved.
func readAndRestore(c *context.Context, max int64) ([]byte, error) {
	body := c.Request.Body
	if body == nil {
		return nil, nil
	}
	captured, err := io.ReadAll(io.LimitReader(body, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(captured)) > max {
		c.Request.Body = &restoredBody{r: io.MultiReader(bytes.NewReader(captured), body), c: body}
		return nil, errBodyTooLarge
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(captured))
	return captured, nil
}

// restoredBody re-serves an already-consumed prefix followed by the untouched
// remainder of the original body, so nothing is lost for the handler.
type restoredBody struct {
	r io.Reader
	c io.Closer
}

func (b *restoredBody) Read(p []byte) (int, error) { return b.r.Read(p) }
func (b *restoredBody) Close() error               { return b.c.Close() }

func methodAllowed(methods []string, m string) bool {
	for _, x := range methods {
		if strings.EqualFold(x, m) {
			return true
		}
	}
	return false
}

func cacheable(cw *captureWriter) bool {
	if cw.overflow {
		return false
	}
	if cw.status < 200 || cw.status >= 300 {
		return false
	}
	ct := strings.ToLower(cw.Header().Get("Content-Type"))
	if strings.Contains(ct, "text/event-stream") || strings.Contains(ct, "application/x-ndjson") {
		return false
	}
	if cw.Header().Get("Set-Cookie") != "" {
		return false
	}
	cc := strings.ToLower(cw.Header().Get("Cache-Control"))
	return !strings.Contains(cc, "no-store") &&
		!strings.Contains(cc, "no-cache") &&
		!strings.Contains(cc, "private")
}

func requestIsPrivate(c *context.Context) bool {
	return c.Request.Header.Get("Authorization") != "" || c.Request.Header.Get("Cookie") != ""
}

func cloneHeader(h http.Header) http.Header {
	out := make(http.Header, len(h))
	for k, v := range h {
		out[k] = append([]string(nil), v...)
	}
	return out
}
