package cors
import (
	"net/http"
	"strconv"
	"strings"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/base"
)
type Config struct {
	base.BaseConfig
	AllowOrigins []string
	AllowMethods []string
	AllowHeaders []string
	AllowCredentials bool
	ExposeHeaders []string
	MaxAge int
}
func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if len(c.AllowOrigins) == 0 {
		return base.NewConfigError("AllowOrigins", "must be explicitly configured - avoid using '*' for production")
	}
	if c.AllowCredentials {
		for _, origin := range c.AllowOrigins {
			if origin == "*" {
				return base.NewConfigError("AllowOrigins", "cannot use '*' when AllowCredentials is true - specify exact origins")
			}
		}
	}
	for _, origin := range c.AllowOrigins {
		if origin == "*" && c.Logger != nil {
			c.Logger.Printf("WARNING: CORS allows all origins (*) - this should only be used in development")
		}
	}
	if len(c.AllowMethods) == 0 {
		c.AllowMethods = []string{"GET", "POST", "HEAD", "PUT", "DELETE", "PATCH"}
	}
	return nil
}
func New(config Config) func(next context.HandlerFunc) context.HandlerFunc {
	if err := config.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "CORS")
			}
		}
	}
	allowMethods := strings.Join(config.AllowMethods, ",")
	allowHeaders := strings.Join(config.AllowHeaders, ",")
	exposeHeaders := strings.Join(config.ExposeHeaders, ",")
	maxAge := strconv.Itoa(config.MaxAge)
	allowAllOrigins := len(config.AllowOrigins) > 0 && config.AllowOrigins[0] == "*"
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if config.Skip != nil && config.Skip(c) {
				next(c)
				return
			}
			origin := c.Request.Header.Get("Origin")
			if origin == "" {
				next(c)
				return
			}
			allowed := false
			if allowAllOrigins {
				allowed = true
			} else {
				for _, o := range config.AllowOrigins {
					if o == origin {
						allowed = true
						break
					}
				}
			}
			if allowed {
				c.Writer.Header().Add("Vary", "Origin")
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				if config.AllowCredentials {
					c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}
			if c.Request.Method == http.MethodOptions {
				c.Writer.Header().Add("Vary", "Access-Control-Request-Method")
				c.Writer.Header().Add("Vary", "Access-Control-Request-Headers")
				if allowed {
					c.Writer.Header().Set("Access-Control-Allow-Methods", allowMethods)
					if allowHeaders != "" {
						c.Writer.Header().Set("Access-Control-Allow-Headers", allowHeaders)
					}
					if exposeHeaders != "" {
						c.Writer.Header().Set("Access-Control-Expose-Headers", exposeHeaders)
					}
					if config.MaxAge > 0 {
						c.Writer.Header().Set("Access-Control-Max-Age", maxAge)
					}
				}
				c.Writer.WriteHeader(http.StatusNoContent)
				return
			}
			next(c)
		}
	}
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{})
}
func WithAllowAll() func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{
		AllowOrigins: []string{"*"},
	})
}
func WithOrigins(origins ...string) func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{
		AllowOrigins: origins,
	})
}
