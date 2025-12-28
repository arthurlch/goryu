package goryu

import (
	"github.com/arthurlch/goryu/middleware/builder"
)

// Deprecated: Use middleware/builder directly if needed, but standard middleware packages are preferred.
type MiddlewareBuilder = builder.MiddlewareBuilder

func NewMiddleware(name ...string) *MiddlewareBuilder {
	return builder.New(name...)
}

// Convenience middleware builders

func NewLoggingMiddleware() *MiddlewareBuilder {
	return builder.NewLogging()
}

func NewTimingMiddleware() *MiddlewareBuilder {
	return builder.NewTiming()
}

func NewCORSMiddleware(allowedOrigins ...string) *MiddlewareBuilder {
	return builder.NewCORS(allowedOrigins...)
}

func NewSecurityMiddleware() *MiddlewareBuilder {
	return builder.NewSecurity()
}

func NewErrorMiddleware() *MiddlewareBuilder {
	return builder.NewError()
}
