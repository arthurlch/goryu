package session

import (
	"testing"
	"time"

	"github.com/arthurlch/goryu/middleware/auth"
)

func newSeamService(t *testing.T) (*auth.AuthService, *auth.User) {
	t.Helper()
	jwt, err := auth.NewJWTAuth("test-secret-key-that-is-long-enough-1234", "test")
	if err != nil {
		t.Fatalf("jwt: %v", err)
	}
	users := auth.NewInMemoryUserStore()
	svc := auth.NewAuthService(jwt, users, auth.NewInMemoryTokenStore(), auth.NewMockEmailSender(), auth.DefaultAuthServiceConfig())
	user, err := users.AddUser("user@example.com", "Str0ng-Password!", nil)
	if err != nil {
		t.Fatalf("add user: %v", err)
	}
	return svc, user
}

func sessionWithLogin(userID string, loginTime int64) *Session {
	return &Session{ID: "s1", Data: map[string]any{"user_id": userID, "login_time": loginTime}}
}

func TestSessionOutdatedRejectsPrePasswordChange(t *testing.T) {
	svc, user := newSeamService(t)
	stale := sessionWithLogin(user.ID, user.PasswordChangedAt.Add(-time.Hour).Unix())
	if !sessionOutdated(svc, stale) {
		t.Fatal("a session established before the password change must be outdated")
	}
}

func TestSessionOutdatedKeepsPostPasswordChange(t *testing.T) {
	svc, user := newSeamService(t)
	fresh := sessionWithLogin(user.ID, user.PasswordChangedAt.Add(time.Hour).Unix())
	if sessionOutdated(svc, fresh) {
		t.Fatal("a session established after the password change must stay valid")
	}
}

func TestSessionOutdatedUnknownUser(t *testing.T) {
	svc, _ := newSeamService(t)
	if sessionOutdated(svc, sessionWithLogin("does-not-exist", 0)) {
		t.Fatal("unknown user should not be treated as outdated")
	}
}

func TestSessionOutdatedMissingLoginTime(t *testing.T) {
	svc, user := newSeamService(t)
	noLogin := &Session{ID: "s1", Data: map[string]any{"user_id": user.ID}}
	if !sessionOutdated(svc, noLogin) {
		t.Fatal("a session without a login_time cannot be proven fresh and must be rejected")
	}
}
