# Cache Middleware

A high-performance, in-memory caching middleware for Goryu. It caches HTTP responses to reduce server load and improve response times for frequently accessed resources.

## Features

- **In-Memory Storage**: Fast access using local memory.
- **LRU Eviction**: Automatically removes least recently used items when limits are reached.
- **Configurable Limits**: Set maximum number of entries and maximum memory usage.
- **TTL Expiration**: Automatically expires entries after a specified duration.
- **Concurrency Safe**: Thread-safe implementation using `sync.RWMutex`.

## Usage

### Basic Setup

Cache GET requests for 5 minutes:

```go
package main

import (
    "time"
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/goryuctx"
    "github.com/arthurlch/goryu/middleware/cache"
)

func main() {
    app := goryu.New()

    // Default configuration (5 min TTL, 1000 entries, 50MB max)
    app.Use(cache.Default())

    app.GET("/slow-data", func(c *goryuctx.Context) {
        time.Sleep(2 * time.Second) // Simulate slow work
        c.JSON(200, map[string]string{"data": "cached"})
    })

    app.Listen(":8080")
}
```

### Advanced Configuration

Customize expiration, memory limits, and cache keys:

```go
app.Use(cache.New(cache.Config{
    Expiration:      10 * time.Minute,
    MaxSize:         5000,       // Max 5000 entries
    MaxMemory:       100 << 20,  // Max 100MB
    CleanupInterval: 1 * time.Minute,
    KeyGenerator: func(c *goryuctx.Context) string {
        // Cache by URL and a custom header
        return c.Request.URL.Path + "|" + c.GetHeader("X-Tenant-ID")
    },
}))
```

## How It Works

1.  **GET Requests Only**: By default, only `GET` requests are cached.
2.  **Key Generation**: Generates a cache key (default: `Method + Path`).
3.  **Cache Lookup**: Checks if a valid entry exists. If found, serves the cached response immediately.
4.  **Response Capture**: If not found, wraps the `ResponseWriter` to capture the status code, headers, and body.
5.  **Storage**: Stores the captured response in the LRU cache if it fits within the memory limits.

## Limitations

- **Single Node**: This is a local, in-memory cache. It is not shared across multiple instances of your application. For distributed caching, consider using Redis.
- **Memory Usage**: Be mindful of the `MaxMemory` setting to prevent your application from consuming too much RAM.
