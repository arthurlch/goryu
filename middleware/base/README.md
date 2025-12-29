# Base Middleware Package

The `base` package provides the foundational building blocks for creating consistent and robust middlewares in Goryu. It handles common concerns like configuration, logging, and error handling, allowing you to focus on the specific logic of your middleware.

## Features

- **Standard Configuration**: A `BaseConfig` struct that all middlewares can embed.
- **Unified Logging**: Consistent logging interface via `DefaultLogger`.
- **Error Handling**: Centralized error handling with `DefaultErrorHandler`.
- **Middleware Helpers**: Helper functions like `StandardMiddleware` and `PostProcessMiddleware` to reduce boilerplate.

## Usage

### Creating a Custom Middleware

Here is how you can use the `base` package to create a custom middleware:

```go
package mymiddleware

import (
    "github.com/arthurlch/goryu/context"
    "github.com/arthurlch/goryu/middleware/base"
)

// 1. Define your Config struct embedding base.BaseConfig
type Config struct {
    base.BaseConfig
    MyOption string
}

// 2. Implement the Configure method
func (c *Config) Configure(baseConfig *base.BaseConfig) {
    c.BaseConfig = *baseConfig
}

// 3. Implement the Validate method
func (c *Config) Validate() error {
    if c.MyOption == "" {
        return base.NewConfigError("MyOption", "cannot be empty")
    }
    return nil
}

// 4. Create the constructor
func New(config Config) func(next context.HandlerFunc) context.HandlerFunc {
    // Validate config
    if err := config.Validate(); err != nil {
        return func(next context.HandlerFunc) context.HandlerFunc {
            return func(c *context.Context) {
                base.DefaultErrorHandler(c, err, "MyMiddleware")
            }
        }
    }

    // Define the handler logic
    handler := func(c *context.Context) error {
        // Your middleware logic here
        config.Logger.Printf("Executing MyMiddleware with option: %s", config.MyOption)
        return nil
    }

    // Use the helper to wrap it
    return base.StandardMiddleware("MyMiddleware", config.BaseConfig, handler)
}
```

### Post-Processing Middleware

If your middleware needs to run logic *after* the next handler returns (e.g., logging response time), use `PostProcessMiddleware`:

```go
func NewTimer(config Config) func(next context.HandlerFunc) context.HandlerFunc {
    preHandler := func(c *context.Context) error {
        c.Set("start_time", time.Now())
        return nil
    }

    postHandler := func(c *context.Context) error {
        start := c.Get("start_time").(time.Time)
        duration := time.Since(start)
        config.Logger.Printf("Request took %v", duration)
        return nil
    }

    return base.PostProcessMiddleware("Timer", config.BaseConfig, preHandler, postHandler)
}
```
