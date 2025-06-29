# Goryu Framework


## Context (Ctx)

In Goryu, the `Context` (or `ctx`) is an object that is passed to every request handler. It holds the underlying HTTP request and response writer, but more importantly, it provides a large set of helper methods to make common web development tasks easier, safer, and more expressive.

Instead of interacting directly with the low-level `http.ResponseWriter` and `*http.Request` objects for every task, you can use the `Context`'s clean and simple API to:

* Parse incoming data (JSON, forms, query strings).
* Send various types of responses (JSON, text, files).
* Manipulate headers, cookies, and status codes.
* Pass data between middleware and your final handler.

Every handler function in Goryu receives a pointer to the `Context`:

```go
func MyHandler(ctx *context.Context) {
    // Use the context to handle the request and send a response
    ctx.Text(http.StatusOK, "Hello, World!")
}
```

## Core Context API

These are the fundamental methods for passing data through the request lifecycle.

### `Set(key string, value interface{})`

Stores a key-value pair in the context. This is the primary way to pass data from middleware to your handlers.

### `Get(key string) (value interface{}, exists bool)`

Retrieves a value from the context by its key. It returns the value and a boolean indicating if the key existed.

**Example: Middleware Authentication**
Here's how you can use `Set` and `Get` to create a simple authentication middleware.

```go
// auth_middleware.go
func AuthMiddleware(next context.HandlerFunc) context.HandlerFunc {
    return func(ctx *context.Context) {
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
func GetUserProfile(ctx *context.Context) {
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

func CreateUser(ctx *context.Context) {
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
func SearchArticles(ctx *context.Context) {
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
func WebhookHandler(ctx *context.Context) {
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
func UploadHandler(ctx *context.Context) {
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
func DownloadHandler(ctx *context.Context) {
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
func DownloadHandler(ctx *context.Context) {
    ctx.Attachment("user-report.csv")
    ctx.SendFile("./reports/report123.csv")
}
```

### `Location(path string)`

Sets the `Location` header, typically used with a `201 Created` status.

```go
func CreateResource(ctx *context.Context) {
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

### `Error(err error)`

A helper to log an error and send a generic `500 Internal Server Error` response.

```go
data, err := someComplexOperation()
if err != nil {
    ctx.Error(err)
    return
}
