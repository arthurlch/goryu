package secure

import (
	"fmt"

	context "github.com/arthurlch/goryu/goryuctx"
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
func New(config ...Config) func(next context.HandlerFunc) context.HandlerFunc {
	cfg := Config{}
	if len(config) > 0 {
		cfg = config[0]
	}

	if err := cfg.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "Secure")
			}
		}
	}
	var hstsValue string
	if cfg.HSTSMaxAge > 0 {
		hstsValue = fmt.Sprintf("max-age=%d", cfg.HSTSMaxAge)
		if cfg.HSTSIncludeSubdomains {
			hstsValue += "; includeSubDomains"
		}
		if cfg.HSTSPreload {
			hstsValue += "; preload"
		}
	}
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if cfg.Skip != nil && cfg.Skip(c) {
				next(c)
				return
			}
			headers := c.Writer.Header()
			if cfg.XSSProtection != "" {
				headers.Set("X-XSS-Protection", cfg.XSSProtection)
			}
			if cfg.ContentTypeNosniff != "" {
				headers.Set("X-Content-Type-Options", cfg.ContentTypeNosniff)
			}
			if cfg.XFrameOptions != "" {
				headers.Set("X-Frame-Options", cfg.XFrameOptions)
			}
			if cfg.ReferrerPolicy != "" {
				headers.Set("Referrer-Policy", cfg.ReferrerPolicy)
			}
			if cfg.ContentSecurityPolicy != "" {
				headers.Set("Content-Security-Policy", cfg.ContentSecurityPolicy)
			}
			if cfg.PermissionsPolicy != "" {
				headers.Set("Permissions-Policy", cfg.PermissionsPolicy)
			}
			if c.Request.TLS != nil && hstsValue != "" {
				headers.Set("Strict-Transport-Security", hstsValue)
			}
			next(c)
		}
	}
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New()
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