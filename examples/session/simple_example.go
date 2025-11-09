package main
import (
	"log"
	"net/http"
	"time"
	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/internal/utils"
	"github.com/arthurlch/goryu/middleware/session"
)
func main() {
	app := goryu.New()
	sessionKey, err := utils.GenerateSecureKey32()
	if err != nil {
		log.Fatal("Failed to generate session key:", err)
	}
	sessionStore, err := session.NewSecureStore(sessionKey,
		session.WithMaxAge(24*time.Hour),
		session.WithFingerprinting("User-Agent", "Accept-Language"),
	)
	if err != nil {
		panic("Failed to create session store: " + err.Error())
	}
	defer sessionStore.Stop()
	sessionConfig := session.Config{
		Store:      sessionStore,
		CookieName: "secure_session",
		Expiration: 24 * time.Hour,
		Secure:     boolPtr(true),
		SameSite:   http.SameSiteStrictMode,
		Path:       "/",
	}
	app.Use(session.New(sessionConfig))
	integration := session.NewAuthIntegration(&sessionConfig, nil)
	_ = integration 
	app.GET("/", func(c *context.Context) {
		c.JSON(200, map[string]string{"message": "Welcome"})
	})
	app.GET("/profile", func(c *context.Context) {
		sess, err := session.Get(c)
		if err != nil || sess.Get("user_id") == nil {
			c.JSON(401, map[string]string{"error": "Valid session required"})
			return
		}
		user := struct {
			ID           string
			LoginTime    time.Time
			LastActivity time.Time
		}{
			ID:           "example-user",
			LoginTime:    time.Now().Add(-1 * time.Hour),
			LastActivity: time.Now(),
		}
		c.JSON(200, map[string]interface{}{
			"user_id":      user.ID,
			"login_time":   user.LoginTime,
			"last_activity": user.LastActivity,
		})
	})
	app.GET("/session", func(c *context.Context) {
		sess, err := session.Get(c)
		if err != nil {
			c.JSON(500, map[string]string{"error": "Failed to get session"})
			return
		}
		info := map[string]interface{}{
			"id":         sess.ID,
			"created_at": sess.Get("created_at"),
			"user_id":    sess.Get("user_id"),
		}
		c.JSON(200, info)
	})
	app.Listen(":8080")
}
func boolPtr(b bool) *bool {
	return &b
}

