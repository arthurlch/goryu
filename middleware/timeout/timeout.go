package timeout

import (
	"context"
	"net/http"
	"time"

	"github.com/arthurlch/goryu"
)

// Config defines the configuration for timeout middleware
type Config struct {
	// Timeout for request processing. Default: 30 seconds
	Timeout time.Duration
	// Handler to call when timeout occurs. If nil, returns 408 Request Timeout
	TimeoutHandler goryu.HandlerFunc
	// Skip defines when to skip timeout middleware
	Skip func(c *goryu.Context) bool
}

// New creates a new timeout middleware
func New(config ...Config) goryu.Middleware {
	cfg := Config{
		Timeout: 30 * time.Second,
		TimeoutHandler: func(c *goryu.Context) {
			c.Status(http.StatusRequestTimeout).Text(http.StatusRequestTimeout, "Request Timeout")
		},
	}

	if len(config) > 0 {
		provided := config[0]
		if provided.Timeout > 0 {
			cfg.Timeout = provided.Timeout
		}
		if provided.TimeoutHandler != nil {
			cfg.TimeoutHandler = provided.TimeoutHandler
		}
		cfg.Skip = provided.Skip
	}

	return func(next goryu.HandlerFunc) goryu.HandlerFunc {
		return func(c *goryu.Context) {
			if cfg.Skip != nil && cfg.Skip(c) {
				next(c)
				return
			}

			// Create context with timeout
			ctx, cancel := context.WithTimeout(c.Request.Context(), cfg.Timeout)
			defer cancel()

			// Update request context
			c.Request = c.Request.WithContext(ctx)

			// Channel to signal completion
			done := make(chan struct{})
			var panicked bool

			// Run handler in goroutine
			go func() {
				defer func() {
					if r := recover(); r != nil {
						panicked = true
					}
					close(done)
				}()
				next(c)
			}()

			select {
			case <-done:
				// Handler completed normally
				if panicked {
					panic("handler panicked") // Re-panic to be caught by recovery middleware
				}
				return
			case <-ctx.Done():
				// Timeout occurred
				if ctx.Err() == context.DeadlineExceeded {
					cfg.TimeoutHandler(c)
				}
				return
			}
		}
	}
}
