package recovery
import (
	"fmt"
	"net/http"
	"runtime/debug"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/base"
)
type Config struct {
	base.BaseConfig
	EnableStackTrace bool
	CustomRecoveryHandler func(c *context.Context, err interface{})
}
func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if !c.EnableStackTrace {
		c.EnableStackTrace = true
	}
	return nil
}
func New(config Config) func(next context.HandlerFunc) context.HandlerFunc {
	if err := config.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "Recovery")
			}
		}
	}
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if config.Skip != nil && config.Skip(c) {
				next(c)
				return
			}
			defer func() {
				if r := recover(); r != nil {
					panicErr, ok := r.(error)
					if !ok {
						panicErr = fmt.Errorf("%v", r)
					}
					if config.CustomRecoveryHandler != nil {
						config.CustomRecoveryHandler(c, r)
						return
					}
					logger := config.Logger
					if logger == nil {
						logger = base.DefaultLogger("Recovery")
					}
					logger.Printf("Panic recovered: %v", panicErr)
					if config.EnableStackTrace {
						logger.Printf("Stack trace:\n%s", debug.Stack())
					}
					if c.Writer.Header().Get("Content-Type") == "" {
						if jsonErr := c.JSON(http.StatusInternalServerError, map[string]string{
							"error": "Internal Server Error",
						}); jsonErr != nil {
							logger.Printf("recovery middleware: could not send error response: %v", jsonErr)
						}
					}
				}
			}()
			next(c)
		}
	}
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{})
}
