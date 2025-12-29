# Middleware Builder

The middleware builder package provides a fluent, intuitive way to create middleware for the Goryu framework.

## Features

- **Fluent API**: Chain methods together for clean, readable middleware definitions
- **Error Handling**: Built-in error handling with customizable error handlers
- **Skip Logic**: Conditionally skip middleware execution
- **Before/After Hooks**: Execute code before and after the main handler
- **Type-Safe**: Full Go type safety with proper error propagation

## Quick Start

```go
import (
    "github.com/arthurlch/goryu"
)

// Simple middleware
middleware := goryu.NewMiddleware("MyMiddleware").
    Before(func(c *goryu.Ctx) error {
        // Pre-processing logic
        return nil
    }).
    Build()

app.Use(middleware)
```

## Examples

### Authentication Middleware

```go
authMiddleware := goryu.NewMiddleware("Auth").
    Before(func(c *goryu.Ctx) error {
        token := c.Request.Header.Get("Authorization")
        if token == "" {
            return errors.New("missing authorization")
        }
        // Validate token...
        return nil
    }).
    OnError(func(c *goryu.Ctx, err error) {
        c.Status(401).FluentJSON(401, map[string]string{
            "error": err.Error(),
        })
    }).
    Build()
```

### Logging Middleware with Timing

```go
loggingMiddleware := goryu.NewMiddleware("Logger").
    Before(func(c *goryu.Ctx) error {
        c.Set("start_time", time.Now())
        return nil
    }).
    After(func(c *goryu.Ctx) error {
        duration := time.Since(c.Get("start_time").(time.Time))
        log.Printf("%s %s - %v", c.Request.Method, c.Request.URL.Path, duration)
        return nil
    }).
    Build()
```

### Conditional Middleware

```go
maintenanceMiddleware := goryu.NewMiddleware("Maintenance").
    Skip(func(c *goryu.Ctx) bool {
        // Skip for admin users
        return c.Get("user_role") == "admin"
    }).
    Before(func(c *goryu.Ctx) error {
        c.Status(503).FluentJSON(503, map[string]string{
            "error": "Service under maintenance",
        })
        return nil
    }).
    Build()
```

## Convenience Builders

The package includes several pre-built middleware builders:

```go
// Security headers
app.Use(goryu.NewSecurityMiddleware().Build())

// CORS
app.Use(goryu.NewCORSMiddleware("https://example.com").Build())

// Request timing
app.Use(goryu.NewTimingMiddleware().Build())

// Request logging
app.Use(goryu.NewLoggingMiddleware().Build())
```

## API Reference

### MiddlewareBuilder Methods

- `Before(func(c *goryu.Ctx) error)`: Set pre-processing function
- `After(func(c *goryu.Ctx) error)`: Set post-processing function
- `Skip(func(c *goryu.Ctx) bool)`: Set skip condition
- `OnError(func(c *goryu.Ctx, err error))`: Set error handler
- `Logger(logger)`: Set custom logger
- `Build()`: Build the middleware
- `BuildSimple(func(c *goryu.Ctx))`: Build simple middleware without error handling