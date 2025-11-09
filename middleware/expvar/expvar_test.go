package expvar_test
import (
	"expvar"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"github.com/arthurlch/goryu/context"
	expvarMW "github.com/arthurlch/goryu/middleware/expvar"
)
func newTestContext(req *http.Request) (*context.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}
func TestExpvarMiddleware(t *testing.T) {
	testCounter := expvar.NewInt("test_requests")
	testCounter.Set(42)
	t.Run("DefaultPath", func(t *testing.T) {
		middleware := expvarMW.New(expvarMW.Config{})
		handler := func(c *context.Context) {
			c.Text(http.StatusNotFound, "Not found")
		}
		req := httptest.NewRequest("GET", "/debug/vars", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		body, _ := io.ReadAll(rr.Body)
		if !strings.Contains(string(body), `"test_requests": 42`) {
			t.Errorf("Expvar body does not contain expected metric. Body: %s", string(body))
		}
	})
	t.Run("CustomPath", func(t *testing.T) {
		config := expvarMW.Config{Path: "/debug/metrics"}
		middleware := expvarMW.New(config)
		handler := func(c *context.Context) {
			c.Text(http.StatusNotFound, "Not found")
		}
		req := httptest.NewRequest("GET", "/debug/metrics", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		body, _ := io.ReadAll(rr.Body)
		if !strings.Contains(string(body), `"test_requests": 42`) {
			t.Errorf("Expvar body does not contain expected metric. Body: %s", string(body))
		}
	})
	t.Run("NonMatchingPath", func(t *testing.T) {
		config := expvarMW.Config{Path: "/debug/vars"}
		middleware := expvarMW.New(config)
		handler := func(c *context.Context) {
			c.Text(http.StatusNotFound, "Not found")
		}
		req := httptest.NewRequest("GET", "/some/other/path", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rr.Code)
		}
		if rr.Body.String() != "Not found" {
			t.Errorf("Expected 'Not found', got %s", rr.Body.String())
		}
	})
	t.Run("WithPathHelper", func(t *testing.T) {
		middleware := expvarMW.WithPath("/custom-debug")
		handler := func(c *context.Context) {
			c.Text(http.StatusNotFound, "Not found")
		}
		req := httptest.NewRequest("GET", "/custom-debug", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		body, _ := io.ReadAll(rr.Body)
		if !strings.HasPrefix(string(body), "{") {
			t.Errorf("Expected JSON response, got: %s", string(body))
		}
	})
	t.Run("DefaultHelper", func(t *testing.T) {
		middleware := expvarMW.Default()
		handler := func(c *context.Context) {
			c.Text(http.StatusNotFound, "Not found")
		}
		req := httptest.NewRequest("GET", "/debug/vars", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})
}