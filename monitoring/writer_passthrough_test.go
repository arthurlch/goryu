package monitoring_test

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	goryuctx "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/monitoring"
)

type fullWriter struct {
	http.ResponseWriter
	flushed  bool
	hijacked bool
}

func (w *fullWriter) Flush() { w.flushed = true }
func (w *fullWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.hijacked = true
	return nil, nil, nil
}

func TestMonitoringWrapperForwardsFlushAndHijack(t *testing.T) {
	m := monitoring.New(monitoring.Config{Enabled: true})
	defer m.Close()

	base := &fullWriter{ResponseWriter: httptest.NewRecorder()}
	c := goryuctx.NewContext(base, httptest.NewRequest("GET", "/", nil))

	handler := func(c *goryuctx.Context) {
		if f, ok := c.Writer.(http.Flusher); ok {
			f.Flush()
		} else {
			t.Fatal("wrapped writer is not an http.Flusher (SSE would break)")
		}
		if h, ok := c.Writer.(http.Hijacker); ok {
			_, _, _ = h.Hijack()
		} else {
			t.Fatal("wrapped writer is not an http.Hijacker (WebSockets would break)")
		}
	}
	m.Middleware()(handler)(c)

	if !base.flushed || !base.hijacked {
		t.Fatalf("calls not forwarded to underlying writer: flushed=%v hijacked=%v", base.flushed, base.hijacked)
	}
}
