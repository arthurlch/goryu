# TLS Redirect Middleware

A middleware that automatically redirects HTTP requests to HTTPS. It supports custom ports, status codes, and proxy headers (e.g., `X-Forwarded-Proto`).

## Features

- **Automatic Redirection**: Redirects all HTTP traffic to HTTPS.
- **Proxy Support**: Respects `X-Forwarded-Proto` and `X-Forwarded-Host` headers for applications behind load balancers.
- **Customizable**: Configurable redirect status code (default: 301) and HTTPS port (default: 443).
- **Flexible Logic**: Supports custom redirect logic via a callback function.

## Usage

### Basic Setup

Redirect all HTTP traffic to HTTPS on port 443:

```go
app.Use(tlsredirect.Default())
```

### Custom Port

If your application listens on a non-standard HTTPS port (e.g., 8443):

```go
app.Use(tlsredirect.WithPort(8443))
```

### Custom Status Code

Use a temporary redirect (307) instead of permanent (301):

```go
app.Use(tlsredirect.WithStatusCode(http.StatusTemporaryRedirect))
```

### Advanced Configuration

```go
app.Use(tlsredirect.New(tlsredirect.Config{
    StatusCode:        http.StatusMovedPermanently,
    CustomPort:        443,
    ForwardedProtocol: "X-Forwarded-Proto", // Header to check for protocol
    ForwardedHost:     "X-Forwarded-Host",  // Header to check for host
}))
```

## Load Balancer Support

This middleware is designed to work behind load balancers and reverse proxies (like Nginx, AWS ALB, Cloudflare). It checks the `X-Forwarded-Proto` header to determine if the original request was secure. If the header indicates "https", the request is allowed to proceed without redirection.
