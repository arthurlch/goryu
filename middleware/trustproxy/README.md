# TrustProxy Middleware

A middleware that securely resolves the real client IP address when your application is running behind a load balancer or reverse proxy. It prevents IP spoofing by only trusting headers from configured proxy networks.

## Features

- **Secure IP Resolution**: Only trusts `X-Forwarded-For` (or custom headers) from whitelisted proxies.
- **CIDR Support**: Easily whitelist entire subnets (e.g., `10.0.0.0/8`, `127.0.0.1`).
- **Context Integration**: Stores the resolved IP in the context for easy access.
- **Header Customization**: Support for custom proxy headers (e.g., `CF-Connecting-IP`).

## Usage

### Basic Setup

Trust local proxies (common for sidecar proxies or local Nginx):

```go
package main

import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/goryuctx"
    "github.com/arthurlch/goryu/middleware/trustproxy"
)

func main() {
    app := goryu.New()

    // Trust loopback addresses
    app.Use(trustproxy.WithProxies([]string{"127.0.0.1", "::1"}))

    app.GET("/", func(c *goryuctx.Context) error {
        clientIP := trustproxy.GetTrustedIPFromContext(c)
        return c.String(200, "Your IP is: "+clientIP)
    })

    app.Run(":8080")
}
```

### Advanced Configuration

Trust private networks and use a custom header:

```go
app.Use(trustproxy.New(trustproxy.Config{
    TrustedProxies: []string{
        "10.0.0.0/8",     // Private network
        "172.16.0.0/12",  // Private network
        "192.168.0.0/16", // Private network
    },
    ProxyHeader: "X-Real-IP", // Use X-Real-IP instead of X-Forwarded-For
}))
```

## Why is this important?

If you blindly trust the `X-Forwarded-For` header, a malicious user can spoof their IP address by sending a request with a fake header:

```
GET / HTTP/1.1
X-Forwarded-For: 8.8.8.8
```

If your application uses IP-based rate limiting or access control, this spoofing can bypass your security measures. `TrustProxy` solves this by verifying that the request actually came from a trusted proxy before accepting the header value.
