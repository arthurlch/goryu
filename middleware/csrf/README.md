# CSRF Middleware

A robust Cross-Site Request Forgery (CSRF) protection middleware for Goryu. It implements the Double Submit Cookie pattern to secure your application against CSRF attacks.

## Features

- **Double Submit Cookie Pattern**: Stateless and scalable protection.
- **Secure Defaults**: Generates cryptographically strong 32-byte tokens.
- **Cookie Security**: Supports `HttpOnly`, `Secure`, and `SameSite` attributes.
- **Safe Methods**: Automatically skips validation for safe HTTP methods (GET, HEAD, OPTIONS).
- **Customizable**: Configurable header names, cookie names, and token properties.

## Usage

### Basic Setup

```go
package main

import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/goryuctx"
    "github.com/arthurlch/goryu/middleware/csrf"
)

func main() {
    app := goryu.New()

    // Apply globally
    app.Use(csrf.Default())

    app.POST("/submit", func(c *goryuctx.Context) {
        c.Status(200).String("CSRF check passed!")
    })

    app.Run(":8080")
}
```

### Client-Side Integration

The middleware sets a `X-CSRF-Token` header and a `csrf-token` cookie on safe requests (GET, HEAD, etc.). Your frontend client must read this token and include it in the header of subsequent unsafe requests (POST, PUT, DELETE).

**Example (JavaScript):**

```javascript
const csrfToken = document.cookie
  .split('; ')
  .find(row => row.startsWith('csrf-token='))
  .split('=')[1];

fetch('/submit', {
  method: 'POST',
  headers: {
    'X-CSRF-Token': csrfToken
  },
  body: JSON.stringify({ data: 'value' })
});
```

### Advanced Configuration

```go
app.Use(csrf.New(csrf.Config{
    TokenHeader: "X-XSRF-TOKEN",
    TokenCookie: "XSRF-TOKEN",
    TokenLength: 64,
    TokenExpiry: 24 * time.Hour,
    Secure:      true, // Set to true in production (requires HTTPS)
    SameSite:    http.SameSiteStrictMode,
}))
```

## How It Works

1.  **Safe Requests (GET, etc.)**: The middleware generates a new random token and sets it as a cookie and a response header.
2.  **Unsafe Requests (POST, etc.)**: The middleware checks for the presence of the token in both the request header and the cookie. It then cryptographically compares them. If they match, the request is allowed; otherwise, it is rejected with `403 Forbidden`.

## Security Note

- **HTTPS Required**: For the `Secure` cookie attribute to work, your application must be served over HTTPS.
- **Subdomains**: If your API and frontend are on different subdomains, ensure `SameSite` and `Domain` cookie attributes are configured correctly.
