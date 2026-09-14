package recorder

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
)

type Record struct {
	Time         time.Time       `json:"time"`
	Method       string          `json:"method"`
	Path         string          `json:"path"`
	Status       int             `json:"status"`
	LatencyMS    int64           `json:"latency_ms"`
	Key          string          `json:"key,omitempty"`
	RequestBody  json.RawMessage `json:"request_body,omitempty"`
	ResponseBody json.RawMessage `json:"response_body,omitempty"`
	Meta         map[string]any  `json:"meta,omitempty"`
}

type Sink interface {
	Write(Record) error
}

type Config struct {
	base.BaseConfig

	Sink            Sink
	MaxBodyBytes    int64
	CaptureRequest  *bool
	CaptureResponse *bool
	KeyFunc         func(c *context.Context) string
	MetaFunc        func(c *context.Context) map[string]any
}

func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}

func (c *Config) Validate() error {
	if c.MaxBodyBytes <= 0 {
		c.MaxBodyBytes = 64 << 10
	}
	return nil
}

func boolOr(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

type captureWriter struct {
	http.ResponseWriter
	status int
	body   *bytes.Buffer
	limit  int64
}

func (w *captureWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *captureWriter) Write(b []byte) (int, error) {
	if w.body != nil && int64(w.body.Len()) < w.limit {
		remaining := w.limit - int64(w.body.Len())
		if int64(len(b)) <= remaining {
			w.body.Write(b)
		} else {
			w.body.Write(b[:remaining])
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
				base.DefaultErrorHandler(c, err, "Recorder")
			}
		}
	}
	captureReq := boolOr(cfg.CaptureRequest, true)
	captureResp := boolOr(cfg.CaptureResponse, true)

	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if cfg.Sink == nil || (cfg.Skip != nil && cfg.Skip(c)) {
				next(c)
				return
			}

			start := time.Now()

			var reqBody []byte
			if captureReq {
				reqBody = readAndRestore(c, cfg.MaxBodyBytes)
			}

			cw := &captureWriter{
				ResponseWriter: c.Writer,
				status:         http.StatusOK,
				limit:          cfg.MaxBodyBytes,
			}
			if captureResp {
				cw.body = bytes.NewBuffer(nil)
			}
			c.Writer = cw

			next(c)

			c.Writer = cw.ResponseWriter
			rec := Record{
				Time:      start,
				Method:    c.Request.Method,
				Path:      c.Request.URL.Path,
				Status:    cw.status,
				LatencyMS: time.Since(start).Milliseconds(),
			}
			if cw.body != nil {
				rec.ResponseBody = toRawJSON(cw.body.Bytes())
			}
			if reqBody != nil {
				rec.RequestBody = toRawJSON(reqBody)
			}
			if cfg.KeyFunc != nil {
				rec.Key = cfg.KeyFunc(c)
			}
			if cfg.MetaFunc != nil {
				rec.Meta = cfg.MetaFunc(c)
			}

			if err := cfg.Sink.Write(rec); err != nil {
				logger := cfg.Logger
				if logger == nil {
					logger = base.DefaultLogger("Recorder")
				}
				logger.Printf("recorder sink write failed: %v", err)
			}
		}
	}
}

// Default builds the recorder with no sink configured, i.e. a pass-through until
// you supply a Sink via New. It exists for API symmetry with other middleware.
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New()
}

// readAndRestore captures up to max bytes of the body for the record, then
// restores the request body so the handler still reads it in full — even when
// the body is larger than max.
func readAndRestore(c *context.Context, max int64) []byte {
	body := c.Request.Body
	if body == nil {
		return nil
	}
	captured, err := io.ReadAll(io.LimitReader(body, max))
	if err != nil {
		return nil
	}
	c.Request.Body = &restoredBody{r: io.MultiReader(bytes.NewReader(captured), body), c: body}
	return captured
}

// restoredBody re-serves an already-consumed prefix followed by the untouched
// remainder of the original body.
type restoredBody struct {
	r io.Reader
	c io.Closer
}

func (b *restoredBody) Read(p []byte) (int, error) { return b.r.Read(p) }
func (b *restoredBody) Close() error               { return b.c.Close() }

func toRawJSON(b []byte) json.RawMessage {
	if len(b) == 0 {
		return nil
	}
	if json.Valid(b) {
		return append(json.RawMessage(nil), b...)
	}
	quoted, err := json.Marshal(string(b))
	if err != nil {
		return nil
	}
	return quoted
}

type FileSink struct {
	mu sync.Mutex
	f  *os.File
	w  *bufio.Writer
}

func NewFileSink(path string) (*FileSink, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	return &FileSink{f: f, w: bufio.NewWriter(f)}, nil
}

func (s *FileSink) Write(r Record) error {
	line, err := json.Marshal(r)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.w.Write(line); err != nil {
		return err
	}
	if err := s.w.WriteByte('\n'); err != nil {
		return err
	}
	return s.w.Flush()
}

func (s *FileSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.w.Flush(); err != nil {
		_ = s.f.Close()
		return err
	}
	return s.f.Close()
}
