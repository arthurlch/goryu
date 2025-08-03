package cors_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/cors"
)

func newTestContext(req *http.Request) (*goryu.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}

func TestCorsMiddleware(t *testing.T) {
	handler := func(c *goryu.Context) {
		_ = c.Text(http.StatusOK, "OK")
	}

	t.Run("AllowAllOrigins", func(t *testing.T) {
		middleware := cors.New(cors.Config{
			AllowOrigins: []string{"*"},
		})
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Origin", "https://example.com")
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
			t.Errorf("Expected ACAO header to be 'https://example.com', got '%s'", rr.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("SpecificOriginAllowed", func(t *testing.T) {
		middleware := cors.New(cors.Config{
			AllowOrigins: []string{"https://allowed.com"},
		})
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Origin", "https://allowed.com")
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Header().Get("Access-Control-Allow-Origin") != "https://allowed.com" {
			t.Errorf("Expected ACAO header for allowed origin, got '%s'", rr.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("SpecificOriginDisallowed", func(t *testing.T) {
		middleware := cors.New(cors.Config{
			AllowOrigins: []string{"https://allowed.com"},
		})
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Origin", "https://disallowed.com")
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("Expected empty ACAO header for disallowed origin, got '%s'", rr.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("PreflightRequest", func(t *testing.T) {
		middleware := cors.New(cors.Config{
			AllowOrigins: []string{"https://example.com"},
			AllowMethods: []string{"GET", "POST"},
			AllowHeaders: []string{"Content-Type"},
			MaxAge:       3600,
		})

		req := httptest.NewRequest("OPTIONS", "/", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type")

		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)

		if rr.Code != http.StatusNoContent {
			t.Errorf("Expected status 204 for preflight, got %d", rr.Code)
		}
		if rr.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
			t.Error("Preflight ACAO header mismatch")
		}
		if rr.Header().Get("Access-Control-Allow-Methods") != "GET,POST" {
			t.Error("Preflight ACAM header mismatch")
		}
		if rr.Header().Get("Access-Control-Allow-Headers") != "Content-Type" {
			t.Error("Preflight ACAH header mismatch")
		}
		if rr.Header().Get("Access-Control-Max-Age") != "3600" {
			t.Errorf("Expected Max-Age to be 3600, got %s", rr.Header().Get("Access-Control-Max-Age"))
		}
	})

	t.Run("NoOriginHeader", func(t *testing.T) {
		middleware := cors.New()
		req := httptest.NewRequest("GET", "/", nil) // No Origin header
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("Expected no ACAO header when no origin is sent")
		}
		if rr.Code != http.StatusOK {
			t.Errorf("Expected handler to be called, got status %d", rr.Code)
		}
	})
}
