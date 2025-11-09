package compress
import (
	"bufio"
	"compress/flate"
	"compress/gzip"
	"net"
	"net/http"
	"strings"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/base"
)
type CompressionLevel int
const (
	LevelBestSpeed          CompressionLevel = flate.BestSpeed          
	LevelBestCompression    CompressionLevel = flate.BestCompression    
	LevelDefaultCompression CompressionLevel = flate.DefaultCompression 
	LevelNoCompression      CompressionLevel = flate.NoCompression      
)
type Config struct {
	base.BaseConfig
	Level CompressionLevel
	MinLength int
	CompressibleTypes []string
}
func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if c.MinLength < 0 {
		return base.NewConfigError("MinLength", "cannot be negative")
	}
	if len(c.CompressibleTypes) == 0 {
		return base.NewConfigError("CompressibleTypes", "must have at least one content type")
	}
	return nil
}
func New(config ...Config) func(next context.HandlerFunc) context.HandlerFunc {
	cfg := Config{
		Level:     LevelDefaultCompression,
		MinLength: 1024,
		CompressibleTypes: []string{
			"text/html",
			"text/css",
			"text/plain",
			"text/javascript",
			"text/xml",
			"application/javascript",
			"application/json",
			"application/xml",
			"application/rss+xml",
			"application/atom+xml",
			"image/svg+xml",
		},
	}
	if len(config) > 0 {
		provided := config[0]
		cfg = provided
		if cfg.Level == 0 {
			cfg.Level = LevelDefaultCompression
		}
		if cfg.MinLength == 0 {
			cfg.MinLength = 1024
		}
		if len(cfg.CompressibleTypes) == 0 {
			cfg.CompressibleTypes = []string{
				"text/html", "text/css", "text/plain", "text/javascript", "text/xml",
				"application/javascript", "application/json", "application/xml",
				"application/rss+xml", "application/atom+xml", "image/svg+xml",
			}
		}
	}
	if err := cfg.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, base.MiddlewareError{
					Middleware: "Compress",
					Err:        err,
					StatusCode: http.StatusInternalServerError,
				}, "Compress")
			}
		}
	}
	preHandler := func(c *context.Context) error {
		acceptEncoding := c.GetHeader("Accept-Encoding")
		if acceptEncoding == "" {
			c.Set("compress.skip", true)
			return nil
		}
		var encoding string
		if strings.Contains(acceptEncoding, "gzip") {
			encoding = "gzip"
		} else if strings.Contains(acceptEncoding, "deflate") {
			encoding = "deflate"
		} else {
			c.Set("compress.skip", true)
			return nil
		}
		buf := &compressBuffer{
			encoding:     encoding,
			level:        int(cfg.Level),
			minLength:    cfg.MinLength,
			compressible: cfg.CompressibleTypes,
		}
		crw := &compressResponseWriter{
			ResponseWriter: c.Writer,
			buffer:         buf,
		}
		c.Writer = crw
		c.Set("compress.writer", crw)
		return nil
	}
	postHandler := func(c *context.Context) error {
		if skip, exists := c.Get("compress.skip"); exists && skip.(bool) {
			return nil
		}
		if writerVal, exists := c.Get("compress.writer"); exists {
			if crw, ok := writerVal.(*compressResponseWriter); ok {
				crw.finalize()
			}
		}
		return nil
	}
	return base.PostProcessMiddleware("Compress", cfg.BaseConfig, preHandler, postHandler)
}
type compressBuffer struct {
	data         []byte
	encoding     string
	level        int
	minLength    int
	compressible []string
	headersSent  bool
}
type compressResponseWriter struct {
	http.ResponseWriter
	buffer     *compressBuffer
	statusCode int
}
func (w *compressResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	if !w.buffer.headersSent {
		w.buffer.headersSent = true
		if statusCode < 200 || statusCode >= 300 {
			w.buffer.encoding = ""
		}
		w.ResponseWriter.WriteHeader(statusCode)
	}
}
func (w *compressResponseWriter) Write(data []byte) (int, error) {
	w.buffer.data = append(w.buffer.data, data...)
	return len(data), nil
}
func (w *compressResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}
func (w *compressResponseWriter) Flush() {
	w.finalize()
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
func (w *compressResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}
func (w *compressResponseWriter) finalize() {
	if len(w.buffer.data) == 0 {
		return
	}
	if !w.buffer.headersSent {
		if w.statusCode == 0 {
			w.statusCode = http.StatusOK
		}
		w.WriteHeader(w.statusCode)
	}
	shouldCompress := w.shouldCompress()
	if shouldCompress && w.buffer.encoding != "" && len(w.buffer.data) >= w.buffer.minLength {
		w.writeCompressed()
	} else {
		w.writeUncompressed()
	}
}
func (w *compressResponseWriter) shouldCompress() bool {
	contentType := w.Header().Get("Content-Type")
	if contentType == "" {
		return false
	}
	mainType := strings.Split(contentType, ";")[0]
	mainType = strings.TrimSpace(strings.ToLower(mainType))
	for _, compressible := range w.buffer.compressible {
		compressible = strings.ToLower(compressible)
		if strings.HasSuffix(compressible, "*") {
			prefix := compressible[:len(compressible)-1]
			if strings.HasPrefix(mainType, prefix) {
				return true
			}
		} else if mainType == compressible {
			return true
		}
	}
	return false
}
func (w *compressResponseWriter) writeCompressed() {
	w.Header().Set("Content-Encoding", w.buffer.encoding)
	w.Header().Del("Content-Length")
	switch w.buffer.encoding {
	case "gzip":
		gzipWriter, err := gzip.NewWriterLevel(w.ResponseWriter, w.buffer.level)
		if err != nil {
			w.writeUncompressed()
			return
		}
		defer func() { _ = gzipWriter.Close() }()
		_, _ = gzipWriter.Write(w.buffer.data)
	case "deflate":
		deflateWriter, err := flate.NewWriter(w.ResponseWriter, w.buffer.level)
		if err != nil {
			w.writeUncompressed()
			return
		}
		defer func() { _ = deflateWriter.Close() }()
		_, _ = deflateWriter.Write(w.buffer.data)
	default:
		w.writeUncompressed()
	}
}
func (w *compressResponseWriter) writeUncompressed() {
	_, _ = w.ResponseWriter.Write(w.buffer.data)
}
func WithGzip(level CompressionLevel) func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{
		Level: level,
		BaseConfig: base.BaseConfig{
			Skip: func(c *context.Context) bool {
				return !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip")
			},
		},
	})
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New()
}
