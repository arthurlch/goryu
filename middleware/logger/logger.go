package logger

import (
	"log"
	"net/http"
	"time"

	"github.com/arthurlch/goryu"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	n, err := lrw.ResponseWriter.Write(b)
	if err == nil {
		lrw.size += n
	}
	return n, err
}

func New() goryu.Middleware {
	return func(next goryu.HandlerFunc) goryu.HandlerFunc {
		return func(c *goryu.Context) {
			start := time.Now()
			lrw := newLoggingResponseWriter(c.Writer)
			c.Writer = lrw
			next(c)
			duration := time.Since(start)
			log.Printf(
				"method=%s path=\"%s\" proto=%s status=%d duration=%v size=%d remote_addr=\"%s\" user_agent=\"%s\"",
				c.Request.Method,
				c.Request.URL.Path,
				c.Request.Proto,
				lrw.statusCode,
				duration,
				lrw.size,
				c.Request.RemoteAddr,
				c.Request.UserAgent(),
			)
		}
	}
}
