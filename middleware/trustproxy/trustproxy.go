package trustproxy

import (
	"net"
	"strings"

	"github.com/arthurlch/goryu"
)

type Config struct {
	TrustedProxies []string
	ProxyHeader    string
	Next           func(c *goryu.Context) bool
}

const TrustedProxyIPKey = "trusted_client_ip"

func New(config Config) goryu.Middleware {
	if config.ProxyHeader == "" {
		config.ProxyHeader = "X-Forwarded-For"
	}

	var trustedNetworks []*net.IPNet
	for _, proxy := range config.TrustedProxies {
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

	return func(next goryu.HandlerFunc) goryu.HandlerFunc {
		return func(c *goryu.Context) {
			if config.Next != nil && config.Next(c) {
				next(c)
				return
			}

			remoteIP, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
			clientIP := remoteIP

			if isTrustedProxy(remoteIP, trustedNetworks) {
				if proxyHeader := c.GetHeader(config.ProxyHeader); proxyHeader != "" {
					// Handle X-Forwarded-For which can contain multiple IPs
					if config.ProxyHeader == "X-Forwarded-For" {
						ips := strings.Split(proxyHeader, ",")
						if len(ips) > 0 {
							clientIP = strings.TrimSpace(ips[0])
						}
					} else {
						clientIP = strings.TrimSpace(proxyHeader)
					}
				}
			}

			c.Set(TrustedProxyIPKey, clientIP)
			next(c)
		}
	}
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

func GetTrustedIP(c *goryu.Context) string {
	if ip, exists := c.Get(TrustedProxyIPKey); exists {
		if ipStr, ok := ip.(string); ok {
			return ipStr
		}
	}
	ip, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
	return ip
}
