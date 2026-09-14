package promptcache_test

import (
	"io"
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

func TestPromptCacheDoesNotCacheOversizeResponse(t *testing.T) {
	var calls atomic.Int32
	mw := promptcache.New(promptcache.Config{MaxBodyBytes: 16})
	handler := mw(func(c *context.Context) {
		calls.Add(1)
		_ = c.Text(200, strings.Repeat("x", 100)) // exceeds the 16-byte cap
	})

	c1, w1 := postCtx(`{"q":"a"}`)
	handler(c1)
	if w1.Body.Len() != 100 {
		t.Fatalf("client must receive full body; got %d", w1.Body.Len())
	}
	c2, _ := postCtx(`{"q":"a"}`)
	handler(c2)
	if calls.Load() != 2 {
		t.Fatalf("oversize response must not be cached; handler ran %d", calls.Load())
	}
}

func TestPromptCachePreservesLargeRequestBody(t *testing.T) {
	mw := promptcache.New(promptcache.Config{MaxBodyBytes: 8})
	var seen int
	handler := mw(func(c *context.Context) {
		b, _ := io.ReadAll(c.Request.Body)
		seen = len(b)
		_ = c.Text(200, "ok")
	})

	body := strings.Repeat("z", 50) // larger than the 8-byte cap
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/chat", strings.NewReader(body))
	handler(context.NewContext(w, req))

	if seen != 50 {
		t.Fatalf("handler must see the full body; got %d want 50", seen)
	}
}

func TestPromptCacheDoesNotLeakAcrossAuthenticatedUsers(t *testing.T) {
	var calls atomic.Int32
	mw := promptcache.New()
	handler := mw(func(c *context.Context) {
		calls.Add(1)
		_ = c.JSON(200, map[string]string{"user": c.GetHeader("Authorization")})
	})

	req1 := httptest.NewRequest("POST", "/chat", strings.NewReader(`{"q":"hi"}`))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer user-A")
	handler(context.NewContext(httptest.NewRecorder(), req1))

	req2 := httptest.NewRequest("POST", "/chat", strings.NewReader(`{"q":"hi"}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer user-B")
	w2 := httptest.NewRecorder()
	handler(context.NewContext(w2, req2))

	if calls.Load() != 2 {
		t.Fatalf("authenticated requests must not be cached with the default key; handler ran %d", calls.Load())
	}
	if strings.Contains(w2.Body.String(), "user-A") {
		t.Fatalf("user B received user A's cached response: %s", w2.Body.String())
	}
	if w2.Header().Get("X-Cache") == "HIT" {
		t.Fatal("authenticated request must not be served from cache")
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
