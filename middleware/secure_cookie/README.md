# SecureCookie Middleware

A middleware for storing encrypted and authenticated data in cookies using AES-256-GCM. It ensures that cookie data cannot be read or tampered with by the client.

## Features

- **Encryption**: Uses AES-256-GCM for authenticated encryption.
- **Tamper Proof**: Ensures data integrity using GCM's authentication tag.
- **Context Integration**: Easy `Set`, `Get`, and `Clear` helper functions.
- **Secure Defaults**: Enforces `HttpOnly`, `Secure`, and `SameSite` attributes.
- **Automatic Decryption**: Decrypts cookie data and makes it available in the request context.

## Usage

### Basic Setup

```go
package main

import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/goryuctx"
    "github.com/arthurlch/goryu/middleware/secure_cookie"
)

func main() {
    app := goryu.New()

    // 32-byte hex-encoded key (64 hex characters)
    // Generate one using: openssl rand -hex 32
    secretKey := "a1b2c3d4e5f6..." 

    app.Use(secure_cookie.Default(secretKey, "my-secure-session"))

    app.GET("/login", func(c *goryuctx.Context) error {
        // Store sensitive data
        err := secure_cookie.Set(c, map[string]string{
            "user_id": "12345",
            "role":    "admin",
        })
        return c.String(200, "Logged in")
    })

    app.GET("/profile", func(c *goryuctx.Context) error {
        // Retrieve data
        data, err := secure_cookie.Get(c)
        if err != nil {
            return c.String(401, "Not logged in")
        }
        return c.JSON(200, data)
    })

    app.GET("/logout", func(c *goryuctx.Context) error {
        // Clear cookie
        secure_cookie.Clear(c)
        return c.String(200, "Logged out")
    })

    app.Run(":8080")
}
```

### Advanced Configuration

```go
app.Use(secure_cookie.New(secure_cookie.Config{
    HexKey:     "your-64-char-hex-key",
    CookieName: "session_id",
    CookiePath: "/",
    CookieTTL:  24 * time.Hour,
    Secure:     true,
    HttpOnly:   true,
    SameSite:   http.SameSiteStrictMode,
}))
```

## Security Note

- **Key Management**: The `HexKey` must be a 64-character hex string representing a 32-byte key. Keep this key secret and rotate it if compromised.
- **Cookie Size**: Cookies have a size limit (usually 4KB). Since encryption adds overhead (IV + Auth Tag + Base64), avoid storing large amounts of data.
