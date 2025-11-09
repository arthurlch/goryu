package basicauth
import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
	"time"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/base"
	"golang.org/x/crypto/bcrypt"
)
type Config struct {
	base.BaseConfig
	Users map[string]string
	Validator func(username, password string) bool
	Realm string
	MaxAttempts int           
	RateWindow  time.Duration 
}
func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if c.Realm == "" {
		c.Realm = "Restricted"
	}
	if c.Users == nil && c.Validator == nil {
		return base.NewConfigError("Users or Validator", "at least one must be provided")
	}
	if c.Users != nil {
		for username, password := range c.Users {
			if !strings.HasPrefix(password, "$2a$") && !strings.HasPrefix(password, "$2b$") && !strings.HasPrefix(password, "$2y$") {
				return base.NewConfigError("Users", "password for user '"+username+"' must be bcrypt hashed (use HashPassword function)")
			}
		}
	}
	if c.MaxAttempts == 0 {
		c.MaxAttempts = 5 
	}
	if c.RateWindow == 0 {
		c.RateWindow = 15 * time.Minute 
	}
	return nil
}
func New(config Config) func(next context.HandlerFunc) context.HandlerFunc {
	if err := config.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "BasicAuth")
			}
		}
	}
	handler := func(c *context.Context) error {
		auth := c.Request.Header.Get("Authorization")
		if auth == "" {
			return unauthorized(c, config.Realm)
		}
		const prefix = "Basic "
		if !strings.HasPrefix(auth, prefix) {
			return unauthorized(c, config.Realm)
		}
		encoded, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
		if err != nil {
			return unauthorized(c, config.Realm)
		}
		creds := string(encoded)
		parts := strings.SplitN(creds, ":", 2)
		if len(parts) != 2 {
			return unauthorized(c, config.Realm)
		}
		username, password := parts[0], parts[1]
		if config.Validator != nil {
			if config.Validator(username, password) {
				return nil
			}
			return unauthorized(c, config.Realm)
		}
		if storedHashedPassword, ok := config.Users[username]; ok {
			err := bcrypt.CompareHashAndPassword([]byte(storedHashedPassword), []byte(password))
			if subtle.ConstantTimeCompare([]byte{func() byte { if err == nil { return 1 } else { return 0 } }()}, []byte{1}) == 1 {
				return nil
			}
		} else {
			_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$dummy.hash.to.prevent.timing.attacks"), []byte(password))
		}
		return unauthorized(c, config.Realm)
	}
	return base.StandardMiddleware("BasicAuth", config.BaseConfig, handler)
}
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
func WithUsers(users map[string]string) func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{
		Users: users,
	})
}
func WithValidator(validator func(string, string) bool) func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{
		Validator: validator,
	})
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{})
}
func unauthorized(c *context.Context, realm string) error {
	c.Writer.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
	c.Writer.WriteHeader(http.StatusUnauthorized)
	if err := c.Text(http.StatusUnauthorized, "Unauthorized"); err != nil {
		return base.MiddlewareError{
			Middleware: "BasicAuth",
			Err:        err,
			StatusCode: http.StatusUnauthorized,
		}
	}
	return base.MiddlewareError{
		Middleware: "BasicAuth",
		Err:        base.NewConfigError("Authentication", "invalid credentials"),
		StatusCode: http.StatusUnauthorized,
	}
}
