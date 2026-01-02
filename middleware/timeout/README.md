# Timeout Middleware

A middleware that adds a timeout to the request context. If the request processing takes longer than the specified duration, it cancels the context and returns a `503 Service Unavailable` (or `408 Request Timeout`) response.

## Features

- **Context Cancellation**: Automatically cancels the request context when the timeout is reached.
- **Custom Handler**: Define a custom response when a timeout occurs.
- **Panic Handling**: Safely handles panics within the timed-out goroutine.
- **Race Condition Protection**: Uses atomic operations to prevent writing to the response after a timeout.

## Usage

### Basic Setup

Enforce a 30-second timeout (default):

```go
import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/goryuctx"
    "github.com/arthurlch/goryu/middleware/timeout"
)

app.Use(timeout.Default())
```

### Custom Duration

Set a specific timeout duration:

```go
app.Use(timeout.New(timeout.Config{
    Timeout: 5 * time.Second,
}))
```

### Custom Error Response

Customize the response sent to the client upon timeout:

```go
app.Use(timeout.New(timeout.Config{
    Timeout: 2 * time.Second,
    TimeoutHandler: func(c *goryuctx.Context) {
        c.JSON(http.StatusGatewayTimeout, map[string]string{
            "error": "Processing took too long",
        })
    },
}))
```

## How It Works

1.  **Goroutine**: The middleware runs the next handler in a separate goroutine.
2.  **Select**: It waits for either the handler to finish or the context to time out.
3.  **Cancellation**: If the timeout is reached first, it writes the timeout response and cancels the context.
4.  **Safety**: It wraps the `ResponseWriter` to ensure that the handler cannot write to the response after the timeout has occurred.

## Important Note

Because the handler runs in a separate goroutine, you must ensure that any data accessed from the `Context` is thread-safe or not modified concurrently. Standard `goryuctx.Context` usage is generally safe, but be careful with shared mutable state.
