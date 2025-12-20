package securecookie

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
)
type contextKey string
const secureCookieContextKey = contextKey("secure-cookie-data")
const secureCookieInstanceKey = contextKey("secure-cookie-instance")
var (
	ErrValueNotFound = errors.New("securecookie: value not found")
	ErrInvalidValue  = errors.New("securecookie: invalid value")
)
type Config struct {
	base.BaseConfig
	HexKey string
	CookieName string
	CookiePath string
	CookieTTL time.Duration
	Secure bool
	SameSite http.SameSite
	HttpOnly bool
}
type SecureCookie struct {
	gcm        cipher.AEAD
	cookieName string
	cookiePath string
	cookieTTL  time.Duration
	secure     bool
	sameSite   http.SameSite
	httpOnly   bool
}
func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if c.HexKey == "" {
		return base.NewConfigError("HexKey", "is required")
	}
	if c.CookieName == "" {
		return base.NewConfigError("CookieName", "is required")
	}
	if c.CookiePath == "" {
		c.CookiePath = "/"
	}
	if c.CookieTTL == 0 {
		c.CookieTTL = 12 * time.Hour
	}
	if !c.Secure && c.HexKey != "" {
		c.Secure = true
	}
	if c.SameSite == 0 {
		c.SameSite = http.SameSiteLaxMode
	}
	if !c.HttpOnly && c.HexKey != "" {
		c.HttpOnly = true
	}
	return nil
}
func New(config ...Config) func(next context.HandlerFunc) context.HandlerFunc {
	cfg := Config{}
	if len(config) > 0 {
		cfg = config[0]
	}

	if err := cfg.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "SecureCookie")
			}
		}
	}
	key, err := hex.DecodeString(cfg.HexKey)
	if err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, base.MiddlewareError{
					Middleware: "SecureCookie",
					Err:        fmt.Errorf("failed to decode hex key: %w", err),
					StatusCode: 500,
				}, "SecureCookie")
			}
		}
	}
	if len(key) != 32 {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, base.MiddlewareError{
					Middleware: "SecureCookie",
					Err:        errors.New("key must be 32 bytes (AES-256)"),
					StatusCode: 500,
				}, "SecureCookie")
			}
		}
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, base.MiddlewareError{
					Middleware: "SecureCookie",
					Err:        fmt.Errorf("failed to create cipher block: %w", err),
					StatusCode: 500,
				}, "SecureCookie")
			}
		}
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, base.MiddlewareError{
					Middleware: "SecureCookie",
					Err:        fmt.Errorf("failed to create GCM: %w", err),
					StatusCode: 500,
				}, "SecureCookie")
			}
		}
	}
	sc := &SecureCookie{
		gcm:        gcm,
		cookieName: cfg.CookieName,
		cookiePath: cfg.CookiePath,
		cookieTTL:  cfg.CookieTTL,
		secure:     cfg.Secure,
		sameSite:   cfg.SameSite,
		httpOnly:   cfg.HttpOnly,
	}
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if cfg.Skip != nil && cfg.Skip(c) {
				next(c)
				return
			}
			cookie, err := c.Request.Cookie(sc.cookieName)
			var value map[string]string
			if err == nil {
				value, err = sc.decrypt(cookie.Value)
				if err != nil {
					logger := cfg.Logger
					if logger == nil {
						logger = base.DefaultLogger("SecureCookie")
					}
					logger.Printf("Failed to decrypt cookie '%s': %v", sc.cookieName, err)
					http.SetCookie(c.Writer, sc.createExpiredCookie())
				}
			}
			c.Set(string(secureCookieContextKey), value)
			c.Set(string(secureCookieInstanceKey), sc)
			next(c)
		}
	}
}
func Default(hexKey, cookieName string) func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{
		HexKey:     hexKey,
		CookieName: cookieName,
	})
}
func (sc *SecureCookie) encrypt(value map[string]string) (string, error) {
	plaintext, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, sc.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := sc.gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}
func (sc *SecureCookie) decrypt(encodedValue string) (map[string]string, error) {
	ciphertext, err := base64.URLEncoding.DecodeString(encodedValue)
	if err != nil {
		return nil, ErrInvalidValue
	}
	nonceSize := sc.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidValue
	}
	nonce, encryptedMessage := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := sc.gcm.Open(nil, nonce, encryptedMessage, nil)
	if err != nil {
		return nil, ErrInvalidValue
	}
	var value map[string]string
	if err := json.Unmarshal(plaintext, &value); err != nil {
		return nil, ErrInvalidValue
	}
	return value, nil
}
func (sc *SecureCookie) createCookie(encodedValue string) *http.Cookie {
	return &http.Cookie{
		Name:     sc.cookieName,
		Value:    encodedValue,
		Path:     sc.cookiePath,
		Expires:  time.Now().Add(sc.cookieTTL),
		HttpOnly: sc.httpOnly,
		Secure:   sc.secure,
		SameSite: sc.sameSite,
	}
}
func (sc *SecureCookie) createExpiredCookie() *http.Cookie {
	return &http.Cookie{
		Name:     sc.cookieName,
		Value:    "",
		Path:     sc.cookiePath,
		MaxAge:   -1, 
		HttpOnly: sc.httpOnly,
		Secure:   sc.secure,
		SameSite: sc.sameSite,
	}
}
func Set(c *context.Context, value map[string]string) error {
	sc, exists := c.Get(string(secureCookieInstanceKey))
	if !exists {
		return errors.New("securecookie: middleware not installed")
	}
	secureCookie, ok := sc.(*SecureCookie)
	if !ok {
		return errors.New("securecookie: invalid instance in context")
	}
	encoded, err := secureCookie.encrypt(value)
	if err != nil {
		return err
	}
	http.SetCookie(c.Writer, secureCookie.createCookie(encoded))
	return nil
}
func Get(c *context.Context) (map[string]string, error) {
	value, exists := c.Get(string(secureCookieContextKey))
	if !exists {
		return nil, ErrValueNotFound
	}
	mapValue, ok := value.(map[string]string)
	if !ok || mapValue == nil {
		return nil, ErrValueNotFound
	}
	return mapValue, nil
}
func Clear(c *context.Context) error {
	sc, exists := c.Get(string(secureCookieInstanceKey))
	if !exists {
		return errors.New("securecookie: middleware not installed")
	}
	secureCookie, ok := sc.(*SecureCookie)
	if !ok {
		return errors.New("securecookie: invalid instance in context")
	}
	http.SetCookie(c.Writer, secureCookie.createExpiredCookie())
	return nil
}