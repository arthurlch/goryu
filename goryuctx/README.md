# 🧠 Goryu Context Package

> **The heart of the Goryu framework.**

The `context` package provides the `Context` struct, which is the central data carrier for every request in Goryu. It is designed with **Developer Experience (DX)**, **Security**, and **Performance** in mind.

Unlike standard `context.Context` or other framework contexts, Goryu's Context offers a **Fluent API**, **Built-in Security Checks**, and **Robust Error Handling** to make your life easier and your application safer.

---

## ✨ Key Features

- **🌊 Fluent API**: Chain methods for expressive, readable, and compact code.
- **🛡️ Security First**: Automatic protection against Path Traversal, IP Spoofing, and malicious file uploads.
- **⚡ Thread-Safe**: Built-in mutexes and atomic operations ensure safety even in concurrent middleware.
- **🔧 Robust Error Handling**: Configurable error modes (Return, Log, Panic, Silent) to suit your debugging needs.
- **📦 Smart Binding**: strict JSON binding and powerful query/form parsing.

---

## 🚀 Quick Start

```go
func HandleLogin(c *context.Context) {
    // Fluent chaining example
    c.Status(200).
      Header("X-App-Version", "1.0").
      JSON(map[string]string{
          "message": "Welcome back!",
      })
}
```

---

## 📥 Request Handling

### Data Access
Easily access query parameters, form values, and path parameters.

```go
// GET /search?q=golang&page=1
query := c.Query("q") // "golang"

// POST form-data
username := c.Form("username")

// Route parameters /users/:id
userID := c.Param("id")
```

### Data Binding
Bind request data to structs with validation.

```go
type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

var user User
// Automatically checks Content-Type, limits body size (1MB default),
// and disallows unknown fields for security.
if err := c.BindJSON(&user); err != nil {
    c.BadRequest("Invalid JSON")
    return
}
```

### 📂 Secure File Uploads
The `SaveUploadedFile` method is a fortress. It performs rigorous security checks:
- ✅ Validates filename length and characters (no null bytes, control chars).
- ✅ Prevents Path Traversal attacks (e.g., `../../etc/passwd`).
- ✅ Checks for dangerous extensions (`.exe`, `.sh`, etc.).
- ✅ Prevents hidden file uploads.

```go
file, header, err := c.FormFile("avatar")
if err != nil {
    c.BadRequest("Upload failed")
    return
}

// Safely saves to "uploads/" directory with all security checks active
if err := c.SaveUploadedFile(header, "user_avatar.png"); err != nil {
    c.InternalError("Could not save file")
    return
}
```

### 🌐 IP Resolution
Get the real client IP, even behind proxies, with spoofing protection.

```go
// Only trusts X-Forwarded-For if the direct IP is in the trusted_proxies list
c.Set("trusted_proxies", []string{"10.0.0.1", "127.0.0.1"})
clientIP := c.RemoteIP()
```

---

## 📤 Response Handling

### Standard Methods
Send responses in various formats.

```go
c.JSON(200, data)
c.Text(200, "Hello World")
c.HTML(200, "<h1>Hello</h1>")
c.XML(200, "<root>Hello</root>")
```

### 🌊 Fluent API
Chain response methods for cleaner handlers.

```go
c.WithStatus(201).
  WithHeader("X-Created-By", "Goryu").
  WithCookie(&http.Cookie{Name: "session", Value: "123"}).
  JSON(map[string]string{"status": "created"})
```

### 📁 File Serving
Serve files securely. `SendFile` automatically cleans paths and checks for directory traversal attempts.

```go
// Safely serves files, returning 404 or 403 as appropriate
c.SendFile("public/index.html")
```

---

## ⚠️ Error Management

Goryu's context uses a sophisticated error handling system.

### Error Handling Modes
You can configure how the context handles errors during fluent chaining or standard operations.

```go
// Available Modes:
// - ErrorModeReturn: Returns the error to the caller (Default)
// - ErrorModeLog: Logs the error but continues execution
// - ErrorModePanic: Panics (useful for development)
// - ErrorModeSilent: Ignores errors

c.SetErrorHandlingMode(context.ErrorModeLog)
```

### Checking Errors
When using the fluent API, errors are collected internally.

```go
c.Status(200).JSON(data)

if c.HasErrors() {
    // Handle any error that occurred in the chain
    log.Println(c.FirstError())
}
```

---

## 🔒 Thread Safety & Concurrency

The `Context` struct is protected by a `sync.RWMutex`.
- **`Set(key, value)`** and **`Get(key)`** are thread-safe.
- Response writing uses atomic flags to prevent "multiple response" race conditions.

```go
// Safe to use in concurrent goroutines
go func() {
    c.Set("background_job", "running")
}()
```

---

## 🧩 Advanced: Context Lifecycle

The `Context` is created via `NewContext`. It is designed to be pooled (future optimization), so avoid storing references to it after the request completes.

```go
ctx := context.NewContext(w, r)
// ... handle request ...
```
