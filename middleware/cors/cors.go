package cors

import (
	"net/http"
	"strconv"
	"strings"

	context "github.com/arthurlch/goryu/goryuctx"
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
func New(config ...Config) func(next context.HandlerFunc) context.HandlerFunc {
	cfg := Config{}
	if len(config) > 0 {
		cfg = config[0]
	}

	if err := cfg.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "CORS")
			}
		}
	}
	allowMethods := strings.Join(cfg.AllowMethods, ",")
	allowHeaders := strings.Join(cfg.AllowHeaders, ",")
	exposeHeaders := strings.Join(cfg.ExposeHeaders, ",")
	maxAge := strconv.Itoa(cfg.MaxAge)
	allowAllOrigins := len(cfg.AllowOrigins) > 0 && cfg.AllowOrigins[0] == "*"
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if cfg.Skip != nil && cfg.Skip(c) {
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
				for _, o := range cfg.AllowOrigins {
					if o == origin {
						allowed = true
						break
					}
				}
			}
			if allowed {
				c.Writer.Header().Add("Vary", "Origin")
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				if cfg.AllowCredentials {
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
					if cfg.MaxAge > 0 {
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
	return New()
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
