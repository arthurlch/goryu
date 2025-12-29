# Logger Middleware

A highly customizable HTTP request logger for Goryu. It provides detailed request/response logging with support for custom formats, colors, and output destinations.

## Features

- **Customizable Format**: Define exactly what you want to log using template variables.
- **Color Coded**: Visual status codes and methods for better readability in development.
- **Request IDs**: Automatically generates or propagates `X-Request-ID`.
- **Latency Tracking**: Measures and logs request duration.
- **Flexible Output**: Write logs to stdout, files, or any `io.Writer`.

## Usage

### Basic Setup

Use the default format (colored output to stdout):

```go
app.Use(logger.Default())
```

### Custom Format

Customize the log output using template variables:

```go
app.Use(logger.New(logger.Config{
    Format: "${time} | ${status} | ${latency} | ${method} ${path} | ${error}\n",
}))
```

**Available Variables:**
- `${time}`: Timestamp (RFC3339 by default)
- `${status}`: HTTP status code (colored)
- `${latency}`: Request duration
- `${ip}`: Client IP address
- `${method}`: HTTP method (colored)
- `${path}`: Request path
- `${proto}`: HTTP protocol version
- `${request_id}`: Request ID
- `${user_agent}`: User Agent string
- `${size}`: Response size in bytes
- `${error}`: Error message (if any)

### JSON Logging (for Production)

For structured logging, you can configure the format to output JSON:

```go
app.Use(logger.New(logger.Config{
    Format: `{"time":"${time}","level":"INFO","msg":"http_request","id":"${request_id}","method":"${method}","uri":"${path}","status":${status},"latency":"${latency}"}` + "\n",
    DisableColors: true,
}))
```

### Advanced Configuration

```go
file, _ := os.OpenFile("requests.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

app.Use(logger.New(logger.Config{
    Output:        file,
    TimeFormat:    time.RFC1123,
    TimeZone:      "UTC",
    DisableColors: true,
}))
```

## Performance Note

The logger uses a buffered writer and efficient string replacement to minimize overhead. However, for extremely high-throughput systems, consider sampling logs or using the `structlog` middleware for zero-allocation structured logging.
