package secure_test

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/secure"
)

func newTestContext(req *http.Request) (*goryu.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}

func TestSecureMiddleware(t *testing.T) {
	handler := func(c *goryu.Context) {
		_ = c.Text(http.StatusOK, "OK")
	}

	t.Run("DefaultHeaders", func(t *testing.T) {
		middleware := secure.New()
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

		expectedHSTS := "max-age=31536000; includeSubdomains"
		if header := rr.Header().Get("Strict-Transport-Security"); header != expectedHSTS {
			t.Errorf("Expected HSTS header to be '%s', got '%s'", expectedHSTS, header)
		}
	})
}
