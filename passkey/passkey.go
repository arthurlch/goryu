// Package passkey adds WebAuthn / passkey auth on top of go-webauthn, exposing
// the four begin/finish endpoints and leaving user and credential storage to the
// app via UserStore.
package passkey

import (
	"errors"
	"net/http"

	"github.com/go-webauthn/webauthn/webauthn"

	goryuctx "github.com/arthurlch/goryu/goryuctx"
)

type (
	User       = webauthn.User
	Credential = webauthn.Credential
)

type UserStore interface {
	User(c *goryuctx.Context) (User, error)
	AddCredential(user User, cred *Credential) error
	// UpdateCredential persists a credential after login: a non-increasing sign
	// counter signals a cloned authenticator, so this must be stored.
	UpdateCredential(user User, cred *Credential) error
}

type Config struct {
	RPID          string
	RPDisplayName string
	RPOrigins     []string
	Users         UserStore
	Sessions      SessionStore // per-ceremony challenge store; default in-memory
	Insecure      bool         // allow the ceremony cookie over plain HTTP (dev only)
	OnLogin       func(c *goryuctx.Context, user User)
}

type Authenticator struct {
	wa       *webauthn.WebAuthn
	users    UserStore
	sessions SessionStore
	onLogin  func(c *goryuctx.Context, user User)
}

func New(cfg Config) (*Authenticator, error) {
	if cfg.Users == nil {
		return nil, errors.New("passkey: Users store is required")
	}
	wa, err := webauthn.New(&webauthn.Config{
		RPID:          cfg.RPID,
		RPDisplayName: cfg.RPDisplayName,
		RPOrigins:     cfg.RPOrigins,
	})
	if err != nil {
		return nil, err
	}
	sessions := cfg.Sessions
	if sessions == nil {
		sessions = newMemorySessionStore(!cfg.Insecure)
	}
	return &Authenticator{wa: wa, users: cfg.Users, sessions: sessions, onLogin: cfg.OnLogin}, nil
}

func (a *Authenticator) BeginRegistration(c *goryuctx.Context) {
	user, err := a.users.User(c)
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	options, session, err := a.wa.BeginRegistration(user)
	if err != nil {
		fail(c, http.StatusInternalServerError, err)
		return
	}
	if err := a.sessions.Save(c, session); err != nil {
		fail(c, http.StatusInternalServerError, err)
		return
	}
	_ = c.JSON(http.StatusOK, options)
}

func (a *Authenticator) FinishRegistration(c *goryuctx.Context) {
	user, session, ok := a.resume(c)
	if !ok {
		return
	}
	cred, err := a.wa.FinishRegistration(user, *session, c.Request)
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	if err := a.users.AddCredential(user, cred); err != nil {
		fail(c, http.StatusInternalServerError, err)
		return
	}
	a.sessions.Clear(c)
	_ = c.JSON(http.StatusOK, map[string]string{"status": "registered"})
}

func (a *Authenticator) BeginLogin(c *goryuctx.Context) {
	user, err := a.users.User(c)
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	options, session, err := a.wa.BeginLogin(user)
	if err != nil {
		fail(c, http.StatusInternalServerError, err)
		return
	}
	if err := a.sessions.Save(c, session); err != nil {
		fail(c, http.StatusInternalServerError, err)
		return
	}
	_ = c.JSON(http.StatusOK, options)
}

func (a *Authenticator) FinishLogin(c *goryuctx.Context) {
	user, session, ok := a.resume(c)
	if !ok {
		return
	}
	cred, err := a.wa.FinishLogin(user, *session, c.Request)
	if err != nil {
		fail(c, http.StatusUnauthorized, err)
		return
	}
	if err := a.users.UpdateCredential(user, cred); err != nil {
		fail(c, http.StatusInternalServerError, err)
		return
	}
	a.sessions.Clear(c)
	if a.onLogin != nil {
		a.onLogin(c, user)
		return
	}
	_ = c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (a *Authenticator) resume(c *goryuctx.Context) (User, *webauthn.SessionData, bool) {
	user, err := a.users.User(c)
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return nil, nil, false
	}
	session, err := a.sessions.Load(c)
	if err != nil {
		fail(c, http.StatusBadRequest, errors.New("no active ceremony"))
		return nil, nil, false
	}
	return user, session, true
}

func fail(c *goryuctx.Context, code int, err error) {
	_ = c.JSON(code, map[string]string{"error": err.Error()})
}
