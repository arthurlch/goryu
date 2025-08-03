package recovery

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/arthurlch/goryu"
)

type Config struct {
	Next func(c *goryu.Context) bool

	EnableStackTrace bool
}

func New(config ...Config) goryu.Middleware {
	var cfg Config
	if len(config) > 0 {
		cfg = config[0]
	}

	if !cfg.EnableStackTrace {
		cfg.EnableStackTrace = true
	}

	return func(next goryu.HandlerFunc) goryu.HandlerFunc {
		return func(c *goryu.Context) {
			if cfg.Next != nil && cfg.Next(c) {
				next(c)
				return
			}

			defer func() {
				if r := recover(); r != nil {
					err, ok := r.(error)
					if !ok {
						err = fmt.Errorf("%v", r)
					}

					log.Printf("Panic recovered: %v", err)
					if cfg.EnableStackTrace {
						log.Printf("Stack trace:\n%s", debug.Stack())
					}

					if c.Writer.Header().Get("Content-Type") == "" {
						if jsonErr := c.JSON(http.StatusInternalServerError, map[string]string{
							"error": "Internal Server Error",
						}); jsonErr != nil {
							log.Printf("recovery middleware: could not send error response: %v", jsonErr)
						}
					}
				}
			}()

			next(c)
		}
	}
}
