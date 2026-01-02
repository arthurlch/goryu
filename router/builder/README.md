# Router Builder - Fluent Route Registration

This package provides an expressive, fluent API for route registration in the Goryu framework, making route definition more readable and maintainable.

## Features

- **Fluent API**: Chain methods for clean route definitions
- **Resource Routes**: Automatic RESTful route generation
- **Nested Groups**: Organize routes with prefixes and middleware
- **Method Chaining**: Configure routes with names, descriptions, and caching
- **Controller Integration**: Automatic method mapping with reflection

## Quick Start

```go
package main

import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/goryuctx"
    "github.com/arthurlch/goryu/router/builder"
)

func main() {
    app := goryu.New()
    
    // Fluent route registration
    app.Route().
        Group("/api/v1", func(v1 *builder.SimpleGroupBuilder) {
            // Individual routes with configuration
            v1.GET("/health", healthHandler).
                Name("health").
                Description("Health check endpoint")
            
            // RESTful resource
            v1.Resource("/users", &UserController{}).
                Name("users").
                Build()
        })
    
    app.Listen(":8080")
}
```

## Examples

### Basic Route Registration

```go
// BEFORE - Traditional approach:
app.GET("/users", getUsersHandler)
app.POST("/users", createUserHandler)
app.GET("/users/:id", getUserHandler)

// AFTER - Fluent approach:
app.Route().
    Group("/", func(root *builder.SimpleGroupBuilder) {
        root.GET("/users", getUsersHandler).Name("users.index")
        root.POST("/users", createUserHandler).Name("users.create")
        root.GET("/users/:id", getUserHandler).Name("users.show")
    })
```

### Resource Routes

```go
type UserController struct{}

func (uc *UserController) Index(c *goryuctx.Context) { /* GET /users */ }
func (uc *UserController) Create(c *goryuctx.Context) { /* POST /users */ }
func (uc *UserController) Show(c *goryuctx.Context) { /* GET /users/:id */ }
func (uc *UserController) Update(c *goryuctx.Context) { /* PUT /users/:id */ }
func (uc *UserController) Destroy(c *goryuctx.Context) { /* DELETE /users/:id */ }

// Automatic RESTful routes
app.Route().
    Group("/api", func(api *builder.SimpleGroupBuilder) {
        api.Resource("/users", &UserController{}).
            Name("users").
            Build()
    })
```

### Advanced Configuration

```go
app.Route().
    Group("/api", func(api *builder.SimpleGroupBuilder) {
        // Group middleware
        api.Middleware(
            auth.Required(),
            cors.New(),
            rateLimit.PerIP(100),
        )
        
        // Versioned API with nested groups
        api.Group("/v1", func(v1 *builder.SimpleGroupBuilder) {
            // Full resource with filtering
            v1.Resource("/posts", &PostController{}).
                Only("index", "show", "create").  // Only these actions
                Name("v1.posts").
                Build()
            
            // Custom routes with configuration
            v1.POST("/auth/login", authController.Login).
                Name("auth.login").
                Description("User authentication endpoint").
                Cache(0) // No caching for auth endpoints
            
            // Admin section
            v1.Group("/admin", func(admin *builder.SimpleGroupBuilder) {
                admin.Middleware(auth.RequireRole("admin"))
                
                admin.Resource("/users", &AdminUserController{}).
                    Except("destroy").  // All except destroy
                    Name("admin.users").
                    Build()
            })
        })
    })
```

### Middleware Integration

```go
app.Route().
    Group("/api", func(api *builder.SimpleGroupBuilder) {
        // Apply middleware to entire group
        api.Middleware(
            logger.New(),
            recovery.New(),
            cors.Default(),
        )
        
        api.Resource("/orders", &OrderController{}).
            Middleware(
                auth.JWT(),           // Resource-specific middleware
                validation.Orders(),
            ).
            Name("orders").
            Build()
    })
```

## Benefits Over Traditional Registration

1. **Readability**: Clear hierarchical structure
2. **Organization**: Group related routes together
3. **Consistency**: Uniform pattern for all route types
4. **Maintainability**: Easy to modify and extend
5. **Documentation**: Built-in route descriptions
6. **Type Safety**: Full Go type checking
7. **Middleware Scoping**: Granular middleware application

## Resource Controller Patterns

### Full Controller Implementation

```go
type ArticleController struct {
    db *Database
}

// GET /articles
func (ac *ArticleController) Index(c *goryuctx.Context) {
    articles := ac.db.GetAllArticles()
    c.OK(articles)
}

// POST /articles
func (ac *ArticleController) Create(c *goryuctx.Context) {
    var article Article
    if err := c.BindJSON(&article); err != nil {
        c.BadRequest("Invalid article data")
        return
    }
    created := ac.db.CreateArticle(article)
    c.Created(created)
}

// GET /articles/:id
func (ac *ArticleController) Show(c *goryuctx.Context) {
    id := c.Param("id")
    article := ac.db.GetArticle(id)
    if article == nil {
        c.NotFound("Article not found")
        return
    }
    c.OK(article)
}

// PUT /articles/:id
func (ac *ArticleController) Update(c *goryuctx.Context) {
    id := c.Param("id")
    var updates Article
    if err := c.BindJSON(&updates); err != nil {
        c.BadRequest("Invalid update data")
        return
    }
    updated := ac.db.UpdateArticle(id, updates)
    c.OK(updated)
}

// DELETE /articles/:id
func (ac *ArticleController) Destroy(c *goryuctx.Context) {
    id := c.Param("id")
    ac.db.DeleteArticle(id)
    c.Status(204)
}
```

### Partial Controller (Only Some Actions)

```go
type ReadOnlyController struct{}

// Only implement the actions you need
func (roc *ReadOnlyController) Index(c *goryuctx.Context) { /* ... */ }
func (roc *ReadOnlyController) Show(c *goryuctx.Context) { /* ... */ }

// Use .Only() to specify which actions to register
app.Route().
    Group("/readonly", func(g *builder.SimpleGroupBuilder) {
        g.Resource("/items", &ReadOnlyController{}).
            Only("index", "show").
            Build()
    })
```

## API Reference

### SimpleGroupBuilder Methods

- `GET(path, handler)` - Register GET route
- `POST(path, handler)` - Register POST route  
- `PUT(path, handler)` - Register PUT route
- `DELETE(path, handler)` - Register DELETE route
- `PATCH(path, handler)` - Register PATCH route
- `Group(prefix, callback)` - Create nested group
- `Resource(path, controller)` - Create RESTful resource
- `Middleware(middlewares...)` - Add group middleware

### SimpleRouteConfig Methods

- `Name(name)` - Set route name for URL generation
- `Description(desc)` - Add route description
- `Cache(ttl)` - Set caching duration

### SimpleResourceBuilder Methods

- `Name(name)` - Set resource name prefix
- `Only(actions...)` - Include only specified actions
- `Except(actions...)` - Exclude specified actions  
- `Middleware(middlewares...)` - Add resource middleware
- `Build()` - Create the routes

This fluent API makes route registration more expressive and maintainable while preserving all the power of the underlying router system.