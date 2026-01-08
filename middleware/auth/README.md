# 🔐 Authentication Middleware

A comprehensive, production-ready authentication service for Goryu applications. This package provides a complete solution for user management, JWT-based authentication, and security best practices.

## ✨ Features

*   **JWT Authentication**: Secure access and refresh token management.
*   **User Management**: Registration, login, profile updates, and account deletion.
*   **Password Security**: Strong password enforcement (Bcrypt) and secure hashing.
*   **Email Verification**: Token-based email verification flow.
*   **Password Reset**: Secure password reset workflow.
*   **🛡️ Security First**:
    *   **Rate Limiting**: Built-in protection against brute-force attacks by IP and Email.
    *   **Audit Logging**: Detailed security event logging (login failures, password changes).
    *   **Secure Cookies**: HttpOnly, Secure, SameSite=Strict cookie management.
    *   **CSRF Protection**: Integration ready.

---

## 🚀 Quick Start

### 1. Install

```bash
go get github.com/arthurlch/goryu/middleware/auth
```

### 2. Implementation

To use this middleware, you need to implement three interfaces: `UserStore`, `TokenStore`, and `EmailSender`.

```go
package main

import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/middleware/auth"
)

func main() {
    app := goryu.New()

    // 1. Setup Dependencies
    // (See "Implementing Stores" section below for full code)
    jwtAuth, _  := auth.NewJWTAuth("your-secret-key-32-chars", "my-app")
    userStore   := NewInMemoryUserStore()
    tokenStore  := NewInMemoryTokenStore()
    emailSender := NewMockEmailSender()

    // 2. Configure Service
    config := auth.DefaultAuthServiceConfig()
    config.AppName = "My Super App"
    config.RequireEmailVerification = true 

    // 3. Initialize Service & Handlers
    authService := auth.NewAuthService(jwtAuth, userStore, tokenStore, emailSender, config)
    authService.SetLogger(auth.NewSimpleLogger()) // Optional: Custom logger
    
    handlers := auth.NewAuthHandlers(authService)

    // 4. Register Routes
    handlers.RegisterRoutes(app)

    app.Listen(":8080")
}
```

---

## ⚙️ Configuration

The `AuthServiceConfig` struct gives you granular control over security policies.

```go
type AuthServiceConfig struct {
    AppNameStr              string
    RequireEmailVerification bool          // Must verify email before login
    PasswordConfig          SecurePasswordConfig
    SessionDuration         time.Duration  // Access Token TTL (e.g. 15m)
    RefreshTokenDuration    time.Duration  // Refresh Token TTL (e.g. 7 days)
    
    // Security Policies
    EnableRateLimit         bool
    MaxLoginAttempts        int            // e.g. 5 attempts
    RateLimitWindow         time.Duration  // e.g. 15 minutes
    RateLimitBlockDuration  time.Duration  // e.g. 30 minutes block
    
    // Cookie Config
    SecureCookies           bool           // Enforce HTTPS
    CookieDomain            string
    CookiePath              string
    CSRFProtection          bool
}
```

---

## 📡 API Reference

The `RegisterRoutes` method sets up a complete auth API.

### Public Endpoints

| Method | Path | Description | Request Body |
|--------|------|-------------|--------------|
| `POST` | `/auth/register` | Create account | `{ "email": "...", "password": "...", "first_name": "..." }` |
| `POST` | `/auth/login` | Login | `{ "email": "...", "password": "...", "remember_me": true }` |
| `POST` | `/auth/forgot-password` | Request Reset | `{ "email": "..." }` |
| `GET` | `/auth/verify-email` | Verify Email | Query: `?token=...` |

### Protected Endpoints (Requires Auth)

| Method | Path | Description | Note |
|--------|------|-------------|------|
| `POST` | `/auth/logout` | Logout | Clears cookies / invalidates refresh token |
| `POST` | `/auth/refresh` | Refresh Token | Uses `refresh_token` cookie or body |
| `GET` | `/auth/profile` | Get User Info | Returns user ID, email, traits |
| `POST` | `/auth/change-password` | Update PW | `{ "current_password": "...", "new_password": "..." }` |

---

## 🛠️ Implementing Stores

You need to implement these interfaces to connect Goryu to your database (Postgres, MongoDB, Redis, etc.).

### 1. UserStore

Manages user persistence.

```go
type UserStore interface {
    AddUser(email, password string, traits map[string]interface{}) (*User, error)
    GetUserByEmail(email string) (*User, bool)
    GetUserByID(id string) (*User, bool)
    UpdatePassword(email, newPassword string) error
    VerifyUserEmail(email string) error
    UpdateUserTraits(id string, traits map[string]interface{}) error
    DeleteUser(id string) error
}
```

### 2. TokenStore

Manages Token allowlist/blocklist (usually Redis).

```go
type TokenStore interface {
    AddToken(jti string) error
    UseToken(jti string) bool // Returns true if token was unused and is now marked used
    IsTokenUsed(jti string) bool
}
```

### 3. EmailSender

Sends transactional emails.

```go
type EmailSender interface {
    SendVerificationEmail(email, token, verifyURL string) error
    SendPasswordResetEmail(email, token, resetURL string) error
    SendSecurityAlert(email, message string) error
}
```

---

## 🔒 Rate Limiting Strategy

The middleware implements a dual-layer rate limiter to prevent abuse:

1.  **IP-based**: Limits total login attempts from a single IP address.
2.  **Email-based**: Limits failed attempts for a specific email account (prevents targeted brute-force).

If limits are exceeded, the user receives a 429 Too Many Requests response with a `Retry-After` header.

---

## 💻 Full Runnable Example

<details>
<summary>Click to view <code>example.go</code></summary>

```go
package main

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/middleware/auth"
)

func main() {
	ExampleUsage()
}

func SetupAuthMiddleware(app *goryu.App, secretKey string) (*auth.AuthService, *auth.AuthHandlers) {
	jwtAuth, err := auth.NewJWTAuth(secretKey, "goryu-app")
	if err != nil {
		panic("Failed to create JWT Auth: " + err.Error())
	}
	
    // These implementations are included in the package for rapid prototyping
	userStore := auth.NewInMemoryUserStore()
	tokenStore := auth.NewInMemoryTokenStore()
	emailSender := auth.NewMockEmailSender()
	
	config := auth.DefaultAuthServiceConfig()
	
	authService := auth.NewAuthService(jwtAuth, userStore, tokenStore, emailSender, config)
	logger := auth.NewSimpleLogger()
	authService.SetLogger(logger)
	
	authHandlers := auth.NewAuthHandlers(authService)
	authHandlers.RegisterRoutes(app)
	return authService, authHandlers
}

func ExampleUsage() {
	app := goryu.New()
	secretKey := generateSecureKey()
	
	authService, _ := SetupAuthMiddleware(app, secretKey)
	defer authService.Cleanup()
	
	app.GET("/protected", func(c *goryu.Ctx) {
		userID, _ := c.Get(auth.UserIDKey)
		c.JSON(200, map[string]interface{}{
			"message": "This is a protected route",
			"user_id": userID,
		})
	})
	
	adminGroup := app.Group("/admin")
	adminGroup.GET("/users", func(c *goryu.Ctx) {
		userID, exists := c.Get(auth.UserIDKey)
		if !exists || userID == nil {
			c.JSON(401, map[string]string{"error": "Authentication required"})
			return
		}
		c.JSON(200, map[string]string{"message": "Admin users endpoint"})
	})
	
	app.Listen(":8080")
}

func generateSecureKey() string {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic("Failed to generate secure key: " + err.Error())
	}
	return hex.EncodeToString(key)
}
```
</details>

<details>
<summary>Click to view <code>example_implementations.go</code> (Mock Helpers)</summary>

```go
package auth

import (
	"fmt"
	"log"
)

type MockEmailSender struct {
	SentEmails []EmailRecord
}
type EmailRecord struct {
	To      string
	Subject string
	Content string
	Type    string
}

func NewMockEmailSender() *MockEmailSender {
	return &MockEmailSender{
		SentEmails: make([]EmailRecord, 0),
	}
}
func (m *MockEmailSender) SendVerificationEmail(email, token, verifyURL string) error {
	m.SentEmails = append(m.SentEmails, EmailRecord{
		To:      email,
		Subject: "Verify Your Email Address",
		Content: fmt.Sprintf("Please verify your email by clicking: %s", verifyURL),
		Type:    "verification",
	})
	log.Printf("MOCK EMAIL: Verification email sent to %s with URL: %s", email, verifyURL)
	return nil
}
func (m *MockEmailSender) SendPasswordResetEmail(email, token, resetURL string) error {
	m.SentEmails = append(m.SentEmails, EmailRecord{
		To:      email,
		Subject: "Reset Your Password",
		Content: fmt.Sprintf("Reset your password by clicking: %s", resetURL),
		Type:    "password_reset",
	})
	log.Printf("MOCK EMAIL: Password reset email sent to %s with URL: %s", email, resetURL)
	return nil
}
func (m *MockEmailSender) SendSecurityAlert(email, message string) error {
	m.SentEmails = append(m.SentEmails, EmailRecord{
		To:      email,
		Subject: "Security Alert",
		Content: message,
		Type:    "security_alert",
	})
	log.Printf("MOCK EMAIL: Security alert sent to %s: %s", email, message)
	return nil
}
func (m *MockEmailSender) GetSentEmails() []EmailRecord {
	return m.SentEmails
}
func (m *MockEmailSender) ClearSentEmails() {
	m.SentEmails = make([]EmailRecord, 0)
}

type SimpleLogger struct {
	EnableSecurityLog bool
	EnableErrorLog    bool
	EnableInfoLog     bool
}

func NewSimpleLogger() *SimpleLogger {
	return &SimpleLogger{
		EnableSecurityLog: true,
		EnableErrorLog:    true,
		EnableInfoLog:     true,
	}
}
func (l *SimpleLogger) LogSecurityEvent(event string, details map[string]interface{}) {
	if !l.EnableSecurityLog {
		return
	}
	log.Printf("SECURITY EVENT: %s - Details: %+v", event, details)
}
func (l *SimpleLogger) LogError(message string, err error) {
	if !l.EnableErrorLog {
		return
	}
	log.Printf("ERROR: %s - %v", message, err)
}
func (l *SimpleLogger) LogInfo(message string, details map[string]interface{}) {
	if !l.EnableInfoLog {
		return
	}
	log.Printf("INFO: %s - Details: %+v", message, details)
}
```
</details>
