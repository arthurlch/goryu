// Package authoidc is an OpenID Connect relying party (authorization-code + PKCE)
// built on coreos/go-oidc and x/oauth2.
package authoidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	oidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	goryuctx "github.com/arthurlch/goryu/goryuctx"
)

type Claims struct {
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
}

type Config struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	CookiePrefix string // namespaces the flow cookies; default "oidc"
	Insecure     bool   // allow flow cookies over plain HTTP (dev only)
	OnSuccess    func(c *goryuctx.Context, claims Claims, token *oauth2.Token)
	OnError      func(c *goryuctx.Context, err error)
}

type RelyingParty struct {
	verifier *oidc.IDTokenVerifier
	oauth    *oauth2.Config
	cfg      Config
}

func New(cfg Config) (*RelyingParty, error) {
	if cfg.ClientID == "" || cfg.Issuer == "" || cfg.RedirectURL == "" {
		return nil, errors.New("authoidc: Issuer, ClientID and RedirectURL are required")
	}
	provider, err := oidc.NewProvider(context.Background(), cfg.Issuer)
	if err != nil {
		return nil, err
	}
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "email", "profile"}
	}
	if cfg.CookiePrefix == "" {
		cfg.CookiePrefix = "oidc"
	}
	return &RelyingParty{
		verifier: provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
		oauth: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       scopes,
		},
		cfg: cfg,
	}, nil
}

// Start binds CSRF (state), replay (nonce), and PKCE (verifier) into short-lived
// cookies, then redirects to the provider.
func (rp *RelyingParty) Start(c *goryuctx.Context) {
	state := randToken()
	nonce := randToken()
	verifier := oauth2.GenerateVerifier()

	rp.setCookie(c, "state", state)
	rp.setCookie(c, "nonce", nonce)
	rp.setCookie(c, "verifier", verifier)

	url := rp.oauth.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier))
	_ = c.Redirect(http.StatusFound, url)
}

// Callback validates state, exchanges the code, verifies the ID token and nonce,
// then hands off to OnSuccess. Each check is security-critical; keep them all.
func (rp *RelyingParty) Callback(c *goryuctx.Context) {
	ctx := c.Request.Context()

	state, err := rp.cookie(c, "state")
	if err != nil || c.Request.URL.Query().Get("state") != state {
		rp.fail(c, errors.New("invalid oauth state"))
		return
	}
	verifier, err := rp.cookie(c, "verifier")
	if err != nil {
		rp.fail(c, errors.New("missing pkce verifier"))
		return
	}

	token, err := rp.oauth.Exchange(ctx, c.Request.URL.Query().Get("code"), oauth2.VerifierOption(verifier))
	if err != nil {
		rp.fail(c, err)
		return
	}
	rawID, ok := token.Extra("id_token").(string)
	if !ok {
		rp.fail(c, errors.New("no id_token in token response"))
		return
	}
	idToken, err := rp.verifier.Verify(ctx, rawID)
	if err != nil {
		rp.fail(c, err)
		return
	}
	nonce, err := rp.cookie(c, "nonce")
	if err != nil || idToken.Nonce != nonce {
		rp.fail(c, errors.New("invalid id_token nonce"))
		return
	}

	var claims Claims
	if err := idToken.Claims(&claims); err != nil {
		rp.fail(c, err)
		return
	}
	rp.clearCookies(c)

	if rp.cfg.OnSuccess != nil {
		rp.cfg.OnSuccess(c, claims, token)
		return
	}
	c.Set("user_id", claims.Subject)
	_ = c.JSON(http.StatusOK, claims)
}

func (rp *RelyingParty) fail(c *goryuctx.Context, err error) {
	rp.clearCookies(c)
	if rp.cfg.OnError != nil {
		rp.cfg.OnError(c, err)
		return
	}
	_ = c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication failed"})
}

func (rp *RelyingParty) setCookie(c *goryuctx.Context, name, value string) {
	_ = c.SetCookie(&http.Cookie{
		Name:     rp.cfg.CookiePrefix + "_" + name,
		Value:    value,
		Path:     "/",
		MaxAge:   int((10 * time.Minute).Seconds()),
		HttpOnly: true,
		Secure:   !rp.cfg.Insecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (rp *RelyingParty) cookie(c *goryuctx.Context, name string) (string, error) {
	ck, err := c.Request.Cookie(rp.cfg.CookiePrefix + "_" + name)
	if err != nil {
		return "", err
	}
	if ck.Value == "" {
		return "", errors.New("empty cookie")
	}
	return ck.Value, nil
}

func (rp *RelyingParty) clearCookies(c *goryuctx.Context) {
	for _, name := range []string{"state", "nonce", "verifier"} {
		_ = c.SetCookie(&http.Cookie{
			Name:     rp.cfg.CookiePrefix + "_" + name,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   !rp.cfg.Insecure,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

func randToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
