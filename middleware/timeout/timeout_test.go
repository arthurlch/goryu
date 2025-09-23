package timeout_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/timeout"
)

func newTestContext(req *http.Request) (*goryu.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}

func TestTimeoutMiddleware(t *testing.T) {
	t.Run("FastHandler", func(t *testing.T) {
		config := timeout.Config{
			Timeout: 100 * time.Millisecond,
		}
		middleware := timeout.New(config)

		handler := func(c *goryu.Context) {
			c.Text(http.StatusOK, "Fast response")
		}

		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		if rr.Body.String() != "Fast response" {
			t.Errorf("Expected 'Fast response', got %s", rr.Body.String())
		}
	})

	t.Run("SlowHandler", func(t *testing.T) {
		config := timeout.Config{
			Timeout: 10 * time.Millisecond,
		}
		middleware := timeout.New(config)

		handler := func(c *goryu.Context) {
			time.Sleep(50 * time.Millisecond) // Sleep longer than timeout
			c.Text(http.StatusOK, "Slow response")
		}

		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Code != http.StatusRequestTimeout {
			t.Errorf("Expected status 408, got %d", rr.Code)
		}

		if rr.Body.String() != "Request Timeout" {
			t.Errorf("Expected 'Request Timeout', got %s", rr.Body.String())
		}
	})

	t.Run("CustomTimeoutHandler", func(t *testing.T) {
		config := timeout.Config{
			Timeout: 10 * time.Millisecond,
			TimeoutHandler: func(c *goryu.Context) {
				c.Text(http.StatusGatewayTimeout, "Custom timeout message")
			},
		}
		middleware := timeout.New(config)

		handler := func(c *goryu.Context) {
			time.Sleep(50 * time.Millisecond)
			c.Text(http.StatusOK, "Should not see this")
		}

		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Code != http.StatusGatewayTimeout {
			t.Errorf("Expected status 504, got %d", rr.Code)
		}

		if rr.Body.String() != "Custom timeout message" {
			t.Errorf("Expected 'Custom timeout message', got %s", rr.Body.String())
		}
	})

	t.Run("SkipMiddleware", func(t *testing.T) {
		config := timeout.Config{
			Timeout: 10 * time.Millisecond,
			Skip: func(c *goryu.Context) bool {
				return c.Request.URL.Path == "/skip"
			},
		}
		middleware := timeout.New(config)

		handler := func(c *goryu.Context) {
			time.Sleep(50 * time.Millisecond) // Sleep longer than timeout
			c.Text(http.StatusOK, "Not timed out")
		}

		req := httptest.NewRequest("GET", "/skip", nil)
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		if rr.Body.String() != "Not timed out" {
			t.Errorf("Expected 'Not timed out', got %s", rr.Body.String())
		}
	})

	t.Run("DefaultConfig", func(t *testing.T) {
		middleware := timeout.New()

		handler := func(c *goryu.Context) {
			c.Text(http.StatusOK, "OK")
		}

		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})
}
