# Router Package

The `router` package implements a high-performance, radix tree-based HTTP router for Goryu. It supports named parameters, wildcards, route grouping, and middleware.

## Features

- **Fast**: Efficient routing algorithm for fast path matching (Radix tree).
- **Parameters**: Named parameters like `/users/:id`.
- **Wildcards**: Catch-all routes like `/files/*path`.
- **Groups**: Organize routes with common prefixes and middlewares.
- **Middleware**: Chainable middleware support at global, group, and route levels.
- **Named Routes**: Reverse URL generation.
- **Customizable**: Configurable behavior for trailing slashes, method not allowed, and more.

## Usage

### Basic Routing

```go
import (
    "github.com/arthurlch/goryu/router"
    "github.com/arthurlch/goryu/goryuctx"
)

r := router.New()

r.GET("/", func(c *goryuctx.Context) error {
    return c.String(200, "Welcome!")
})

r.POST("/users", func(c *goryuctx.Context) error {
    return c.String(201, "User created")
})
```

### Parameters and Wildcards

```go
// Named parameter
r.GET("/users/:id", func(c *goryuctx.Context) error {
    id := c.Param("id")
    return c.String(200, "User ID: "+id)
})

// Wildcard
r.GET("/files/*path", func(c *goryuctx.Context) error {
    path := c.Param("path")
    return c.String(200, "File path: "+path)
})
```

### Route Groups

Group routes to share a common prefix and middlewares:

```go
api := r.Group("/api")
api.Use(authMiddleware)

v1 := api.Group("/v1")
v1.GET("/users", listUsers)
v1.GET("/users/:id", getUser)
```

### Configuration

You can customize the router's behavior:

```go
r := router.New(router.RouterConfig{
    StrictRouting:          true,  // /foo and /foo/ are different
    RedirectTrailingSlash:  false, // Disable automatic redirects
    HandleMethodNotAllowed: true,  // Return 405 for invalid methods
    MaxRouteDepth:          32,    // Security limit
})
```

## Security

The router includes built-in security limits to prevent resource exhaustion attacks:
- `MaxRouteDepth`: Limits the depth of the routing tree.
- `MaxTotalRoutes`: Limits the total number of registered routes.
- `MaxParametersPerRoute`: Limits the complexity of individual routes.
