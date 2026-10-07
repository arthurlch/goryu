package passkey_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-webauthn/webauthn/webauthn"

	goryuctx "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/passkey"
)

type testUser struct{ id []byte }

func (u *testUser) WebAuthnID() []byte                         { return u.id }
func (u *testUser) WebAuthnName() string                       { return "alice" }
func (u *testUser) WebAuthnDisplayName() string                { return "Alice" }
func (u *testUser) WebAuthnCredentials() []webauthn.Credential { return nil }

type testStore struct {
	user  *testUser
	added []*webauthn.Credential
}

func (s *testStore) User(c *goryuctx.Context) (passkey.User, error) { return s.user, nil }
func (s *testStore) AddCredential(u passkey.User, cred *passkey.Credential) error {
	s.added = append(s.added, cred)
	return nil
}
func (s *testStore) UpdateCredential(u passkey.User, cred *passkey.Credential) error { return nil }

func newAuth(t *testing.T, store passkey.UserStore) *passkey.Authenticator {
	t.Helper()
	pk, err := passkey.New(passkey.Config{
		RPID:          "example.com",
		RPDisplayName: "Example",
		RPOrigins:     []string{"https://example.com"},
		Users:         store,
		Insecure:      true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return pk
}

func TestNewRequiresUsers(t *testing.T) {
	if _, err := passkey.New(passkey.Config{RPID: "example.com", RPOrigins: []string{"https://example.com"}}); err == nil {
		t.Fatal("expected error without a Users store")
	}
}

func TestBeginRegistrationSetsChallenge(t *testing.T) {
	pk := newAuth(t, &testStore{user: &testUser{id: []byte("user-handle-1234")}})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/webauthn/register/begin", nil)
	pk.BeginRegistration(goryuctx.NewContext(w, req))

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "challenge") {
		t.Fatalf("creation options missing challenge: %s", w.Body.String())
	}
	found := false
	for _, ck := range w.Result().Cookies() {
		if ck.Name == "webauthn_ceremony" {
			found = true
		}
	}
	if !found {
		t.Fatal("ceremony cookie was not set")
	}
}

func TestFinishWithoutCeremonyFails(t *testing.T) {
	pk := newAuth(t, &testStore{user: &testUser{id: []byte("user-handle-1234")}})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/webauthn/register/finish", strings.NewReader("{}"))
	pk.FinishRegistration(goryuctx.NewContext(w, req))

	if w.Code != 400 {
		t.Fatalf("finish without an active ceremony must be 400, got %d", w.Code)
	}
}
