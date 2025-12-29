package session_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/middleware/session"
)

func TestSecureStore(t *testing.T) {
	store, err := session.NewSecureStore("test-encryption-key-must-be-at-least-32-chars",
		session.WithMaxAge(1*time.Hour),
		session.WithMaxSize(1024),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Stop()
	t.Run("EncryptDecrypt", func(t *testing.T) {
		testSession := &session.Session{
			ID: "test-session-id",
			Data: map[string]any{
				"user_id": "123",
				"email":   "test@example.com",
				"roles":   []string{"user", "admin"},
			},
		}
		if err := store.Save(testSession); err != nil {
			t.Fatal(err)
		}
		retrieved, err := store.Get("test-session-id")
		if err != nil {
			t.Fatal(err)
		}
		if retrieved.ID != testSession.ID {
			t.Errorf("Expected ID %s, got %s", testSession.ID, retrieved.ID)
		}
		if retrieved.Data["user_id"] != "123" {
			t.Errorf("Expected user_id 123, got %v", retrieved.Data["user_id"])
		}
		if retrieved.Data["email"] != "test@example.com" {
			t.Errorf("Expected email test@example.com, got %v", retrieved.Data["email"])
		}
	})
	t.Run("Expiration", func(t *testing.T) {
		shortStore, err := session.NewSecureStore("another-test-key-that-is-long-enough-for-aes",
			session.WithMaxAge(100*time.Millisecond),
		)
		if err != nil {
			t.Fatal(err)
		}
		defer shortStore.Stop()
		testSession := &session.Session{
			ID:   "expiring-session",
			Data: map[string]any{"test": "data"},
		}
		if err := shortStore.Save(testSession); err != nil {
			t.Fatal(err)
		}
		_, err = shortStore.Get("expiring-session")
		if err != nil {
			t.Error("Expected session to exist immediately after save")
		}
		time.Sleep(150 * time.Millisecond)
		_, err = shortStore.Get("expiring-session")
		if err == nil {
			t.Error("Expected session to be expired")
		}
	})
}
func TestFingerprinting(t *testing.T) {
	store, err := session.NewSecureStore("fingerprint-test-key-must-be-32-characters-long",
		session.WithFingerprinting("User-Agent", "Accept-Language"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Stop()
	testSession := &session.Session{
		ID:   "fingerprint-session",
		Data: map[string]any{"user_id": "123"},
	}
	fingerprint1 := session.GenerateFingerprint(map[string]string{
		"User-Agent":      "Mozilla/5.0",
		"Accept-Language": "en-US",
	}, []string{"User-Agent", "Accept-Language"})
	if err := store.SaveWithFingerprint(testSession, fingerprint1); err != nil {
		t.Fatal(err)
	}
	if err := store.ValidateFingerprint("fingerprint-session", fingerprint1); err != nil {
		t.Error("Expected fingerprint validation to pass with same fingerprint")
	}
	fingerprint2 := session.GenerateFingerprint(map[string]string{
		"User-Agent":      "Chrome/96.0",
		"Accept-Language": "en-US",
	}, []string{"User-Agent", "Accept-Language"})
	if err := store.ValidateFingerprint("fingerprint-session", fingerprint2); err == nil {
		t.Error("Expected fingerprint validation to fail with different fingerprint")
	}
}
func TestSessionSecurity(t *testing.T) {
	app := goryu.New()
	store, _ := session.NewSecureStore("security-test-key-32-characters-minimum-required")
	defer store.Stop()
	app.Use(session.New(session.Config{
		Store:      store,
		CookieName: "test_session",
		Expiration: 1 * time.Hour,
		Secure:     func(b bool) *bool { return &b }(false),
		SameSite:   http.SameSiteStrictMode,
	}))
	app.Use(session.SecureSessionMiddleware(session.SecurityConfig{
		TrackActivity:   true,
		IdleTimeout:     100 * time.Millisecond,
		AbsoluteTimeout: 1 * time.Hour,
		BindToUserAgent: true,
	}))
	app.GET("/test", func(c *goryu.Ctx) {
		sess, _ := session.Get(c)
		sess.Set("test", "value")
		c.JSON(200, map[string]string{"status": "ok"})
	})
	t.Run("IdleTimeout", func(t *testing.T) {
		// t.Skip("Session middleware integration needs debugging - cookie not being set")
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		var sessionCookie *http.Cookie
		cookies := rr.Result().Cookies()
		t.Logf("Found %d cookies", len(cookies))
		for i, cookie := range cookies {
			t.Logf("Cookie %d: Name=%s, Value=%s", i, cookie.Name, cookie.Value)
			if cookie.Name == "test_session" {
				sessionCookie = cookie
				break
			}
		}
		if sessionCookie == nil {
			t.Errorf("Session cookie 'test_session' not found in response")
			return
		}
		time.Sleep(150 * time.Millisecond)
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.AddCookie(sessionCookie)
		rr2 := httptest.NewRecorder()
		app.ServeHTTP(rr2, req2)
		if rr2.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 due to idle timeout, got %d", rr2.Code)
		}
	})
	t.Run("UserAgentBinding", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var sessionCookie *http.Cookie
		for _, cookie := range rr.Result().Cookies() {
			if cookie.Name == "test_session" {
				sessionCookie = cookie
				break
			}
		}
		if sessionCookie == nil {
			t.Fatal("Session cookie not found")
		}

		// Same User-Agent -> Should pass
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.Header.Set("User-Agent", "Mozilla/5.0")
		req2.AddCookie(sessionCookie)
		rr2 := httptest.NewRecorder()
		app.ServeHTTP(rr2, req2)
		if rr2.Code != http.StatusOK {
			t.Errorf("Expected status 200 with same UA, got %d", rr2.Code)
		}

		// Different User-Agent -> Should fail (401)
		req3 := httptest.NewRequest("GET", "/test", nil)
		req3.Header.Set("User-Agent", "EvilBot/1.0")
		req3.AddCookie(sessionCookie)
		rr3 := httptest.NewRecorder()
		app.ServeHTTP(rr3, req3)
		if rr3.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 with different UA, got %d", rr3.Code)
		}
	})
}
func TestSessionToken(t *testing.T) {
	secret := []byte("test-secret-key")
	sessionID := "test-session-123"
	token := session.GenerateSessionToken(sessionID, secret)
	validatedID, err := session.ValidateSessionToken(token, secret)
	if err != nil {
		t.Fatal(err)
	}
	if validatedID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, validatedID)
	}
	_, err = session.ValidateSessionToken("invalid.token", secret)
	if err == nil {
		t.Error("Expected error for invalid token")
	}
	tamperedToken := "fake-session-id." + token[len("test-session-123")+1:]
	_, err = session.ValidateSessionToken(tamperedToken, secret)
	if err == nil {
		t.Error("Expected error for tampered token")
	}
}
func TestAnomalyDetection(t *testing.T) {
	config := session.SecurityConfig{
		MaxSessionsPerUser: 2,
		MaxSessionsPerIP:   3,
	}
	detector := session.NewSessionAnomalyDetector(config)
	t.Run("UserSessionLimit", func(t *testing.T) {
		userID := "user1"
		err := detector.CheckAnomaly(userID, "session1", "192.168.1.1")
		if err != nil {
			t.Error("Expected first session to be allowed")
		}
		err = detector.CheckAnomaly(userID, "session2", "192.168.1.2")
		if err != nil {
			t.Error("Expected second session to be allowed")
		}
		err = detector.CheckAnomaly(userID, "session3", "192.168.1.3")
		if err == nil {
			t.Error("Expected anomaly for too many user sessions")
		}
	})
	t.Run("IPSessionLimit", func(t *testing.T) {
		ip := "192.168.1.100"
		for i := 1; i <= 3; i++ {
			err := detector.CheckAnomaly(
				"user"+string(rune(i)),
				"session"+string(rune(i)),
				ip,
			)
			if err != nil {
				t.Errorf("Expected session %d to be allowed", i)
			}
		}
		err := detector.CheckAnomaly("user4", "session4", ip)
		if err == nil {
			t.Error("Expected anomaly for too many IP sessions")
		}
	})
}
