# Secure Session Middleware for Goryu

Enhanced session middleware with enterprise-grade security features that integrates seamlessly with the auth middleware.

## Features

### 🔐 Advanced Security
- **AES-256 Encryption**: All session data is encrypted at rest
- **Session Fingerprinting**: Detects and prevents session hijacking
- **IP Binding**: Optional IP address validation
- **Automatic Rotation**: Prevents session fixation attacks
- **Anomaly Detection**: Identifies suspicious session patterns

### 🛡️ Attack Prevention
- **Session Fixation**: Automatic ID regeneration on login
- **Session Hijacking**: Client fingerprinting and IP validation
- **Replay Attacks**: HMAC-signed session tokens
- **CSRF Protection**: Integrates with existing CSRF middleware
- **Timing Attacks**: Constant-time token validation

### ⚡ Performance & Reliability
- **Encrypted Storage**: Fast in-memory store with encryption
- **Automatic Cleanup**: Expired session removal
- **Concurrent Safety**: Thread-safe operations
- **Configurable Limits**: Size and count restrictions

### 🔄 Auth Integration
- **Seamless Integration**: Works perfectly with auth middleware
- **Session Sync**: Automatic user session management
- **Privilege Escalation**: Secure handling of permission changes
- **Activity Tracking**: Monitor user activity patterns

## Quick Start

```go
package main

import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/middleware/session"
    "github.com/arthurlch/goryu/middleware/auth"
)

func main() {
    app := goryu.New()
    
    // Generate secure keys
    sessionKey, _ := auth.GenerateSecureKey()
    
    // Create encrypted session store
    store, _ := session.NewSecureStore(sessionKey,
        session.WithMaxAge(24*time.Hour),
        session.WithFingerprinting("User-Agent", "Accept-Language"),
    )
    
    // Apply session middleware
    app.Use(session.New(session.Config{
        Store:      store,
        CookieName: "secure_session",
        Secure:     true,
        SameSite:   http.SameSiteStrictMode,
    }))
    
    // Apply security features
    app.Use(session.SecureSessionMiddleware(session.DefaultSecurityConfig()))
    
    app.Listen(":8080")
}
```

## Security Configuration

```go
config := session.SecurityConfig{
    // Session rotation
    RotateOnLogin:          true,
    RotateOnPrivilegeChange: true,
    RotationInterval:       1 * time.Hour,
    
    // IP security
    BindToIP:       true,
    AllowIPChange:  false,
    TrustedProxies: []string{"127.0.0.1/8"},
    
    // Activity monitoring
    TrackActivity:   true,
    IdleTimeout:     30 * time.Minute,
    AbsoluteTimeout: 24 * time.Hour,
    
    // Anomaly detection
    DetectAnomalies:    true,
    MaxSessionsPerUser: 5,
    MaxSessionsPerIP:   20,
}
```

## Integration with Auth Middleware

```go
// Full setup with auth integration
authService, integration := session.CreateIntegratedAuthSetup(
    app, 
    jwtSecret, 
    sessionKey,
)

// Protected routes require both auth and session
protected := app.Group("/api", session.RequireSession())

protected.GET("/profile", func(c *goryu.Ctx) {
    user, _ := session.GetSessionUser(c)
    c.JSON(200, user)
})
```

## Session Management

### Creating Sessions
```go
session, _ := session.Get(c)
session.Set("user_id", userID)
session.Set("roles", []string{"user", "admin"})
```

### Reading Sessions
```go
session, _ := session.Get(c)
userID := session.Get("user_id")
roles := session.Get("roles")
```

### Destroying Sessions
```go
// Properly destroy session (e.g., on logout)
session.Destroy(c)
```

### Session Regeneration
```go
// Regenerate session ID (important after login)
session.Regenerate(c)
```

## Security Best Practices

### 1. Always Regenerate on Authentication State Change
```go
// On login
if loginSuccessful {
    session.Regenerate(c)
    session.Set("user_id", user.ID)
}

// On privilege escalation
if err := session.OnPrivilegeEscalation(c, config); err != nil {
    // Handle error
}
```

### 2. Use Fingerprinting in High-Security Environments
```go
store, _ := session.NewSecureStore(key,
    session.WithFingerprinting(
        "User-Agent",
        "Accept-Language",
        "Accept-Encoding",
    ),
)
```

### 3. Implement Proper Logout
```go
app.POST("/logout", func(c *goryu.Ctx) {
    // Destroy session
    session.Destroy(c)
    
    // Clear auth tokens
    // ... auth cleanup ...
    
    c.JSON(200, map[string]string{"message": "Logged out"})
})
```

### 4. Monitor for Anomalies
```go
detector := session.NewSessionAnomalyDetector(config)

// Check for suspicious patterns
if err := detector.CheckAnomaly(userID, sessionID, clientIP); err != nil {
    // Log security event
    // Terminate suspicious session
    session.Destroy(c)
}
```

## Advanced Features

### Secure Temporary Tokens
```go
// Generate HMAC-signed session token
token := session.GenerateSessionToken(sessionID, secret)

// Validate token
validID, err := session.ValidateSessionToken(token, secret)
```

### Activity Tracking
```go
session, _ := session.Get(c)
loginTime := session.Get("login_time")
lastActivity := session.Get("last_activity")
loginCount := session.Get("login_count")
```

### IP Change Detection
```go
// Sessions can track IP changes
ipChanges := session.Get("ip_changes")
// Returns: [{from: "1.2.3.4", to: "5.6.7.8", time: 1234567890}, ...]
```

## Testing

```go
// Create test store
store, _ := session.NewSecureStore("test-key-32-chars-minimum")

// Test with mock sessions
testSession := &session.Session{
    ID: "test-123",
    Data: map[string]any{
        "user_id": "user-123",
    },
}

store.Save(testSession)
retrieved, _ := store.Get("test-123")
```

## Performance Considerations

- **Encryption Overhead**: ~0.1ms per operation
- **Memory Usage**: ~1KB per active session
- **Cleanup Interval**: Configurable (default: 1 hour)
- **Max Session Size**: Configurable (default: 1MB)

## Security Checklist

- ✅ Use HTTPS in production (`Secure: true`)
- ✅ Set appropriate `SameSite` policy
- ✅ Enable session rotation
- ✅ Configure timeouts appropriately
- ✅ Use strong encryption keys (min 32 chars)
- ✅ Enable fingerprinting for high-security apps
- ✅ Monitor for anomalies
- ✅ Implement proper logout
- ✅ Test session security regularly

## Common Pitfalls

1. **Not regenerating session IDs**: Always regenerate after login
2. **Weak encryption keys**: Use `auth.GenerateSecureKey()`
3. **No timeout configuration**: Set both idle and absolute timeouts
4. **Missing CSRF protection**: Always use with CSRF middleware
5. **Ignoring anomalies**: Monitor and respond to suspicious patterns

## License

Same as Goryu framework.