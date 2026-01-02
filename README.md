<div align="center">
  <img src="https://i.ibb.co/YBfgFnG0/goryu-v3.png" alt="Goryu Logo" width="300"/>
  
---
*WARNING: This project is in alpha stage. Use at your own risk.*
---

</div>

## What is Goryu?

A Go web framework that gives you the productivity of Rails/Phoenix with the speed of Golang.

**Actually production-ready out of the box.**

```go
app := goryu.New()

app.GET("/", func(c *goryuctx.Context) {
    c.JSON(goryuctx.Map{"message": "Hello, World!"})
})

app.Listen(":3000")
```

That's it. You now have health checks, metrics, structured logging, and graceful shutdown.

## Why?

Because setting up a production Go service takes too long. You need:
- Router ✓
- Validation ✓
- Error handling ✓
- Logging ✓
- Health checks ✓
- Metrics ✓
- Graceful shutdown ✓
- Good project structure ✓
- A powerful CLI  ✓
- Scaffolding ✓

Goryu gives you all of this. No setup needed.

## Performance

```
BenchmarkGoryu_JSON-8         1,245,364 ops/sec    967.2 ns/op
BenchmarkGin_JSON-8           1,232,113 ops/sec    977.2 ns/op
BenchmarkGoryu_Param-8        2,036,600 ops/sec    593.3 ns/op
BenchmarkGin_Param-8          2,015,674 ops/sec    590.9 ns/op
```

As fast as Gin. With 10x more features built-in.

## Developer Experience

### 1. Smart Context

```go
// One context object. No request/response split.
app.POST("/users", func(c *goryuctx.Context) {
    var user CreateUserRequest
    c.BodyParser(&user)  // Parses + validates
    
    // Do stuff...
    
    c.Status(201).JSON(user)  // Chainable responses
})
```

### 2. Real Generators

```bash
# Generate handlers with tests
goryu generate handler user --crud

# Generate models  
goryu generate model product --fields="name:string,price:float"

# Scaffold entire features
goryu scaffold blog title:string content:text --api
```

Generated code is clean, tested, and yours to modify.

### 3. Built-in Monitoring

```go
// Automatic at /_health
{
  "status": "healthy",
  "uptime": "2h15m",
  "memory": "45MB",
  "goroutines": 12
}

// Prometheus metrics at /_metrics
http_requests_total{method="GET",path="/users",status="200"} 1543
```

### 4. Phoenix-style Resources

```go
// One line for full CRUD using the builder pattern
app.Route().Group("/api", func(api *builder.SimpleGroupBuilder) {
    api.Resource("/products", &ProductController{})
})

// GET    /products
// GET    /products/:id
// POST   /products
// PUT    /products/:id
// DELETE /products/:id
```

### 5. Middleware that Makes Sense

```go
app.Use(
    logger.New(),           // Structured logging
    cors.Default(),         // CORS with sane defaults
    recovery.New(),         // Panic recovery
    limiter.New(),          // Rate limiting
)
```

## Quick Start

```bash
# Install
go get github.com/arthurlch/goryu

# CLI (Highly recommended)
go install github.com/arthurlch/goryu/cmd/goryu@latest

# Initialize a new project
goryu init myapp

# Start development server
cd myapp
goryu dev
```

## Real Example

```go
package main

import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/goryuctx"
    "github.com/arthurlch/goryu/middleware/logger"
    "github.com/arthurlch/goryu/middleware/cors"
)

type Product struct {
    ID    string  `json:"id"`
    Name  string  `json:"name" validate:"required"`
    Price float64 `json:"price" validate:"required,min=0"`
}

func main() {
    app := goryu.New()
    
    // Middleware
    app.Use(logger.New(), cors.Default())
    
    // Routes
    products := app.Group("/api/products")
    products.GET("/", listProducts)
    products.GET("/:id", getProduct)
    products.POST("/", createProduct)
    
    app.Listen(":8080")
}

func createProduct(c *goryuctx.Context) {
    var product Product
    if err := c.BodyParser(&product); err != nil {
        c.Status(400).JSON(goryuctx.Map{"error": err.Error()})
        return
    }
    
    // Validation happens automatically
    // Save product...
    
    c.Status(201).JSON(product)
}
```

## What You Get

### Core
- **High-performance router** - As fast as Gin 
- **Context API** - Clean, chainable, one object
- **Validation** - Automatic with struct tags
- **Error handling** - Consistent error responses

### Production Features
- **Health checks** - `/_health` endpoint
- **Metrics** - Prometheus-ready at `/_metrics`  
- **Structured logging** - JSON logs with request context
- **Graceful shutdown** - Never drop a connection
- **Panic recovery** - Your app stays up

### Developer Tools
- **CLI generators** - Generate handlers, models, full features
- **Hot reload** - `goryu dev` watches your code
- **Project structure** - Scalable layout from day one
- **Testing helpers** - Test your HTTP handlers easily

### Middleware
- Authentication (JWT, Basic, API Key)
- Rate limiting
- CORS
- Request ID
- Compression
- Timeout
- Recovery
- Logging
- And 15+ more...

## Philosophy

1. **Batteries included** - Everything you need to ship
2. **No magic** - You can read the source
3. **Performance matters** - Because blazingly fast performance is essential
4. **Developer happiness** - Great DX is a feature

## Documentation

- [Tutorial](./TUTORIAL.md)
- [CLI Reference](./CLI.md)
- [API Reference](https://pkg.go.dev/github.com/arthurlch/goryu)
- [Examples](./examples)

## Contributing

PRs welcome! Please follow the project's coding standards and include tests.

## License

MIT

---

<div align="center">
  <strong>Build something awesome.</strong>
</div>