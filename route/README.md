# Route Package

The `route` package defines the core `Route` structure used by Goryu. It is designed to be a lightweight, dependency-free package to avoid circular imports between the `router` and `context` packages.

## Purpose

In complex frameworks, circular dependencies can occur when:
- The `Router` needs to know about `Routes`.
- The `Context` needs to know about the current `Route`.
- The `Route` might need to reference the `Router`.

To solve this, the `Route` struct is defined in this isolated package.

## The Route Struct

The `Route` struct holds all the metadata for a registered route:

```go
type Route struct {
    Method      string      // HTTP Method (GET, POST, etc.)
    Path        string      // URL Path (e.g., /users/:id)
    Handler     interface{} // The handler function
    Name        string      // Unique name for the route
    Description string      // Documentation description
    Router      interface{} // Reference to the parent router
}
```

## Usage

While you typically interact with routes via the `goryu` or `router` packages, you might encounter the `Route` struct when inspecting the current route from the context:

```go
import "github.com/arthurlch/goryu/goryuctx"

func MyHandler(c *goryuctx.Context) error {
    // Access the current route
    currentRoute := c.Route
    
    fmt.Printf("Handling request for: %s %s\n", 
        currentRoute.Method, 
        currentRoute.Path,
    )
    
    return nil
}
```

### Naming Routes

You can name routes for easier lookup or reverse routing (if supported):

```go
app.GET("/users/:id", GetUser).SetName("get_user")
```

This updates the `Name` field of the underlying `Route` struct.
