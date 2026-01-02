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
import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/middleware/limiter"
)

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

## Configuration Options

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| Max | `int` | Maximum requests allowed | `60` |
| Expiration | `time.Duration` | Time window duration | `1 * time.Minute` |
| KeyGenerator | `func(*goryuctx.Context) string` | Function to extract client key | IP-based |
| LimitReached | `func(*goryuctx.Context)` | Custom response when limit is reached | JSON error response |

## How It Works

The middleware tracks requests using an in-memory map where:
1. **Client Identification**: Uses `KeyGenerator` function (default: client IP)
2. **Fixed Window**: Counts requests within specified time intervals
3. **Automatic Cleanup**: Removes expired client entries to manage memory
4. **Thread-Safe**: Uses mutexes for concurrent request handling

## Security Note

When using IP-based limiting (default), be aware of:
- Users behind NAT may share IP addresses
- Consider using API keys or authenticated user IDs for more accurate limiting
- Set appropriate limits to balance security and user experience