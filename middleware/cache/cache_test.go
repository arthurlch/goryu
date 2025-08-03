package cache_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/cache"
)

func newTestContext(req *http.Request) (*goryu.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}

func TestCacheMiddleware(t *testing.T) {
	dynamicHandler := func(c *goryu.Context) {
		c.Writer.Header().Set("X-Test-Header", "true")
		_ = c.Text(http.StatusOK, time.Now().String())
	}

	config := cache.Config{
		Expiration: 100 * time.Millisecond,
	}
	middleware := cache.New(config)

	t.Run("CachesFirstResponse", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/dynamic", nil)
		ctx, rr := newTestContext(req)

		middleware(dynamicHandler)(ctx)
		firstBody := rr.Body.String()
		firstHeader := rr.Header().Get("X-Test-Header")

		if firstBody == "" {
			t.Fatal("First response body is empty")
		}
		if firstHeader != "true" {
			t.Fatal("First response header not set")
		}

		ctx, rr = newTestContext(req)
		middleware(dynamicHandler)(ctx)
		secondBody := rr.Body.String()
		secondHeader := rr.Header().Get("X-Test-Header")

		if firstBody != secondBody {
			t.Errorf("Expected cached response body to be the same. Got '%s', want '%s'", secondBody, firstBody)
		}
		if secondHeader != "true" {
			t.Errorf("Expected cached header to be present. Got '%s'", secondHeader)
		}
	})

	t.Run("CacheExpires", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/expire", nil)
		ctx, rr := newTestContext(req)

		middleware(dynamicHandler)(ctx)
		firstBody := rr.Body.String()

		time.Sleep(150 * time.Millisecond)

		ctx, rr = newTestContext(req)
		middleware(dynamicHandler)(ctx)
		secondBody := rr.Body.String()

		if firstBody == secondBody {
			t.Error("Expected cache to expire and get a new response, but bodies were the same.")
		}
	})

	t.Run("DoesNotCacheNonGetRequests", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/dynamic", nil)
		ctx, rr := newTestContext(req)

		middleware(dynamicHandler)(ctx)
		firstBody := rr.Body.String()

		ctx, rr = newTestContext(req)
		middleware(dynamicHandler)(ctx)
		secondBody := rr.Body.String()

		if firstBody == secondBody {
			t.Error("POST request was cached, but it should not have been.")
		}
	})
}
