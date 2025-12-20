package secure_test
import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/secure"
)
func newTestContext(req *http.Request) (*context.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}
func TestSecureMiddleware(t *testing.T) {
	handler := func(c *context.Context) {
		c.Text(http.StatusOK, "OK")
	}
	t.Run("DefaultHeaders", func(t *testing.T) {
		middleware := secure.New(secure.Config{})
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if header := rr.Header().Get("X-XSS-Protection"); header != "1; mode=block" {
			t.Errorf("Expected X-XSS-Protection to be '1; mode=block', got '%s'", header)
		}
		if header := rr.Header().Get("X-Content-Type-Options"); header != "nosniff" {
			t.Errorf("Expected X-Content-Type-Options to be 'nosniff', got '%s'", header)
		}
		if header := rr.Header().Get("X-Frame-Options"); header != "SAMEORIGIN" {
			t.Errorf("Expected X-Frame-Options to be 'SAMEORIGIN', got '%s'", header)
		}
		if header := rr.Header().Get("Referrer-Policy"); header != "strict-origin-when-cross-origin" {
			t.Errorf("Expected Referrer-Policy to be 'strict-origin-when-cross-origin', got '%s'", header)
		}
		if header := rr.Header().Get("Strict-Transport-Security"); header != "" {
			t.Errorf("Expected HSTS header to be empty for non-TLS request, got '%s'", header)
		}
	})
	t.Run("CustomHeaders", func(t *testing.T) {
		config := secure.Config{
			XFrameOptions: "DENY",
			HSTSMaxAge:    31536000,
		}
		middleware := secure.New(config)
		req := httptest.NewRequest("GET", "/", nil)
		req.TLS = &tls.ConnectionState{}
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if header := rr.Header().Get("X-Frame-Options"); header != "DENY" {
			t.Errorf("Expected X-Frame-Options to be 'DENY', got '%s'", header)
		}
		expectedHSTS := "max-age=31536000"
		if header := rr.Header().Get("Strict-Transport-Security"); header != expectedHSTS {
			t.Errorf("Expected HSTS header to be '%s', got '%s'", expectedHSTS, header)
		}
	})
	t.Run("HSTSWithSubdomains", func(t *testing.T) {
		config := secure.Config{
			HSTSMaxAge:            31536000,
			HSTSIncludeSubdomains: true,
		}
		middleware := secure.New(config)
		req := httptest.NewRequest("GET", "/", nil)
		req.TLS = &tls.ConnectionState{}
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		expectedHSTS := "max-age=31536000; includeSubDomains"
		if header := rr.Header().Get("Strict-Transport-Security"); header != expectedHSTS {
			t.Errorf("Expected HSTS header to be '%s', got '%s'", expectedHSTS, header)
		}
	})
	t.Run("HSTSWithPreload", func(t *testing.T) {
		config := secure.Config{
			HSTSMaxAge:            31536000,
			HSTSIncludeSubdomains: true,
			HSTSPreload:           true,
		}
		middleware := secure.New(config)
		req := httptest.NewRequest("GET", "/", nil)
		req.TLS = &tls.ConnectionState{}
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		expectedHSTS := "max-age=31536000; includeSubDomains; preload"
		if header := rr.Header().Get("Strict-Transport-Security"); header != expectedHSTS {
			t.Errorf("Expected HSTS header to be '%s', got '%s'", expectedHSTS, header)
		}
	})
	t.Run("ContentSecurityPolicy", func(t *testing.T) {
		config := secure.Config{
			ContentSecurityPolicy: "default-src 'self'",
		}
		middleware := secure.New(config)
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		expectedCSP := "default-src 'self'"
		if header := rr.Header().Get("Content-Security-Policy"); header != expectedCSP {
			t.Errorf("Expected CSP header to be '%s', got '%s'", expectedCSP, header)
		}
	})
	t.Run("PermissionsPolicy", func(t *testing.T) {
		config := secure.Config{
			PermissionsPolicy: "camera=(), microphone=()",
		}
		middleware := secure.New(config)
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		expectedPP := "camera=(), microphone=()"
		if header := rr.Header().Get("Permissions-Policy"); header != expectedPP {
			t.Errorf("Expected Permissions-Policy header to be '%s', got '%s'", expectedPP, header)
		}
	})
	t.Run("DefaultHelper", func(t *testing.T) {
		middleware := secure.Default()
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		if header := rr.Header().Get("X-XSS-Protection"); header == "" {
			t.Error("Expected X-XSS-Protection header to be set")
		}
	})
	t.Run("WithXSSProtectionHelper", func(t *testing.T) {
		middleware := secure.WithXSSProtection("0")
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if header := rr.Header().Get("X-XSS-Protection"); header != "0" {
			t.Errorf("Expected X-XSS-Protection to be '0', got '%s'", header)
		}
	})
	t.Run("WithHSTSHelper", func(t *testing.T) {
		middleware := secure.WithHSTS(86400, true)
		req := httptest.NewRequest("GET", "/", nil)
		req.TLS = &tls.ConnectionState{}
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		expectedHSTS := "max-age=86400; includeSubDomains"
		if header := rr.Header().Get("Strict-Transport-Security"); header != expectedHSTS {
			t.Errorf("Expected HSTS header to be '%s', got '%s'", expectedHSTS, header)
		}
	})
	t.Run("WithCSPHelper", func(t *testing.T) {
		csp := "default-src 'none'; script-src 'self'"
		middleware := secure.WithCSP(csp)
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if header := rr.Header().Get("Content-Security-Policy"); header != csp {
			t.Errorf("Expected CSP header to be '%s', got '%s'", csp, header)
		}
	})
}