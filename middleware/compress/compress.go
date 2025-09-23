package compress

import (
	"bufio"
	"compress/flate"
	"compress/gzip"
	"net"
	"net/http"
	"strings"

	"github.com/arthurlch/goryu"
)

// CompressionLevel defines the compression level
type CompressionLevel int

const (
	LevelBestSpeed          CompressionLevel = flate.BestSpeed          // 1
	LevelBestCompression    CompressionLevel = flate.BestCompression    // 9
	LevelDefaultCompression CompressionLevel = flate.DefaultCompression // -1
	LevelNoCompression      CompressionLevel = flate.NoCompression      // 0
)

// Config defines the configuration for compression middleware
type Config struct {
	// Next defines when to skip compression
	Next func(c *goryu.Context) bool
	// Level of compression. Default: LevelDefaultCompression
	Level CompressionLevel
	// MinLength is the minimum content length required for compression. Default: 1024 bytes
	MinLength int
	// CompressibleTypes defines which content types should be compressed
	CompressibleTypes []string
}

// New creates a new compression middleware
func New(config ...Config) goryu.Middleware {
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
		if provided.Level != 0 {
			cfg.Level = provided.Level
		}
		if provided.MinLength > 0 {
			cfg.MinLength = provided.MinLength
		}
		if len(provided.CompressibleTypes) > 0 {
			cfg.CompressibleTypes = provided.CompressibleTypes
		}
		cfg.Next = provided.Next
	}

	return func(next goryu.HandlerFunc) goryu.HandlerFunc {
		return func(c *goryu.Context) {
			if cfg.Next != nil && cfg.Next(c) {
				next(c)
				return
			}

			// Check if client supports compression
			acceptEncoding := c.GetHeader("Accept-Encoding")
			if acceptEncoding == "" {
				next(c)
				return
			}

			// Determine compression type based on client support
			var encoding string
			if strings.Contains(acceptEncoding, "gzip") {
				encoding = "gzip"
			} else if strings.Contains(acceptEncoding, "deflate") {
				encoding = "deflate"
			} else {
				next(c)
				return
			}

			// Create a buffer to capture response
			buf := &compressBuffer{
				encoding:     encoding,
				level:        int(cfg.Level),
				minLength:    cfg.MinLength,
				compressible: cfg.CompressibleTypes,
			}

			// Wrap the response writer
			crw := &compressResponseWriter{
				ResponseWriter: c.Writer,
				buffer:         buf,
			}

			c.Writer = crw
			next(c)

			// Finalize compression
			crw.finalize()
		}
	}
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
		// Don't compress error responses
		if statusCode < 200 || statusCode >= 300 {
			w.buffer.encoding = ""
		}
		w.ResponseWriter.WriteHeader(statusCode)
	}
}

func (w *compressResponseWriter) Write(data []byte) (int, error) {
	// Buffer the data
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

	// Ensure headers are sent with correct status code
	if !w.buffer.headersSent {
		if w.statusCode == 0 {
			w.statusCode = http.StatusOK
		}
		w.WriteHeader(w.statusCode)
	}

	// Check if content should be compressed
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

	// Extract main content type (before semicolon)
	mainType := strings.Split(contentType, ";")[0]
	mainType = strings.TrimSpace(strings.ToLower(mainType))

	for _, compressible := range w.buffer.compressible {
		compressible = strings.ToLower(compressible)
		if strings.HasSuffix(compressible, "*") {
			// Wildcard matching (e.g., "text/*")
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
	w._, _ = ResponseWriter.Write(w.buffer.data)
}

// WithGzip is a convenience function to create gzip-only compression
func WithGzip(level CompressionLevel) goryu.Middleware {
	return New(Config{
		Level: level,
		Next: func(c *goryu.Context) bool {
			return !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip")
		},
	})
}
