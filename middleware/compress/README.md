# Compress Middleware

A middleware that compresses HTTP responses using Gzip or Deflate. It automatically negotiates the best encoding based on the client's `Accept-Encoding` header.

## Features

- **Multiple Algorithms**: Supports `gzip` and `deflate`.
- **Content Filtering**: Only compresses specific content types (e.g., HTML, JSON, XML).
- **Size Threshold**: Skips compression for small responses (configurable `MinLength`) to avoid overhead.
- **Configurable Levels**: Adjust compression level (BestSpeed vs BestCompression).

## Usage

### Basic Setup

Enable compression with default settings (Gzip/Deflate, default level):

```go
app.Use(compress.Default())
```

### Gzip Only

Force Gzip compression:

```go
app.Use(compress.WithGzip(compress.LevelBestCompression))
```

### Advanced Configuration

```go
app.Use(compress.New(compress.Config{
    Level:     compress.LevelBestSpeed, // Prioritize speed
    MinLength: 2048,                    // Only compress if > 2KB
    CompressibleTypes: []string{
        "text/html",
        "application/json",
        "image/svg+xml",
    },
}))
```

## How It Works

1.  **Negotiation**: Checks the `Accept-Encoding` header.
2.  **Buffering**: Buffers the response to determine its size.
3.  **Filtering**: Checks if the `Content-Type` is in the allowed list and if the size exceeds `MinLength`.
4.  **Compression**: If criteria are met, compresses the body and sets `Content-Encoding`. Otherwise, sends the original response.

## Performance Note

Compression consumes CPU. While it reduces bandwidth usage and improves client load times, be mindful of the CPU overhead on high-traffic servers. `LevelBestSpeed` is often a good trade-off.
