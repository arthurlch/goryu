# Graceful Middleware

A middleware and utility package for handling graceful shutdowns in Goryu applications. It ensures that active connections are drained and cleanup tasks are executed before the server stops.

## Features

- **Graceful Shutdown**: Waits for active requests to complete before shutting down.
- **Signal Handling**: Automatically listens for `SIGINT` and `SIGTERM`.
- **Cleanup Hooks**: Register functions to run during shutdown (e.g., closing DB connections).
- **Timeout**: Enforces a maximum shutdown duration to prevent hanging.
- **Connection Tracking**: Middleware to track the number of active connections.

## Usage

### Basic Setup

Use the `RunWithGracefulShutdown` helper to start your server:

```go
package main

import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/goryuctx"
    "github.com/arthurlch/goryu/middleware/graceful"
    "time"
)

func main() {
    app := goryu.New()

    app.GET("/", func(c *goryuctx.Context) {
        // Simulate work
        time.Sleep(2 * time.Second)
        c.Status(200).String("Done")
    })

    // Starts the server and handles shutdown signals
    graceful.RunWithGracefulShutdown(app, ":8080")
}
```

### With Cleanup Tasks

Register functions to close resources (Database, Redis, etc.) during shutdown:

```go
graceful.RunWithGracefulShutdown(app, ":8080", graceful.ShutdownConfig{
    Timeout: 10 * time.Second,
    CleanupFuncs: []func() error{
        func() error {
            fmt.Println("Closing database connection...")
            return db.Close()
        },
        func() error {
            fmt.Println("Flushing logs...")
            return logger.Sync()
        },
    },
})
```

### Connection Tracking Middleware

If you want to track active connections in your application logic (e.g., for health checks or metrics):

```go
app.Use(graceful.New(graceful.Config{
    ContextKey: "active_conns",
}))

app.GET("/status", func(c *goryuctx.Context) {
    count, _ := c.Get("active_conns")
    c.JSON(200, map[string]interface{}{
        "active_connections": count,
    })
})
```

## How It Works

1.  **Signal Listener**: A goroutine listens for OS signals (`SIGINT`, `SIGTERM`).
2.  **Shutdown Trigger**: When a signal is received, it calls `server.Shutdown(ctx)`.
3.  **Draining**: The HTTP server stops accepting new connections and waits for existing handlers to return.
4.  **Cleanup**: After the server stops (or times out), the registered `CleanupFuncs` are executed in order.
