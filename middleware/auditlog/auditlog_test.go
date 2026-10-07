package auditlog_test

import (
	"net/http/httptest"
	"sync"
	"testing"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/auditlog"
)

type memSink struct {
	mu     sync.Mutex
	events []auditlog.Event
}

func (s *memSink) Write(e auditlog.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e)
	return nil
}

func TestAuditCapturesRequest(t *testing.T) {
	sink := &memSink{}
	mw := auditlog.New(auditlog.Config{Sink: sink})
	handler := mw(func(c *context.Context) {
		c.Set("user_id", "u-42")
		_ = c.JSON(201, map[string]string{"ok": "yes"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/orders", nil)
	handler(context.NewContext(w, req))

	if len(sink.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(sink.events))
	}
	e := sink.events[0]
	if e.Method != "POST" || e.Path != "/orders" || e.Status != 201 {
		t.Fatalf("event fields wrong: %+v", e)
	}
	if e.UserID != "u-42" {
		t.Fatalf("user id not captured: %q", e.UserID)
	}
}

func TestAuditAnonymizeIP(t *testing.T) {
	sink := &memSink{}
	mw := auditlog.New(auditlog.Config{Sink: sink, AnonymizeIP: true})
	handler := mw(func(c *context.Context) { _ = c.Text(200, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.RemoteAddr = "203.0.113.7:5555"
	handler(context.NewContext(w, req))

	if len(sink.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(sink.events))
	}
	if got := sink.events[0].IP; got != "203.0.113.0" {
		t.Fatalf("IPv4 should be masked to /24; got %q", got)
	}
}

func TestAuditKeepsFullIPByDefault(t *testing.T) {
	sink := &memSink{}
	mw := auditlog.New(auditlog.Config{Sink: sink})
	handler := mw(func(c *context.Context) { _ = c.Text(200, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.RemoteAddr = "203.0.113.7:5555"
	handler(context.NewContext(w, req))

	if got := sink.events[0].IP; got != "203.0.113.7" {
		t.Fatalf("default should keep the full IP; got %q", got)
	}
}

func TestAuditRespectsSkip(t *testing.T) {
	sink := &memSink{}
	cfg := auditlog.Config{Sink: sink}
	cfg.Skip = func(c *context.Context) bool { return c.Request.URL.Path == "/health" }
	mw := auditlog.New(cfg)

	handler := mw(func(c *context.Context) { _ = c.Text(200, "ok") })
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	handler(context.NewContext(w, req))

	if len(sink.events) != 0 {
		t.Fatalf("skipped request must not be audited, got %d", len(sink.events))
	}
}
