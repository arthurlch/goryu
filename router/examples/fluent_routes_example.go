package goryu

// COMPLETE WORKING EXAMPLE - Fluent Route Registration
// This file demonstrates the final working implementation

/*

func main() {
    app := goryu.New()

    // NEW: Fluent route registration with the Route() method
    app.Route().
        Group("/api/v1", func(v1 *builder.SimpleGroupBuilder) {
            // Group-level middleware
            v1.Middleware(
                logger.New(),
                cors.Default(),
                rateLimit.PerIP(100),
            )

            // Individual routes with fluent configuration
            v1.GET("/health", func(c *goryu.Ctx) {
                c.OK(map[string]string{"status": "healthy"})
            }).
                Name("api.health").
                Description("Health check endpoint")

            // RESTful resource with automatic route generation
            v1.Resource("/users", &UserController{}).
                Name("users").
                Build()

            // Partial resource with filtering
            v1.Resource("/settings", &SettingsController{}).
                Only("index", "update").
                Name("settings").
                Build()

            // Nested groups
            v1.Group("/admin", func(admin *builder.SimpleGroupBuilder) {
                admin.Middleware(auth.RequireRole("admin"))

                admin.GET("/dashboard", adminDashboard).
                    Name("admin.dashboard")

                admin.Resource("/audit", &AuditController{}).
                    Except("destroy").
                    Name("admin.audit").
                    Build()
            })
        })

    app.Listen(":8080")
}

// Example controllers with automatic method mapping:

type UserController struct {
    db *Database
}

func (uc *UserController) Index(c *goryu.Ctx) {    // GET /api/v1/users
    users := uc.db.GetAllUsers()
    c.OK(users)
}

func (uc *UserController) Create(c *goryu.Ctx) {   // POST /api/v1/users
    var user User
    if err := c.BindJSON(&user); err != nil {
        c.BadRequest("Invalid user data")
        return
    }
    created := uc.db.CreateUser(user)
    c.Created(created)
}

func (uc *UserController) Show(c *goryu.Ctx) {     // GET /api/v1/users/:id
    id := c.Param("id")
    user := uc.db.GetUser(id)
    if user == nil {
        c.NotFound("User not found")
        return
    }
    c.OK(user)
}

func (uc *UserController) Update(c *goryu.Ctx) {   // PUT /api/v1/users/:id
    id := c.Param("id")
    var updates User
    if err := c.BindJSON(&updates); err != nil {
        c.BadRequest("Invalid update data")
        return
    }
    updated := uc.db.UpdateUser(id, updates)
    c.OK(updated)
}

func (uc *UserController) Destroy(c *goryu.Ctx) {  // DELETE /api/v1/users/:id
    id := c.Param("id")
    uc.db.DeleteUser(id)
    c.Status(204)
}

// COMPARISON:

// BEFORE - Traditional route registration:
func setupRoutesOld(app *goryu.App) {
    api := app.Group("/api/v1")
    api.Use(logger.New())
    api.Use(cors.Default())

    api.GET("/health", healthHandler)
    api.GET("/users", userController.Index)
    api.POST("/users", userController.Create)
    api.GET("/users/:id", userController.Show)
    api.PUT("/users/:id", userController.Update)
    api.DELETE("/users/:id", userController.Destroy)

    admin := api.Group("/admin")
    admin.Use(auth.RequireRole("admin"))
    admin.GET("/dashboard", adminDashboard)
    // ... more routes
}

// AFTER - Fluent route registration:
func setupRoutesNew(app *goryu.App) {
    app.Route().
        Group("/api/v1", func(v1 *builder.SimpleGroupBuilder) {
            v1.Middleware(logger.New(), cors.Default())

            v1.GET("/health", healthHandler).Name("api.health")

            v1.Resource("/users", &UserController{}).
                Name("users").
                Build()

            v1.Group("/admin", func(admin *builder.SimpleGroupBuilder) {
                admin.Middleware(auth.RequireRole("admin"))
                admin.GET("/dashboard", adminDashboard).Name("admin.dashboard")
            })
        })
}

// BENEFITS:
// ✅ More expressive and readable
// ✅ Automatic RESTful route generation
// ✅ Hierarchical middleware application
// ✅ Built-in route naming and descriptions
// ✅ Type-safe controller method mapping
// ✅ Clean nested group organization
// ✅ Consistent API patterns
// ✅ Better maintainability

*/
