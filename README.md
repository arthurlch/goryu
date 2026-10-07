<div align="center">
  <img src="https://i.ibb.co/YBfgFnG0/goryu-v3.png" alt="Goryu Logo" width="300"/>
  
---
*WARNING: This project is in alpha stage. Use at your own risk.*
---

</div>

## The Go backend your AI gets right the first time

You don't hand-write boilerplate anymore — you direct an agent. So the framework's
job changed. It's no longer just about being pleasant to *type*; it has to be the
thing your coding agent **generates correctly** and that you can **read well enough
to trust shipping.**

Goryu is built for that:

- **Agents get it right the first time.** A small, regular API constrains codegen — fewer ways to call it means fewer ways to get it wrong. Go's compiler rejects the rest on the spot.
- **Secure by default.** The one thing agents reliably get wrong — the security basics — is handled on every response path, whether the agent thinks about it or not.
- **No magic.** Explicit, readable Go. You (or a non-technical reviewer) can read what the agent shipped and believe it's safe to deploy.
- **Batteries included.** Fewer dependencies means fewer versions and APIs for an agent to hallucinate. One binary, zero assembly.

```go
app := goryu.New()

app.GET("/", func(c *goryu.Ctx) {
    c.JSON(200, goryu.Map{"message": "Hello, World!"})
})

app.Run(":3000")
```

That's it. You now have health checks, metrics, structured logging, and graceful shutdown.

## Why this matters now

Point an agent at raw `net/http` and you get a different shape every time, with
different security gaps, and nobody can review fifty bespoke services by hand.
Goryu makes agent output **uniform, safe, and auditable** — so the person directing
the agent can actually ship it. Everything below is wired from day one:

- Router ✓
- Validation ✓
- Error handling ✓
- Logging ✓
- Health checks ✓
- Metrics ✓
- Graceful shutdown ✓
- Good project structure ✓
- A powerful CLI ✓
- Scaffolding ✓

## Performance

```
BenchmarkGoryu_JSON-8         1,245,364 ops/sec    967.2 ns/op
BenchmarkGin_JSON-8           1,232,113 ops/sec    977.2 ns/op
BenchmarkGoryu_Param-8        2,036,600 ops/sec    593.3 ns/op
BenchmarkGin_Param-8          2,015,674 ops/sec    590.9 ns/op
```

As fast as Gin. With 10x more features built-in — and none of the readability your agent depends on traded away for it.

## Developer Experience

### 1. Smart Context

```go
// One context object. No request/response split.
app.POST("/users", func(c *goryu.Ctx) {
    var user CreateUserRequest
    if err := c.BodyParser(&user); err != nil {  // Parses + validates
        c.JSON(400, goryu.Map{"error": err.Error()})
        return
    }

    // Do stuff...

    c.JSON(201, user)
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

Generated code is clean, tested, and yours to modify. A human and a coding agent produce the same output.

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

    app.Run(":8080")
}

func createProduct(c *goryu.Ctx) {
    product, err := goryu.Bind[Product](c) // parse + validate
    if err != nil {
        c.JSON(400, goryu.Map{"error": err.Error()})
        return
    }

    // Save product...

    c.JSON(201, product)
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

### Secure by default
- **Security headers & CSRF** - on every response path
- **JWT & sessions** - with a unified invalidation seam
- **OAuth2 / OIDC** - and passkeys / WebAuthn
- **RBAC & audit log** - authorization and a tamper-evident trail

### Developer Tools
- **CLI generators** - Generate handlers, models, full features
- **Hot reload** - `goryu dev` watches your code
- **Project structure** - Scalable layout from day one
- **Testing helpers** - Test your HTTP handlers easily

### Self-describing to the next agent
- **`llms.txt` + machine-readable API reference** (`App.MountLLMs`)
- **MCP server** - `goryu mcp` lets agents list routes, scaffold, and run
- **AI scaffolds** - `goryu scaffold ai <name> --kind=chat|rag|agent`

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

## REST first, realtime when you need it

Plain REST stays plain — a handler is just a function. Typed input with validation
is one call:

```go
type Signup struct {
    Email string `json:"email"`
}

func (s Signup) Validate() error {
    if s.Email == "" {
        return errors.New("email is required")
    }
    return nil
}

app.POST("/signup", func(c *goryu.Ctx) {
    body, err := goryu.Bind[Signup](c)
    if err != nil {
        c.JSON(400, goryu.Map{"error": err.Error()})
        return
    }
    c.JSON(201, goryu.Map{"email": body.Email})
})
```

Streaming is first-class — great for progress updates and for AI token streaming:

```go
app.GET("/events", func(c *goryu.Ctx) {
    c.SSE(func(send func(goryu.SSEvent) error) error {
        for i := 0; i < 3; i++ {
            if err := send(goryu.SSEvent{Event: "tick", Data: fmt.Sprint(i)}); err != nil {
                return err
            }
        }
        return nil
    })
})
```

WebSockets use the same `Context`:

```go
app.GET("/ws", func(c *goryu.Ctx) {
    conn, err := websocket.Upgrade(c)
    if err != nil {
        return
    }
    defer conn.Close()
    for {
        _, msg, err := conn.ReadMessage()
        if err != nil {
            return
        }
        _ = conn.WriteText(string(msg))
    }
})
```

These are additive — you never pay for them in a plain REST app.

## Philosophy

1. **If an LLM can't reliably generate correct Goryu code, the API is too clever** - so it isn't. The regular, predictable API is a feature for people and agents alike.
2. **Built to be directed, not just typed** - the same clean API serves the human reviewing and the agent writing.
3. **No magic** - read the source and understand it; trusting what ships now matters more than typing it.
4. **Secure by default** - on every response path, not opt-in.
5. **REST first** - the simple case stays simple; AI/realtime features are additive.
6. **Performance matters** - because blazingly fast performance is essential.

## Documentation

- [Tutorial](./TUTORIAL.md)
- [CLI Reference](./CLI.md)
- [Direction & Roadmap](./docs/DIRECTION.md)
- [API Stability](./docs/STABILITY.md)
- [API Reference](https://pkg.go.dev/github.com/arthurlch/goryu)
- [Examples](./examples)

## Contributing

PRs welcome! Please follow the project's coding standards and include tests.

## License

MIT

---

<div align="center">
  <strong>Build something your agent gets right.</strong>
</div>
