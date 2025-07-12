package limiter_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/limiter"
)

func newTestContext(req *http.Request) (*goryu.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}

func TestLimiterMiddleware(t *testing.T) {
	handler := func(c *goryu.Context) {
		_ = c.Text(http.StatusOK, "OK")
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
