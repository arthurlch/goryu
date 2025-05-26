package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/arthurlch/goryu/pkg/context"
)

type loggingResponseWriter struct {
	http.ResponseWriter // oringinal
	statusCode          int
	size                int
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

func Logger() Middleware {
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			start := time.Now()

			lrw := newLoggingResponseWriter(c.Writer)

			// WARUNINGU: Replace the original ResponseWriter in the context with our wrapper.
			// Need to  ensures that subsequent calls to c.Writer.WriteHeader() or c.Writer.Write()
			// (e.g., from c.JSON(), c.Text(), render.Render()) go through the wrapper
			c.Writer = lrw

			next(c)

			duration := time.Since(start)

			// structured format is the best 
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