package auth_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/auth"
	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "my-test-secret"

func newTestContext(req *http.Request) (*goryu.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}

func createTestToken(userID string, secret string, expiresAt time.Time) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func TestAuthMiddleware(t *testing.T) {
	handler := func(c *goryu.Context) {
		userID, exists := c.Get(auth.UserIDKey)
		if !exists {
			t.Error("UserID not found in context")
		}
		if userID.(string) != "user-123" {
			t.Errorf("Expected userID 'user-123', got '%s'", userID)
		}
		_ = c.Text(http.StatusOK, "OK")
	}

	config := auth.Config{
		SecretKey: testSecret,
	}
	middleware := auth.New(config)

	t.Run("ValidToken", func(t *testing.T) {
		validToken, err := createTestToken("user-123", testSecret, time.Now().Add(1*time.Hour))
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 for valid token, got %d", rr.Code)
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

	t.Run("InvalidTokenFormat", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bear token") // Invalid, not Bearer
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})

	t.Run("ExpiredToken", func(t *testing.T) {
		expiredToken, err := createTestToken("user-123", testSecret, time.Now().Add(-1*time.Hour))
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+expiredToken)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for expired token, got %d", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "Token has expired") {
			t.Errorf("Expected 'Token has expired' error, got: %s", rr.Body.String())
		}
	})

	t.Run("WrongSecret", func(t *testing.T) {
		wrongToken, err := createTestToken("user-123", "another-secret", time.Now().Add(1*time.Hour))
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+wrongToken)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for wrong secret, got %d", rr.Code)
		}
	})
}
