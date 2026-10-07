# passkey

WebAuthn / passkey authentication built on `github.com/go-webauthn/webauthn`. It
exposes the four begin/finish endpoints and leaves user lookup and credential
storage to your app. The library handles CBOR/COSE, attestation, assertion, and
the sign counter — we don't hand-roll crypto.

## Usage

Implement `UserStore` (resolve the ceremony's user, persist credentials) and wire
the endpoints:

```go
pk, err := passkey.New(passkey.Config{
    RPID:          "example.com",
    RPDisplayName: "Example",
    RPOrigins:     []string{"https://example.com"},
    Users:         myUserStore,
})
if err != nil { log.Fatal(err) }

app.POST("/webauthn/register/begin",  pk.BeginRegistration)
app.POST("/webauthn/register/finish", pk.FinishRegistration)
app.POST("/webauthn/login/begin",     pk.BeginLogin)
app.POST("/webauthn/login/finish",    pk.FinishLogin)
```

Your user type implements `webauthn.User` (re-exported as `passkey.User`).

## Notes

- The per-ceremony challenge is stored server-side. The default `SessionStore` is
  in-memory (process-local) — provide your own for multi-instance deployments.
- `UpdateCredential` must persist the credential after login: a non-increasing
  sign counter indicates a cloned authenticator.
- Set `Insecure: true` only in local development.
