package basicauth_test

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/basicauth"
)

func newTestContext(req *http.Request) (*goryu.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}

func TestBasicAuthMiddleware(t *testing.T) {
	handler := func(c *goryu.Context) {
		_ = c.Text(http.StatusOK, "OK")
	}

	config := basicauth.Config{
		Users: map[string]string{
			"admin": "password123",
		},
	}
	middleware := basicauth.New(config)

	t.Run("ValidCredentials", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		auth := base64.StdEncoding.EncodeToString([]byte("admin:password123"))
		req.Header.Set("Authorization", "Basic "+auth)
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 for valid credentials, got %d", rr.Code)
		}
	})

	t.Run("InvalidPassword", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		auth := base64.StdEncoding.EncodeToString([]byte("admin:wrongpassword"))
		req.Header.Set("Authorization", "Basic "+auth)
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for invalid password, got %d", rr.Code)
		}
		if rr.Header().Get("WWW-Authenticate") == "" {
			t.Error("Expected WWW-Authenticate header to be set for failed auth")
		}
	})

	t.Run("NoAuthHeader", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for missing auth header, got %d", rr.Code)
		}
	})

	t.Run("WithValidator", func(t *testing.T) {
		validatorConfig := basicauth.Config{
			Validator: func(username, password string) bool {
				return username == "validator" && password == "valid"
			},
		}
		validatorMiddleware := basicauth.New(validatorConfig)

		req := httptest.NewRequest("GET", "/", nil)
		auth := base64.StdEncoding.EncodeToString([]byte("validator:valid"))
		req.Header.Set("Authorization", "Basic "+auth)
		ctx, rr := newTestContext(req)
		validatorMiddleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 for valid validator credentials, got %d", rr.Code)
		}

		req = httptest.NewRequest("GET", "/", nil)
		auth = base64.StdEncoding.EncodeToString([]byte("validator:invalid"))
		req.Header.Set("Authorization", "Basic "+auth)
		ctx, rr = newTestContext(req)
		validatorMiddleware(handler)(ctx)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for invalid validator credentials, got %d", rr.Code)
		}
	})
}
