package tlsredirect

import (
	"net/http"
	"strconv"
	"strings"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
)

type Config struct {
	base.BaseConfig
	StatusCode        int
	CustomPort        int
	ForwardedProtocol string
	ForwardedHost     string
	RedirectFunc      func(c *context.Context, httpsURL string)
}

func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if c.StatusCode == 0 {
		c.StatusCode = http.StatusMovedPermanently
	}
	if c.CustomPort == 0 {
		c.CustomPort = 443
	}
	if c.ForwardedProtocol == "" {
		c.ForwardedProtocol = "X-Forwarded-Proto"
	}
	if c.ForwardedHost == "" {
		c.ForwardedHost = "X-Forwarded-Host"
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
				base.DefaultErrorHandler(c, err, "TLSRedirect")
			}
		}
	}
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if cfg.Skip != nil && cfg.Skip(c) {
				next(c)
				return
			}
			if isSecureRequest(c, cfg) {
				next(c)
				return
			}
			httpsURL := buildHTTPSURL(c, cfg)
			if cfg.RedirectFunc != nil {
				cfg.RedirectFunc(c, httpsURL)
				return
			}
			c.Writer.Header().Set("Location", httpsURL)
			c.Writer.WriteHeader(cfg.StatusCode)
		}
	}
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New()
}
func WithPort(port int) func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{CustomPort: port})
}
func WithStatusCode(statusCode int) func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{StatusCode: statusCode})
}
func isSecureRequest(c *context.Context, config Config) bool {
	if c.Request.TLS != nil {
		return true
	}
	proto := c.Request.Header.Get(config.ForwardedProtocol)
	if proto != "" {
		return strings.ToLower(proto) == "https"
	}
	return false
}
func buildHTTPSURL(c *context.Context, config Config) string {
	host := c.Request.Header.Get(config.ForwardedHost)
	if host == "" {
		host = c.Request.Host
	}
	if strings.HasSuffix(host, ":80") {
		host = strings.TrimSuffix(host, ":80")
	}
	if config.CustomPort != 443 {
		if colonIndex := strings.LastIndex(host, ":"); colonIndex != -1 {
			host = host[:colonIndex]
		}
		host = host + ":" + strconv.Itoa(config.CustomPort)
	}
	uri := c.Request.URL.Path
	if c.Request.URL.RawQuery != "" {
		uri += "?" + c.Request.URL.RawQuery
	}
	return "https://" + host + uri
}
