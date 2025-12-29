# Errors Middleware

A comprehensive error handling middleware for Goryu. It centralizes error management, provides consistent response formatting, and handles panics gracefully.

## Features

- **Centralized Handling**: Catches errors returned by handlers or stored in the context.
- **Panic Recovery**: Automatically recovers from panics and converts them into internal server errors.
- **Consistent Formatting**: Returns errors in a standardized JSON format.
- **Dev Mode**: Optionally exposes stack traces and internal error details for debugging.
- **Customization**: Supports custom error transformers and response formatters.

## Usage

### Basic Setup

Enable default error handling:

```go
app.Use(errors.New())
```

### Custom Configuration

Enable development mode to see stack traces:

```go
app.Use(errors.NewWithConfig(errors.Config{
    ShowDetails:    true,
    ShowStackTrace: true, // Only use in development!
    DevMode:        true,
    LogErrors:      true,
}))
```

### Returning Errors

In your handlers, simply return an error:

```go
app.GET("/users/:id", func(c *goryuctx.Context) {
    user, err := findUser(c.Param("id"))
    if err != nil {
        // Return a 404 error
        c.Status(http.StatusNotFound).JSON(errors.NewError("USER_NOT_FOUND", "User not found").
            Build())
        return
    }
    c.JSON(200, user)
})
```

### Standard Error Format

The middleware returns errors in the following JSON structure:

```json
{
  "error": {
    "code": "USER_NOT_FOUND",
    "message": "User not found",
    "details": { ... }, // Optional
    "request_id": "req-123",
    "timestamp": "2023-10-27T10:00:00Z"
  }
}
```

## Advanced Features

- **Error Transformer**: Modify errors before they are processed (e.g., map database errors to application errors).
- **Custom Handler**: Execute custom logic when an error occurs (e.g., logging to an external service).
- **Response Formatter**: Completely change the JSON structure of the error response.