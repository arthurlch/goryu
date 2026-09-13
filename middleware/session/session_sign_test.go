package session_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arthurlch/goryu"
	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/session"
)

func TestSessionCookieMustBeSigned(t *testing.T) {
	store, err := session.NewSecureStore("test-encryption-key-must-be-at-least-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Stop()

	mw := session.New(session.Config{Store: store, CookieName: "sid"})

	// Handler records the loaded value, sets one, and writes so the cookie is saved.
	var seen string
	var id string
	handler := func(c *goryu.Ctx) {
		s, err := session.Get(c)
		if err != nil {
			t.Fatalf("no session: %v", err)
		}
		if v, ok := s.Get("user").(string); ok {
			seen = v
		} else {
			seen = ""
		}
		id = s.ID
		s.Set("user", "alice")
		_ = c.Text(http.StatusOK, "ok")
	}

	do := func(cookie string) *http.Cookie {
		req := httptest.NewRequest("GET", "/", nil)
		if cookie != "" {
			req.Header.Set("Cookie", "sid="+cookie)
		}
		rr := httptest.NewRecorder()
		mw(handler)(context.NewContext(rr, req))
		for _, ck := range rr.Result().Cookies() {
			if ck.Name == "sid" {
				return ck
			}
		}
		t.Fatal("no session cookie set")
		return nil
	}

	// 1) fresh session
	valid := do("")
	firstID := id

	// 2) valid signed cookie round-trips: value and ID persist
	do(valid.Value)
	if seen != "alice" || id != firstID {
		t.Fatalf("valid cookie did not restore session: seen=%q id=%q firstID=%q", seen, id, firstID)
	}

	// 3) tampered cookie must be rejected -> new session, old value gone
	tampered := valid.Value[:len(valid.Value)-1] + flip(valid.Value[len(valid.Value)-1])
	do(tampered)
	if seen == "alice" {
		t.Fatalf("tampered cookie was accepted and restored the session")
	}
	if id == firstID {
		t.Fatalf("tampered cookie reused the original session id (fixation)")
	}
}

func flip(b byte) string {
	if b == 'A' {
		return "B"
	}
	return "A"
}
