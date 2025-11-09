package limiter_test
import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/limiter"
)
func newTestContext(req *http.Request) (*context.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}
func TestLimiterMiddleware(t *testing.T) {
	handler := func(c *context.Context) {
		c.Text(http.StatusOK, "OK")
	}
	t.Run("AllowsRequestsWithinLimit", func(t *testing.T) {
		config := limiter.Config{
			Max:        2,
			Expiration: 1 * time.Second,
		}
		middleware := limiter.New(config)
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 on first request, got %d", rr.Code)
		}
		ctx, rr = newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 on second request, got %d", rr.Code)
		}
	})
	t.Run("BlocksRequestsOverLimit", func(t *testing.T) {
		config := limiter.Config{
			Max:        1,
			Expiration: 1 * time.Second,
		}
		middleware := limiter.New(config)
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 on first request, got %d", rr.Code)
		}
		ctx, rr = newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusTooManyRequests {
			t.Errorf("Expected status 429 on blocked request, got %d", rr.Code)
		}
	})
	t.Run("ResetsAfterExpiration", func(t *testing.T) {
		config := limiter.Config{
			Max:        1,
			Expiration: 100 * time.Millisecond,
		}
		middleware := limiter.New(config)
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 on first request, got %d", rr.Code)
		}
		ctx, rr = newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusTooManyRequests {
			t.Errorf("Expected status 429, got %d", rr.Code)
		}
		time.Sleep(150 * time.Millisecond)
		ctx, rr = newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 after expiration, got %d", rr.Code)
		}
	})
}
func TestLimiterSecurityLimits(t *testing.T) {
	handler := func(c *context.Context) {
		c.Text(http.StatusOK, "OK")
	}
	t.Run("Memory Exhaustion Prevention", func(t *testing.T) {
		config := limiter.Config{
			Max:         10,
			Expiration:  10 * time.Minute, 
			MaxClients:  5,                
			CleanupInterval: 10 * time.Minute, 
			KeyGenerator: func(c *context.Context) string {
				return c.Request.Header.Get("X-Client-ID")
			},
		}
		middleware := limiter.New(config)
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("X-Client-ID", fmt.Sprintf("client-%d", i))
			ctx, rr := newTestContext(req)
			middleware(handler)(ctx)
			if rr.Code != http.StatusOK {
				t.Errorf("Expected request %d to succeed, got status %d", i, rr.Code)
			}
		}
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Client-ID", "final-client")
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected final request to succeed, got status %d", rr.Code)
		}
	})
	t.Run("Cleanup Expired Entries", func(t *testing.T) {
		config := limiter.Config{
			Max:         1,
			Expiration:  50 * time.Millisecond,  
			MaxClients:  100,
			CleanupInterval: 60 * time.Millisecond, 
			KeyGenerator: func(c *context.Context) string {
				return c.Request.Header.Get("X-Client-ID")
			},
		}
		middleware := limiter.New(config)
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("X-Client-ID", fmt.Sprintf("temp-client-%d", i))
			ctx, _ := newTestContext(req)
			middleware(handler)(ctx)
		}
		time.Sleep(150 * time.Millisecond)
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Client-ID", "cleanup-trigger")
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected cleanup trigger request to succeed, got status %d", rr.Code)
		}
		req = httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Client-ID", "temp-client-0")
		ctx, rr = newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected cleaned up client to be able to make new requests, got status %d", rr.Code)
		}
	})
	t.Run("Limits Configuration", func(t *testing.T) {
		config := limiter.Config{
			MaxClients: 200000, 
		}
		middleware := limiter.New(config)
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code == http.StatusOK {
			t.Error("Expected configuration error for excessive MaxClients, but request succeeded")
		}
	})
}
