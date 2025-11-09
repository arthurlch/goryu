package secure
import (
	"fmt"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/base"
)
type Config struct {
	base.BaseConfig
	XSSProtection string
	ContentTypeNosniff string
	XFrameOptions string
	HSTSMaxAge int
	HSTSIncludeSubdomains bool
	HSTSPreload bool
	ContentSecurityPolicy string
	ReferrerPolicy string
	PermissionsPolicy string
}
func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if c.XSSProtection == "" {
		c.XSSProtection = "1; mode=block"
	}
	if c.ContentTypeNosniff == "" {
		c.ContentTypeNosniff = "nosniff"
	}
	if c.XFrameOptions == "" {
		c.XFrameOptions = "SAMEORIGIN"
	}
	if c.ReferrerPolicy == "" {
		c.ReferrerPolicy = "strict-origin-when-cross-origin"
	}
	return nil
}
func New(config Config) func(next context.HandlerFunc) context.HandlerFunc {
	if err := config.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "Secure")
			}
		}
	}
	var hstsValue string
	if config.HSTSMaxAge > 0 {
		hstsValue = fmt.Sprintf("max-age=%d", config.HSTSMaxAge)
		if config.HSTSIncludeSubdomains {
			hstsValue += "; includeSubDomains"
		}
		if config.HSTSPreload {
			hstsValue += "; preload"
		}
	}
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if config.Skip != nil && config.Skip(c) {
				next(c)
				return
			}
			headers := c.Writer.Header()
			if config.XSSProtection != "" {
				headers.Set("X-XSS-Protection", config.XSSProtection)
			}
			if config.ContentTypeNosniff != "" {
				headers.Set("X-Content-Type-Options", config.ContentTypeNosniff)
			}
			if config.XFrameOptions != "" {
				headers.Set("X-Frame-Options", config.XFrameOptions)
			}
			if config.ReferrerPolicy != "" {
				headers.Set("Referrer-Policy", config.ReferrerPolicy)
			}
			if config.ContentSecurityPolicy != "" {
				headers.Set("Content-Security-Policy", config.ContentSecurityPolicy)
			}
			if config.PermissionsPolicy != "" {
				headers.Set("Permissions-Policy", config.PermissionsPolicy)
			}
			if c.Request.TLS != nil && hstsValue != "" {
				headers.Set("Strict-Transport-Security", hstsValue)
			}
			next(c)
		}
	}
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{})
}
func WithXSSProtection(protection string) func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{XSSProtection: protection})
}
func WithHSTS(maxAge int, includeSubdomains bool) func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{
		HSTSMaxAge:            maxAge,
		HSTSIncludeSubdomains: includeSubdomains,
	})
}
func WithCSP(csp string) func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{ContentSecurityPolicy: csp})
}