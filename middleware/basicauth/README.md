# BasicAuth Middleware

A robust HTTP Basic Authentication middleware for Goryu. It supports both static user configuration with secure bcrypt hashing and dynamic validation logic.

## Features

- **Secure by Default**: Enforces bcrypt hashing for static passwords.
- **Flexible Validation**: Supports custom validator functions for dynamic checks (e.g., database lookups).
- **Rate Limiting**: Built-in protection against brute-force attacks.
- **Timing Attack Protection**: Uses constant-time comparisons for password verification.
- **Configurable Realm**: Custom authentication realm support.

## Usage

### Static Users

For simple use cases, you can define a static map of users. **Note: Passwords must be bcrypt hashed.**

```go
package main

import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/goryuctx"
    "github.com/arthurlch/goryu/middleware/basicauth"
)

func main() {
    app := goryu.New()

    // Helper to generate hash (do not use in production runtime)
    hash, _ := basicauth.HashPassword("secret")

    app.Use(basicauth.WithUsers(map[string]string{
        "admin": hash,
    }))

    app.GET("/", func(c *goryuctx.Context) {
        c.Status(200).String("Welcome Admin!")
    })

    app.Run(":8080")
}
```

### Custom Validator

For more complex scenarios, use a custom validator function:

```go
app.Use(basicauth.WithValidator(func(username, password string) bool {
    // Check credentials against database, LDAP, etc.
    if username == "john" && password == "doe" {
        return true
    }
    return false
}))
```

### Advanced Configuration

Use the `Config` struct for full control:

```go
app.Use(basicauth.New(basicauth.Config{
    Realm: "My Secure App",
    MaxAttempts: 3,
    RateWindow: 10 * time.Minute,
    Validator: myCustomValidator,
}))
```

## Security Note

When using static configuration, `basicauth` **requires** passwords to be hashed using bcrypt (starting with `$2a$`, `$2b$`, or `$2y$`). Plain text passwords in configuration will cause the middleware to fail validation on startup, ensuring you don't accidentally expose credentials.
