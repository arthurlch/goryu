# StructLog Middleware

A high-performance, structured logging middleware for Goryu, built on top of Go's standard `log/slog` package. It provides zero-allocation JSON logging suitable for production environments.

## Features

- **Structured Logging**: Outputs logs in JSON format by default.
- **Request Correlation**: Automatically adds `request_id` to all log entries.
- **Context Integration**: Helper functions to log with context (`LogInfo`, `LogError`).
- **Performance**: Built on `log/slog` for minimal allocation overhead.
- **Detailed Attributes**: Logs status, method, path, duration, IP, user agent, and response size.

## Usage

### Basic Setup

```go
package main

import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/goryuctx"
    "github.com/arthurlch/goryu/middleware/structlog"
)

func main() {
    app := goryu.New()

    // Use default configuration (JSON output to stdout)
    app.Use(structlog.Default())

    app.GET("/", func(c *goryuctx.Context) {
        // Log with correlation ID
        structlog.LogInfo(c, "handling_request", "user_id", "123")
        
        c.JSON(200, map[string]string{"status": "ok"})
    })

    app.Listen(":8080")
}
```

### Advanced Configuration

Customize the logger instance or output:

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

app.Use(structlog.New(structlog.Config{
    Logger: logger,
    CustomFields: func(c *goryuctx.Context) map[string]any {
        return map[string]any{
            "tenant_id": c.GetHeader("X-Tenant-ID"),
        }
    },
}))
```

### Context Logging

The middleware injects a correlated logger into the context. You can use helper functions to log messages that automatically include the `request_id`:

- `structlog.LogInfo(c, msg, args...)`
- `structlog.LogWarn(c, msg, args...)`
- `structlog.LogError(c, msg, err, args...)`

Or get the logger instance directly:

```go
logger := structlog.CorrelatedLogger(c)
logger.Info("custom message")
```

## Output Example

```json
{
  "time": "2023-10-27T10:00:00Z",
  "level": "INFO",
  "msg": "GET /api/users 200",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "GET",
  "path": "/api/users",
  "status_code": 200,
  "duration_ms": "12.5",
  "remote_ip": "192.168.1.1",
  "user_agent": "Mozilla/5.0...",
  "custom": {
    "tenant_id": "tenant_1"
  }
}
```
