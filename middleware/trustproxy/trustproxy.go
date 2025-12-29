package trustproxy

import (
	"net"
	"strings"

	"github.com/arthurlch/goryu"
	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
)

type Config struct {
	base.BaseConfig
	TrustedProxies []string
	ProxyHeader    string
	ContextKey     string
}

const TrustedProxyIPKey = "trusted_client_ip"

func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if c.ProxyHeader == "" {
		c.ProxyHeader = "X-Forwarded-For"
	}
	if c.ContextKey == "" {
		c.ContextKey = TrustedProxyIPKey
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
				base.DefaultErrorHandler(c, err, "TrustProxy")
			}
		}
	}
	trustedNetworks := parseTrustedProxies(cfg.TrustedProxies)
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if cfg.Skip != nil && cfg.Skip(c) {
				next(c)
				return
			}
			remoteIP, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
			clientIP := remoteIP
			if isTrustedProxy(remoteIP, trustedNetworks) {
				if proxyHeader := c.Request.Header.Get(cfg.ProxyHeader); proxyHeader != "" {
					clientIP = extractClientIP(proxyHeader, cfg.ProxyHeader)
				}
			}
			c.Set(cfg.ContextKey, clientIP)
			next(c)
		}
	}
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New()
}
func WithProxies(proxies []string) func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{TrustedProxies: proxies})
}
func NewLegacy(config Config) goryu.Middleware {
	trustedNetworks := parseTrustedProxies(config.TrustedProxies)
	if config.ProxyHeader == "" {
		config.ProxyHeader = "X-Forwarded-For"
	}
	return func(next goryu.Handler) goryu.Handler {
		return func(c *goryu.Ctx) {
			remoteIP, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
			clientIP := remoteIP
			if isTrustedProxy(remoteIP, trustedNetworks) {
				if proxyHeader := c.GetHeader(config.ProxyHeader); proxyHeader != "" {
					clientIP = extractClientIP(proxyHeader, config.ProxyHeader)
				}
			}
			c.Set(TrustedProxyIPKey, clientIP)
			next(c)
		}
	}
}
func parseTrustedProxies(proxies []string) []*net.IPNet {
	var trustedNetworks []*net.IPNet
	for _, proxy := range proxies {
		if strings.Contains(proxy, "/") {
			_, network, err := net.ParseCIDR(proxy)
			if err == nil {
				trustedNetworks = append(trustedNetworks, network)
			}
		} else {
			ip := net.ParseIP(proxy)
			if ip != nil {
				if ip.To4() != nil {
					_, network, _ := net.ParseCIDR(proxy + "/32")
					trustedNetworks = append(trustedNetworks, network)
				} else {
					_, network, _ := net.ParseCIDR(proxy + "/128")
					trustedNetworks = append(trustedNetworks, network)
				}
			}
		}
	}
	return trustedNetworks
}
func extractClientIP(proxyHeader, headerName string) string {
	if headerName == "X-Forwarded-For" {
		ips := strings.Split(proxyHeader, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	return strings.TrimSpace(proxyHeader)
}
func isTrustedProxy(ipStr string, trustedNetworks []*net.IPNet) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, network := range trustedNetworks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
func GetTrustedIP(c *goryu.Ctx) string {
	if ip, exists := c.Get(TrustedProxyIPKey); exists {
		if ipStr, ok := ip.(string); ok {
			return ipStr
		}
	}
	ip, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
	return ip
}
func GetTrustedIPFromContext(c *context.Context) string {
	return GetTrustedIPFromContextWithKey(c, TrustedProxyIPKey)
}
func GetTrustedIPFromContextWithKey(c *context.Context, key string) string {
	if ip, exists := c.Get(key); exists {
		if ipStr, ok := ip.(string); ok {
			return ipStr
		}
	}
	ip, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
	return ip
}
