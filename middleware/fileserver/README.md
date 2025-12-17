# Fileserver Middleware

A secure and flexible middleware for serving static files from a directory. It includes built-in protection against directory traversal attacks and supports directory browsing.

## Features

- **Secure Serving**: Validates paths to prevent directory traversal attacks (e.g., `../../etc/passwd`).
- **Path Stripping**: Serves files from a specific URL prefix.
- **Directory Browsing**: Optionally allows listing directory contents.
- **Index Files**: Automatically serves `index.html` or `index.htm` for directories.
- **Sanitization**: Cleans and validates file paths before access.

## Usage

### Basic Setup

Serve files from the `./static` directory at the `/static` URL path:

```go
app.Use(fileserver.WithPrefix("/static", "./static"))
```

### Serve Root Directory

Serve files from the current directory at the root URL (use with caution):

```go
app.Use(fileserver.WithRoot("."))
```

### Enable Directory Browsing

Allow users to view file listings:

```go
app.Use(fileserver.WithBrowse("./public", true))
```

### Advanced Configuration

```go
app.Use(fileserver.New(fileserver.Config{
    Root:       "./assets",
    PathPrefix: "/assets",
    Browse:     false,
    Index:      []string{"main.html", "index.html"},
}))
```

## Security Note

This middleware implements strict path validation to ensure that users cannot access files outside the specified `Root` directory. It checks for:
- Path traversal sequences (`..`, `%2e%2e`, etc.)
- Null bytes and other suspicious characters
- Absolute path escaping

However, always ensure that your `Root` directory does not contain sensitive files (like configuration files or source code) that you do not intend to expose.
