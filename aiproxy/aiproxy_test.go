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

func TestProxyStripsClientSecrets(t *testing.T) {
	var gotAuth, gotCookie, gotAPIKey, gotContentType string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotCookie = r.Header.Get("Cookie")
		gotAPIKey = r.Header.Get("X-Api-Key")
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(200)
	}))
	defer upstream.Close()

	px := aiproxy.New(aiproxy.Config{BaseURL: upstream.URL, APIKey: "provider-key"})

	req := httptest.NewRequest("POST", "/v1/chat", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer client-secret")
	req.Header.Set("Cookie", "session=abc")
	req.Header.Set("X-Api-Key", "client-key")
	req.Header.Set("Content-Type", "application/json")
	c := context.NewContext(httptest.NewRecorder(), req)

	if err := px.Forward(c, "/v1/chat"); err != nil {
		t.Fatalf("Forward error: %v", err)
	}

	if gotAuth != "Bearer provider-key" {
		t.Fatalf("upstream auth should be the proxy credential, got %q", gotAuth)
	}
	if gotCookie != "" {
		t.Fatalf("client cookie leaked upstream: %q", gotCookie)
	}
	if gotAPIKey != "" {
		t.Fatalf("client X-Api-Key leaked upstream: %q", gotAPIKey)
	}
	if gotContentType != "application/json" {
		t.Fatalf("content-type should pass through, got %q", gotContentType)
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
