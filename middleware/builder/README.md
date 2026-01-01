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
    "github.com/arthurlch/goryu/goryuctx"
)

// Simple middleware
middleware := goryu.NewMiddleware("MyMiddleware").
    Before(func(c *goryuctx.Context) error {
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
    Before(func(c *goryuctx.Context) error {
        token := c.Request.Header.Get("Authorization")
        if token == "" {
            return errors.New("missing authorization")
        }
        // Validate token...
        return nil
    }).
    OnError(func(c *goryuctx.Context, err error) {
        c.Status(401).JSON(goryuctx.Map{
            "error": err.Error(),
        })
    }).
    Build()
```

### Logging Middleware with Timing

```go
loggingMiddleware := goryu.NewMiddleware("Logger").
    Before(func(c *goryuctx.Context) error {
        c.Set("start_time", time.Now())
        return nil
    }).
    After(func(c *goryuctx.Context) error {
        duration := time.Since(c.Get("start_time").(time.Time))
        log.Printf("%s %s - %v", c.Request.Method, c.Request.URL.Path, duration)
        return nil
    }).
    Build()
```

### Conditional Middleware

```go
maintenanceMiddleware := goryu.NewMiddleware("Maintenance").
    Skip(func(c *goryuctx.Context) bool {
        // Skip for admin users
        return c.Get("user_role") == "admin"
    }).
    Before(func(c *goryuctx.Context) error {
        c.Status(503).JSON(goryuctx.Map{
            "error": "Service under maintenance",
        })
        return nil
    }).
    Build()
```

## Additional Features

The builder pattern allows for complex middleware composition and provides a clean, readable way to define middleware behavior.

## API Reference

### MiddlewareBuilder Methods

- `Before(func(c *goryuctx.Context) error)`: Set pre-processing function
- `After(func(c *goryuctx.Context) error)`: Set post-processing function
- `Skip(func(c *goryuctx.Context) bool)`: Set skip condition
- `OnError(func(c *goryuctx.Context, err error))`: Set error handler
- `Logger(logger)`: Set custom logger
- `Build()`: Build the middleware
- `BuildSimple(func(c *goryuctx.Context))`: Build simple middleware without error handling