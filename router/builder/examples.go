package builder

// EXAMPLES OF FLUENT ROUTE REGISTRATION
// This file demonstrates the new expressive route builder patterns

/*

// BEFORE - Traditional route registration:
func setupRoutesOld(app *goryu.App) {
	// Create groups
	v1 := app.Group("/api/v1")

	// Add middleware to group manually
	v1.Use(authMiddleware)
	v1.Use(loggingMiddleware)

	// Register routes one by one
	v1.GET("/users", userController.Index)
	v1.POST("/users", userController.Create)
	v1.GET("/users/:id", userController.Show)
	v1.PUT("/users/:id", userController.Update)
	v1.DELETE("/users/:id", userController.Destroy)

	// Name routes manually
	v1.GET("/health", healthHandler).SetName("health")

	// Nested groups
	admin := v1.Group("/admin")
	admin.Use(adminAuthMiddleware)
	admin.GET("/dashboard", dashboardHandler)
}

// AFTER - Fluent route registration:
func setupRoutesNew(app *goryu.App) {
	app.Route().
		Group("/api/v1", func(v1 *GroupBuilder) {
			// Middleware applied to entire group
			v1.Middleware(auth.Required(), logger.New())

			// RESTful resource with automatic route generation
			v1.Resource("/users", &UserController{}).
				Middleware(rateLimit.PerUser(100)).
				Name("users").
				Build()

			// Individual routes with fluent configuration
			v1.GET("/health", health.Handler).
				Name("health").
				Cache(5 * time.Minute).
				Description("Health check endpoint")

			// Nested groups with fluent API
			v1.Group("/admin", func(admin *GroupBuilder) {
				admin.Middleware(auth.RequireRole("admin"))

				admin.GET("/dashboard", dashboard.Handler).
					Name("admin.dashboard").
					Middleware(audit.Log())

				admin.Resource("/settings", &SettingsController{}).
					Only("index", "update").
					Name("admin.settings").
					Build()
			})

			// WebSocket route
			v1.GET("/ws", websocket.Handler).
				Name("websocket").
				Description("WebSocket endpoint for real-time updates")
		})
}

// Example 1: Simple API with resources
func Example_SimpleAPI() {
	app := goryu.New()

	app.Route().
		Group("/api", func(api *GroupBuilder) {
			// Global API middleware
			api.Middleware(
				cors.New(),
				rateLimit.Global(1000),
				logger.New(),
			)

			// User resource
			api.Resource("/users", &UserController{}).
				Middleware(auth.Required()).
				Build()

			// Public endpoints
			api.POST("/login", auth.LoginHandler).
				Name("auth.login").
				Description("User authentication")

			api.POST("/register", auth.RegisterHandler).
				Name("auth.register").
				Description("User registration")
		})
}

// Example 2: Versioned API
func Example_VersionedAPI() {
	app := goryu.New()

	app.Route().
		Group("/api/v1", func(v1 *GroupBuilder) {
			v1.Middleware(middleware.Version("1.0"))

			v1.Resource("/posts", &PostControllerV1{}).
				Name("v1.posts").
				Build()
		}).
		Group("/api/v2", func(v2 *GroupBuilder) {
			v2.Middleware(middleware.Version("2.0"))

			v2.Resource("/posts", &PostControllerV2{}).
				Name("v2.posts").
				Middleware(validation.New()).
				Build()

			// V2 exclusive features
			v2.GET("/posts/:id/reactions", reactions.List).
				Name("v2.posts.reactions")
		})
}

// Example 3: Complex application with multiple modules
func Example_ComplexApp() {
	app := goryu.New()

	app.Route().
		// Public routes
		Group("/", func(public *GroupBuilder) {
			public.GET("/", home.Handler).Name("home")
			public.GET("/about", pages.About).Name("about")
			public.GET("/contact", pages.Contact).Name("contact")
		}).
		// API routes
		Group("/api", func(api *GroupBuilder) {
			api.Middleware(
				cors.New(),
				headers.Secure(),
			)

			// Auth endpoints
			api.Group("/auth", func(auth *GroupBuilder) {
				auth.POST("/login", authController.Login).Name("auth.login")
				auth.POST("/logout", authController.Logout).
					Middleware(authMiddleware).
					Name("auth.logout")
				auth.POST("/refresh", authController.Refresh).Name("auth.refresh")
			})

			// Protected resources
			api.Group("/", func(protected *GroupBuilder) {
				protected.Middleware(authMiddleware)

				protected.Resource("/profile", &ProfileController{}).
					Only("show", "update").
					Name("profile").
					Build()

				protected.Resource("/projects", &ProjectController{}).
					Name("projects").
					Build()

				// Nested resources
				protected.Group("/projects/:project_id", func(project *GroupBuilder) {
					project.Resource("/tasks", &TaskController{}).
						Name("project.tasks").
						Build()

					project.Resource("/members", &MemberController{}).
						Except("destroy").
						Name("project.members").
						Build()
				})
			})
		}).
		// Admin panel
		Group("/admin", func(admin *GroupBuilder) {
			admin.Middleware(
				auth.Required(),
				auth.RequireRole("admin"),
				audit.Log(),
			)

			admin.GET("/", adminDashboard).Name("admin.dashboard")

			admin.Resource("/users", &AdminUserController{}).
				Name("admin.users").
				Build()

			admin.Resource("/settings", &SettingsController{}).
				Only("index", "update").
				Name("admin.settings").
				Build()
		})
}

// Example 4: Microservice with health checks and metrics
func Example_Microservice() {
	app := goryu.New()

	app.Route().
		Group("/", func(root *GroupBuilder) {
			// Health and metrics endpoints
			root.GET("/health", health.Handler).
				Name("health").
				Cache(10 * time.Second)

			root.GET("/metrics", metrics.Handler).
				Name("metrics").
				Middleware(auth.APIKey())

			// Service API
			root.Group("/api", func(api *GroupBuilder) {
				api.Middleware(
					tracing.New(),
					logger.Structured(),
					recovery.New(),
				)

				// Service endpoints
				api.Resource("/orders", &OrderService{}).
					Middleware(
						auth.JWT(),
						validation.New(),
					).
					Name("orders").
					Build()

				// Async operations
				api.POST("/orders/:id/process", orderProcessor.Process).
					Name("orders.process").
					Middleware(queue.Async())
			})
		})
}

// Example UserController implementation
type UserController struct {
	db *Database
}

func (uc *UserController) Index(c *goryu.Ctx) {
	users := uc.db.GetAllUsers()
	c.JSON(200, users)
}

func (uc *UserController) Create(c *goryu.Ctx) {
	var user User
	if err := c.BindJSON(&user); err != nil {
		c.BadRequest("Invalid user data")
		return
	}

	created := uc.db.CreateUser(user)
	c.Created(created)
}

func (uc *UserController) Show(c *goryu.Ctx) {
	id := c.Param("id")
	user := uc.db.GetUser(id)
	if user == nil {
		c.NotFound("User not found")
		return
	}
	c.OK(user)
}

func (uc *UserController) Update(c *goryu.Ctx) {
	id := c.Param("id")
	var updates User
	if err := c.BindJSON(&updates); err != nil {
		c.BadRequest("Invalid update data")
		return
	}

	updated := uc.db.UpdateUser(id, updates)
	c.OK(updated)
}

func (uc *UserController) Destroy(c *goryu.Ctx) {
	id := c.Param("id")
	uc.db.DeleteUser(id)
	c.Status(204)
}

*/
