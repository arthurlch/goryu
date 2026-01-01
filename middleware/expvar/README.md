# Expvar Middleware

A middleware that exposes Go's public variables (via the `expvar` package) over HTTP. This is useful for monitoring internal application state, memory statistics, and custom metrics.

## Features

- **Standard Endpoint**: Exposes metrics at `/debug/vars` by default.
- **Custom Path**: Configurable endpoint path.
- **JSON Output**: Returns metrics in JSON format, compatible with many monitoring tools.

## Usage

### Basic Setup

Expose metrics at `/debug/vars`:

```go
package main

import (
    "expvar"
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/goryuctx"
    "github.com/arthurlch/goryu/middleware/expvar"
)

func main() {
    app := goryu.New()

    // Publish a custom metric
    requests := expvar.NewInt("requests")
    
    app.Use(expvar.Default())

    app.GET("/", func(c *goryuctx.Context) error {
        requests.Add(1)
        return c.String(200, "Hello Expvar")
    })

    app.Run(":8080")
}
```

### Custom Path

Expose metrics at a different path:

```go
app.Use(expvar.WithPath("/metrics/vars"))
```

### Advanced Configuration

```go
app.Use(expvar.New(expvar.Config{
    Path: "/admin/stats",
}))
```

## Security Note

The `expvar` endpoint exposes internal application details. In a production environment, you should protect this endpoint using authentication middleware or restrict access to internal networks only.

```go
// Example: Protect expvar with Basic Auth
app.Use(func(next goryuctx.HandlerFunc) goryuctx.HandlerFunc {
    return func(c *goryuctx.Context) {
        if c.Request.URL.Path == "/debug/vars" {
            // Apply auth check here
        }
        next(c)
    }
})
app.Use(expvar.Default())
```
