package cache_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/arthurlch/goryu"
	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/cache"
)

func TestCacheDoesNotServePrivateResponses(t *testing.T) {
	var calls int64
	handler := func(c *goryu.Ctx) {
		n := atomic.AddInt64(&calls, 1)
		_ = c.Text(http.StatusOK, fmt.Sprintf("body-%d", n))
	}
	mw := cache.New(cache.Config{})

	run := func(cookie string) string {
		req := httptest.NewRequest("GET", "/acct", nil)
		if cookie != "" {
			req.Header.Set("Cookie", cookie)
		}
		rr := httptest.NewRecorder()
		mw(handler)(context.NewContext(rr, req))
		return rr.Body.String()
	}

	authed := run("session=secret-user-a")
	anon := run("")
	if anon == authed {
		t.Fatalf("anonymous request received the authenticated user's cached body %q", authed)
	}

	// A response carrying Set-Cookie must never be cached either.
	var setCookieCalls int64
	scHandler := func(c *goryu.Ctx) {
		n := atomic.AddInt64(&setCookieCalls, 1)
		c.Writer.Header().Set("Set-Cookie", "sid=abc")
		_ = c.Text(http.StatusOK, fmt.Sprintf("sc-%d", n))
	}
	first := func() string {
		req := httptest.NewRequest("GET", "/sc", nil)
		rr := httptest.NewRecorder()
		mw(scHandler)(context.NewContext(rr, req))
		return rr.Body.String()
	}
	a := first()
	b := first()
	if a == b {
		t.Fatalf("response with Set-Cookie was cached and replayed: %q", a)
	}
}
