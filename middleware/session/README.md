# 🔐 Session Middleware

A flexible, secure, and production-ready session management middleware for Goryu. It provides encrypted session storage, advanced security features (fingerprinting, anomaly detection), and seamless integration with the Auth package.

## ✨ Features

*   **🔒 Secure by Default**: AES-GCM encryption for all session data.
*   **💾 Pluggable Storage**: Interfaces for Memory, Redis, or Database storage.
*   **🛡️ Advanced Protection**:
    *   **Fingerprinting**: Binds sessions to User-Agent and IPs to prevent hijacking.
    *   **Anomaly Detection**: Tracks IP changes and multiple concurrent sessions.
    *   **Rotation**: Automatic session ID rotation to prevent fixation.
*   **🚦 strict Cookie Policy**: Enforces `HttpOnly`, `Secure`, and `SameSite`.

---

## 🚀 Quick Start

### 1. Basic Setup

```go
package main

import (
    "time"
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/middleware/session"
)

func main() {
    app := goryu.New()

    // 1. Create a Secure Store (In-Memory for this example)
    // In production, use a persistent store (e.g. Redis)
    store, _ := session.NewSecureStore(
        "your-32-byte-secret-key-keep-safe", 
        session.WithMaxAge(24 * time.Hour),
    )

    // 2. Register Middleware
    app.Use(session.New(session.Config{
        Store:      store,
        CookieName: "sid",
        Expiration: 24 * time.Hour,
        Secure:     boolPtr(true), // Enforce HTTPS
    }))

    // 3. Use in Routes
    app.GET("/login", func(c *goryu.Ctx) {
        sess, _ := session.Get(c)
        sess.Set("user_id", "123")
        c.String(200, "Logged in")
    })

    app.Listen(":8080")
}

func boolPtr(b bool) *bool { return &b }
```

---

## ⚙️ Configuration

### Core Config (`session.Config`)

| Field | Type | Description |
|-------|------|-------------|
| `Store` | `Store` | Storage backend (Required). |
| `CookieName` | `string` | Name of the cookie (default: "goryu_session"). |
| `Expiration` | `Duration` | Session TTL. |
| `Secure` | `*bool` | Enforce HTTPS (Recommended: true). |
| `HttpOnly` | `bool` | Prevent JS access (Recommended: true). |
| `SameSite` | `SameSite` | CSRF protection (Recommended: Strict). |

### Security Config (`session.SecurityConfig`)

For high-security apps, use `SecureSessionMiddleware` wrapper:

```go
type SecurityConfig struct {
    RotateOnLogin           bool          // Re-issue ID on privilege change
    BindToIP                bool          // Lock session to IP
    BindToUserAgent         bool          // Lock session to Browser
    AllowIPChange           bool          // Allow roaming (e.g. mobile)
    DetectAnomalies         bool          // Check for multi-session abuse
    MaxSessionsPerUser      int           // Limit concurrent sessions
    IdleTimeout             time.Duration // Auto-logout on inactivity
    AbsoluteTimeout         time.Duration // Force re-login after N hours
}
```

---

## 🛡️ Advanced Security

### Session Fixation Protection
Always regenerate the session ID after a user logs in (privilege escalation).

```go
app.POST("/login", func(c *goryu.Ctx) {
    // ... verify credentials ...
    
    // 🛑 Critical: Prevent Session Fixation
    if err := session.Regenerate(c); err != nil {
        return c.Status(500).Send("Error")
    }
    
    sess, _ := session.Get(c)
    sess.Set("user_id", user.ID)
})
```

### Anomaly Detection
The middleware can detect if a single user has too many active sessions or if a session is hopping IPs suspiciously.

```go
// Add the Security Middleware layer
app.Use(session.SecureSessionMiddleware(session.SecurityConfig{
    BindToIP:        true,
    DetectAnomalies: true,
    MaxSessionsPerUser: 5,
}))
```

---

## 💻 Full Integrated Example

This example demonstrates a complete secure setup including key generation, encryption, CSRF protection, and anomaly detection.

<details>
<summary>Click to view <code>example_secure.go</code></summary>

```go
package session

import (
	"net/http"
	"time"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/middleware/auth"
	"github.com/arthurlch/goryu/middleware/csrf"
)

// SecureSessionExample demonstrates secure session setup with auth integration
func SecureSessionExample() {
	app := goryu.New()
	
	// 1. Generate secure keys
	jwtSecret, _ := auth.GenerateSecureKey()
	sessionKey, _ := auth.GenerateSecureKey()
	csrfSecret, _ := auth.GenerateSecureKey()
	
	// 2. Create secure session store with encryption
	sessionStore, err := NewSecureStore(sessionKey,
		WithMaxAge(24*time.Hour),
		WithMaxSize(1024*1024), // 1MB
		WithFingerprinting("User-Agent", "Accept-Language"), // Prevent hijacking
	)
	if err != nil {
		panic("Failed to create session store: " + err.Error())
	}
	defer sessionStore.Stop()
	
	// 3. Session configuration with security defaults (will be used by CreateIntegratedAuthSetup)
	_ = sessionStore // Store will be used by integration setup
	
	// 4. Apply CSRF middleware (works with sessions)
	app.Use(csrf.New(csrf.Config{
		TokenHeader: "X-CSRF-Token",
		TokenCookie: "csrf_token",
		Secure:      true,
		SameSite:    http.SameSiteStrictMode,
	}))
	
	// 5. Setup auth with session integration (includes session and security middleware)
	authService, _ := CreateIntegratedAuthSetup(app, jwtSecret, sessionKey)
	defer authService.Cleanup()
	
	// Apply additional security middleware
	app.Use(SecureSessionMiddleware(DefaultSecurityConfig()))
	
	// 6. Example routes showing session usage
	
	// Public route - no session required
	app.GET("/", func(c *goryu.Ctx) {
		c.JSON(200, map[string]string{"message": "Welcome to secure app"})
	})
	
	// Login with session management - note: auth routes are already set up by CreateIntegratedAuthSetup
	// This is just for demonstration of how to wrap handlers manually
	loginWithSession := func(c *goryu.Ctx) {
		// Apply RequireNoSession check inline
		session, err := Get(c)
		if err == nil && session.Get("user_id") != nil {
			c.Redirect(302, "/dashboard")
			return
		}
		// Since auth routes are auto-registered, this is just an example
		c.JSON(200, map[string]string{"message": "Login endpoint - use POST /auth/login"})
	}
	app.GET("/login-demo", loginWithSession)
	
	// Protected routes requiring valid session
	// Note: Groups don't have Use method, apply middleware to individual routes
	requireSessionMW := RequireSession()
	
	protected.GET("/profile", func(c *goryu.Ctx) {
		// Get user from session
		user, err := GetSessionUser(c)
		if err != nil {
			c.JSON(401, map[string]string{"error": "Invalid session"})
			return
		}
		
		// Get session data
		session, _ := Get(c)
		loginCount := 0
		if count := session.Get("login_count"); count != nil {
			loginCount = count.(int)
		}
		
		c.JSON(200, map[string]interface{}{
			"user_id":      user.ID,
			"login_time":   user.LoginTime,
			"last_activity": user.LastActivity,
			"login_count":  loginCount,
		})
	})
	
	// Admin route with privilege escalation
	protected.POST("/admin/action", func(c *goryu.Ctx) {
		// Require recent authentication for sensitive actions
		session, _ := Get(c)
		if escalated := session.Get("privilege_escalated_at"); escalated != nil {
			if escalatedTime, ok := escalated.(int64); ok {
				// Require re-authentication after 5 minutes for sensitive actions
				if time.Since(time.Unix(escalatedTime, 0)) > 5*time.Minute {
					c.JSON(401, map[string]string{"error": "Please re-authenticate for this action"})
					return
				}
			}
		}
		
		// Perform sensitive action
		c.JSON(200, map[string]string{"message": "Admin action completed"})
	})
	
	// Logout with proper session cleanup - note: auth routes auto-registered
	// This demonstrates custom logout logic
	protected.POST("/logout", func(c *goryu.Ctx) {
		// Destroy session
		if err := Destroy(c); err != nil {
			c.JSON(500, map[string]string{"error": "Failed to destroy session"})
			return
		}
		c.JSON(200, map[string]string{"message": "Logged out successfully"})
	})
	
	// Session info endpoint
	protected.GET("/session/info", func(c *goryu.Ctx) {
		session, err := Get(c)
		if err != nil {
			c.JSON(500, map[string]string{"error": "Failed to get session"})
			return
		}
		
		info := map[string]interface{}{
			"id":         session.ID,
			"created_at": session.Get("created_at"),
			"ip_bound":   session.Get("bound_ip"),
			"rotated_at": session.Get("rotated_at"),
		}
		
		// Show IP changes if any
		if changes := session.Get("ip_changes"); changes != nil {
			info["ip_changes"] = changes
		}
		
		c.JSON(200, info)
	})
	
	// Start server
	app.Listen(":8080")
}

// AdvancedSecurityExample shows advanced security features
func AdvancedSecurityExample() {
	app := goryu.New()
	
	// Create anomaly detector
	anomalyDetector := NewSessionAnomalyDetector(SecurityConfig{
		MaxSessionsPerUser: 3,
		MaxSessionsPerIP:   10,
	})
	
	// Custom session validation middleware
	anomalyMiddleware := func(next goryu.Handler) goryu.Handler {
		return func(c *goryu.Ctx) {
			session, err := Get(c)
			if err != nil {
				next(c)
				return
			}
			
			// Check for anomalies
			if userID := session.Get("user_id"); userID != nil {
				clientIP := getClientIP(c, []string{"127.0.0.1", "::1"})
				if err := anomalyDetector.CheckAnomaly(
					userID.(string),
					session.ID,
					clientIP,
				); err != nil {
					// Potential attack detected
					Destroy(c)
					c.JSON(401, map[string]string{
						"error": "Security violation: " + err.Error(),
					})
					return
				}
			}
			
			next(c)
		}
	}
	app.Use(anomalyMiddleware)
	
	// Privilege escalation example with inline session check
	sudoHandler := func(c *goryu.Ctx) {
		// Check session inline
		session, err := Get(c)
		if err != nil || session.Get("user_id") == nil {
			c.JSON(401, map[string]string{"error": "Valid session required"})
			return
		}
		// Verify password for privilege escalation
		var req struct {
			Password string `json:"password"`
		}
		
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, map[string]string{"error": "Invalid request"})
			return
		}
		
		// Verify password (integrate with auth service)
		// ... password verification logic ...
		
		// On success, escalate privileges
		if err := OnPrivilegeEscalation(c, DefaultSecurityConfig()); err != nil {
			c.JSON(500, map[string]string{"error": "Failed to escalate privileges"})
			return
		}
		
		c.JSON(200, map[string]string{"message": "Privileges escalated"})
	}
	app.POST("/sudo", sudoHandler)
	
	app.Listen(":8080")
}

// SessionMetricsExample shows how to track session metrics
func SessionMetricsExample() {
	app := goryu.New()
	
	type SessionMetrics struct {
		ActiveSessions   int
		TotalLogins      int
		FailedLogins     int
		SessionsCreated  int
		SessionsDestroyed int
	}
	
	metrics := &SessionMetrics{}
	
	// Track session creation
	app.Use(func(next goryu.Handler) goryu.Handler {
		return func(c *goryu.Ctx) {
			// Before
			session, _ := Get(c)
			isNew := session.Get("created_at") == nil
			
			next(c)
			
			// After
			if isNew {
				metrics.SessionsCreated++
				metrics.ActiveSessions++
			}
		}
	})
	
	// Metrics endpoint
	app.GET("/metrics/sessions", func(c *goryu.Ctx) {
		c.JSON(200, metrics)
	})
	
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}
```
</details>