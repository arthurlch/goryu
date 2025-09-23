package session

import (
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/arthurlch/goryu"
	"github.com/google/uuid"
)

type Logger interface {
	Printf(format string, v ...any)
}

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
	Store Store
	// CookieName is the name of the session cookie.
	// Default: "goryu_session"
	CookieName string
	// Expiration is the duration for which the session is valid.
	// Default: 24 * time.Hour
	Expiration time.Duration
	// Logger is the logger instance for the middleware.
	// If nil, it defaults to the standard log package.
	Logger Logger
	// Next is a function to skip this middleware.
	Next func(c *goryu.Context) bool
}

const (
	sessionKey          = "goryu.session"
	sessionIDKey        = "goryu.session.id"
	sessionCfgKey       = "goryu.session.config"
	sessionDestroyedKey = "goryu.session.destroyed"
)

func New(config Config) goryu.Middleware {
	if config.Store == nil {
		return func(next goryu.HandlerFunc) goryu.HandlerFunc {
			return func(c *goryu.Context) {
				c.Status(http.StatusInternalServerError).Text(http.StatusInternalServerError, "Session middleware error: no store configured")
			}
		}
	}
	if config.CookieName == "" {
		config.CookieName = "goryu_session"
	}
	if config.Expiration == 0 {
		config.Expiration = 24 * time.Hour
	}

	logger := config.Logger
	if logger == nil {
		logger = log.New(os.Stderr, "[goryu-session] ", log.LstdFlags)
	}

	return func(next goryu.HandlerFunc) goryu.HandlerFunc {
		return func(c *goryu.Context) {
			c.Set(sessionCfgKey, &config)

			cookie, err := c.Cookie(config.CookieName)
			var session *Session

			if err == nil {
				// Existing session
				sessionID, err := base64.StdEncoding.DecodeString(cookie.Value)
				if err == nil {
					s, err := config.Store.Get(string(sessionID))
					if err == nil && s != nil {
						session = s
						c.Set(sessionIDKey, s.ID)
					}
				}
			}

			if session == nil {
				// Create a new session if none exists or if it's invalid
				newID := uuid.New().String()
				session = &Session{
					ID:       newID,
					Data:     make(map[string]any),
					modified: true,
				}
				c.Set(sessionIDKey, newID)
			}

			c.Set(sessionKey, session)

			next(c)

			if destroyed, _ := c.Get(sessionDestroyedKey); destroyed == true {
				return // .
			}

			finalSessionVal, _ := c.Get(sessionKey)
			finalSession := finalSessionVal.(*Session)

			if finalSession.modified {
				if err := config.Store.Save(finalSession); err != nil {
					logger.Printf("Error saving session: %v", err)
				}
			}

			c.SetCookie(&http.Cookie{
				Name:     config.CookieName,
				Value:    base64.StdEncoding.EncodeToString([]byte(finalSession.ID)),
				Expires:  time.Now().Add(config.Expiration),
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
		}
	}
}

func Get(c *goryu.Context) (*Session, error) {
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

func Destroy(c *goryu.Context) error {
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
		Name:   cfg.CookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	c.Set(sessionDestroyedKey, true)

	return cfg.Store.Destroy(sessionID)
}
