# RequestID Middleware

A middleware that adds a unique identifier to every HTTP request. It can generate a new ID or propagate an existing one from the request headers.

## Features

- **Automatic Generation**: Generates a cryptographically secure random UUID (or hex string) if no ID is present.
- **Propagation**: Respects existing IDs passed in headers (e.g., from a load balancer).
- **Context Integration**: Stores the ID in the context for use by other middleware (like loggers).
- **Response Header**: Adds the ID to the response headers for client-side tracking.

## Usage

### Basic Setup

```go
package main

import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/goryuctx"
    "github.com/arthurlch/goryu/middleware/requestid"
)

func main() {
    app := goryu.New()

    app.Use(requestid.Default())

    app.GET("/", func(c *goryuctx.Context) {
        id, exists := c.Get("requestid")
        if exists {
            c.Status(200).String("Request ID: "+id.(string))
        } else {
            c.Status(200).String("No request ID found")
        }
    })

    app.Listen(":8080")
}
```

### Advanced Configuration

Customize the header name or ID generator:

```go
app.Use(requestid.New(requestid.Config{
    Header: "X-Correlation-ID",
    Generator: func() string {
        return "custom-id-" + time.Now().String()
    },
}))
```

## Why is this useful?

- **Debugging**: Correlate logs across different services or components using a single ID.
- **Tracing**: Track a request's journey through your distributed system.
- **Support**: Provide the Request ID to users in error responses so they can reference it when contacting support.
