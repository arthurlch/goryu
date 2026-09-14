package aiproxy_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arthurlch/goryu/aiproxy"
	context "github.com/arthurlch/goryu/goryuctx"
)

func TestProxyStreamPassesThrough(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("auth header not forwarded: %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		f, _ := w.(http.Flusher)
		for i := range 3 {
			fmt.Fprintf(w, "data: chunk%d\n\n", i)
			if f != nil {
				f.Flush()
			}
		}
	}))
	defer upstream.Close()

	px := aiproxy.New(aiproxy.Config{BaseURL: upstream.URL, APIKey: "sk-test"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/chat", strings.NewReader(`{"m":"hi"}`))
	c := context.NewContext(w, req)

	if err := px.Stream(c, "/v1/chat"); err != nil {
		t.Fatalf("Stream error: %v", err)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type not copied: %q", ct)
	}
	body := w.Body.String()
	for i := range 3 {
		if !strings.Contains(body, fmt.Sprintf("chunk%d", i)) {
			t.Fatalf("missing chunk%d in %q", i, body)
		}
	}
}

func TestProxyForwardBuffered(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		fmt.Fprintf(w, `{"echo":%s}`, string(b))
	}))
	defer upstream.Close()

	px := aiproxy.New(aiproxy.Config{BaseURL: upstream.URL})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/echo", strings.NewReader(`"hello"`))
	c := context.NewContext(w, req)

	if err := px.Forward(c, "/echo"); err != nil {
		t.Fatalf("Forward error: %v", err)
	}
	if w.Code != 201 {
		t.Fatalf("status not copied: %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"echo":"hello"`) {
		t.Fatalf("body wrong: %q", w.Body.String())
	}
}

func TestProxyEmptyBaseURL(t *testing.T) {
	px := aiproxy.New(aiproxy.Config{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/x", nil)
	c := context.NewContext(w, req)
	if err := px.Forward(c, "/x"); err == nil {
		t.Fatal("expected error for empty BaseURL")
	}
}
