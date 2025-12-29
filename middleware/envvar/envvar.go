package envvar

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
)

type Config struct {
	base.BaseConfig
	Path    string
	Expose  []string
	Exclude []string
}

func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if c.Path == "" {
		c.Path = "/envvar"
	}
	if !strings.HasPrefix(c.Path, "/") {
		c.Path = "/" + c.Path
	}
	c.Path = "/" + strings.Trim(c.Path, "/")
	return nil
}
func New(config ...Config) func(next context.HandlerFunc) context.HandlerFunc {
	cfg := Config{}
	if len(config) > 0 {
		cfg = config[0]
	}

	if err := cfg.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "Envvar")
			}
		}
	}
	envMap := collectEnvVars(cfg.Expose, cfg.Exclude)
	jsonResponse, err := json.Marshal(envMap)
	if err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, base.MiddlewareError{
					Middleware: "Envvar",
					Err:        err,
					StatusCode: http.StatusInternalServerError,
				}, "Envvar")
			}
		}
	}
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if cfg.Skip != nil && cfg.Skip(c) {
				next(c)
				return
			}
			if c.Request.URL.Path == cfg.Path {
				c.Writer.Header().Set("Content-Type", "application/json")
				c.Writer.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				if _, err := c.Writer.Write(jsonResponse); err != nil {
					logger := cfg.Logger
					if logger == nil {
						logger = base.DefaultLogger("Envvar")
					}
					logger.Printf("could not write envvar response: %v", err)
				}
				return
			}
			next(c)
		}
	}
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New()
}
func WithPath(path string) func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{Path: path})
}
func WithExpose(expose []string) func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{Expose: expose})
}
func WithExclude(exclude []string) func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{Exclude: exclude})
}
func collectEnvVars(expose, exclude []string) map[string]string {
	envMap := make(map[string]string)
	exposeMap := make(map[string]bool)
	for _, key := range expose {
		exposeMap[key] = true
	}
	excludeMap := make(map[string]bool)
	for _, key := range exclude {
		excludeMap[key] = true
	}
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) != 2 {
			continue
		}
		key, value := pair[0], pair[1]
		if excludeMap[key] {
			continue
		}
		if len(exposeMap) > 0 && !exposeMap[key] {
			continue
		}
		envMap[key] = value
	}
	return envMap
}
