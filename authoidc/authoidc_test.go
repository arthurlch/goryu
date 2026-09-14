package authoidc_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arthurlch/goryu/authoidc"
	goryuctx "github.com/arthurlch/goryu/goryuctx"
)

func mockProvider(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                srv.URL,
			"authorization_endpoint":                srv.URL + "/auth",
			"token_endpoint":                        srv.URL + "/token",
			"jwks_uri":                              srv.URL + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestStartRedirectsWithPKCEAndState(t *testing.T) {
	srv := mockProvider(t)
	rp, err := authoidc.New(authoidc.Config{
		Issuer:       srv.URL,
		ClientID:     "client",
		ClientSecret: "secret",
		RedirectURL:  "https://app.example.com/callback",
		Insecure:     true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/login", nil)
	rp.Start(goryuctx.NewContext(w, req))

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	for _, want := range []string{"code_challenge=", "code_challenge_method=S256", "state=", "nonce="} {
		if !strings.Contains(loc, want) {
			t.Fatalf("redirect missing %q: %s", want, loc)
		}
	}
	names := map[string]bool{}
	for _, ck := range w.Result().Cookies() {
		names[ck.Name] = true
	}
	for _, want := range []string{"oidc_state", "oidc_nonce", "oidc_verifier"} {
		if !names[want] {
			t.Fatalf("missing flow cookie %q", want)
		}
	}
}

func TestCallbackRejectsBadState(t *testing.T) {
	srv := mockProvider(t)
	rp, err := authoidc.New(authoidc.Config{
		Issuer:      srv.URL,
		ClientID:    "client",
		RedirectURL: "https://app.example.com/callback",
		Insecure:    true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/callback?state=evil&code=x", nil)
	req.AddCookie(&http.Cookie{Name: "oidc_state", Value: "good"})
	rp.Callback(goryuctx.NewContext(w, req))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("state mismatch must be rejected with 401, got %d", w.Code)
	}
}

func TestNewValidatesConfig(t *testing.T) {
	if _, err := authoidc.New(authoidc.Config{ClientID: "x"}); err == nil {
		t.Fatal("expected error for missing Issuer/RedirectURL")
	}
}
