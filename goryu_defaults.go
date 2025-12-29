package goryu

import (
	"github.com/arthurlch/goryu/middleware/logger"
	"github.com/arthurlch/goryu/middleware/recovery"
	"github.com/arthurlch/goryu/middleware/requestid"
)

// Default returns an App instance with the default configuration and middleware.
// It includes:
// - Recovery middleware to handle panics
// - Logger middleware for request logging
// - RequestID middleware for request tracing
// - Monitoring (Health, Metrics, Events) via app.New()
func Default(config ...Config) *App {
	app := New(config...)

	// Add default middleware
	// Order matters: Recovery should be first (outermost) to catch panics from everything else
	app.Use(recovery.New())
	app.Use(requestid.New())
	app.Use(logger.New())

	return app
}
