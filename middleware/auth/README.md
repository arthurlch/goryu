# Auth Middleware

A comprehensive, production-ready authentication service for Goryu applications. This package provides a complete solution for user management, JWT-based authentication, and security best practices.

## Features

- **JWT Authentication**: Secure access and refresh token management.
- **User Management**: Registration, login, profile updates, and account deletion.
- **Password Security**: Strong password enforcement and secure hashing.
- **Email Verification**: Token-based email verification flow.
- **Password Reset**: Secure password reset workflow.
- **Rate Limiting**: Built-in protection against brute-force attacks on login and registration.
- **Security Events**: Detailed logging of security-critical events (login failures, password changes, etc.).
- **Secure Cookies**: HttpOnly and Secure cookie support for token storage.

## Usage

### Basic Setup

```go
package main

import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/middleware/auth"
)

func main() {
    app := goryu.New()

    // 1. Initialize dependencies (stores, email sender, etc.)
    jwtAuth := auth.NewJWTAuth("your-secret-key", "your-app-name")
    userStore := NewYourUserStore()   // Implement auth.UserStore
    tokenStore := NewYourTokenStore() // Implement auth.TokenStore
    emailSender := NewYourEmailSender() // Implement auth.EmailSender

    // 2. Configure the service
    config := auth.DefaultAuthServiceConfig()
    config.AppName = "My App"
    config.RequireEmailVerification = true

    // 3. Create the service and handlers
    authService := auth.NewAuthService(jwtAuth, userStore, tokenStore, emailSender, config)
    authHandlers := auth.NewAuthHandlers(authService)

    // 4. Register routes
    authHandlers.RegisterRoutes(app)

    app.Run(":8080")
}
```

### Configuration

The `AuthServiceConfig` struct allows you to customize various aspects of the authentication service:

```go
type AuthServiceConfig struct {
    AppName                string
    RequireEmailVerification bool
    PasswordConfig         SecurePasswordConfig
    SessionDuration        time.Duration
    RefreshTokenDuration   time.Duration
    EnableRateLimit        bool
    MaxLoginAttempts       int
    RateLimitWindow        time.Duration
    RateLimitBlockDuration time.Duration
    EnableAuditLog         bool
    SecureCookies          bool
    CookieDomain           string
    CookiePath             string
    CSRFProtection         bool
}
```

### Interfaces

To use this middleware, you need to implement the following interfaces to connect with your database and email service:

- `UserStore`: For user persistence (Create, Get, Update).
- `TokenStore`: For managing refresh and verification tokens (Add, Use).
- `EmailSender`: For sending transactional emails (Verification, Password Reset).

### API Endpoints

The `RegisterRoutes` method sets up the following endpoints:

**Public:**
- `POST /auth/register`: Create a new account.
- `POST /auth/login`: Authenticate and receive tokens.
- `POST /auth/forgot-password`: Request a password reset link.
- `POST /auth/reset-password`: Reset password using a token.
- `GET /auth/verify-email`: Verify email address using a token.
- `POST /auth/resend-verification`: Resend verification email.

**Protected (Requires Authentication):**
- `POST /auth/logout`: Invalidate tokens and clear cookies.
- `POST /auth/refresh`: Get a new access token using a refresh token.
- `POST /auth/change-password`: Update password.
- `GET /auth/profile`: Get current user profile.
- `PUT /auth/profile`: Update user profile traits.
- `DELETE /auth/account`: Delete user account.

## Security Features

- **Rate Limiting**: Prevents abuse by limiting login attempts by IP and email.
- **Token Rotation**: Refresh tokens are one-time use (optional implementation detail in store).
- **Secure Headers**: Cookies are set with `HttpOnly`, `Secure`, and `SameSite=Strict`.
- **Audit Logging**: Tracks all security-relevant actions for monitoring and forensics.
