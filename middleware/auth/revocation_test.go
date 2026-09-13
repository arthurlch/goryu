package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/middleware/auth"
)

func TestPasswordChangeRevokesAccessToken(t *testing.T) {
	users := auth.NewInMemoryUserStore()
	user, err := users.AddUser("a@b.com", "StrongPassw0rd!xyz", nil)
	if err != nil {
		t.Fatal(err)
	}

	jwtAuth, err := auth.NewJWTAuth("0123456789abcdef0123456789abcdef", "test")
	if err != nil {
		t.Fatal(err)
	}
	token, err := jwtAuth.CreateAuthToken(user.ID)
	if err != nil {
		t.Fatal(err)
	}

	cfg := auth.DefaultAuthServiceConfig()
	cfg.RequireEmailVerification = false
	svc := auth.NewAuthService(jwtAuth, users, auth.NewInMemoryTokenStore(), auth.NewMockEmailSender(), cfg)
	handlers := auth.NewAuthHandlers(svc)
	app := goryu.New()
	handlers.RegisterRoutes(app)

	call := func() int {
		req := httptest.NewRequest("GET", "/auth/profile", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)
		return rr.Code
	}

	if code := call(); code != http.StatusOK {
		t.Fatalf("expected 200 before password change, got %d", code)
	}

	time.Sleep(1100 * time.Millisecond)
	if err := users.UpdatePassword("a@b.com", "NewStrongPassw0rd!xyz"); err != nil {
		t.Fatal(err)
	}

	if code := call(); code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after password change, token should be revoked, got %d", code)
	}
}
