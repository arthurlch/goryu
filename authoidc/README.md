# authoidc

OpenID Connect relying party built on `coreos/go-oidc` + `golang.org/x/oauth2`.
Runs the authorization-code flow with PKCE, verifies the ID token, and hands your
app the verified claims. We don't hand-roll any crypto — the audited libraries do
discovery, JWKS rotation, and signature verification.

## Usage

```go
rp, err := authoidc.New(authoidc.Config{
    Issuer:       "https://accounts.google.com",
    ClientID:     os.Getenv("OIDC_CLIENT_ID"),
    ClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
    RedirectURL:  "https://app.example.com/auth/callback",
})
if err != nil { log.Fatal(err) }

app.GET("/auth/login", rp.Start)
app.GET("/auth/callback", rp.Callback)
```

By default `Callback` sets the `user_id` context key to the subject and returns
the claims as JSON. Supply `OnSuccess` to create your own session / redirect.

## Security

`Start` binds three values into short-lived, HttpOnly, `SameSite=Lax` cookies and
`Callback` verifies them all: **state** (CSRF), **nonce** (ID-token replay), and a
**PKCE verifier** (code interception). The ID token's signature, issuer, audience
and expiry are verified against the provider's JWKS by `go-oidc`.

Set `Insecure: true` only in local development (allows the flow cookies over HTTP).
