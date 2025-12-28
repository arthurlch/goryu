package base

import (
	"errors"
	context "github.com/arthurlch/goryu/goryuctx"
	"net/http"
	"time"
)

type ExampleConfig struct {
	BaseConfig
	Timeout        time.Duration
	CustomHeader   string
	AllowedMethods []string
}

func (c *ExampleConfig) Configure(base *BaseConfig) {
	c.BaseConfig = *base
}
func (c *ExampleConfig) Validate() error {
	if c.Timeout <= 0 {
		return NewConfigError("Timeout", "must be greater than 0")
	}
	if c.CustomHeader == "" {
		return NewConfigError("CustomHeader", "cannot be empty")
	}
	return nil
}
func NewExampleMiddleware(config ExampleConfig) func(next context.HandlerFunc) context.HandlerFunc {
	if err := config.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				DefaultErrorHandler(c, MiddlewareError{
					Middleware: "Example",
					Err:        err,
					StatusCode: http.StatusInternalServerError,
				}, "Example")
			}
		}
	}
	return StandardMiddleware("Example", config.BaseConfig, func(c *context.Context) error {
		if len(config.AllowedMethods) > 0 {
			methodAllowed := false
			for _, method := range config.AllowedMethods {
				if c.Request.Method == method {
					methodAllowed = true
					break
				}
			}
			if !methodAllowed {
				return MiddlewareError{
					Middleware: "Example",
					Err:        errors.New("method not allowed"),
					StatusCode: http.StatusMethodNotAllowed,
				}
			}
		}
		c.SetHeader(config.CustomHeader, "processed")
		if config.Timeout > 0 {
			c.Set("middleware.timeout", config.Timeout)
		}
		return nil
	})
}
func NewExamplePostProcessMiddleware(config ExampleConfig) func(next context.HandlerFunc) context.HandlerFunc {
	if err := config.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				DefaultErrorHandler(c, err, "ExamplePostProcess")
			}
		}
	}
	preHandler := func(c *context.Context) error {
		c.SetHeader("X-Pre-Process", "true")
		c.Set("start_time", time.Now())
		return nil
	}
	postHandler := func(c *context.Context) error {
		if startTime, exists := c.Get("start_time"); exists {
			if t, ok := startTime.(time.Time); ok {
				duration := time.Since(t)
				c.SetHeader("X-Processing-Time", duration.String())
			}
		}
		c.SetHeader("X-Post-Process", "true")
		return nil
	}
	return PostProcessMiddleware("ExamplePostProcess", config.BaseConfig, preHandler, postHandler)
}
func Example(timeout time.Duration, customHeader string) func(next context.HandlerFunc) context.HandlerFunc {
	return NewExampleMiddleware(ExampleConfig{
		Timeout:        timeout,
		CustomHeader:   customHeader,
		AllowedMethods: []string{"GET", "POST"},
	})
}
