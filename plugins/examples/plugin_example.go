package goryu

// COMPLETE WORKING EXAMPLE - Plugin System for Middleware
// This file demonstrates the comprehensive plugin system for middleware

/*

import (
	"time"
	"github.com/arthurlch/goryu"
)

func main() {
	// NEW: Fluent middleware configuration with the plugin system
	app := goryu.New().
		Use(goryu.Logger().
			Development().
			JSON().
			Output(os.Stdout).
			Build()).
		Use(goryu.Recovery().
			Development().
			EnableStackTrace(true).
			JSONResponse().
			Build()).
		Use(goryu.CORS().
			AllowOrigins("https://mydomain.com", "https://app.mydomain.com").
			AllowMethods("GET", "POST", "PUT", "DELETE").
			AllowHeaders("Authorization", "Content-Type").
			AllowCredentials(true).
			MaxAge(3600).
			Build()).
		Use(goryu.RateLimit(100, time.Minute).
			ByIP().
			JSONResponse().
			Build())
	
	// Alternative: Create and configure before using
	corsMiddleware := goryu.CORS().
		Production().
		AllowOrigins("https://mydomain.com").
		AllowMethods("GET", "POST").
		MaxAge(86400).
		Build()
	
	app.Use(corsMiddleware)
	
	// REST API with configured middleware
	app.GET("/api/users", func(c *goryu.Ctx) {
		c.JSON(200, map[string]string{"message": "Users endpoint"})
	})
	
	app.Listen(":8080")
}

// EXAMPLE CONFIGURATIONS

// Logger Examples
func exampleLoggerConfigurations() {
	// Development logging with colors
	devLogger := goryu.Logger().
		Development().
		EnableColors().
		TimeFormat("15:04:05").
		Build()
	
	// Production JSON logging
	prodLogger := goryu.Logger().
		Production().
		JSON().
		DisableColors().
		TimeFormat(time.RFC3339).
		Build()
	
	// Custom format logging
	customLogger := goryu.Logger().
		Format("[CUSTOM] ${time} | ${status} | ${method} ${path} | ${latency}").
		TimeZone("UTC").
		Build()
	
	// Common log format
	commonLogger := goryu.Logger().
		CommonLog().
		Build()
}

// Recovery Examples
func exampleRecoveryConfigurations() {
	// Development recovery with full stack traces
	devRecovery := goryu.Recovery().
		Development().
		EnableStackTrace(true).
		Build()
	
	// Production recovery with minimal info
	prodRecovery := goryu.Recovery().
		Production().
		DisableStackTrace().
		JSONResponse().
		Build()
	
	// Custom recovery handler
	customRecovery := goryu.Recovery().
		Handler(func(c *goryu.Ctx, err interface{}) {
			// Log the error
			log.Printf("Panic recovered: %v", err)
			// Send custom response
			c.JSON(500, map[string]interface{}{
				"error": "Something went wrong",
				"id":    generateErrorID(),
			})
		}).
		Build()
}

// CORS Examples
func exampleCORSConfigurations() {
	// Restrictive CORS for production
	restrictiveCORS := goryu.CORS().
		Restrictive().
		AllowOrigins("https://mydomain.com").
		AllowMethods("GET", "POST").
		Build()
	
	// Development CORS (permissive)
	devCORS := goryu.CORS().
		Development().
		AllowOrigins("http://localhost:3000", "http://localhost:8080").
		Build()
	
	// API CORS with credentials
	apiCORS := goryu.CORS().
		AllowOrigins("https://app.mydomain.com").
		AllowMethods("GET", "POST", "PUT", "DELETE", "PATCH").
		AllowHeaders("Authorization", "Content-Type", "X-API-Key").
		AllowCredentials(true).
		ExposeHeaders("X-Total-Count", "X-Page-Count").
		MaxAge(3600).
		Build()
	
	// Microservice CORS
	microserviceCORS := goryu.CORS().
		AllowOrigins("https://gateway.mydomain.com").
		AllowMethods("GET", "POST", "PUT", "DELETE").
		AddCommonHeaders().
		MaxAge(86400).
		Build()
}

// Rate Limiting Examples
func exampleRateLimitConfigurations() {
	// IP-based rate limiting
	ipRateLimit := goryu.RateLimit(100, time.Minute).
		ByIP().
		JSONResponse().
		Build()
	
	// API key-based rate limiting
	apiKeyRateLimit := goryu.RateLimit(1000, time.Hour).
		ByAPIKey("X-API-Key").
		JSONResponse().
		Build()
	
	// User-based rate limiting
	userRateLimit := goryu.RateLimit(200, time.Minute).
		ByUserID().
		CustomMessage("Rate limit exceeded for your account").
		Build()
	
	// Strict rate limiting for expensive operations
	strictRateLimit := goryu.RateLimit(5, time.Minute).
		ByIP().
		Handler(func(c *goryu.Ctx) {
			c.JSON(429, map[string]interface{}{
				"error": "Too many requests",
				"message": "This operation is rate limited to 5 requests per minute",
				"retry_after": 60,
			})
		}).
		Build()
	
	// Burst rate limiting
	burstRateLimit := goryu.RateLimit(50, time.Minute).
		Burst(100, 10*time.Second). // Allow 100 requests in 10 seconds
		ByIP().
		Build()
	
	// Preset configurations
	conservativeLimit := goryu.RateLimit(0, 0).Conservative().Build() // 60/min by IP
	moderateLimit := goryu.RateLimit(0, 0).Moderate().Build()         // 200/min by IP
	generousLimit := goryu.RateLimit(0, 0).Generous().Build()         // 1000/min by IP
}

// Plugin Discovery and Registration
func examplePluginSystem() {
	// List all registered plugins
	plugins := goryu.ListPlugins()
	fmt.Printf("Available plugins: %v\n", plugins)
	
	// Get a specific plugin
	if corsPlugin, exists := goryu.Plugin("cors"); exists {
		middleware := corsPlugin.(*plugins.CORSBuilder).
			AllowOrigins("https://example.com").
			Build()
		app.Use(middleware)
	}
	
	// Register a custom plugin
	goryu.RegisterPlugin("custom-auth", func() plugins.Builder {
		return &CustomAuthBuilder{}
	})
}

// Environment-based Configuration
func exampleEnvironmentConfiguration() {
	env := os.Getenv("ENVIRONMENT")
	
	var logger, recovery, cors, rateLimit goryu.Middleware
	
	switch env {
	case "production":
		logger = goryu.Logger().Production().Build()
		recovery = goryu.Recovery().Production().Build()
		cors = goryu.CORS().Restrictive().
			AllowOrigins("https://mydomain.com").
			Build()
		rateLimit = goryu.RateLimit(200, time.Minute).
			ByAPIKey("X-API-Key").
			Build()
	
	case "staging":
		logger = goryu.Logger().JSON().Build()
		recovery = goryu.Recovery().EnableStackTrace(false).Build()
		cors = goryu.CORS().
			AllowOrigins("https://staging.mydomain.com").
			Build()
		rateLimit = goryu.RateLimit(500, time.Minute).ByIP().Build()
	
	default: // development
		logger = goryu.Logger().Development().Build()
		recovery = goryu.Recovery().Development().Build()
		cors = goryu.CORS().Development().Build()
		rateLimit = goryu.RateLimit(1000, time.Minute).ByIP().Build()
	}
	
	app := goryu.New().
		Use(logger).
		Use(recovery).
		Use(cors).
		Use(rateLimit)
	
	app.Listen(":8080")
}

// Conditional Middleware
func exampleConditionalMiddleware() {
	app := goryu.New()
	
	// Always use logger and recovery
	app.Use(goryu.Logger().Development().Build())
	app.Use(goryu.Recovery().Development().Build())
	
	// Conditional CORS
	if os.Getenv("ENABLE_CORS") == "true" {
		app.Use(goryu.CORS().Development().Build())
	}
	
	// Conditional rate limiting
	if rateLimit := os.Getenv("RATE_LIMIT"); rateLimit != "" {
		if limit, err := strconv.Atoi(rateLimit); err == nil {
			app.Use(goryu.RateLimit(limit, time.Minute).ByIP().Build())
		}
	}
}

// COMPARISON:

// BEFORE - Manual middleware configuration:
func setupMiddlewareOld(app *goryu.App) {
	// Logger with manual config
	loggerConfig := logger.Config{
		Format: "[GORYU] ${time} | ${status} | ${latency} | ${ip} | ${method} ${path}",
		Output: os.Stdout,
		DisableColors: false,
	}
	app.Use(logger.New(loggerConfig))
	
	// Recovery with manual config
	recoveryConfig := recovery.Config{
		EnableStackTrace: true,
	}
	app.Use(recovery.New(recoveryConfig))
	
	// CORS with manual config
	corsConfig := cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders: []string{"*"},
	}
	app.Use(cors.New(corsConfig))
	
	// Rate limiting with manual config
	limiterConfig := limiter.Config{
		Max: 100,
		Expiration: time.Minute,
		KeyGenerator: func(c *context.Context) string {
			return getClientIP(c)
		},
	}
	app.Use(limiter.New(limiterConfig))
}

// AFTER - Fluent plugin system:
func setupMiddlewareNew(app *goryu.App) {
	app.Use(goryu.Logger().Development().Build())
	app.Use(goryu.Recovery().Development().Build())
	app.Use(goryu.CORS().
		AllowOrigins("https://mydomain.com").
		AllowMethods("GET", "POST", "PUT", "DELETE").
		Build())
	app.Use(goryu.RateLimit(100, time.Minute).ByIP().Build())
}

// BENEFITS:
// ✅ Fluent, chainable configuration API
// ✅ Type-safe middleware builders with validation
// ✅ Preset configurations (Development, Production, etc.)
// ✅ Plugin registration and discovery system
// ✅ Comprehensive configuration options
// ✅ Built-in security defaults and warnings
// ✅ Consistent API across all middleware
// ✅ Easy to test and mock
// ✅ Better error messages and validation
// ✅ Self-documenting configuration

*/