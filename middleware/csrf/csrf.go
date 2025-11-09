package csrf
import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"time"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/base"
)
const (
	DefaultCSRFTokenHeader = "X-CSRF-Token"
	DefaultCSRFTokenCookie = "csrf-token"
	DefaultTokenByteLength = 32
)
type Config struct {
	base.BaseConfig
	TokenHeader string
	TokenCookie string
	TokenLength int
	TokenExpiry time.Duration
	Secure bool
	SameSite http.SameSite
	SafeMethods []string
	TokenGenerator func(int) (string, error)
}
func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if c.TokenHeader == "" {
		c.TokenHeader = DefaultCSRFTokenHeader
	}
	if c.TokenCookie == "" {
		c.TokenCookie = DefaultCSRFTokenCookie
	}
	if c.TokenLength <= 0 {
		c.TokenLength = DefaultTokenByteLength
	}
	if c.TokenExpiry <= 0 {
		c.TokenExpiry = 12 * time.Hour
	}
	if c.SafeMethods == nil {
		c.SafeMethods = []string{"GET", "HEAD", "OPTIONS"}
	}
	if c.TokenGenerator == nil {
		c.TokenGenerator = generateSecureToken
	}
	return nil
}
func New(config Config) func(next context.HandlerFunc) context.HandlerFunc {
	if err := config.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "CSRF")
			}
		}
	}
	handler := func(c *context.Context) error {
		isSafeMethod := isSafe(c.Request.Method, config.SafeMethods)
		if isSafeMethod {
			token, err := config.TokenGenerator(config.TokenLength)
			if err != nil {
				return base.MiddlewareError{
					Middleware: "CSRF",
					Err:        err,
					StatusCode: http.StatusInternalServerError,
				}
			}
			cookie := &http.Cookie{
				Name:     config.TokenCookie,
				Value:    token,
				Expires:  time.Now().Add(config.TokenExpiry),
				Secure:   config.Secure,
				HttpOnly: true,
				Path:     "/",
				SameSite: config.SameSite,
			}
			http.SetCookie(c.Writer, cookie)
			c.Writer.Header().Set(config.TokenHeader, token)
		} else {
			tokenFromHeader := c.Request.Header.Get(config.TokenHeader)
			if tokenFromHeader == "" {
				return base.MiddlewareError{
					Middleware: "CSRF",
					Err:        base.NewConfigError("CSRF Token", "missing token in header"),
					StatusCode: http.StatusForbidden,
				}
			}
			cookie, err := c.Request.Cookie(config.TokenCookie)
			if err != nil {
				return base.MiddlewareError{
					Middleware: "CSRF",
					Err:        base.NewConfigError("CSRF Token", "missing token cookie"),
					StatusCode: http.StatusForbidden,
				}
			}
			if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(tokenFromHeader)) != 1 {
				return base.MiddlewareError{
					Middleware: "CSRF",
					Err:        base.NewConfigError("CSRF Token", "token mismatch"),
					StatusCode: http.StatusForbidden,
				}
			}
		}
		return nil
	}
	return base.StandardMiddleware("CSRF", config.BaseConfig, handler)
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{})
}
func generateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
func isSafe(method string, safeMethods []string) bool {
	for _, safe := range safeMethods {
		if method == safe {
			return true
		}
	}
	return false
}