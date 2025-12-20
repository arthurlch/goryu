package auth

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/arthurlch/goryu"
)
func SetupAuthMiddleware(app *goryu.App, secretKey string) (*AuthService, *AuthHandlers) {
	jwtAuth, err := NewJWTAuth(secretKey, "goryu-app")
	if err != nil {
		panic("Failed to create JWT Auth: " + err.Error())
	}
	userStore := NewInMemoryUserStore()
	tokenStore := NewInMemoryTokenStore()
	emailSender := NewMockEmailSender()
	config := DefaultAuthServiceConfig()
	authService := NewAuthService(jwtAuth, userStore, tokenStore, emailSender, config)
	logger := NewSimpleLogger()
	authService.SetLogger(logger)
	authHandlers := NewAuthHandlers(authService)
	authHandlers.RegisterRoutes(app)
	return authService, authHandlers
}
func ExampleUsage() {
	app := goryu.New()
	secretKey := generateSecureKey()
	authService, _ := SetupAuthMiddleware(app, secretKey)
	defer authService.Cleanup()
	app.GET("/protected", func(c *goryu.Ctx) {
		userID, _ := c.Get(UserIDKey)
		c.JSON(200, map[string]interface{}{
			"message": "This is a protected route",
			"user_id": userID,
		})
	})
	adminGroup := app.Group("/admin")
	adminGroup.GET("/users", func(c *goryu.Ctx) {
		userID, exists := c.Get(UserIDKey)
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
func ProductionConfig() AuthServiceConfig {
	return AuthServiceConfig{
		AppName:                  "your-app-name",
		RequireEmailVerification: true,
		PasswordConfig:           DefaultPasswordConfig(),
		SessionDuration:          15 * time.Minute,
		RefreshTokenDuration:     7 * 24 * time.Hour,
		EnableRateLimit:          true,
		MaxLoginAttempts:         3,
		RateLimitWindow:          15 * time.Minute,
		RateLimitBlockDuration:   1 * time.Hour,
		EnableAuditLog:           true,
		SecureCookies:            true,
		CookieDomain:             "",
		CookiePath:               "/",
		CSRFProtection:           true,
	}
}