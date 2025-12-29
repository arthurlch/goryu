# Secure Middleware

A middleware that secures your Goryu application by setting various HTTP security headers. It helps prevent XSS, clickjacking, and other common web attacks.

## Features

- **XSS Protection**: Enables browser XSS filtering.
- **Content Type Options**: Prevents MIME type sniffing (`nosniff`).
- **Frame Options**: Protects against clickjacking (`SAMEORIGIN`).
- **HSTS**: Enforces HTTPS connections (Strict-Transport-Security).
- **CSP**: Configurable Content Security Policy.
- **Referrer Policy**: Controls referrer information sent in requests.
- **Permissions Policy**: Controls browser features and APIs.

## Usage

### Basic Setup

Apply default security headers:

```go
app.Use(secure.Default())
```

This sets:
- `X-XSS-Protection: 1; mode=block`
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: SAMEORIGIN`
- `Referrer-Policy: strict-origin-when-cross-origin`

### HTTPS Enforcement (HSTS)

Enable Strict-Transport-Security for 1 year, including subdomains:

```go
app.Use(secure.WithHSTS(31536000, true))
```

### Content Security Policy (CSP)

Define a custom CSP:

```go
app.Use(secure.WithCSP("default-src 'self'; script-src 'self' https://trusted.cdn.com"))
```

### Advanced Configuration

```go
app.Use(secure.New(secure.Config{
    XSSProtection:         "1; mode=block",
    ContentTypeNosniff:    "nosniff",
    XFrameOptions:         "DENY",
    HSTSMaxAge:            31536000,
    HSTSIncludeSubdomains: true,
    HSTSPreload:           true,
    ContentSecurityPolicy: "default-src 'self'",
    ReferrerPolicy:        "no-referrer",
    PermissionsPolicy:     "geolocation=(), camera=()",
}))
```

## Security Note

- **HSTS Preload**: Be careful when enabling `HSTSPreload`. Once your domain is on the browser preload list, it is difficult to remove. Ensure you are committed to HTTPS for the long term.
- **CSP**: A poorly configured CSP can break your application. Test thoroughly in `Content-Security-Policy-Report-Only` mode (if supported by your setup) or development environment before enforcing.
