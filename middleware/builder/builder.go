package builder

import (
	"log"
	"time"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
)

// MiddlewareBuilder provides a fluent interface for building middleware.
// Deprecated: Use the standard middleware packages (e.g. middleware/logger, middleware/cors) instead.
type MiddlewareBuilder struct {
	name         string
	beforeFunc   func(c *context.Context) error
	afterFunc    func(c *context.Context) error
	skipFunc     func(c *context.Context) bool
	errorHandler func(c *context.Context, err error)
	config       base.BaseConfig
}

func New(name ...string) *MiddlewareBuilder {
	middlewareName := "Custom"
	if len(name) > 0 && name[0] != "" {
		middlewareName = name[0]
	}
	return &MiddlewareBuilder{
		name:   middlewareName,
		config: base.BaseConfig{},
	}
}
func (mb *MiddlewareBuilder) Before(handler func(c *context.Context) error) *MiddlewareBuilder {
	mb.beforeFunc = handler
	return mb
}
func (mb *MiddlewareBuilder) After(handler func(c *context.Context) error) *MiddlewareBuilder {
	mb.afterFunc = handler
	return mb
}
func (mb *MiddlewareBuilder) Skip(skipFunc func(c *context.Context) bool) *MiddlewareBuilder {
	mb.skipFunc = skipFunc
	mb.config.Skip = func(c *context.Context) bool {
		return skipFunc(c)
	}
	return mb
}
func (mb *MiddlewareBuilder) OnError(errorHandler func(c *context.Context, err error)) *MiddlewareBuilder {
	mb.errorHandler = errorHandler
	mb.config.ErrorHandler = func(c *context.Context, err error, middlewareName string) {
		errorHandler(c, err)
	}
	return mb
}
func (mb *MiddlewareBuilder) Logger(logger base.Logger) *MiddlewareBuilder {
	mb.config.Logger = logger
	return mb
}
func (mb *MiddlewareBuilder) Build() context.Middleware {
	if mb.beforeFunc == nil && mb.afterFunc == nil {
		mb.beforeFunc = func(c *context.Context) error {
			return nil
		}
	}
	if mb.afterFunc == nil {
		return base.StandardMiddleware(mb.name, mb.config, mb.beforeFunc)
	}
	return base.PostProcessMiddleware(mb.name, mb.config, mb.beforeFunc, mb.afterFunc)
}
func (mb *MiddlewareBuilder) BuildSimple(handler func(c *context.Context)) context.Middleware {
	return mb.Before(func(c *context.Context) error {
		handler(c)
		return nil
	}).Build()
}
func NewLogging() *MiddlewareBuilder {
	return New("Logging").
		Before(func(c *context.Context) error {
			c.Set("start_time", time.Now())
			return nil
		}).
		After(func(c *context.Context) error {
			startTime, exists := c.Get("start_time")
			if !exists {
				return nil
			}
			duration := time.Since(startTime.(time.Time))
			log.Printf("%s %s - %v",
				c.Request.Method,
				c.Request.URL.Path,
				duration,
			)
			return nil
		})
}
func NewTiming() *MiddlewareBuilder {
	return New("Timing").
		Before(func(c *context.Context) error {
			c.Set("request_start", time.Now())
			return nil
		}).
		After(func(c *context.Context) error {
			if startTime, exists := c.Get("request_start"); exists {
				duration := time.Since(startTime.(time.Time))
				c.Writer.Header().Set("X-Response-Time", duration.String())
			}
			return nil
		})
}
func NewCORS(allowedOrigins ...string) *MiddlewareBuilder {
	origin := "*"
	if len(allowedOrigins) > 0 {
		origin = allowedOrigins[0]
	}
	return New("CORS").
		Before(func(c *context.Context) error {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if c.Request.Method == "OPTIONS" {
				c.Status(204)
				return nil
			}
			return nil
		})
}
func NewSecurity() *MiddlewareBuilder {
	return New("Security").
		Before(func(c *context.Context) error {
			c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
			c.Writer.Header().Set("X-Frame-Options", "DENY")
			c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")
			c.Writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			return nil
		})
}
func NewError() *MiddlewareBuilder {
	return New("ErrorHandler").
		OnError(func(c *context.Context, err error) {
			c.JSON(500, map[string]interface{}{
				"error": map[string]interface{}{
					"message": err.Error(),
				},
			})
		})
}
