package goryu_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arthurlch/goryu"
	goryuctx "github.com/arthurlch/goryu/goryuctx"
)

type chunk struct {
	Index int    `json:"index"`
	Text  string `json:"text"`
}

func TestStreamJSONNDJSON(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stream", nil)
	c := goryuctx.NewContext(w, req)

	err := goryu.StreamJSON(c, func(emit func(chunk) error) error {
		for i := range 3 {
			if err := emit(chunk{Index: i, Text: "x"}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("StreamJSON error: %v", err)
	}

	if ct := w.Header().Get("Content-Type"); ct != "application/x-ndjson" {
		t.Fatalf("content-type wrong: %q", ct)
	}
	lines := strings.Split(strings.TrimSpace(w.Body.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 NDJSON lines, got %d: %q", len(lines), w.Body.String())
	}
	if !strings.Contains(lines[0], `"index":0`) {
		t.Fatalf("first line wrong: %q", lines[0])
	}
}

func TestSSEJSON(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sse", nil)
	c := goryuctx.NewContext(w, req)

	err := goryu.SSEJSON(c, "token", func(send func(chunk) error) error {
		return send(chunk{Index: 1, Text: "hi"})
	})
	if err != nil {
		t.Fatalf("SSEJSON error: %v", err)
	}

	body := w.Body.String()
	if !strings.Contains(body, "event: token") {
		t.Fatalf("missing event line: %q", body)
	}
	if !strings.Contains(body, `data: {"index":1,"text":"hi"}`) {
		t.Fatalf("missing json data line: %q", body)
	}
}
