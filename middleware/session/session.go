package session
import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/base"
	"github.com/google/uuid"
)
type Session struct {
	ID       string
	Data     map[string]any
	modified bool
}
func (s *Session) Set(key string, value any) {
	s.Data[key] = value
	s.modified = true
}
func (s *Session) Get(key string) any {
	return s.Data[key]
}
type Store interface {
	Get(id string) (*Session, error)
	Save(session *Session) error
	Destroy(id string) error
}
type Config struct {
	base.BaseConfig
	Store Store
	CookieName string
	Expiration time.Duration
	Secure *bool
	SameSite http.SameSite
	Domain string
	Path string
}
func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if c.Store == nil {
		return base.NewConfigError("Store", "cannot be nil")
	}
	if c.CookieName == "" {
		c.CookieName = "goryu_session"
	}
	if c.Expiration == 0 {
		c.Expiration = 24 * time.Hour
	}
	if c.Secure == nil {
		secure := true
		c.Secure = &secure
	}
	if c.SameSite == 0 {
		c.SameSite = http.SameSiteStrictMode
	}
	if c.Path == "" {
		c.Path = "/"
	}
	return nil
}
const (
	sessionKey          = "goryu.session"
	sessionIDKey        = "goryu.session.id"
	sessionCfgKey       = "goryu.session.config"
	sessionDestroyedKey = "goryu.session.destroyed"
)
func New(config Config) func(next context.HandlerFunc) context.HandlerFunc {
	if err := config.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, base.MiddlewareError{
					Middleware: "Session",
					Err:        err,
					StatusCode: http.StatusInternalServerError,
				}, "Session")
			}
		}
	}
	preHandler := func(c *context.Context) error {
		c.Set(sessionCfgKey, &config)
		cookie, err := c.Cookie(config.CookieName)
		var session *Session
		if err == nil {
			sessionID, decodeErr := base64.StdEncoding.DecodeString(cookie.Value)
			if decodeErr == nil {
				s, storeErr := config.Store.Get(string(sessionID))
				if storeErr == nil && s != nil {
					session = s
					c.Set(sessionIDKey, s.ID)
				}
			}
		}
		if session == nil {
			newID := uuid.New().String()
			session = &Session{
				ID:       newID,
				Data:     make(map[string]any),
				modified: true,
			}
			c.Set(sessionIDKey, newID)
		}
		c.Set(sessionKey, session)
		return nil
	}
	postHandler := func(c *context.Context) error {
		if destroyed, _ := c.Get(sessionDestroyedKey); destroyed == true {
			return nil 
		}
		finalSessionVal, exists := c.Get(sessionKey)
		if !exists {
			return nil 
		}
		finalSession, ok := finalSessionVal.(*Session)
		if !ok {
			return errors.New("invalid session type in context")
		}
		if finalSession.modified {
			if err := config.Store.Save(finalSession); err != nil {
				if config.Logger != nil {
					config.Logger.Printf("Error saving session: %v", err)
				}
				return nil 
			}
		}
		cookie := &http.Cookie{
			Name:     config.CookieName,
			Value:    base64.StdEncoding.EncodeToString([]byte(finalSession.ID)),
			Expires:  time.Now().Add(config.Expiration),
			Path:     config.Path,
			Domain:   config.Domain,
			HttpOnly: true,           
			Secure:   *config.Secure, 
			SameSite: config.SameSite, 
		}
		c.SetCookie(cookie)
		return nil
	}
	return base.PostProcessMiddleware("Session", config.BaseConfig, preHandler, postHandler)
}
func Get(c *context.Context) (*Session, error) {
	s, exists := c.Get(sessionKey)
	if !exists {
		return nil, errors.New("session not found in context")
	}
	session, ok := s.(*Session)
	if !ok {
		return nil, errors.New("invalid session type in context")
	}
	return session, nil
}
func Destroy(c *context.Context) error {
	cfgVal, exists := c.Get(sessionCfgKey)
	if !exists {
		return errors.New("session config not found in context")
	}
	cfg, ok := cfgVal.(*Config)
	if !ok {
		return errors.New("session config is of invalid type in context")
	}
	sessionIDVal, exists := c.Get(sessionIDKey)
	if !exists {
		return errors.New("session ID not found in context")
	}
	sessionID, ok := sessionIDVal.(string)
	if !ok {
		return errors.New("session ID is of invalid type in context")
	}
	c.SetCookie(&http.Cookie{
		Name:     cfg.CookieName,
		Value:    "",
		Path:     cfg.Path,
		Domain:   cfg.Domain,
		MaxAge:   -1,
		HttpOnly: true,           
		Secure:   *cfg.Secure,    
		SameSite: cfg.SameSite,   
	})
	c.Set(sessionDestroyedKey, true)
	return cfg.Store.Destroy(sessionID)
}
func Regenerate(c *context.Context) error {
	session, err := Get(c)
	if err != nil {
		return err
	}
	cfgVal, exists := c.Get(sessionCfgKey)
	if !exists {
		return errors.New("session config not found in context")
	}
	cfg, ok := cfgVal.(*Config)
	if !ok {
		return errors.New("session config is of invalid type in context")
	}
	oldSessionIDVal, exists := c.Get(sessionIDKey)
	if !exists {
		return errors.New("session ID not found in context")
	}
	oldSessionID, ok := oldSessionIDVal.(string)
	if !ok {
		return errors.New("session ID is of invalid type in context")
	}
	newSessionID := uuid.New().String()
	newSession := &Session{
		ID:       newSessionID,
		Data:     make(map[string]any),
		modified: true,
	}
	for key, value := range session.Data {
		newSession.Data[key] = value
	}
	err = cfg.Store.Save(newSession)
	if err != nil {
		return fmt.Errorf("failed to save regenerated session: %w", err)
	}
	c.Set(sessionKey, newSession)
	c.Set(sessionIDKey, newSessionID)
	cookie := &http.Cookie{
		Name:     cfg.CookieName,
		Value:    base64.StdEncoding.EncodeToString([]byte(newSessionID)),
		Expires:  time.Now().Add(cfg.Expiration),
		Path:     cfg.Path,
		Domain:   cfg.Domain,
		HttpOnly: true,           
		Secure:   *cfg.Secure,    
		SameSite: cfg.SameSite,   
	}
	err = c.SetCookie(cookie)
	if err != nil {
		if cfg.Logger != nil {
			cfg.Logger.Printf("Warning: failed to set cookie during session regeneration: %v", err)
		}
	}
	go func() {
		time.Sleep(100 * time.Millisecond)
		err := cfg.Store.Destroy(oldSessionID)
		if err != nil && cfg.Logger != nil {
			cfg.Logger.Printf("Warning: failed to destroy old session during regeneration: %v", err)
		}
	}()
	return nil
}
func Default(store Store) func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{
		Store: store,
	})
}
func Middleware(config Config) func(next context.HandlerFunc) context.HandlerFunc {
	return New(config)
}
