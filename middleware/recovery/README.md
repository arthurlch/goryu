# Recovery Middleware

A critical middleware that recovers from panics anywhere in the middleware chain or handler. It prevents the server from crashing and returns a `500 Internal Server Error` response to the client.

## Features

- **Crash Prevention**: Catches panics and keeps the server running.
- **Stack Traces**: Logs detailed stack traces for debugging (configurable).
- **Custom Handling**: Define custom logic for handling panics (e.g., sending alerts).
- **Safe Response**: Ensures a valid HTTP response is sent even after a panic.

## Usage

### Basic Setup

Recover from panics and log stack traces (default behavior):

```go
app.Use(recovery.Default())
```

### Custom Recovery Handler

Customize the response or perform side effects (like reporting to Sentry):

```go
app.Use(recovery.New(recovery.Config{
    EnableStackTrace: true,
    CustomRecoveryHandler: func(c *goryuctx.Context, err interface{}) {
        // Log to external service
        reportToSentry(err)
        
        // Send custom response
        c.Status(http.StatusInternalServerError).JSON(map[string]string{
            "error": "Unexpected error occurred. We have been notified.",
        })
    },
}))
```

## How It Works

1.  **Defer**: The middleware uses a `defer` statement to catch any panic that occurs during the execution of the `next` handler.
2.  **Recover**: If a panic occurs, `recover()` captures the error value.
3.  **Log**: It logs the error and the stack trace (if enabled).
4.  **Respond**: It sends a `500` response to the client if headers haven't already been written.

## Best Practices

- **Always Use It**: This middleware should be one of the first in your chain (after `Logger` and `RequestID`) to ensure it catches panics from all other middleware and handlers.
- **Production**: Keep stack traces enabled in logs but **never** expose them to the client in the HTTP response.
