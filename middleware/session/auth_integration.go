package session

import (
	"fmt"
	"net/http"
	"time"

	"github.com/arthurlch/goryu"
	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/auth"
)
type AuthIntegration struct {
	sessionConfig *Config
	authService   *auth.AuthService
}
func NewAuthIntegration(sessionConfig *Config, authService *auth.AuthService) *AuthIntegration {
	return &AuthIntegration{
		sessionConfig: sessionConfig,
		authService:   authService,
	}
}
func (ai *AuthIntegration) WrapLoginHandler(originalHandler context.HandlerFunc) context.HandlerFunc {
	return func(c *context.Context) {
		wrapped := &responseWriter{ResponseWriter: c.Writer, status: http.StatusOK}
		c.Writer = wrapped
		originalHandler(c)
		if wrapped.status == http.StatusOK {
			if err := Regenerate(c); err != nil {
				if ai.sessionConfig.Logger != nil {
					ai.sessionConfig.Logger.Printf("Failed to regenerate session on login: %v", err)
				}
			}
			if userID, exists := c.Get(auth.UserIDKey); exists {
				session, err := Get(c)
				if err == nil {
					session.Set("user_id", userID)
					session.Set("login_time", time.Now().Unix())
					session.Set("last_activity", time.Now().Unix())
					if store, ok := ai.sessionConfig.Store.(*SecureStore); ok && store.enableFingerprinting {
						headers := make(map[string]string)
						for _, field := range store.fingerprintFields {
							headers[field] = c.GetHeader(field)
						}
						fingerprint := GenerateFingerprint(headers, store.fingerprintFields)
						store.SaveWithFingerprint(session, fingerprint)
					}
				}
			}
		}
	}
}
func (ai *AuthIntegration) WrapLogoutHandler(originalHandler context.HandlerFunc) context.HandlerFunc {
	return func(c *context.Context) {
		if err := Destroy(c); err != nil {
			if ai.sessionConfig.Logger != nil {
				ai.sessionConfig.Logger.Printf("Failed to destroy session on logout: %v", err)
			}
		}
		originalHandler(c)
	}
}
func SessionAuthMiddleware(sessionConfig *Config) func(next context.HandlerFunc) context.HandlerFunc {
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			session, err := Get(c)
			if err != nil {
				next(c)
				return
			}
			if userID := session.Get("user_id"); userID != nil {
				session.Set("last_activity", time.Now().Unix())
				if lastActivity := session.Get("last_activity"); lastActivity != nil {
					if lastActivityTime, ok := lastActivity.(int64); ok {
						idleTimeout := 30 * time.Minute 
						if time.Since(time.Unix(lastActivityTime, 0)) > idleTimeout {
							Destroy(c)
							c.JSON(401, map[string]string{"error": "Session expired due to inactivity"})
							return
						}
					}
				}
				c.Set(auth.UserIDKey, userID)
			}
			if store, ok := sessionConfig.Store.(*SecureStore); ok && store.enableFingerprinting {
				headers := make(map[string]string)
				for _, field := range store.fingerprintFields {
					headers[field] = c.GetHeader(field)
				}
				fingerprint := GenerateFingerprint(headers, store.fingerprintFields)
				if err := store.ValidateFingerprint(session.ID, fingerprint); err != nil {
					Destroy(c)
					if sessionConfig.Logger != nil {
						sessionConfig.Logger.Printf("Session hijacking detected for session %s: %v", session.ID, err)
					}
					c.JSON(401, map[string]string{"error": "Session security violation"})
					return
				}
			}
			next(c)
		}
	}
}
type responseWriter struct {
	http.ResponseWriter
	status int
}
func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}
func CreateIntegratedAuthSetup(app *goryu.App, jwtSecret, sessionKey string) (*auth.AuthService, *AuthIntegration) {
	sessionStore, err := NewSecureStore(sessionKey, 
		WithMaxAge(24*time.Hour),
		WithFingerprinting("User-Agent", "Accept-Language"),
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to create session store: %v", err))
	}
	sessionConfig := &Config{
		Store:      sessionStore,
		CookieName: "goryu_secure_session",
		Expiration: 24 * time.Hour,
		Secure:     func(b bool) *bool { return &b }(true),
		SameSite:   http.SameSiteStrictMode,
		Path:       "/",
	}
	app.Use(New(*sessionConfig))
	authService, _ := auth.SetupAuthMiddleware(app, jwtSecret)
	integration := NewAuthIntegration(sessionConfig, authService)
	app.Use(SessionAuthMiddleware(sessionConfig))
	return authService, integration
}
type SessionUser struct {
	ID           string
	Email        string
	LoginTime    time.Time
	LastActivity time.Time
}
func GetSessionUser(c *context.Context) (*SessionUser, error) {
	session, err := Get(c)
	if err != nil {
		return nil, err
	}
	userID := session.Get("user_id")
	if userID == nil {
		return nil, fmt.Errorf("no user in session")
	}
	user := &SessionUser{
		ID: fmt.Sprintf("%v", userID),
	}
	if loginTime := session.Get("login_time"); loginTime != nil {
		if lt, ok := loginTime.(int64); ok {
			user.LoginTime = time.Unix(lt, 0)
		}
	}
	if lastActivity := session.Get("last_activity"); lastActivity != nil {
		if la, ok := lastActivity.(int64); ok {
			user.LastActivity = time.Unix(la, 0)
		}
	}
	return user, nil
}
func RequireSession() func(next context.HandlerFunc) context.HandlerFunc {
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			session, err := Get(c)
			if err != nil || session.Get("user_id") == nil {
				c.JSON(401, map[string]string{"error": "Valid session required"})
				return
			}
			next(c)
		}
	}
}
func RequireNoSession() func(next context.HandlerFunc) context.HandlerFunc {
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			session, err := Get(c)
			if err == nil && session.Get("user_id") != nil {
				c.Redirect(302, "/dashboard")
				return
			}
			next(c)
		}
	}
}