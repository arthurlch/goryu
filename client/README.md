# Goryu HTTP Client

A fluent and chainable HTTP client for Go with a clean API design, built into the Goryu framework.

## Overview

The Goryu client provides an intuitive way to make HTTP requests with method chaining, automatic error handling, and built-in support for common patterns like JSON payloads, form data, authentication, and more.

## Features

- **Fluent API**: Chain methods for clean and readable code
- **All HTTP Methods**: Support for GET, POST, PUT, DELETE, PATCH, HEAD
- **JSON Support**: Easy JSON request/response handling
- **Form Data**: Built-in form encoding support
- **Authentication**: Basic auth and custom headers
- **Cookie Management**: Simple cookie handling
- **Query Parameters**: Chainable query parameter building
- **Timeouts**: Configurable request timeouts
- **TLS Options**: Support for custom TLS configurations
- **Error Aggregation**: Collect errors throughout the request chain

## Quick Start

```go
import "github.com/arthurlch/goryu/client"

// Simple GET request
code, body, errs := client.Get("https://api.example.com/users").String()

// POST with JSON
type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

user := User{Name: "John", Email: "john@example.com"}
code, body, errs := client.Post("https://api.example.com/users").
    JSON(user).
    String()

// Complex request with multiple options
var result ResponseType
code, body, errs := client.Get("https://api.example.com/data").
    UserAgent("MyApp/1.0").
    BasicAuth("user", "pass").
    Query("page", "1").
    Query("limit", "10").
    Timeout(30 * time.Second).
    Struct(&result)
```

## API Reference

### Creating Requests

```go
// Using package-level functions
client.Get(url)
client.Post(url)
client.Put(url)
client.Delete(url)
client.Patch(url)
client.Head(url)

// Or create an agent first
agent := client.New()
agent.Get(url)
```

### Request Configuration

#### Headers
```go
// Set a header (replaces existing)
agent.Set("Authorization", "Bearer token")

// Add a header (appends to existing)
agent.Add("Accept", "application/json")

// Convenience methods
agent.UserAgent("MyApp/1.0")
agent.ContentType("application/json")
```

#### Request Body
```go
// Raw bytes
agent.Body([]byte("raw data"))

// JSON payload
agent.JSON(structOrMap)

// Form data
formData := url.Values{}
formData.Add("key", "value")
agent.Form(formData)
```

#### Authentication
```go
// Basic authentication
agent.BasicAuth("username", "password")

// Bearer token (via Set)
agent.Set("Authorization", "Bearer your-token")
```

#### Other Options
```go
// Query parameters
agent.Query("search", "goryu")

// Cookies
agent.Cookie("session_id", "abc123")

// Timeout
agent.Timeout(10 * time.Second)

// Skip TLS verification (use with caution)
agent.InsecureSkipVerify()
```

### Getting Responses

```go
// Get response as string
statusCode, body, errors := agent.String()

// Get response as bytes
statusCode, body, errors := agent.Bytes()

// Decode JSON response into struct
var data MyStruct
statusCode, body, errors := agent.Struct(&data)
```

## Error Handling

The client aggregates errors throughout the request chain. Always check the errors slice:

```go
code, body, errs := client.Get("https://api.example.com").String()
if len(errs) > 0 {
    for _, err := range errs {
        log.Printf("Error: %v", err)
    }
    return
}
```

## Examples

### JSON API Request
```go
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

type CreateUserResponse struct {
    ID        int    `json:"id"`
    CreatedAt string `json:"created_at"`
}

req := CreateUserRequest{
    Name:  "Jane Doe",
    Email: "jane@example.com",
}

var resp CreateUserResponse
code, _, errs := client.Post("https://api.example.com/users").
    JSON(req).
    Set("X-API-Key", "secret").
    Struct(&resp)

if len(errs) > 0 {
    // Handle errors
}

if code == 201 {
    fmt.Printf("User created with ID: %d\n", resp.ID)
}
```

### Form Submission
```go
formData := url.Values{}
formData.Add("username", "johndoe")
formData.Add("password", "secret123")

code, body, errs := client.Post("https://example.com/login").
    Form(formData).
    Cookie("csrf_token", "xyz789").
    String()
```

### Downloading with Progress
```go
code, data, errs := client.Get("https://example.com/large-file.zip").
    UserAgent("Goryu-Downloader/1.0").
    Timeout(5 * time.Minute).
    Bytes()

if len(errs) == 0 && code == 200 {
    // Save to file
    err := os.WriteFile("downloaded.zip", data, 0644)
}
```

## Testing

The client is designed to work well with Go's `httptest` package:

```go
func TestMyHandler(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(myHandler))
    defer server.Close()
    
    code, body, errs := client.Get(server.URL + "/endpoint").String()
    // Assert on code, body, errs
}
```

## Performance Considerations

- The client creates a new `http.Client` for each request chain by default
- For multiple requests to the same host, consider reusing an agent:
  ```go
  agent := client.New()
  agent.Timeout(30 * time.Second)
  
  // Reuse agent for multiple requests
  agent.Get(url1).String()
  agent.Get(url2).String()
  ```
- Response bodies are fully read into memory; for large responses, consider using the standard `net/http` package directly

## Notes

- All errors are collected and returned together for better debugging
- The client is not goroutine-safe; create separate agents for concurrent requests
- TLS verification should only be disabled in development/testing environments