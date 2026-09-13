package goryuctx_test

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	goryuctx "github.com/arthurlch/goryu/goryuctx"
)

func TestSSEStreamsEvents(t *testing.T) {
	req := httptest.NewRequest("GET", "/events", nil)
	rr := httptest.NewRecorder()
	c := goryuctx.NewContext(rr, req)

	err := c.SSE(func(send func(goryuctx.SSEvent) error) error {
		if err := send(goryuctx.SSEvent{Event: "tick", Data: "1"}); err != nil {
			return err
		}
		return send(goryuctx.SSEvent{Data: "line1\nline2"})
	})
	if err != nil {
		t.Fatal(err)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type = %q", ct)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "event: tick\ndata: 1\n\n") {
		t.Fatalf("first event missing: %q", body)
	}
	if !strings.Contains(body, "data: line1\ndata: line2\n\n") {
		t.Fatalf("multiline data wrong: %q", body)
	}
}

func TestStreamWritesChunks(t *testing.T) {
	req := httptest.NewRequest("GET", "/dl", nil)
	rr := httptest.NewRecorder()
	c := goryuctx.NewContext(rr, req)

	err := c.Stream(func(w io.Writer) error {
		_, err := io.WriteString(w, "chunk-1")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if rr.Body.String() != "chunk-1" {
		t.Fatalf("stream body = %q", rr.Body.String())
	}
}
