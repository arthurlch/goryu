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
	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/session"
)
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
		return nil, nil 
	}
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
func newTestContext(req *http.Request) (*goryu.Ctx, *httptest.ResponseRecorder) {
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
		handler := func(c *goryu.Ctx) {
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
		setCookieHeaders := rr.Header()["Set-Cookie"]
		if len(setCookieHeaders) == 0 {
			t.Fatal("Session cookie not set")
		}
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
		var sessionCookie *http.Cookie
		var sessionID string
		setupHandler := func(c *goryu.Ctx) {
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
		handler := func(c *goryu.Ctx) {
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
		var sessionCookie *http.Cookie
		var sessionID string
		setupHandler := func(c *goryu.Ctx) {
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
		destroyHandler := func(c *goryu.Ctx) {
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
		setCookieHeaders := rr2.Header()["Set-Cookie"]
		foundExpired := false
		for _, cookieHeader := range setCookieHeaders {
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
