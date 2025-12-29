# CORS Middleware

A flexible Cross-Origin Resource Sharing (CORS) middleware for Goryu. It handles preflight requests and validates cross-origin requests according to the configured policy.

## Features

- **Origin Validation**: Strict allowlist checking or wildcard support (dev only).
- **Credentials Support**: Securely handles `Access-Control-Allow-Credentials`.
- **Preflight Handling**: Automatically handles `OPTIONS` requests with correct headers.
- **Security Checks**: Prevents insecure configurations (e.g., wildcard origin with credentials).
- **Caching**: Configurable `Access-Control-Max-Age`.

## Usage

### Basic Setup

Allow all origins (useful for development):

```go
app.Use(cors.WithAllowAll())
```

### Production Setup

Restrict access to specific domains:

```go
app.Use(cors.WithOrigins("https://myapp.com", "https://api.myapp.com"))
```

### Advanced Configuration

Full control over headers, methods, and exposure:

```go
app.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"https://myapp.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Authorization", "Content-Type"},
    ExposeHeaders:    []string{"X-Custom-Header"},
    AllowCredentials: true,
    MaxAge:           3600, // 1 hour
}))
```

## Security Best Practices

1.  **Avoid Wildcards in Production**: Never use `AllowOrigins: []string{"*"}` in production environments. It allows any website to make requests to your API.
2.  **Credentials & Wildcards**: The middleware explicitly forbids combining `AllowCredentials: true` with wildcard origins, as this is a security risk and is disallowed by the CORS spec.
3.  **Vary Header**: The middleware automatically adds `Vary: Origin` to responses to ensure correct caching behavior by CDNs and browsers.
