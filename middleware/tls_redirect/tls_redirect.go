package tlsredirect
import (
	"net/http"
	"strconv"
	"strings"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/base"
)
type Config struct {
	base.BaseConfig
	StatusCode int
	CustomPort int
	ForwardedProtocol string
	ForwardedHost string
	RedirectFunc func(c *context.Context, httpsURL string)
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
func New(config Config) func(next context.HandlerFunc) context.HandlerFunc {
	if err := config.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "TLSRedirect")
			}
		}
	}
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if config.Skip != nil && config.Skip(c) {
				next(c)
				return
			}
			if isSecureRequest(c, config) {
				next(c)
				return
			}
			httpsURL := buildHTTPSURL(c, config)
			if config.RedirectFunc != nil {
				config.RedirectFunc(c, httpsURL)
				return
			}
			c.Writer.Header().Set("Location", httpsURL)
			c.Writer.WriteHeader(config.StatusCode)
		}
	}
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{})
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
	return "https://" + host + c.Request.RequestURI
}