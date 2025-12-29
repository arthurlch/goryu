# Plugins Package

The `plugins` package provides a fluent, type-safe builder API for configuring Goryu middlewares. It simplifies complex configurations and provides sensible defaults for different environments (Development vs. Production).

## Features

- **Fluent API**: Chainable methods for easy configuration.
- **Type Safety**: Compile-time checks for configuration options.
- **Presets**: Built-in configurations for `Development`, `Production`, and more.
- **Validation**: Automatic validation of configuration values before building.
- **Registry**: A system to register and discover plugins.

## Available Plugins

### Logger Plugin

Configure the request logger with ease.

```go
// Development Logger (Colors, Human-readable time)
app.Use(plugins.NewLoggerBuilder().
    Development().
    Build())

// Production Logger (JSON, No colors, RFC3339 time)
app.Use(plugins.NewLoggerBuilder().
    Production().
    Output(os.Stdout).
    Build())

// Custom Format
app.Use(plugins.NewLoggerBuilder().
    Format("[${status}] ${method} ${path} (${latency})").
    Build())
```

### Recovery Plugin

Manage panic recovery and stack traces.

```go
// Development (Show stack traces)
app.Use(plugins.NewRecoveryBuilder().
    Development().
    Build())

// Production (JSON response, no stack traces)
app.Use(plugins.NewRecoveryBuilder().
    Production().
    JSONResponse().
    Build())
```

### CORS Plugin

Configure Cross-Origin Resource Sharing policies.

```go
// Development (Permissive)
app.Use(plugins.NewCORSBuilder().
    Development().
    Build())

// Production (Restrictive)
app.Use(plugins.NewCORSBuilder().
    Production().
    AllowOrigins("https://example.com").
    AllowMethods("GET", "POST").
    Build())

// Custom API Configuration
app.Use(plugins.NewCORSBuilder().
    AllowOrigins("https://app.example.com").
    AllowMethods("GET", "POST", "PUT", "DELETE").
    AllowHeaders("Authorization", "Content-Type").
    AllowCredentials(true).
    MaxAge(3600).
    Build())
```

### Rate Limit Plugin

Protect your application from abuse.

```go
// Simple IP-based limit (100 req/min)
app.Use(plugins.NewRateLimitBuilder().
    PerMinute(100).
    ByIP().
    Build())

// API Key-based limit
app.Use(plugins.NewRateLimitBuilder().
    Rate(1000, time.Hour).
    ByAPIKey("X-API-Key").
    JSONResponse().
    Build())

// Presets
app.Use(plugins.NewRateLimitBuilder().Conservative().Build()) // 60/min
app.Use(plugins.NewRateLimitBuilder().Generous().Build())     // 1000/min
```

## Plugin Registry

The package includes a registry system for discovering and creating plugins dynamically.

```go
// List available plugins
plugins := plugins.List()

// Get a builder by name
if builder, ok := plugins.Get("cors"); ok {
    // Cast to specific builder if needed, or use generic interface
    middleware := builder.(plugins.Builder).Build()
    app.Use(middleware)
}
```

## Creating Custom Plugins

You can create your own plugins by implementing the `Builder` interface and registering them:

```go
func init() {
    plugins.Register("my-plugin", func() plugins.Builder {
        return NewMyPluginBuilder()
    })
}
```
