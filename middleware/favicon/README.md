# Favicon Middleware

A middleware for efficiently serving the favicon of your Goryu application. It supports in-memory caching and automatic `Cache-Control` header management.

## Features

- **Efficient Serving**: Serves the favicon file directly.
- **Caching**: Caches the file content in memory to reduce disk I/O.
- **Browser Caching**: Automatically sets `Cache-Control` headers for client-side caching.
- **No Content**: Can serve a `204 No Content` response if no file is provided (to silence browser 404s).

## Usage

### Basic Setup

Serve a `204 No Content` response for `/favicon.ico` requests (useful for APIs):

```go
app.Use(favicon.Default())
```

### Serve a File

Serve a specific file from disk:

```go
app.Use(favicon.WithFile("./assets/favicon.ico"))
```

### Advanced Configuration

```go
app.Use(favicon.New(favicon.Config{
    File:      "./static/icon.png",
    URL:       "/favicon.ico",
    CacheFile: true,            // Cache in memory
    MaxAge:    3600,            // Cache-Control max-age in seconds
}))
```

## Why use this?

Browsers automatically request `/favicon.ico` when visiting a website. If you don't handle this request, your logs will be cluttered with `404 Not Found` errors. This middleware handles it efficiently, either by serving the file or by returning a "No Content" response, keeping your logs clean and your users happy.
