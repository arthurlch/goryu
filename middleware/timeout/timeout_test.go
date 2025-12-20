package timeout_test
import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
	"github.com/arthurlch/goryu/middleware/timeout"
)
func newTestContext(req *http.Request) (*context.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}
func TestTimeoutMiddleware(t *testing.T) {
	t.Run("FastHandler", func(t *testing.T) {
		config := timeout.Config{
			Timeout: 100 * time.Millisecond,
		}
		middleware := timeout.New(config)
		handler := func(c *context.Context) {
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
		handler := func(c *context.Context) {
			time.Sleep(50 * time.Millisecond) 
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
			TimeoutHandler: func(c *context.Context) {
				c.Text(http.StatusGatewayTimeout, "Custom timeout message")
			},
		}
		middleware := timeout.New(config)
		handler := func(c *context.Context) {
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
			BaseConfig: base.BaseConfig{
				Skip: func(c *context.Context) bool {
					return c.Request.URL.Path == "/skip"
				},
			},
		}
		middleware := timeout.New(config)
		handler := func(c *context.Context) {
			time.Sleep(50 * time.Millisecond) 
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
		middleware := timeout.New(timeout.Config{})
		handler := func(c *context.Context) {
			c.Text(http.StatusOK, "OK")
		}
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})
	t.Run("HandlerPanic", func(t *testing.T) {
		config := timeout.Config{
			Timeout: 100 * time.Millisecond,
		}
		middleware := timeout.New(config)
		handler := func(c *context.Context) {
			panic("handler panic test")
		}
		req := httptest.NewRequest("GET", "/", nil)
		ctx, _ := newTestContext(req)
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic to propagate from handler")
			}
		}()
		middleware(handler)(ctx)
	})
	t.Run("TimeoutHandlerPanic", func(t *testing.T) {
		config := timeout.Config{
			Timeout: 10 * time.Millisecond,
			TimeoutHandler: func(c *context.Context) {
				panic("timeout handler panic")
			},
		}
		middleware := timeout.New(config)
		handler := func(c *context.Context) {
			time.Sleep(50 * time.Millisecond)
			c.Text(http.StatusOK, "Should not see this")
		}
		req := httptest.NewRequest("GET", "/", nil)
		ctx, _ := newTestContext(req)
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic from timeout handler")
			}
		}()
		middleware(handler)(ctx)
	})
	t.Run("ResponseAlreadyWritten", func(t *testing.T) {
		config := timeout.Config{
			Timeout: 50 * time.Millisecond,
		}
		middleware := timeout.New(config)
		handler := func(c *context.Context) {
			c.Text(http.StatusOK, "Quick response")
			time.Sleep(100 * time.Millisecond)
		}
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		body := rr.Body.String()
		if !strings.Contains(body, "Quick response") {
			t.Errorf("Expected body to contain 'Quick response', got %s", body)
		}
	})
	t.Run("ZeroTimeout", func(t *testing.T) {
		config := timeout.Config{
			Timeout: 0, 
		}
		middleware := timeout.New(config)
		handler := func(c *context.Context) {
			time.Sleep(50 * time.Millisecond)
			c.Text(http.StatusOK, "No timeout")
		}
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 for zero timeout, got %d", rr.Code)
		}
		if rr.Body.String() != "No timeout" {
			t.Errorf("Expected 'No timeout', got %s", rr.Body.String())
		}
	})
	t.Run("NegativeTimeout", func(t *testing.T) {
		config := timeout.Config{
			Timeout: -10 * time.Millisecond, 
		}
		middleware := timeout.New(config)
		handler := func(c *context.Context) {
			time.Sleep(20 * time.Millisecond)
			c.Text(http.StatusOK, "Negative timeout ignored")
		}
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 for negative timeout, got %d", rr.Code)
		}
		if rr.Body.String() != "Negative timeout ignored" {
			t.Errorf("Expected 'Negative timeout ignored', got %s", rr.Body.String())
		}
	})
}
