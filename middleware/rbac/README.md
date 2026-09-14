# RBAC Middleware

Per-route role checks. Populate the caller's roles once after authentication with
`SetRoles`, then guard routes with `Require` / `RequireAll`.

## Usage

```go
import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/middleware/rbac"
)

// After auth, load the caller's roles into the context.
app.Use(func(next goryu.HandlerFunc) goryu.HandlerFunc {
    return func(c *goryu.Ctx) {
        rbac.SetRoles(c, loadRolesFor(c)...)
        next(c)
    }
})

admin := app.Group("/admin", rbac.Require("admin"))     // any of
billing := app.Group("/billing", rbac.RequireAll("admin", "billing")) // all of
```

Custom role source / denial response:

```go
guard := rbac.New(rbac.Config{
    RolesFunc: func(c *goryuctx.Context) []string { return rolesFromJWT(c) },
    Forbidden: func(c *goryuctx.Context) { c.Status(403).JSON(403, goryu.Map{"error": "nope"}) },
})
app.Group("/admin", guard.Require("admin"))
```

## API

| Function | Description |
|----------|-------------|
| `SetRoles(c, roles...)` | Store the caller's roles for downstream checks |
| `GetRoles(c) []string` | Read the stored roles |
| `Require(roles...)` | Allow if the caller has **any** of the roles |
| `RequireAll(roles...)` | Allow only if the caller has **every** role |
| `New(Config)` | Build an enforcer with a custom `RolesFunc` / `Forbidden` |

Denied requests return `403` by default. Callers with no roles are always denied.
