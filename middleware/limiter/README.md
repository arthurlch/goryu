# Limiter Middleware

A rate limiting middleware for Goryu. It protects your application from abuse by limiting the number of requests a client can make within a specified time window.

## Features

- **Fixed Window**: Limits requests per time interval (e.g., 60 requests per minute).
- **In-Memory Storage**: Fast and efficient local storage.
- **Client Management**: Automatically cleans up old client entries to manage memory.
- **Custom Keys**: Rate limit by IP (default), API key, or any other identifier.
- **Custom Responses**: Define what happens when the limit is reached.

## Usage

### Basic Setup

Limit to 60 requests per minute per IP:

```go
app.Use(limiter.Default())
```

### Custom Limit

Limit to 10 requests per second:

```go
app.Use(limiter.New(limiter.Config{
    Max:        10,
    Expiration: 1 * time.Second,
}))
```

### Advanced Configuration

Rate limit by API key with a custom error response:

```go
app.Use(limiter.New(limiter.Config{
    Max:        1000,
    Expiration: 1 * time.Hour,
    KeyGenerator: func(c *goryuctx.Context) string {
        return c.GetHeader("X-API-Key")
    },
    LimitReached: func(c *goryuctx.Context) {
        c.Status(http.StatusTooManyRequests).JSON(map[string]string{
            "error": "Rate limit exceeded. Try again later.",
        })
    },
}))
```

## How It Works

1.  **Identification**: The middleware identifies the client using the `KeyGenerator` (default: Remote IP).
2.  **Tracking**: It maintains a counter for each client in memory.
3.  **Windowing**: If the time since the last access exceeds `Expiration`, the counter is reset.
4.  **Enforcement**: If the counter exceeds `Max`, the `LimitReached` handler is called.
5.  **Cleanup**: Periodically removes inactive clients to prevent memory leaks.

## Limitations

- **Single Node**: This is a local, in-memory limiter. It does not synchronize limits across multiple instances of your application. For distributed rate limiting, use an external store like Redis.
