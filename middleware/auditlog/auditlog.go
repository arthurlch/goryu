// Package auditlog records an access-audit trail (who, what, when, outcome) as
// Events written to a Sink.
package auditlog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
)

// userIDKey mirrors auth.UserIDKey / the session user key without importing them.
const userIDKey = "user_id"

type Event struct {
	Time      time.Time `json:"time"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Status    int       `json:"status"`
	LatencyMS int64     `json:"latency_ms"`
	UserID    string    `json:"user_id,omitempty"`
	IP        string    `json:"ip"`
}

// Sink receives audit events; implementations must be safe for concurrent use.
type Sink interface {
	Write(Event) error
}

type Config struct {
	base.BaseConfig
	Sink       Sink
	UserIDFunc func(c *context.Context) string
}

func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}

type loggerSink struct{ logger base.Logger }

func (s loggerSink) Write(e Event) error {
	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	s.logger.Printf("%s", line)
	return nil
}

type captureWriter struct {
	http.ResponseWriter
	status int
}

func (w *captureWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
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

// New builds the audit middleware; without a Sink it logs JSON via the logger.
func New(config ...Config) func(next context.HandlerFunc) context.HandlerFunc {
	cfg := Config{}
	if len(config) > 0 {
		cfg = config[0]
	}
	if cfg.Sink == nil {
		logger := cfg.Logger
		if logger == nil {
			logger = base.DefaultLogger("Audit")
		}
		cfg.Sink = loggerSink{logger: logger}
	}
	if cfg.UserIDFunc == nil {
		cfg.UserIDFunc = defaultUserID
	}

	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if cfg.Skip != nil && cfg.Skip(c) {
				next(c)
				return
			}
			start := time.Now()
			cw := &captureWriter{ResponseWriter: c.Writer, status: http.StatusOK}
			c.Writer = cw

			next(c)

			c.Writer = cw.ResponseWriter
			event := Event{
				Time:      start,
				Method:    c.Request.Method,
				Path:      c.Request.URL.Path,
				Status:    cw.status,
				LatencyMS: time.Since(start).Milliseconds(),
				UserID:    cfg.UserIDFunc(c),
				IP:        c.RemoteIP(),
			}
			if err := cfg.Sink.Write(event); err != nil {
				logger := cfg.Logger
				if logger == nil {
					logger = base.DefaultLogger("Audit")
				}
				logger.Printf("audit sink write failed: %v", err)
			}
		}
	}
}

func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New()
}

func defaultUserID(c *context.Context) string {
	if v, ok := c.Get(userIDKey); ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}
