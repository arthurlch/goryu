package promptcache_test

import (
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/promptcache"
)

func postCtx(body string) (*context.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return context.NewContext(w, req), w
}

func TestPromptCacheHitOnSameBody(t *testing.T) {
	var calls atomic.Int32
	mw := promptcache.New()
	handler := mw(func(c *context.Context) {
		calls.Add(1)
		c.JSON(200, map[string]string{"answer": "42"})
	})

	c1, w1 := postCtx(`{"q":"life"}`)
	handler(c1)
	if w1.Header().Get("X-Cache") != "MISS" {
		t.Fatalf("first call should be MISS, got %q", w1.Header().Get("X-Cache"))
	}

	c2, w2 := postCtx(`{"q":"life"}`)
	handler(c2)
	if w2.Header().Get("X-Cache") != "HIT" {
		t.Fatalf("second call should be HIT, got %q", w2.Header().Get("X-Cache"))
	}
	if calls.Load() != 1 {
		t.Fatalf("handler should run once, ran %d times", calls.Load())
	}
	if !strings.Contains(w2.Body.String(), "42") {
		t.Fatalf("cached body not replayed: %q", w2.Body.String())
	}
}

func TestPromptCacheMissOnDifferentBody(t *testing.T) {
	var calls atomic.Int32
	mw := promptcache.New()
	handler := mw(func(c *context.Context) {
		calls.Add(1)
		c.JSON(200, map[string]string{"answer": "x"})
	})

	c1, _ := postCtx(`{"q":"a"}`)
	handler(c1)
	c2, _ := postCtx(`{"q":"b"}`)
	handler(c2)
	if calls.Load() != 2 {
		t.Fatalf("different bodies should both hit handler, ran %d", calls.Load())
	}
}

func TestPromptCacheSkipsGET(t *testing.T) {
	var calls atomic.Int32
	mw := promptcache.New()
	handler := mw(func(c *context.Context) {
		calls.Add(1)
		c.JSON(200, map[string]string{"a": "b"})
	})

	for range 2 {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/chat", nil)
		handler(context.NewContext(w, req))
	}
	if calls.Load() != 2 {
		t.Fatalf("GET should not be cached, handler ran %d", calls.Load())
	}
}

func TestPromptCacheDoesNotCacheErrors(t *testing.T) {
	var calls atomic.Int32
	mw := promptcache.New()
	handler := mw(func(c *context.Context) {
		calls.Add(1)
		c.JSON(500, map[string]string{"error": "boom"})
	})

	c1, _ := postCtx(`{"q":"z"}`)
	handler(c1)
	c2, _ := postCtx(`{"q":"z"}`)
	handler(c2)
	if calls.Load() != 2 {
		t.Fatalf("5xx must not be cached, handler ran %d", calls.Load())
	}
}
