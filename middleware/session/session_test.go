package session_test

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/session"
)

// --- In-Memory Store for Testing ---
type memoryStore struct {
	sync.RWMutex
	sessions map[string]*session.Session
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		sessions: make(map[string]*session.Session),
	}
}

func (s *memoryStore) Get(id string) (*session.Session, error) {
	s.RLock()
	defer s.RUnlock()
	sess, ok := s.sessions[id]
	if !ok {
		return nil, nil // Not found
	}
	// Return a copy to prevent race conditions on the session data map
	newSess := &session.Session{
		ID:   sess.ID,
		Data: make(map[string]any),
	}
	maps.Copy(newSess.Data, sess.Data)
	return newSess, nil
}

func (s *memoryStore) Save(sess *session.Session) error {
	s.Lock()
	defer s.Unlock()
	s.sessions[sess.ID] = sess
	return nil
}

func (s *memoryStore) Destroy(id string) error {
	s.Lock()
	defer s.Unlock()
	delete(s.sessions, id)
	return nil
}

// --- Test Setup ---
func newTestContext(req *http.Request) (*goryu.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}

func TestSessionMiddleware(t *testing.T) {
	store := newMemoryStore()
	config := session.Config{
		Store:      store,
		Expiration: 1 * time.Hour,
	}
	middleware := session.New(config)

	t.Run("CreateNewSession", func(t *testing.T) {
		handler := func(c *goryu.Context) {
			sess, err := session.Get(c)
			if err != nil {
				t.Fatalf("Failed to get session: %v", err)
			}
			sess.Set("username", "goryu_user")
			_ = c.Text(http.StatusOK, "Session created")
		}

		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		// Check if cookie was set in the response
		// Parse cookies from the Set-Cookie header
		setCookieHeaders := rr.Header()["Set-Cookie"]
		if len(setCookieHeaders) == 0 {
			t.Fatal("Session cookie not set")
		}

		// Verify the cookie name
		foundSessionCookie := false
		for _, cookieHeader := range setCookieHeaders {
			if len(cookieHeader) > len(config.CookieName) && cookieHeader[:len(config.CookieName)] == config.CookieName {
				foundSessionCookie = true
				break
			}
		}
		if !foundSessionCookie {
			t.Fatalf("Expected cookie named %s not found in Set-Cookie headers", config.CookieName)
		}
		if len(store.sessions) != 1 {
			t.Errorf("Expected 1 session in store, got %d", len(store.sessions))
		}
	})

	t.Run("RetrieveAndModifySession", func(t *testing.T) {
		// First, create a session to retrieve
		var sessionCookie *http.Cookie
		var sessionID string
		setupHandler := func(c *goryu.Context) {
			sess, _ := session.Get(c)
			sess.Set("visits", 1)
			sessionID = sess.ID
		}
		req1 := httptest.NewRequest("GET", "/", nil)
		ctx1, rr1 := newTestContext(req1)
		middleware(setupHandler)(ctx1)

		resp1 := rr1.Result()
		cookies := resp1.Cookies()
		if len(cookies) == 0 {
			t.Fatal("No cookie was set in the setup request")
		}
		sessionCookie = cookies[0]

		// Now, test retrieving and modifying it
		handler := func(c *goryu.Context) {
			sess, err := session.Get(c)
			if err != nil {
				t.Fatalf("Failed to get session: %v", err)
			}
			visits := sess.Get("visits").(int)
			if visits != 1 {
				t.Errorf("Expected visits to be 1, got %d", visits)
			}
			sess.Set("visits", visits+1)
			_ = c.Text(http.StatusOK, "Session modified")
		}

		req2 := httptest.NewRequest("GET", "/", nil)
		req2.AddCookie(sessionCookie)
		ctx2, _ := newTestContext(req2)

		middleware(handler)(ctx2)

		finalSess, err := store.Get(sessionID)
		if err != nil || finalSess == nil {
			t.Fatalf("Could not retrieve final session from store")
		}

		if finalSess.Get("visits").(int) != 2 {
			t.Errorf("Expected visits to be updated to 2, got %d", finalSess.Get("visits"))
		}
	})

	t.Run("DestroySession", func(t *testing.T) {
		// Create a session to destroy
		var sessionCookie *http.Cookie
		var sessionID string
		setupHandler := func(c *goryu.Context) {
			sess, _ := session.Get(c)
			sessionID = sess.ID
		}
		req1 := httptest.NewRequest("GET", "/", nil)
		ctx1, rr1 := newTestContext(req1)
		middleware(setupHandler)(ctx1)

		resp1 := rr1.Result()
		cookies := resp1.Cookies()
		if len(cookies) == 0 {
			t.Fatal("No cookie was set in the setup request")
		}
		sessionCookie = cookies[0]

		if _, exists := store.sessions[sessionID]; !exists {
			t.Fatal("Session was not created in the store")
		}

		// Now, destroy it
		destroyHandler := func(c *goryu.Context) {
			if err := session.Destroy(c); err != nil {
				t.Fatalf("Failed to destroy session: %v", err)
			}
			_ = c.Text(http.StatusOK, "Session destroyed")
		}

		req2 := httptest.NewRequest("GET", "/", nil)
		req2.AddCookie(sessionCookie)
		ctx2, rr2 := newTestContext(req2)
		middleware(destroyHandler)(ctx2)

		if rr2.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr2.Code)
		}
		if _, exists := store.sessions[sessionID]; exists {
			t.Error("Session was not removed from the store")
		}

		// Check if the cookie was expired by examining Set-Cookie headers
		setCookieHeaders := rr2.Header()["Set-Cookie"]
		foundExpired := false
		for _, cookieHeader := range setCookieHeaders {
			// Check if it contains our cookie name and has Max-Age=0 or -1
			if strings.Contains(cookieHeader, config.CookieName) &&
				(strings.Contains(cookieHeader, "Max-Age=0") || strings.Contains(cookieHeader, "Max-Age=-1")) {
				foundExpired = true
				break
			}
		}

		if !foundExpired {
			t.Errorf("Session cookie was not expired. Set-Cookie headers: %v", setCookieHeaders)
		}
	})
}
