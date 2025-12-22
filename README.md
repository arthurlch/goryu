# 🐉 Goryu Framework

> **A GOated Web Framework for Go. Built for Developer Happiness.**

[![Go Report Card](https://goreportcard.com/badge/github.com/arthurlch/goryu)](https://goreportcard.com/report/github.com/arthurlch/goryu)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Goryu is an **opinionated, batteries-included** web framework that prioritizes **Developer Experience (DX)** without sacrificing performance. It feels like express/fiber, but with the robustness of Go.

## 🚀 Why Goryu?

*   **⚡ Zero-Config Start**: `goryu.Default()` gives you a production-ready app with logging, recovery, and monitoring instantly.
*   **👁️ Visual Observability**: Built-in **Real-time Dashboard**, colorful logs, and startup **Route Tables**.
*   **🧠 Smart Context**: Unified `BodyParser`, Auto-Validation, and generic `goryu.Map` for clean code.
*   **🛡️ Production Grade**: Graceful shutdown, health checks, and security headers enabled by default.

[**🎓 Start the Tutorial: Build a Todo App in 5 min**](./TUTORIAL.md)

---

## 📦 Installation

```bash
# The library
go get github.com/arthurlch/goryu

# The CLI (Recommended for new projects)
go install github.com/arthurlch/goryu/cmd/goryu@latest
```

## ⚡ Quick Start

Create a `main.go` and experience the magic:

```go
package main

import (
    "github.com/arthurlch/goryu"
)

// define a structured request with validation
type CreateUser struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (u *CreateUser) Validate() error {
    if len(u.Name) < 3 { return fmt.Errorf("name too short") }
    return nil
}

func main() {
    // 1. Create app with Logger, Recovery, Metrics & Dashboard
    app := goryu.Default()

    // 2. Define routes with smart binding
    app.POST("/users", func(c *goryu.Ctx) {
        var req CreateUser
        
        // Auto-binds JSON/Form AND runs Validate()
        if err := c.BodyParser(&req); err != nil {
            c.JSON(400, goryu.Map{"error": err.Error()})
            return
        }

        c.JSON(201, goryu.Map{
            "status": "created", 
            "user": req.Name,
        })
    })

    // 3. Run it! (Prints route table on start)
    app.Listen(":8080")
}
```

### What just happened?
When you run this, Goryu automatically:
1.  Started a **Real-time Monitoring Dashboard** at `http://localhost:8080/_dashboard`.
2.  printed a **Route Table** to your console so you know what's running.
3.  Enabled **Panic Recovery** so your server never crashes.
4.  Set up **Request Logging** with colors and tracing IDs.

---

## 🌟 Key Features

### 1. Smart Request Binding
Stop writing boilerplate. `BodyParser` handles JSON, Forms, and Query params in one line. Plus, if your struct implements `Validate() error`, it runs automatically!

```go
// One line to bind AND validate
if err := c.BodyParser(&payload); err != nil {
    return c.Status(400).JSON(goryu.Map{"error": err.Error()})
}
```

### 2. Built-in Monitoring UI
No need for Prometheus/Grafana for simple apps. Goryu ships with a zero-dependency SPA dashboard.
- **/_dashboard**: Visual stats & event log.
- **/_health**: JSON health check.
- **/_metrics**: Prometheus-compatible metrics.

### 3. Developer Experience (DX) First
- **goryu.Map**: `map[string]interface{}` is too long. Use `goryu.Map{}`.
- **Route Table**:
    ```text
    📍 Registered Routes:
       METHOD   PATH             NAME
       -----------------------------------
       GET      /_dashboard      Goryu Monitoring...
       POST     /users           -
    ```

---

## 📚 Documentation

The Goryu router can be configured with various options to control its behavior:

```go
app := goryu.New(goryu.Config{
    AppName:               "MyApp",
    RedirectTrailingSlash: &[]bool{true}[0],  // Enable trailing slash redirection
    EnableHEADFallback:    &[]bool{true}[0],  // Enable HEAD fallback to GET
})
```

### Trailing Slash Handling

By default, Goryu handles trailing slashes intelligently:

- When `RedirectTrailingSlash` is true (default), requests to `/users/` will redirect to `/users` if only `/users` is defined
- When `StrictRouting` is enabled, `/users` and `/users/` are treated as different routes
- Redirects use `301 Moved Permanently` for GET requests and `308 Permanent Redirect` for other methods

### HEAD Method Support

- By default, HEAD requests automatically fall back to GET handlers
- You can disable this by setting `EnableHEADFallback` to false
- Explicit HEAD routes take precedence over GET fallback

### ALL Method Enhancement

The `ALL` method now returns a `RouteCollection` containing all registered routes:

```go
collection := app.ALL("/api", handler)
collection.SetName("api_endpoints") // Sets names like "api_endpoints_get", "api_endpoints_post", etc.
```

## Core Context API

The Context object provides a clean, consistent API for handling HTTP requests and responses. All response methods now include proper error handling and return errors for robust error management.

### Key Improvements

- **Simplified Structure**: Removed redundant `Req` field - use `ctx.Request` instead
- **Enhanced Error Handling**: All response methods return errors and include detailed logging
- **Security Enhancements**: File serving includes path traversal protection and proper error handling
- **Flexible Error Responses**: Multiple error handling methods for different use cases

These are the fundamental methods for passing data through the request lifecycle.

### `Set(key string, value interface{})`

Stores a key-value pair in the context. This is the primary way to pass data from middleware to your handlers.

### `Get(key string) (value interface{}, exists bool)`

Retrieves a value from the context by its key. It returns the value and a boolean indicating if the key existed.

**Example: Middleware Authentication**
Here's how you can use `Set` and `Get` to create a simple authentication middleware.

```go
// auth_middleware.go
func AuthMiddleware(next goryu.Handler) goryu.Handler {
    return func(ctx *goryu.Ctx) {
        // Imagine you validate a token from the "Authorization" header
        token := ctx.GetHeader("Authorization")
        if user, valid := validateToken(token); valid {
            // If valid, set the user's data in the context
            ctx.Set("user", user)
            next(ctx) // Call the next handler in the chain
        } else {
            ctx.Status(http.StatusUnauthorized).JSON(map[string]string{"error": "Unauthorized"})
        }
    }
}

// user_handler.go
func GetUserProfile(ctx *goryu.Ctx) {
    // Retrieve the user data set by the middleware
    user, exists := ctx.Get("user")
    if !exists {
        ctx.Status(http.StatusInternalServerError).JSON(map[string]string{"error": "User not found in context"})
        return
    }

    // Now you can use the user data
    currentUser := user.(YourUserStruct)
    ctx.JSON(http.StatusOK, currentUser)
}
```

## Request Handling

These methods help you inspect and parse the incoming HTTP request.

### `Query(name string) string`

Gets a URL query parameter by name.

```go
// Request: /search?q=goryu&page=2
q := ctx.Query("q") // "goryu"
page := ctx.Query("page") // "2"
```

### `Form(name string) string`

Gets a form field value by name from `application/x-www-form-urlencoded` or `multipart/form-data`.

```go
// POST /users with form data: name=Goryu
name := ctx.Form("name") // "Goryu"
```

### `BindJSON(i interface{}) error`

Parses a JSON request body and populates a struct. It automatically checks if the `Content-Type` is `application/json`.

```go
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func CreateUser(ctx *goryu.Ctx) {
    var req CreateUserRequest
    if err := ctx.BindJSON(&req); err != nil {
        ctx.Status(http.StatusBadRequest).JSON(map[string]string{"error": "Invalid request"})
        return
    }
    // Use req.Name and req.Email
    ctx.JSON(http.StatusCreated, req)
}
```

### `QueryParser(out interface{}) error`

Automatically parses URL query parameters into a struct based on `query` tags.

```go
type SearchFilters struct {
    Topic  string `query:"topic"`
    Limit  int    `query:"limit"`
    Strict bool   `query:"strict"`
}

// Request: /articles?topic=golang&limit=20&strict=true
func SearchArticles(ctx *goryu.Ctx) {
    var filters SearchFilters
    if err := ctx.QueryParser(&filters); err != nil {
        ctx.Status(http.StatusBadRequest).JSON(map[string]string{"error": "Invalid query params"})
        return
    }
    // filters.Topic == "golang"
    // filters.Limit == 20
    // filters.Strict == true
}
```

### `GetHeader(key string) string`

Gets a request header value by key. The key is case-insensitive.

```go
userAgent := ctx.GetHeader("User-Agent")
```

### `Cookie(name string) (*http.Cookie, error)`

Gets a cookie by name from the request.

```go
sessionCookie, err := ctx.Cookie("session_id")
if err != nil {
    // Handle error (e.g., cookie not found)
}
```

### `RemoteIP() string`

Returns the client's IP address. It safely checks `X-Forwarded-For` and `X-Real-IP` for requests behind a trusted proxy.

```go
ip := ctx.RemoteIP() // e.g., "192.168.1.100"
```

### `BaseURL() string`

Returns the base URL, including the protocol and host (e.g., `https://example.com`).

```go
url := ctx.BaseURL()
```

### `BodyRaw() ([]byte, error)`

Returns the raw request body as a byte slice. This is useful when you need to process the body directly, without parsing it as JSON or a form.

```go
func WebhookHandler(ctx *goryu.Ctx) {
    body, err := ctx.BodyRaw()
    if err != nil {
        ctx.Status(http.StatusInternalServerError).JSON(map[string]string{"error": "Could not read body"})
        return
    }
    // e.g., Validate an HMAC signature using the raw body
    if !validateSignature(ctx.GetHeader("Webhook-Signature"), body) {
        ctx.Status(http.StatusUnauthorized).JSON(map[string]string{"error": "Invalid signature"})
        return
    }
    // Process the webhook...
}
```

### `Hostname() string`

Returns just the hostname from the request (e.g., `example.com`).

```go
host := ctx.Hostname()
```

### `Protocol() string`

Returns the request protocol: `http` or `https`.

```go
proto := ctx.Protocol()
```

### `Is(extension string) bool`

Checks if the request's `Content-Type` header matches a given type (e.g., `json`, `.html`, `application/xml`).

```go
if ctx.Is("json") {
    // Process JSON request
}
```

### `FormFile(key string)` & `SaveUploadedFile(file *multipart.FileHeader, dst string)`

Handles file uploads from a multipart form.

```go
func UploadHandler(ctx *goryu.Ctx) {
    file, header, err := ctx.FormFile("profile_picture")
    if err != nil {
        ctx.Status(http.StatusBadRequest).JSON(map[string]string{"error": "File upload failed"})
        return
    }
    defer file.Close()

    // Save the file to ./uploads/ with its original name
    if err := ctx.SaveUploadedFile(header, header.Filename); err != nil {
        ctx.Status(http.StatusInternalServerError).JSON(map[string]string{"error": "Could not save file"})
        return
    }

    ctx.Text(http.StatusOK, "File uploaded successfully!")
}
```

## Response Handling

These methods help you build and send the HTTP response.

### `JSON(code int, obj interface{}) error`

Sends a JSON response.

```go
user := map[string]string{"name": "Goryu", "status": "active"}
if err := ctx.JSON(http.StatusOK, user); err != nil {
    // Handle potential encoding error
}
```

### `Text(code int, text string) error`

Sends a plain text response.

```go
ctx.Text(http.StatusOK, "OK")
```

### `Data(code int, contentType string, data []byte) error`

Sends a response with raw byte data and a custom content type.

```go
imageData, _ := os.ReadFile("logo.png")
ctx.Data(http.StatusOK, "image/png", imageData)
```

### `Status(code int) *Context`

Sets the HTTP status code for the response. This method is chainable.

```go
// These two lines are equivalent:
ctx.Status(http.StatusNotFound).Text(http.StatusNotFound, "Not Found")
ctx.JSON(http.StatusNotFound, map[string]string{"error": "Not Found"})
```

### `SendFile(path string)`

Streams a file to the client, automatically setting the correct `Content-Type`.

```go
func DownloadHandler(ctx *goryu.Ctx) {
    ctx.SendFile("./static/downloads/document.pdf")
}
```

### `Redirect(code int, location string)`

Redirects the client to a new URL.

```go
ctx.Redirect(http.StatusFound, "/login")
```

### `SetCookie(cookie *http.Cookie)` & `ClearCookie(name string)`

Manages response cookies.

```go
// Set a session cookie
sessionCookie := &http.Cookie{
    Name: "session_id",
    Value: "some-random-string",
    Expires: time.Now().Add(24 * time.Hour),
}
ctx.SetCookie(sessionCookie)

// Clear a cookie
ctx.ClearCookie("old_session")
```

### `Append(field string, values ...string)`

Adds a value to a response header. Unlike `Set`, this will not overwrite existing values.

```go
ctx.Append("Link", "[http://example.com/api/v1](http://example.com/api/v1); rel=\"version\"")
ctx.Append("Link", "[http://example.com/docs](http://example.com/docs); rel=\"documentation\"")
```

### `Attachment(filename ...string)`

Tells the browser to prompt a download for the response.

```go
func DownloadHandler(ctx *goryu.Ctx) {
    ctx.Attachment("user-report.csv")
    ctx.SendFile("./reports/report123.csv")
}
```

### `Location(path string)`

Sets the `Location` header, typically used with a `201 Created` status.

```go
func CreateResource(ctx *goryu.Ctx) {
    // ... create a new resource with ID 456 ...
    ctx.Location("/api/resources/456")
    ctx.Status(http.StatusCreated)
}
```

### `Type(ext string) *Context`

Sets the `Content-Type` header based on a file extension. This is chainable.

```go
ctx.Type("xml").Data(http.StatusOK, "application/xml", []byte("<user>Goryu</user>"))
```

### `Vary(fields ...string)`

Adds fields to the `Vary` response header, which is important for caching.

```go
// Tell caches that the response depends on these headers
ctx.Vary("Accept-Encoding", "Accept-Language")
```

### Error Handling Methods

Goryu provides several methods for handling and responding to errors:

#### `Error(err error, statusCode ...int) error`

A flexible helper to log an error and send an HTTP error response. The status code is optional and defaults to 500.

```go
// Default 500 error
data, err := someComplexOperation()
if err != nil {
    return ctx.Error(err)
}

// Custom status code
if !isValid {
    return ctx.Error(errors.New("validation failed"), http.StatusBadRequest)
}
```

#### `ErrorWithMessage(err error, statusCode int, message string) error`

Send a custom error message to the client while logging the original error.

```go
if err := validateUser(user); err != nil {
    return ctx.ErrorWithMessage(err, http.StatusUnprocessableEntity, "Invalid user data provided")
}
```

#### `Abort(statusCode int, message ...string)`

Stop request processing and send an error response without requiring an error object.

```go
if !userHasPermission {
    ctx.Abort(http.StatusForbidden, "Access denied")
    return
}
```

## Best Practices

### Context Reuse

Goryu uses a `sync.Pool` to reuse `Context` objects. This reduces Garbage Collection pressure and improves performance. However, it means:

*   **Do not store** `Context` objects across goroutines.
*   The `Context` is only valid during the execution of the handler.
*   If you need to pass data to a background goroutine, extract the necessary values from the `Context` first.

### Error Handling

Always check errors returned by context methods (like `JSON`, `Text`, `BindJSON`). Even though Goryu handles many errors internally, checking them allows for better observability and custom error responses.

### Configuration

Use `goryu.Config` to fine-tune your application settings, such as `MaxMultipartMemory` for form uploads, which defaults to 10MB to prevent memory exhaustion attacks.

