package tlsredirect_test
import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/arthurlch/goryu/context"
	tlsredirect "github.com/arthurlch/goryu/middleware/tls_redirect"
)
func newTestContext(req *http.Request) (*context.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}
func TestTLSRedirectMiddleware(t *testing.T) {
	handler := func(c *context.Context) {
		c.Text(http.StatusOK, "OK")
	}
	t.Run("RedirectsHTTPRequest", func(t *testing.T) {
		middleware := tlsredirect.New(tlsredirect.Config{})
		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusMovedPermanently {
			t.Errorf("Expected status 301, got %d", rr.Code)
		}
		expectedURL := "https://example.com/test"
		if location := rr.Header().Get("Location"); location != expectedURL {
			t.Errorf("Expected redirect to %s, got %s", expectedURL, location)
		}
	})
	t.Run("RedirectsWithForwardedProto", func(t *testing.T) {
		middleware := tlsredirect.New(tlsredirect.Config{})
		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		req.Header.Set("X-Forwarded-Proto", "http")
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusMovedPermanently {
			t.Errorf("Expected status 301, got %d", rr.Code)
		}
		expectedURL := "https://example.com/test"
		if location := rr.Header().Get("Location"); location != expectedURL {
			t.Errorf("Expected redirect to %s, got %s", expectedURL, location)
		}
	})
	t.Run("AllowsHTTPSRequest", func(t *testing.T) {
		middleware := tlsredirect.New(tlsredirect.Config{})
		req := httptest.NewRequest("GET", "https://example.com/test", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		if rr.Body.String() != "OK" {
			t.Errorf("Expected 'OK', got %s", rr.Body.String())
		}
	})
	t.Run("AllowsTLSConnection", func(t *testing.T) {
		middleware := tlsredirect.New(tlsredirect.Config{})
		req := httptest.NewRequest("GET", "https://example.com/test", nil)
		req.TLS = &tls.ConnectionState{}
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})
	t.Run("CustomStatusCode", func(t *testing.T) {
		config := tlsredirect.Config{StatusCode: http.StatusFound}
		middleware := tlsredirect.New(config)
		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusFound {
			t.Errorf("Expected status 302, got %d", rr.Code)
		}
	})
	t.Run("CustomPort", func(t *testing.T) {
		config := tlsredirect.Config{CustomPort: 8443}
		middleware := tlsredirect.New(config)
		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusMovedPermanently {
			t.Errorf("Expected status 301, got %d", rr.Code)
		}
		expectedURL := "https://example.com:8443/test"
		if location := rr.Header().Get("Location"); location != expectedURL {
			t.Errorf("Expected redirect to %s, got %s", expectedURL, location)
		}
	})
	t.Run("ForwardedHost", func(t *testing.T) {
		middleware := tlsredirect.New(tlsredirect.Config{})
		req := httptest.NewRequest("GET", "http://localhost/test", nil)
		req.Header.Set("X-Forwarded-Host", "example.com")
		req.Header.Set("X-Forwarded-Proto", "http")
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusMovedPermanently {
			t.Errorf("Expected status 301, got %d", rr.Code)
		}
		expectedURL := "https://example.com/test"
		if location := rr.Header().Get("Location"); location != expectedURL {
			t.Errorf("Expected redirect to %s, got %s", expectedURL, location)
		}
	})
	t.Run("DefaultHelper", func(t *testing.T) {
		middleware := tlsredirect.Default()
		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusMovedPermanently {
			t.Errorf("Expected status 301, got %d", rr.Code)
		}
	})
	t.Run("WithPortHelper", func(t *testing.T) {
		middleware := tlsredirect.WithPort(9443)
		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		expectedURL := "https://example.com:9443/test"
		if location := rr.Header().Get("Location"); location != expectedURL {
			t.Errorf("Expected redirect to %s, got %s", expectedURL, location)
		}
	})
	t.Run("WithStatusCodeHelper", func(t *testing.T) {
		middleware := tlsredirect.WithStatusCode(http.StatusFound)
		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusFound {
			t.Errorf("Expected status 302, got %d", rr.Code)
		}
	})
	t.Run("CustomRedirectFunc", func(t *testing.T) {
		customCalled := false
		config := tlsredirect.Config{
			RedirectFunc: func(c *context.Context, httpsURL string) {
				customCalled = true
				c.Writer.Header().Set("Custom-Header", "test")
				c.Writer.Header().Set("Location", httpsURL)
				c.Writer.WriteHeader(http.StatusPermanentRedirect)
			},
		}
		middleware := tlsredirect.New(config)
		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if !customCalled {
			t.Error("Expected custom redirect function to be called")
		}
		if rr.Code != http.StatusPermanentRedirect {
			t.Errorf("Expected status 308, got %d", rr.Code)
		}
		if header := rr.Header().Get("Custom-Header"); header != "test" {
			t.Errorf("Expected Custom-Header to be 'test', got %s", header)
		}
	})
}