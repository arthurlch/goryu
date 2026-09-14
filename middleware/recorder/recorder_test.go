package recorder_test

import (
	"bufio"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/recorder"
)

type memSink struct {
	mu      sync.Mutex
	records []recorder.Record
}

func (s *memSink) Write(r recorder.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, r)
	return nil
}

func TestRecorderCapturesExchange(t *testing.T) {
	sink := &memSink{}
	mw := recorder.New(recorder.Config{
		Sink:    sink,
		KeyFunc: func(c *context.Context) string { return c.GetHeader("X-API-Key") },
		MetaFunc: func(c *context.Context) map[string]any {
			return map[string]any{"model": "test-model"}
		},
	})
	handler := mw(func(c *context.Context) {
		c.JSON(200, map[string]string{"answer": "42"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/chat", strings.NewReader(`{"q":"life"}`))
	req.Header.Set("X-API-Key", "k-123")
	handler(context.NewContext(w, req))

	if len(sink.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(sink.records))
	}
	rec := sink.records[0]
	if rec.Method != "POST" || rec.Path != "/chat" || rec.Status != 200 {
		t.Fatalf("record fields wrong: %+v", rec)
	}
	if rec.Key != "k-123" {
		t.Fatalf("key not captured: %q", rec.Key)
	}
	if !strings.Contains(string(rec.RequestBody), "life") {
		t.Fatalf("request body not captured: %s", rec.RequestBody)
	}
	if !strings.Contains(string(rec.ResponseBody), "42") {
		t.Fatalf("response body not captured: %s", rec.ResponseBody)
	}
	if rec.Meta["model"] != "test-model" {
		t.Fatalf("meta not captured: %v", rec.Meta)
	}
}

func TestRecorderRestoresRequestBody(t *testing.T) {
	sink := &memSink{}
	mw := recorder.New(recorder.Config{Sink: sink})
	var seen string
	handler := mw(func(c *context.Context) {
		var body map[string]string
		_ = c.BodyParser(&body)
		seen = body["q"]
		c.Text(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/chat", strings.NewReader(`{"q":"restored"}`))
	req.Header.Set("Content-Type", "application/json")
	handler(context.NewContext(w, req))

	if seen != "restored" {
		t.Fatalf("handler could not read body after capture: %q", seen)
	}
}

func TestFileSinkWritesNDJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evals.jsonl")
	sink, err := recorder.NewFileSink(path)
	if err != nil {
		t.Fatalf("NewFileSink: %v", err)
	}

	mw := recorder.New(recorder.Config{Sink: sink})
	handler := mw(func(c *context.Context) { c.JSON(200, map[string]int{"n": 1}) })

	for range 2 {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/x", strings.NewReader(`{}`))
		handler(context.NewContext(w, req))
	}
	if err := sink.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	f, _ := os.Open(path)
	defer f.Close()
	lines := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var rec recorder.Record
		if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
			t.Fatalf("line not valid JSON: %v", err)
		}
		lines++
	}
	if lines != 2 {
		t.Fatalf("expected 2 NDJSON lines, got %d", lines)
	}
}
