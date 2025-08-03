package secure

import (
	"fmt"

	"github.com/arthurlch/goryu"
)

type Config struct {
	Next                  func(c *goryu.Context) bool
	XSSProtection         string
	ContentTypeNosniff    string
	XFrameOptions         string
	HSTSMaxAge            int
	HSTSIncludeSubdomains bool
}

func New(config ...Config) goryu.Middleware {
	var cfg Config
	if len(config) > 0 {
		cfg = config[0]
	}

	if cfg.XSSProtection == "" {
		cfg.XSSProtection = "1; mode=block"
	}
	if cfg.ContentTypeNosniff == "" {
		cfg.ContentTypeNosniff = "nosniff"
	}
	if cfg.XFrameOptions == "" {
		cfg.XFrameOptions = "SAMEORIGIN"
	}

	return func(next goryu.HandlerFunc) goryu.HandlerFunc {
		return func(c *goryu.Context) {
			if cfg.Next != nil && cfg.Next(c) {
				next(c)
				return
			}

			if cfg.XSSProtection != "" {
				c.Writer.Header().Set("X-XSS-Protection", cfg.XSSProtection)
			}
			if cfg.ContentTypeNosniff != "" {
				c.Writer.Header().Set("X-Content-Type-Options", cfg.ContentTypeNosniff)
			}
			if cfg.XFrameOptions != "" {
				c.Writer.Header().Set("X-Frame-Options", cfg.XFrameOptions)
			}
			if c.Request.TLS != nil && cfg.HSTSMaxAge > 0 {
				subdomains := ""
				if cfg.HSTSIncludeSubdomains {
					subdomains = "; includeSubdomains"
				}
				c.Writer.Header().Set("Strict-Transport-Security", fmt.Sprintf("max-age=%d%s", cfg.HSTSMaxAge, subdomains))
			}

			next(c)
		}
	}
}
