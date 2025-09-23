package trustproxy_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/trustproxy"
)

func newTestContext(req *http.Request) (*goryu.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}

func TestTrustProxyMiddleware(t *testing.T) {
	t.Run("DirectRequest", func(t *testing.T) {
		config := trustproxy.Config{
			TrustedProxies: []string{"10.0.0.0/8"},
		}
		middleware := trustproxy.New(config)

		handler := func(c *goryu.Context) {
			ip := trustproxy.GetTrustedIP(c)
			c.Text(http.StatusOK, ip)
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Body.String() != "192.168.1.100" {
			t.Errorf("Expected IP 192.168.1.100, got %s", rr.Body.String())
		}
	})

	t.Run("TrustedProxyWithXForwardedFor", func(t *testing.T) {
		config := trustproxy.Config{
			TrustedProxies: []string{"10.0.0.0/8", "172.16.0.0/12"},
		}
		middleware := trustproxy.New(config)

		handler := func(c *goryu.Context) {
			ip := trustproxy.GetTrustedIP(c)
			c.Text(http.StatusOK, ip)
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Body.String() != "203.0.113.195" {
			t.Errorf("Expected IP 203.0.113.195, got %s", rr.Body.String())
		}
	})

	t.Run("UntrustedProxyWithXForwardedFor", func(t *testing.T) {
		config := trustproxy.Config{
			TrustedProxies: []string{"10.0.0.0/8"},
		}
		middleware := trustproxy.New(config)

		handler := func(c *goryu.Context) {
			ip := trustproxy.GetTrustedIP(c)
			c.Text(http.StatusOK, ip)
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.100:12345" // Not in trusted range
		req.Header.Set("X-Forwarded-For", "203.0.113.195")
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		// Should ignore the header and use RemoteAddr
		if rr.Body.String() != "192.168.1.100" {
			t.Errorf("Expected IP 192.168.1.100, got %s", rr.Body.String())
		}
	})

	t.Run("CustomProxyHeader", func(t *testing.T) {
		config := trustproxy.Config{
			TrustedProxies: []string{"10.0.0.1"},
			ProxyHeader:    "X-Real-IP",
		}
		middleware := trustproxy.New(config)

		handler := func(c *goryu.Context) {
			ip := trustproxy.GetTrustedIP(c)
			c.Text(http.StatusOK, ip)
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		req.Header.Set("X-Real-IP", "203.0.113.195")
		req.Header.Set("X-Forwarded-For", "192.168.1.100") // Should be ignored
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Body.String() != "203.0.113.195" {
			t.Errorf("Expected IP 203.0.113.195, got %s", rr.Body.String())
		}
	})

	t.Run("IPv6Support", func(t *testing.T) {
		config := trustproxy.Config{
			TrustedProxies: []string{"2001:db8::/32"},
		}
		middleware := trustproxy.New(config)

		handler := func(c *goryu.Context) {
			ip := trustproxy.GetTrustedIP(c)
			c.Text(http.StatusOK, ip)
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "[2001:db8::1]:12345"
		req.Header.Set("X-Forwarded-For", "2001:db8:85a3::8a2e:370:7334")
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Body.String() != "2001:db8:85a3::8a2e:370:7334" {
			t.Errorf("Expected IP 2001:db8:85a3::8a2e:370:7334, got %s", rr.Body.String())
		}
	})
}
