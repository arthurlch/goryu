package base

import (
	"log"
	"net/http"
	"os"

	context "github.com/arthurlch/goryu/goryuctx"
)

type Logger interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}
type BaseConfig struct {
	Skip         func(c *context.Context) bool
	Logger       Logger
	ErrorHandler func(c *context.Context, err error, middlewareName string)
}
type MiddlewareError struct {
	Middleware string
	Err        error
	StatusCode int
}

func (e MiddlewareError) Error() string {
	return e.Err.Error()
}
func DefaultErrorHandler(c *context.Context, err error, middlewareName string) {
	statusCode := http.StatusInternalServerError
	if mErr, ok := err.(MiddlewareError); ok && mErr.StatusCode != 0 {
		statusCode = mErr.StatusCode
	}
	log.Printf("[MIDDLEWARE:%s] Error: %v", middlewareName, err)
	if statusCode >= 500 {
		_ = c.Error(err, statusCode)
	} else {
		_ = c.ErrorWithMessage(err, statusCode, err.Error())
	}
}
func DefaultLogger(middlewareName string) Logger {
	return log.New(os.Stderr, "[GORYU:"+middlewareName+"] ", log.LstdFlags)
}

type ConfigurableMiddleware interface {
	Configure(base *BaseConfig)
	Validate() error
}

func StandardMiddleware(middlewareName string, base BaseConfig, handler func(c *context.Context) error) func(next context.HandlerFunc) context.HandlerFunc {
	if base.Logger == nil {
		base.Logger = DefaultLogger(middlewareName)
	}
	if base.ErrorHandler == nil {
		base.ErrorHandler = DefaultErrorHandler
	}
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if base.Skip != nil && base.Skip(c) {
				next(c)
				return
			}
			if err := handler(c); err != nil {
				base.ErrorHandler(c, err, middlewareName)
				return
			}
			next(c)
		}
	}
}
func PostProcessMiddleware(middlewareName string, base BaseConfig,
	preHandler func(c *context.Context) error,
	postHandler func(c *context.Context) error) func(next context.HandlerFunc) context.HandlerFunc {
	if base.Logger == nil {
		base.Logger = DefaultLogger(middlewareName)
	}
	if base.ErrorHandler == nil {
		base.ErrorHandler = DefaultErrorHandler
	}
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if base.Skip != nil && base.Skip(c) {
				next(c)
				return
			}
			if preHandler != nil {
				if err := preHandler(c); err != nil {
					base.ErrorHandler(c, err, middlewareName)
					return
				}
			}
			next(c)
			if postHandler != nil {
				if err := postHandler(c); err != nil {
					base.Logger.Printf("Post-processing error: %v", err)
				}
			}
		}
	}
}
func ValidationError(field string, message string) MiddlewareError {
	return MiddlewareError{
		Err:        NewConfigError(field, message),
		StatusCode: http.StatusBadRequest,
	}
}

type ConfigError struct {
	Field   string
	Message string
}

func (e ConfigError) Error() string {
	return "config error for field '" + e.Field + "': " + e.Message
}
func NewConfigError(field, message string) ConfigError {
	return ConfigError{
		Field:   field,
		Message: message,
	}
}

type ResponseWriterWrapper interface {
	http.ResponseWriter
	Status() int
	Size() int
	Written() bool
}
type StandardResponseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
	written    bool
}

func NewStandardResponseWriter(w http.ResponseWriter) *StandardResponseWriter {
	return &StandardResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}
func (w *StandardResponseWriter) WriteHeader(statusCode int) {
	if !w.written {
		w.statusCode = statusCode
		w.written = true
		w.ResponseWriter.WriteHeader(statusCode)
	}
}
func (w *StandardResponseWriter) Write(data []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(data)
	w.size += n
	return n, err
}
func (w *StandardResponseWriter) Status() int {
	return w.statusCode
}
func (w *StandardResponseWriter) Size() int {
	return w.size
}
func (w *StandardResponseWriter) Written() bool {
	return w.written
}
