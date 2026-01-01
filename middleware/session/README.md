# Session Middleware

A flexible and secure session management middleware for Goryu. It supports pluggable storage backends (memory, Redis, database) and secure cookie handling.

## Features

- **Pluggable Storage**: Define your own `Store` interface for saving session data.
- **Secure Cookies**: Enforces `HttpOnly`, `Secure`, and `SameSite` attributes.
- **Session Regeneration**: Helper to prevent Session Fixation attacks.
- **Context Integration**: Easy access to session data via `session.Get(c)` and `session.Set(c)`.

## Usage

### Basic Setup

You need to provide a `Store` implementation. Here is a conceptual example:

```go
package main

import (
    "time"
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/goryuctx"
    "github.com/arthurlch/goryu/middleware/session"
)

// In-memory store implementation (for testing only)
type MemoryStore struct { ... }
// ... implement Get, Save, Destroy

func main() {
    app := goryu.New()

    store := NewMemoryStore()

    app.Use(session.New(session.Config{
        Store:      store,
        CookieName: "sid",
        Expiration: 24 * time.Hour,
    }))

    app.GET("/login", func(c *goryuctx.Context) {
        sess, _ := session.Get(c)
        sess.Set("user_id", "123")
        c.Status(200).String("Logged in")
    })

    app.GET("/profile", func(c *goryuctx.Context) {
        sess, _ := session.Get(c)
        userID := sess.Get("user_id")
        c.Status(200).String("User ID: " + userID.(string))
    })

    app.Run(":8080")
}
```

### Session Regeneration

To prevent Session Fixation attacks, regenerate the session ID after privilege changes (e.g., login):

```go
app.POST("/login", func(c *goryuctx.Context) {
    // ... verify credentials ...
    
    // Regenerate session ID
    if err := session.Regenerate(c); err != nil {
        c.Status(500).String("Failed to regenerate session")
        return
    }
    
    sess, _ := session.Get(c)
    sess.Set("user_id", user.ID)
    c.Status(200).String("Logged in securely")
})
```

### Interface Definition

To implement a custom store (e.g., Redis), implement this interface:

```go
type Store interface {
    Get(id string) (*Session, error)
    Save(session *Session) error
    Destroy(id string) error
}
```

## Security Best Practices

- **HTTPS**: Always use HTTPS in production. The middleware defaults to `Secure: true` for cookies.
- **Regeneration**: Always call `session.Regenerate(c)` after a user logs in.
- **Expiration**: Set a reasonable expiration time for sessions.