package cors

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/arthurlch/goryu"
)

type Config struct {
	Next func(c *goryu.Context) bool

	// AllowOrigins defines a list of origins that are allowed to access the resources.
	// Default: []string{"*"}
	AllowOrigins []string

	// AllowMethods defines a list of methods that are allowed when accessing the resources.
	// Default: []string{"GET", "POST", "HEAD", "PUT", "DELETE", "PATCH"}
	AllowMethods []string

	// AllowHeaders defines a list of headers that are allowed to be sent by the client.
	// Default: []string{}
	AllowHeaders []string

	// AllowCredentials indicates whether the request can include user credentials.
	// Default: false
	AllowCredentials bool

	// ExposeHeaders defines a list of headers that clients can access.
	// Default: []string{}
	ExposeHeaders []string

	// MaxAge indicates how long the results of a preflight request can be cached.
	// Default: 0
	MaxAge int
}

func New(config ...Config) goryu.Middleware {
	var cfg Config
	if len(config) > 0 {
		cfg = config[0]
	}

	if len(cfg.AllowOrigins) == 0 {
		cfg.AllowOrigins = []string{"*"}
	}
	if len(cfg.AllowMethods) == 0 {
		cfg.AllowMethods = []string{"GET", "POST", "HEAD", "PUT", "DELETE", "PATCH"}
	}

	allowMethods := strings.Join(cfg.AllowMethods, ",")
	allowHeaders := strings.Join(cfg.AllowHeaders, ",")
	exposeHeaders := strings.Join(cfg.ExposeHeaders, ",")
	maxAge := strconv.Itoa(cfg.MaxAge)
	allowAllOrigins := cfg.AllowOrigins[0] == "*"

	return func(next goryu.HandlerFunc) goryu.HandlerFunc {
		return func(c *goryu.Context) {
			if cfg.Next != nil && cfg.Next(c) {
				next(c)
				return
			}

			origin := c.GetHeader("Origin")
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
				c.Append("Vary", "Origin")
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				if cfg.AllowCredentials {
					c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}

			// preflight req
			if c.Request.Method == http.MethodOptions {
				c.Append("Vary", "Access-Control-Request-Method")
				c.Append("Vary", "Access-Control-Request-Headers")

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

				c.Status(http.StatusNoContent)
				return
			}

			next(c)
		}
	}
}
