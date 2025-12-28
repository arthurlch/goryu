package cli

type MiddlewareInfo struct {
	Name         string
	Description  string
	Category     string
	Package      string
	Dependencies []string
	Usage        string
	Example      string
	Options      []MiddlewareOption
}

type MiddlewareOption struct {
	Name        string
	Description string
	Default     string
}

func getMiddlewareInfo(name string) *MiddlewareInfo {
	middlewareData := map[string]*MiddlewareInfo{
		"auth": {
			Name:        "Authentication",
			Description: "Provides authentication middleware for protecting routes",
			Category:    "Security",
			Package:     "github.com/arthurlch/goryu/middleware/auth",
			Usage:       "app.Use(auth.New(auth.Config{...}))",
			Example: `app.Use(auth.New(auth.Config{
    Secret: "your-secret-key",
    TokenLookup: "header:Authorization",
    ContextKey: "user",
}))`,
			Options: []MiddlewareOption{
				{Name: "Secret", Description: "Secret key for JWT signing", Default: ""},
				{Name: "TokenLookup", Description: "Token lookup method", Default: "header:Authorization"},
				{Name: "ContextKey", Description: "Context key for user data", Default: "user"},
				{Name: "Expiry", Description: "Token expiry duration", Default: "24h"},
			},
		},
		"basicauth": {
			Name:        "Basic Authentication",
			Description: "HTTP Basic Authentication middleware",
			Category:    "Security",
			Package:     "github.com/arthurlch/goryu/middleware/basicauth",
			Usage:       "app.Use(basicauth.New(basicauth.Config{...}))",
			Example: `app.Use(basicauth.New(basicauth.Config{
    Users: map[string]string{
        "admin": "password",
        "user":  "secret",
    },
    Realm: "Protected Area",
}))`,
			Options: []MiddlewareOption{
				{Name: "Users", Description: "Username:password pairs", Default: ""},
				{Name: "Realm", Description: "Authentication realm", Default: "Restricted"},
			},
		},
		"cors": {
			Name:        "CORS",
			Description: "Cross-Origin Resource Sharing middleware",
			Category:    "Security",
			Package:     "github.com/arthurlch/goryu/middleware/cors",
			Usage:       "app.Use(cors.New(cors.Config{...}))",
			Example: `app.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"https://example.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Content-Type", "Authorization"},
    AllowCredentials: true,
}))`,
			Options: []MiddlewareOption{
				{Name: "AllowOrigins", Description: "Allowed origins", Default: "*"},
				{Name: "AllowMethods", Description: "Allowed HTTP methods", Default: "GET,POST,HEAD,PUT,DELETE,OPTIONS"},
				{Name: "AllowHeaders", Description: "Allowed headers", Default: "Accept,Authorization,Content-Type,X-CSRF-Token"},
				{Name: "AllowCredentials", Description: "Allow credentials", Default: "false"},
			},
		},
		"csrf": {
			Name:        "CSRF Protection",
			Description: "Cross-Site Request Forgery protection middleware",
			Category:    "Security",
			Package:     "github.com/arthurlch/goryu/middleware/csrf",
			Usage:       "app.Use(csrf.New(csrf.Config{...}))",
			Example: `app.Use(csrf.New(csrf.Config{
    Secret: "your-csrf-secret",
    TokenLookup: "form:_token",
    ContextKey: "csrf",
}))`,
			Options: []MiddlewareOption{
				{Name: "Secret", Description: "Secret for token generation", Default: ""},
				{Name: "TokenLookup", Description: "Token lookup method", Default: "form:_token"},
				{Name: "ContextKey", Description: "Context key for token", Default: "csrf"},
			},
		},
		"cache": {
			Name:        "Cache",
			Description: "Response caching middleware with TTL support",
			Category:    "Performance",
			Package:     "github.com/arthurlch/goryu/middleware/cache",
			Usage:       "app.Use(cache.New(cache.Config{...}))",
			Example: `app.Use(cache.New(cache.Config{
    TTL: 300 * time.Second,
    Store: memory.New(),
}))`,
			Options: []MiddlewareOption{
				{Name: "TTL", Description: "Time to live for cached responses", Default: "300s"},
				{Name: "Store", Description: "Cache store implementation", Default: "memory"},
				{Name: "KeyGenerator", Description: "Custom key generation function", Default: "URL+Method"},
			},
		},
		"compress": {
			Name:        "Compression",
			Description: "Response compression middleware (gzip, deflate, brotli)",
			Category:    "Performance",
			Package:     "github.com/arthurlch/goryu/middleware/compress",
			Usage:       "app.Use(compress.New(compress.Config{...}))",
			Example: `app.Use(compress.New(compress.Config{
    Level: compress.LevelDefault,
}))`,
			Options: []MiddlewareOption{
				{Name: "Level", Description: "Compression level", Default: "DefaultCompression"},
				{Name: "MinLength", Description: "Minimum response size to compress", Default: "1024"},
			},
		},
		"limiter": {
			Name:        "Rate Limiter",
			Description: "Rate limiting middleware with customizable strategies",
			Category:    "Performance",
			Package:     "github.com/arthurlch/goryu/middleware/limiter",
			Usage:       "app.Use(limiter.New(limiter.Config{...}))",
			Example: `app.Use(limiter.New(limiter.Config{
    Max: 100,
    Expiration: 60 * time.Second,
    KeyGenerator: func(c *goryu.Context) string {
        return c.IP()
    },
}))`,
			Options: []MiddlewareOption{
				{Name: "Max", Description: "Maximum requests per duration", Default: "20"},
				{Name: "Expiration", Description: "Rate limit window duration", Default: "60s"},
				{Name: "KeyGenerator", Description: "Function to generate rate limit keys", Default: "IP-based"},
			},
		},
		"logger": {
			Name:        "Logger",
			Description: "HTTP request logging middleware with customizable formats",
			Category:    "Monitoring",
			Package:     "github.com/arthurlch/goryu/middleware/logger",
			Usage:       "app.Use(logger.New(logger.Config{...}))",
			Example: `app.Use(logger.New(logger.Config{
    Format: "${time} | ${status} | ${latency} | ${ip} | ${method} ${path}",
    TimeFormat: "15:04:05",
    Output: os.Stdout,
}))`,
			Options: []MiddlewareOption{
				{Name: "Format", Description: "Log format string", Default: "Combined format"},
				{Name: "TimeFormat", Description: "Time format layout", Default: "15:04:05"},
				{Name: "Output", Description: "Output destination", Default: "os.Stdout"},
			},
		},
		"metrics": {
			Name:         "Metrics",
			Description:  "Prometheus metrics collection middleware",
			Category:     "Monitoring",
			Package:      "github.com/arthurlch/goryu/middleware/metrics",
			Dependencies: []string{"github.com/prometheus/client_golang"},
			Usage:        "app.Use(metrics.New(metrics.Config{...}))",
			Example: `app.Use(metrics.New(metrics.Config{
    Namespace: "myapp",
    Subsystem: "http",
}))`,
			Options: []MiddlewareOption{
				{Name: "Namespace", Description: "Prometheus namespace", Default: "goryu"},
				{Name: "Subsystem", Description: "Prometheus subsystem", Default: "http"},
				{Name: "SkipPaths", Description: "Paths to skip from metrics", Default: "/metrics,/health"},
			},
		},
		"recovery": {
			Name:        "Recovery",
			Description: "Panic recovery middleware with stack trace logging",
			Category:    "Monitoring",
			Package:     "github.com/arthurlch/goryu/middleware/recovery",
			Usage:       "app.Use(recovery.New(recovery.Config{...}))",
			Example: `app.Use(recovery.New(recovery.Config{
    EnableStackTrace: true,
    StackTraceHandler: func(c *goryu.Context, err interface{}) {
        log.Printf("Panic: %v", err)
    },
}))`,
			Options: []MiddlewareOption{
				{Name: "EnableStackTrace", Description: "Enable stack trace logging", Default: "true"},
				{Name: "StackTraceHandler", Description: "Custom stack trace handler", Default: "log to stderr"},
			},
		},
		"secure": {
			Name:        "Security Headers",
			Description: "Security headers middleware (HSTS, X-Frame-Options, etc.)",
			Category:    "Security",
			Package:     "github.com/arthurlch/goryu/middleware/secure",
			Usage:       "app.Use(secure.New(secure.Config{...}))",
			Example: `app.Use(secure.New(secure.Config{
    XSSProtection: "1; mode=block",
    ContentTypeNoSniff: "nosniff",
    XFrameOptions: "DENY",
    HSTSMaxAge: 31536000,
}))`,
			Options: []MiddlewareOption{
				{Name: "XSSProtection", Description: "X-XSS-Protection header", Default: "1; mode=block"},
				{Name: "ContentTypeNoSniff", Description: "X-Content-Type-Options header", Default: "nosniff"},
				{Name: "XFrameOptions", Description: "X-Frame-Options header", Default: "SAMEORIGIN"},
				{Name: "HSTSMaxAge", Description: "HSTS max-age in seconds", Default: "0"},
			},
		},
		"timeout": {
			Name:        "Timeout",
			Description: "Request timeout middleware with context cancellation",
			Category:    "Performance",
			Package:     "github.com/arthurlch/goryu/middleware/timeout",
			Usage:       "app.Use(timeout.New(timeout.Config{...}))",
			Example: `app.Use(timeout.New(timeout.Config{
    Timeout: 30 * time.Second,
}))`,
			Options: []MiddlewareOption{
				{Name: "Timeout", Description: "Request timeout duration", Default: "30s"},
				{Name: "ErrorHandler", Description: "Custom timeout error handler", Default: "HTTP 408"},
			},
		},
		"tracing": {
			Name:         "OpenTelemetry Tracing",
			Description:  "Distributed tracing middleware with OpenTelemetry",
			Category:     "Monitoring",
			Package:      "github.com/arthurlch/goryu/middleware/tracing",
			Dependencies: []string{"go.opentelemetry.io/otel"},
			Usage:        "app.Use(tracing.New(tracing.Config{...}))",
			Example: `app.Use(tracing.New(tracing.Config{
    ServiceName: "my-service",
    ServiceVersion: "1.0.0",
}))`,
			Options: []MiddlewareOption{
				{Name: "ServiceName", Description: "Service name for tracing", Default: "goryu-app"},
				{Name: "ServiceVersion", Description: "Service version", Default: "1.0.0"},
				{Name: "SkipPaths", Description: "Paths to skip from tracing", Default: "/health,/metrics"},
			},
		},
		"healthcheck": {
			Name:        "Health Check",
			Description: "Health check endpoint middleware",
			Category:    "Monitoring",
			Package:     "github.com/arthurlch/goryu/middleware/healthcheck",
			Usage:       "app.Use(healthcheck.New(healthcheck.Config{...}))",
			Example: `app.Use(healthcheck.New(healthcheck.Config{
    Path: "/health",
    Checks: []healthcheck.Check{
        healthcheck.DatabaseCheck(db),
        healthcheck.RedisCheck(redis),
    },
}))`,
			Options: []MiddlewareOption{
				{Name: "Path", Description: "Health check endpoint path", Default: "/health"},
				{Name: "Checks", Description: "Health check functions", Default: "Basic ping"},
			},
		},
		"errors": {
			Name:        "Error Handling",
			Description: "Elegant error handling middleware with structured errors",
			Category:    "Core",
			Package:     "github.com/arthurlch/goryu/middleware/errors",
			Usage:       "app.Use(errors.New()) or app.Use(errors.NewWithConfig(errors.Config{...}))",
			Example: `// Basic usage
app.Use(errors.New())

// With custom config
app.Use(errors.NewWithConfig(errors.Config{
    ShowDetails:    true,
    ShowStackTrace: false,
    LogErrors:      true,
    DevMode:        false,
}))

// Using error helpers in handlers
app.GET("/users/:id", errors.Handle(func(c *goryu.Context) error {
    id := c.Param("id")
    user, err := getUserByID(id)
    if err != nil {
        return errors.NotFound("user")
    }
    return c.JSON(200, user)
}))

// Using fluent error API
app.POST("/login", func(c *goryu.Context) {
    var req LoginRequest
    if err := c.Bind(&req); err != nil {
        errors.Error(c).BadRequest("Invalid request body")
        return
    }
    
    if req.Email == "" {
        errors.Error(c).Validation("email", "Email is required")
        return
    }
    
    // Process login...
})`,
			Options: []MiddlewareOption{
				{Name: "ShowDetails", Description: "Show error details in response", Default: "true"},
				{Name: "ShowStackTrace", Description: "Include stack trace (dev mode only)", Default: "false"},
				{Name: "LogErrors", Description: "Log errors to console", Default: "true"},
				{Name: "DevMode", Description: "Development mode with extra debugging", Default: "false"},
				{Name: "CustomHandler", Description: "Custom error handler function", Default: ""},
				{Name: "ErrorTransformer", Description: "Transform errors before response", Default: ""},
			},
		},
	}

	return middlewareData[name]
}
