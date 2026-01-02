# 🎓 Goryu: Building a Modern E-Commerce API

Welcome to the **Goryu Masterclass**. In this guide, we will build a complete, production-ready E-Commerce Backend (`shop-api`) featuring:

*   **Configuration** via the Builder pattern.
*   **Authentication** (JWT, Password Hashing) using Goryu's Auth Middleware.
*   **Public & Protected APIs** for Products and Orders.
*   **Observability** with the built-in Dashboard.

---

## Prerequisites

*   **Go 1.24+**
*   **Goryu CLI**: `go install github.com/arthurlch/goryu/cmd/goryu@latest`

---

## Part 1: The Foundation 🏗️

Let's start by initializing a clean project structure.

```bash
goryu init shop-api
cd shop-api
go mod tidy
```

### 1.1 Configuration Strategy

Create a `config.json` in the root directory. We'll use this to manage our environment settings.

```json
{
  "app": {
    "name": "Goryu Shop",
    "environment": "development"
  },
  "server": {
    "port": 3000
  },
  "security": {
    "csrf_protection": false
  }
}
```

### 1.2 Main Entrypoint

Open `cmd/server/main.go` and set up the Config Builder.

```go
package main

import (
    "log"
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/config/builder"
)

func main() {
    // 1. Load Configuration (Defaults < File < Env)
    cfg, err := builder.New().
        WithDefaults().
        WithFile("config.json").
        WithEnvironment("SHOP"). // e.g. SHOP_SERVER_PORT=8080
        Build()

    if err != nil {
        log.Fatal("❌ Configuration error:", err)
    }

    // 2. Initialize App
    app := goryu.New(*cfg)
    app.Use(goryu.Recovery())
    app.Use(goryu.Logger())

    // 3. Start Server
    log.Printf("🚀 Starting %s on %s", cfg.App.Name, cfg.GetServerAddress())
    log.Fatal(app.Listen(cfg.GetServerAddress()))
}
```

---

## Part 2: Authentication & Users 🔐

We will use Goryu's powerful `middleware/auth` package. For this tutorial, we'll use the built-in **In-Memory Stores**, but in production, you would wrap your database (Postgres, Mongo, etc.).

### 2.1 Setup Auth Service

Update `main.go` to initialize the Auth Service.

```go
    // ... imports
    "github.com/arthurlch/goryu/middleware/auth"

    // ... inside main()

    // --- Authentication Setup ---
    
    // 1. Dependencies
    // In production, implement these interfaces with your DB
    userStore := auth.NewInMemoryUserStore()
    tokenStore := auth.NewInMemoryTokenStore()
    
    // Mock Email Sender (prints to console)
    emailSender := &MockEmailSender{} 

    // 2. JWT Setup
    jwtAuth := auth.NewJWTAuth("super-secret-key-change-me", cfg.App.Name)

    // 3. Service Config
    authConfig := auth.DefaultAuthServiceConfig()
    authConfig.RequireEmailVerification = false // Simplified for tutorial

    // 4. Initialize Service
    authService := auth.NewAuthService(jwtAuth, userStore, tokenStore, emailSender, authConfig)
    authHandlers := auth.NewAuthHandlers(authService)

    // 5. Register Routes (/auth/login, /auth/register, etc.)
    authHandlers.RegisterRoutes(app)
```

Add the Mock Email Sender at the bottom of the file:

```go
type MockEmailSender struct{}
func (m *MockEmailSender) SendVerificationEmail(email, token, link string) error {
    log.Printf("📧 [Email] Verify %s: %s", email, link)
    return nil
}
func (m *MockEmailSender) SendPasswordResetEmail(email, token, link string) error {
    log.Printf("📧 [Email] Reset Password %s: %s", email, link)
    return nil
}
func (m *MockEmailSender) SendSecurityAlert(email, msg string) error {
    log.Printf("📧 [Email] Alert %s: %s", email, msg)
    return nil
}
```

---

## Part 3: The Product Catalog 📦

Let's build the public API for browsing products.

### 3.1 The Product Model

Create `internal/models/product.go`:

```go
package models

type Product struct {
    ID          string  `json:"id"`
    Name        string  `json:"name"`
    Price       float64 `json:"price"`
    Description string  `json:"description"`
}

// Simple validation
func (p *Product) Validate() error {
    // Goryu calls this automatically
    if len(p.Name) < 3 {
        return fmt.Errorf("name too short")
    }
    if p.Price <= 0 {
        return fmt.Errorf("price must be positive")
    }
    return nil
}
```

## 3.2 Product Handlers

Add a simple in-memory product handler in `main.go`:

```go
    // ... imports
    "github.com/arthurlch/goryu/goryuctx"
    "shop-api/internal/models"

    // ... inside main() (or separate file)
    
    // Seed some data
    products := []models.Product{
        {ID: "1", Name: "Goryu T-Shirt", Price: 25.0, Description: "Official Swag"},
        {ID: "2", Name: "Gopher Plush", Price: 15.0, Description: "Cuddly friend"},
    }

    // Public API Group
    api := app.Group("/api/v1")
    
    api.GET("/products", func(c *goryuctx.Context) {
        c.JSON(goryuctx.Map{
            "data": products,
            "count": len(products),
        })
    })
```

---

## Part 4: Orders & Protection 🛡️

Now for the critical part: **Protected Routes**. Users must be logged in to place orders.

### 4.1 Protected Group

Use the `authService.Middleware()` to protect routes.

```go
    // Protected API Group
    // All routes here require a valid JWT Access Token
    protected := api.Group("/user", authService.Middleware())

    protected.POST("/orders", func(c *goryuctx.Context) {
        // 1. Get User ID from Context (set by middleware)
        userID := c.Get(auth.UserIDKey) // "user_id"
        
        // 2. Bind Order Data
        type OrderRequest struct {
            ProductID string `json:"product_id"`
            Quantity  int    `json:"quantity"`
        }
        var req OrderRequest
        if err := c.BodyParser(&req); err != nil {
            c.Status(400).JSON(goryuctx.Map{"error": "Invalid request"})
            return
        }

        // 3. Process Order (Mock)
        log.Printf("💰 New Order! User: %v, Product: %s, Qty: %d", userID, req.ProductID, req.Quantity)

        c.Status(201).JSON(goryuctx.Map{
            "status": "confirmed",
            "order_id": "ord_12345",
            "user_id": userID,
        })
    })
```

---

## Part 5: Run & Verify 🚀

Your E-Commerce API is ready!

### 5.1 Start the Server

```bash
goryu dev
```

### 5.2 Test the Flow

1.  **Register a User**:
    ```bash
    curl -X POST http://localhost:3000/auth/register \
      -H "Content-Type: application/json" \
      -d '{"email": "jane@example.com", "password": "securePassword123!"}'
    ```

2.  **Login**:
    ```bash
    curl -X POST http://localhost:3000/auth/login \
      -H "Content-Type: application/json" \
      -d '{"email": "jane@example.com", "password": "securePassword123!"}'
    ```
    *Copy the `access_token` from the response.*

3.  **Browse Products** (Public):
    ```bash
    curl http://localhost:3000/api/v1/products
    ```

4.  **Place Order** (Protected):
    ```bash
    curl -X POST http://localhost:3000/api/v1/user/orders \
      -H "Authorization: Bearer <YOUR_ACCESS_TOKEN>" \
      -H "Content-Type: application/json" \
      -d '{"product_id": "1", "quantity": 2}'
    ```

If you try step 4 without the token, you'll get a `401 Unauthorized`.

---

## Part 5: Deep Dive: Security & Sessions 🛒

The user requested we focus on **Sessions** and **Security**. Let's add a Shopping Cart feature to demonstrate stateful sessions alongside our stateless JWT auth.

### 5.1 Why Sessions?
While JWTs are great for **Authentication** (stateless identity), Sessions are often better for **Temporary State** (like a shopping cart) that you don't want to store in a database permanently until an order is placed.

### 5.2 Setup Session Middleware
We'll use Goryu's `SecureStore` which keeps session data in memory but **encrypted** — a best practice to prevent leaking user data even if memory is dumped.

Update `main.go`:

```go
    // ... imports
    "github.com/arthurlch/goryu/middleware/session"

    // ... inside main()
    
    // 1. Initialize Secure Store
    // In production, use a Redis store for persistence across restarts
    store, err := session.NewSecureStore("my-very-long-encryption-key-must-be-32-bytes")
    if err != nil {
        log.Fatal(err)
    }

    // 2. Add Session Middleware
    // We add it to the API group so all API routes have session support
    api.Use(session.New(session.Config{
        Store: store,
        CookieName: "shop_session",
        Secure: &[]bool{false}[0], // Set to true in production (requires HTTPS)
        HttpOnly: true, // 🛡️ JavaScript cannot access this cookie (prevents XSS)
    }))
```

### 5.3 Shopping Cart Implementation
Now let's add routes to manage a cart. This interaction happens *without* login!

```go
    // Public Cart Routes
    api.POST("/cart", func(c *goryuctx.Context) {
        // Get existing session or create new one
        sess, _ := session.Get(c)
        
        // Simple Cart Logic
        type CartItem struct {
            ProductID string `json:"product_id"`
            Qty       int    `json:"qty"`
        }
        var item CartItem
        if err := c.BodyParser(&item); err != nil {
            c.Status(400).JSON(goryuctx.Map{"error": "bad request"})
            return
        }
        
        // Add to session data
        cart := sess.Get("cart")
        if cart == nil {
            cart = []CartItem{}
        }
        // (In real app, append properly)
        items := cart.([]CartItem)
        items = append(items, item)
        sess.Set("cart", items)
        
        c.JSON(goryuctx.Map{"status": "added", "cart_size": len(items)})
    })

    api.GET("/cart", func(c *goryuctx.Context) {
        sess, _ := session.Get(c)
        cart := sess.Get("cart")
        c.JSON(goryuctx.Map{"cart": cart})
    })
```

### 5.4 Security Concepts Explained 🛡️

1.  **HttpOnly Cookies**: We set `HttpOnly: true`. This means your frontend JavaScript code (e.g., React/Vue) **cannot** read the session cookie. If an attacker injects malicious JS (XSS), they cannot steal the session ID.
2.  **Secure Cookies**: In production (`Secure: true`), cookies are only sent over HTTPS, preventing "Man-in-the-Middle" attacks.
3.  **Encrypted Store**: Goryu's `SecureStore` uses AES-GCM encryption. Even if an attacker gains access to your server's RAM, they cannot read user session data.

---

## Part 6: Advanced Features 🚀

Goryu isn't just for JSON APIs. Let's look at some advanced capabilities.

### 6.1 Route Naming & Reversal
In large apps, hardcoding URLs like `/api/v1/user/orders` is brittle. Use **Named Routes** instead.

```go
    // Name the route
    api.GET("/products/:id", getProduct).SetName("product_detail")

    // Generate URL dynamically in your code
    url := app.Router.Reverse("product_detail", "123") 
    // url == "/api/v1/products/123"
```

### 6.2 File Uploads
Let's allow admins to upload product images. Goryu's `SaveUploadedFile` includes built-in security checks against path traversal and malicious filenames.

```go
    admin := api.Group("/admin")
    // ... add auth middleware ...

    admin.POST("/products/:id/image", func(c *goryuctx.Context) {
        // 1. Parse Multipart Form (10MB limit by default)
        file, header, err := c.FormFile("image")
        if err != nil {
            c.BadRequest("Image required")
            return
        }

        // 2. Validate File (Content-Type, Size, etc.)
        if header.Size > 5*1024*1024 { // 5MB limit
            c.BadRequest("File too large")
            return
        }

        // 3. Secure Save
        // Automatically prevents "../" attacks
        savePath := fmt.Sprintf("uploads/products/%s_%s", c.Param("id"), header.Filename)
        if err := c.SaveUploadedFile(header, savePath); err != nil {
             c.InternalError("Failed to save image")
             return
        }

        c.Success("Image uploaded: " + savePath)
    })
```

### 6.3 Fluent XML/HTML
Sometimes you need to support old systems (XML) or return rendered fragments (HTML). Goryu's Context has a **Fluent API** for this.

```go
    api.GET("/legacy/products", func(c *goryuctx.Context) {
        c.FluentXML(200, "<products><product id='1'>Goryu T-Shirt</product></products>")
    })
```

---

## Part 7: The Plugin System 🧩

Goryu comes with a powerful **Plugin System** to standardize middleware configuration across teams.

Instead of manually configuring every middleware, use Builders.

### 7.1 Production Logger
The Plugin Builder ensures you don't forget critical production settings (like disabling colors or using JSON format).

```go
    // Replace app.Use(goryu.Logger()) with:
    
    import "github.com/arthurlch/goryu/plugins"

    app.Use(plugins.NewLoggerBuilder().
        Production().            // Enforces JSON, UTC timestamps
        Output(os.Stdout).
        Build())
```

### 7.2 Rate Limiting
Protect your API with the Rate Limit plugin.

```go
    // 100 requests per minute per IP
    app.Use(plugins.NewRateLimitBuilder().
        PerMinute(100).
        ByIP().
        Build())
```

---

## Conclusion 🎓

You have toured the entire Goryu ecosystem!

**What you have built:**
1.  **Shop API**: A structured, configured backend.
2.  **Auth System**: Secure JWT + In-Memory Store.
3.  **Shopping Cart**: Stateful Session management.
4.  **Advanced Features**: File Uploads, Reverse Routing.
5.  **Production Ready**: Using Plugins for standard logging and rate limiting.

**Next Steps:**
*   Explore `middleware/` for more tools (CORS, CSRF, Cache).
*   Check `examples/` for more code patterns.
*   Join the community and build something amazing!
