package session

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/arthurlch/goryu/goryuctx"
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
	Store      Store
	CookieName string
	Expiration time.Duration
	Secure     *bool
	SameSite   http.SameSite
	Domain     string
	Path       string
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

// sessionResponseWriter wraps http.ResponseWriter to intercept writes
type sessionResponseWriter struct {
	http.ResponseWriter
	beforeWrite func()
	written     bool
}

func (w *sessionResponseWriter) WriteHeader(code int) {
	if !w.written {
		w.beforeWrite()
		w.written = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *sessionResponseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.beforeWrite()
		w.written = true
	}
	return w.ResponseWriter.Write(b)
}

func New(config ...Config) func(next goryuctx.HandlerFunc) goryuctx.HandlerFunc {
	cfg := Config{}
	if len(config) > 0 {
		cfg = config[0]
	}

	if err := cfg.Validate(); err != nil {
		return func(next goryuctx.HandlerFunc) goryuctx.HandlerFunc {
			return func(c *goryuctx.Context) {
				base.DefaultErrorHandler(c, base.MiddlewareError{
					Middleware: "Session",
					Err:        err,
					StatusCode: http.StatusInternalServerError,
				}, "Session")
			}
		}
	}

	// Pre-handler logic extracted
	loadSession := func(c *goryuctx.Context) error {
		c.Set(sessionCfgKey, &cfg)
		cookie, err := c.Cookie(cfg.CookieName)
		var session *Session
		if err == nil {
			sessionID, decodeErr := base64.StdEncoding.DecodeString(cookie.Value)
			if decodeErr == nil {
				s, storeErr := cfg.Store.Get(string(sessionID))
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

	// Post-handler logic extracted
	saveSession := func(c *goryuctx.Context) {
		if destroyed, _ := c.Get(sessionDestroyedKey); destroyed == true {
			return
		}
		finalSessionVal, exists := c.Get(sessionKey)
		if !exists {
			return
		}
		finalSession, ok := finalSessionVal.(*Session)
		if !ok {
			// Should log error?
			return
		}

		// Only save/set cookie if the writer hasn't been written to OR if we are just about to write
		// Ideally we do this exactly once.

		if finalSession.modified {
			if err := cfg.Store.Save(finalSession); err != nil {
				if cfg.Logger != nil {
					cfg.Logger.Printf("Error saving session: %v", err)
				}
				// If save fails, we probably shouldn't set the cookie to the new ID if it was new?
				// But we continue for now.
			}
		}

		cookie := &http.Cookie{
			Name:     cfg.CookieName,
			Value:    base64.StdEncoding.EncodeToString([]byte(finalSession.ID)),
			Expires:  time.Now().Add(cfg.Expiration),
			Path:     cfg.Path,
			Domain:   cfg.Domain,
			HttpOnly: true,
			Secure:   *cfg.Secure,
			SameSite: cfg.SameSite,
		}
		c.SetCookie(cookie)
	}

	return func(next goryuctx.HandlerFunc) goryuctx.HandlerFunc {
		return func(c *goryuctx.Context) {
			// 1. Run Pre-handler
			if err := loadSession(c); err != nil {
				if cfg.Logger != nil {
					cfg.Logger.Printf("Session load error: %v", err)
				}
				// Continue? or Abort? Usually continue with new session if failed.
			}

			// 2. Wrap Writer to capture response start
			originalWriter := c.Writer
			w := &sessionResponseWriter{
				ResponseWriter: originalWriter,
				beforeWrite: func() {
					saveSession(c)
				},
			}
			c.Writer = w

			// 3. Next
			next(c)

			// 4. If nothing was written, ensure we save (e.g. 404s or empty 200 OKs not using Write)
			if !w.written {
				w.beforeWrite()
				w.written = true
			}
		}
	}
}
func Get(c *goryuctx.Context) (*Session, error) {
	s, exists := c.Get(sessionKey)
	if !exists {
		return nil, errors.New("session not found in goryuctx")
	}
	session, ok := s.(*Session)
	if !ok {
		return nil, errors.New("invalid session type in goryuctx")
	}
	return session, nil
}
func Destroy(c *goryuctx.Context) error {
	cfgVal, exists := c.Get(sessionCfgKey)
	if !exists {
		return errors.New("session config not found in goryuctx")
	}
	cfg, ok := cfgVal.(*Config)
	if !ok {
		return errors.New("session config is of invalid type in goryuctx")
	}
	sessionIDVal, exists := c.Get(sessionIDKey)
	if !exists {
		return errors.New("session ID not found in goryuctx")
	}
	sessionID, ok := sessionIDVal.(string)
	if !ok {
		return errors.New("session ID is of invalid type in goryuctx")
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
func Regenerate(c *goryuctx.Context) error {
	session, err := Get(c)
	if err != nil {
		return err
	}
	cfgVal, exists := c.Get(sessionCfgKey)
	if !exists {
		return errors.New("session config not found in goryuctx")
	}
	cfg, ok := cfgVal.(*Config)
	if !ok {
		return errors.New("session config is of invalid type in goryuctx")
	}
	oldSessionIDVal, exists := c.Get(sessionIDKey)
	if !exists {
		return errors.New("session ID not found in goryuctx")
	}
	oldSessionID, ok := oldSessionIDVal.(string)
	if !ok {
		return errors.New("session ID is of invalid type in goryuctx")
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
func Default(store Store) func(next goryuctx.HandlerFunc) goryuctx.HandlerFunc {
	return New(Config{
		Store: store,
	})
}
func Middleware(config Config) func(next goryuctx.HandlerFunc) goryuctx.HandlerFunc {
	return New(config)
}
