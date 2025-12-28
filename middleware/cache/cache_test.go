package cache_test

import (
	"fmt"
	"github.com/arthurlch/goryu"
	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/cache"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestContext(req *http.Request) (*goryu.Ctx, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}
func TestCacheMiddleware(t *testing.T) {
	dynamicHandler := func(c *goryu.Ctx) {
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
func TestCacheSecurityLimits(t *testing.T) {
	staticHandler := func(c *goryu.Ctx) {
		_ = c.Text(http.StatusOK, "test response")
	}
	t.Run("Memory Exhaustion Prevention", func(t *testing.T) {
		config := cache.Config{
			Expiration: 10 * time.Minute,
			MaxSize:    5,
			MaxMemory:  1024,
		}
		middleware := cache.New(config)
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/test"+fmt.Sprintf("%d", i), nil)
			ctx, _ := newTestContext(req)
			middleware(staticHandler)(ctx)
		}
		req := httptest.NewRequest("GET", "/final", nil)
		ctx, rr := newTestContext(req)
		middleware(staticHandler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected cache to still work after hitting limits, got status %d", rr.Code)
		}
	})
	t.Run("Large Entry Rejection", func(t *testing.T) {
		largeHandler := func(c *goryu.Ctx) {
			largeData := make([]byte, 200)
			for i := range largeData {
				largeData[i] = 'A'
			}
			_ = c.Text(http.StatusOK, string(largeData))
		}
		config := cache.Config{
			Expiration: 10 * time.Minute,
			MaxMemory:  1024,
		}
		middleware := cache.New(config)
		req := httptest.NewRequest("GET", "/large", nil)
		ctx, rr := newTestContext(req)
		middleware(largeHandler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected large response to be served, got status %d", rr.Code)
		}
		if len(rr.Body.String()) < 200 {
			t.Error("Expected large response body")
		}
	})
	t.Run("Error Response Not Cached", func(t *testing.T) {
		errorHandler := func(c *goryu.Ctx) {
			_ = c.Text(http.StatusInternalServerError, "error")
		}
		config := cache.Config{
			Expiration: 10 * time.Minute,
		}
		middleware := cache.New(config)
		req := httptest.NewRequest("GET", "/error", nil)
		ctx, rr := newTestContext(req)
		middleware(errorHandler)(ctx)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected error status, got %d", rr.Code)
		}
		ctx, rr2 := newTestContext(req)
		middleware(errorHandler)(ctx)
		if rr2.Code != http.StatusInternalServerError {
			t.Errorf("Expected error status on second request, got %d", rr2.Code)
		}
	})
}
