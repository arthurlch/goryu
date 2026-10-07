package passkey

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"

	goryuctx "github.com/arthurlch/goryu/goryuctx"
)

// SessionStore persists the per-ceremony challenge between the begin and finish
// steps. Implementations must be safe for concurrent use.
type SessionStore interface {
	Save(c *goryuctx.Context, data *webauthn.SessionData) error
	Load(c *goryuctx.Context) (*webauthn.SessionData, error)
	Clear(c *goryuctx.Context)
}

const ceremonyCookie = "webauthn_ceremony"
const ceremonyTTL = 5 * time.Minute

type memorySessionStore struct {
	mu     sync.Mutex
	data   map[string]memoryEntry
	secure bool
}

type memoryEntry struct {
	session   *webauthn.SessionData
	expiresAt time.Time
}

func newMemorySessionStore(secure bool) *memorySessionStore {
	return &memorySessionStore{data: make(map[string]memoryEntry), secure: secure}
}

func (s *memorySessionStore) Save(c *goryuctx.Context, data *webauthn.SessionData) error {
	id := randID()

	s.mu.Lock()
	s.pruneLocked()
	s.data[id] = memoryEntry{session: data, expiresAt: time.Now().Add(ceremonyTTL)}
	s.mu.Unlock()

	return c.SetCookie(&http.Cookie{
		Name:     ceremonyCookie,
		Value:    id,
		Path:     "/",
		MaxAge:   int(ceremonyTTL.Seconds()),
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *memorySessionStore) Load(c *goryuctx.Context) (*webauthn.SessionData, error) {
	ck, err := c.Request.Cookie(ceremonyCookie)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.data[ck.Value]
	if !ok || time.Now().After(entry.expiresAt) {
		delete(s.data, ck.Value)
		return nil, errors.New("ceremony expired")
	}
	return entry.session, nil
}

func (s *memorySessionStore) Clear(c *goryuctx.Context) {
	if ck, err := c.Request.Cookie(ceremonyCookie); err == nil {
		s.mu.Lock()
		delete(s.data, ck.Value)
		s.mu.Unlock()
	}
	_ = c.SetCookie(&http.Cookie{
		Name:     ceremonyCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *memorySessionStore) pruneLocked() {
	now := time.Now()
	for id, entry := range s.data {
		if now.After(entry.expiresAt) {
			delete(s.data, id)
		}
	}
}

func randID() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
