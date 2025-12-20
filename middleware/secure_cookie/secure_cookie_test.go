package securecookie_test
import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/secure_cookie"
)
const testHexKey = "a1b2c3d4e5f6a7b8c9d0a1b2c3d4e5f6a7b8c9d0a1b2c3d4e5f6a7b8c9d0a1b2"
func newTestContext(req *http.Request) (*context.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}
func TestSecureCookieMiddleware(t *testing.T) {
	t.Run("BasicFunctionality", func(t *testing.T) {
		config := securecookie.Config{
			HexKey:     testHexKey,
			CookieName: "test-cookie",
		}
		middleware := securecookie.New(config)
		handler := func(c *context.Context) {
			data, err := securecookie.Get(c)
			if err == nil {
				expected := map[string]string{"hello": "world"}
				if !reflect.DeepEqual(data, expected) {
					t.Errorf("Got unexpected data from cookie: got %v, want %v", data, expected)
				}
				c.Writer.WriteHeader(http.StatusOK)
				return
			}
			if errors.Is(err, securecookie.ErrValueNotFound) {
				if err := securecookie.Set(c, map[string]string{"hello": "world"}); err != nil {
					t.Errorf("Failed to set cookie: %v", err)
				}
				c.Writer.WriteHeader(http.StatusCreated)
				return
			}
			c.Writer.WriteHeader(http.StatusInternalServerError)
		}
		req1 := httptest.NewRequest("GET", "/", nil)
		ctx1, rr1 := newTestContext(req1)
		middleware(handler)(ctx1)
		if rr1.Code != http.StatusCreated {
			t.Fatalf("First request: expected status %d, got %d", http.StatusCreated, rr1.Code)
		}
		cookieHeader := rr1.Header().Get("Set-Cookie")
		if !strings.Contains(cookieHeader, "test-cookie=") {
			t.Fatal("First request: cookie was not set in response")
		}
		req2 := httptest.NewRequest("GET", "/", nil)
		req2.Header.Set("Cookie", cookieHeader)
		ctx2, rr2 := newTestContext(req2)
		middleware(handler)(ctx2)
		if rr2.Code != http.StatusOK {
			t.Fatalf("Second request: expected status %d, got %d", http.StatusOK, rr2.Code)
		}
	})
	t.Run("TamperedCookie", func(t *testing.T) {
		config := securecookie.Config{
			HexKey:     testHexKey,
			CookieName: "tamper-test",
		}
		middleware := securecookie.New(config)
		handler := func(c *context.Context) {
			data, err := securecookie.Get(c)
			if err != nil && errors.Is(err, securecookie.ErrValueNotFound) {
				c.Writer.WriteHeader(http.StatusUnauthorized)
				return
			}
			if data != nil {
				t.Error("Should not have valid data from tampered cookie")
			}
			c.Writer.WriteHeader(http.StatusOK)
		}
		req1 := httptest.NewRequest("GET", "/", nil)
		ctx1, rr1 := newTestContext(req1)
		middleware(func(c *context.Context) {
			securecookie.Set(c, map[string]string{"valid": "data"})
		})(ctx1)
		cookieHeader := rr1.Header().Get("Set-Cookie")
		tamperedCookie := strings.Replace(cookieHeader, "=", "=TAMPERED", 1)
		req2 := httptest.NewRequest("GET", "/", nil)
		req2.Header.Set("Cookie", tamperedCookie)
		ctx2, rr2 := newTestContext(req2)
		middleware(handler)(ctx2)
		if rr2.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d for tampered cookie, got %d", http.StatusUnauthorized, rr2.Code)
		}
	})
	t.Run("ClearCookie", func(t *testing.T) {
		config := securecookie.Config{
			HexKey:     testHexKey,
			CookieName: "clear-test",
		}
		middleware := securecookie.New(config)
		handler := func(c *context.Context) {
			if err := securecookie.Clear(c); err != nil {
				t.Errorf("Failed to clear cookie: %v", err)
			}
			c.Writer.WriteHeader(http.StatusOK)
		}
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		cookieHeader := rr.Header().Get("Set-Cookie")
		if !strings.Contains(cookieHeader, "clear-test=") {
			t.Fatal("Clear cookie header not found")
		}
		if !strings.Contains(cookieHeader, "Max-Age=0") {
			t.Fatal("Cookie was not properly expired")
		}
	})
	t.Run("InvalidHexKey", func(t *testing.T) {
		config := securecookie.Config{
			HexKey:     "invalid-hex",
			CookieName: "invalid-test",
		}
		middleware := securecookie.New(config)
		handler := func(c *context.Context) {
			t.Error("Handler should not be called with invalid config")
			c.Writer.WriteHeader(http.StatusOK)
		}
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code == http.StatusOK {
			t.Error("Expected error status for invalid hex key")
		}
	})
	t.Run("DefaultHelper", func(t *testing.T) {
		middleware := securecookie.Default(testHexKey, "default-test")
		handler := func(c *context.Context) {
			c.Writer.WriteHeader(http.StatusOK)
		}
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})
	t.Run("CustomConfig", func(t *testing.T) {
		config := securecookie.Config{
			HexKey:     testHexKey,
			CookieName: "custom-test",
			CookiePath: "/admin",
			Secure:     false, 
			HttpOnly:   false, 
			SameSite:   http.SameSiteStrictMode,
		}
		middleware := securecookie.New(config)
		handler := func(c *context.Context) {
			securecookie.Set(c, map[string]string{"test": "data"})
			c.Writer.WriteHeader(http.StatusOK)
		}
		req := httptest.NewRequest("GET", "/admin", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		cookieHeader := rr.Header().Get("Set-Cookie")
		if !strings.Contains(cookieHeader, "Path=/admin") {
			t.Error("Cookie path not set correctly")
		}
		if !strings.Contains(cookieHeader, "SameSite=Strict") {
			t.Error("SameSite not set correctly")
		}
	})
}