package csrf_test

import (
	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/csrf"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestContext(req *http.Request) (*context.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}
func TestCSRFMiddleware(t *testing.T) {
	t.Run("SafeMethodGeneratesToken", func(t *testing.T) {
		middleware := csrf.New(csrf.Config{})
		handler := func(c *context.Context) {
			c.Text(http.StatusOK, "OK")
		}
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		token := rr.Header().Get(csrf.DefaultCSRFTokenHeader)
		if token == "" {
			t.Error("Expected CSRF token header to be set")
		}
		cookies := rr.Result().Cookies()
		found := false
		for _, cookie := range cookies {
			if cookie.Name == csrf.DefaultCSRFTokenCookie && cookie.Value == token {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected CSRF token cookie to be set")
		}
	})
	t.Run("UnsafeMethodRequiresToken", func(t *testing.T) {
		middleware := csrf.New(csrf.Config{})
		handler := func(c *context.Context) {
			c.Text(http.StatusOK, "OK")
		}
		req := httptest.NewRequest("POST", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", rr.Code)
		}
	})
	t.Run("UnsafeMethodWithValidToken", func(t *testing.T) {
		middleware := csrf.New(csrf.Config{})
		handler := func(c *context.Context) {
			c.Text(http.StatusOK, "OK")
		}
		getReq := httptest.NewRequest("GET", "/", nil)
		getCtx, getResp := newTestContext(getReq)
		middleware(handler)(getCtx)
		token := getResp.Header().Get(csrf.DefaultCSRFTokenHeader)
		if token == "" {
			t.Fatal("Failed to get CSRF token from GET request")
		}
		postReq := httptest.NewRequest("POST", "/", nil)
		postReq.Header.Set(csrf.DefaultCSRFTokenHeader, token)
		postReq.AddCookie(&http.Cookie{
			Name:  csrf.DefaultCSRFTokenCookie,
			Value: token,
		})
		postCtx, postResp := newTestContext(postReq)
		middleware(handler)(postCtx)
		if postResp.Code != http.StatusOK {
			t.Errorf("Expected status 200 with valid token, got %d", postResp.Code)
		}
	})
	t.Run("TokenMismatchFails", func(t *testing.T) {
		middleware := csrf.New(csrf.Config{})
		handler := func(c *context.Context) {
			c.Text(http.StatusOK, "OK")
		}
		req := httptest.NewRequest("POST", "/", nil)
		req.Header.Set(csrf.DefaultCSRFTokenHeader, "header-token")
		req.AddCookie(&http.Cookie{
			Name:  csrf.DefaultCSRFTokenCookie,
			Value: "cookie-token",
		})
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusForbidden {
			t.Errorf("Expected status 403 with token mismatch, got %d", rr.Code)
		}
	})
	t.Run("CustomConfiguration", func(t *testing.T) {
		config := csrf.Config{
			TokenHeader: "X-Custom-Token",
			TokenCookie: "custom-token",
			SafeMethods: []string{"GET", "HEAD"},
		}
		middleware := csrf.New(config)
		handler := func(c *context.Context) {
			c.Text(http.StatusOK, "OK")
		}
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		token := rr.Header().Get("X-Custom-Token")
		if token == "" {
			t.Error("Expected custom CSRF token header to be set")
		}
		cookies := rr.Result().Cookies()
		found := false
		for _, cookie := range cookies {
			if cookie.Name == "custom-token" && cookie.Value == token {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected custom CSRF token cookie to be set")
		}
	})
}
