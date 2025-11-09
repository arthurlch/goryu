package favicon
import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/base"
)
type Config struct {
	base.BaseConfig
	File string
	URL string
	CacheFile bool
	MaxAge int
}
type faviconCache struct {
	data        []byte
	contentType string
	mu          sync.RWMutex
}
func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if c.URL == "" {
		c.URL = "/favicon.ico"
	}
	if c.MaxAge <= 0 {
		c.MaxAge = 86400 
	}
	return nil
}
func New(config Config) func(next context.HandlerFunc) context.HandlerFunc {
	if err := config.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "Favicon")
			}
		}
	}
	var cache *faviconCache
	if config.File != "" && config.CacheFile {
		cache = &faviconCache{}
		if err := loadFaviconFile(config.File, cache); err != nil {
			return func(next context.HandlerFunc) context.HandlerFunc {
				return func(c *context.Context) {
					base.DefaultErrorHandler(c, base.MiddlewareError{
						Middleware: "Favicon",
						Err:        err,
						StatusCode: http.StatusInternalServerError,
					}, "Favicon")
				}
			}
		}
	}
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if config.Skip != nil && config.Skip(c) {
				next(c)
				return
			}
			if c.Request.URL.Path != config.URL {
				next(c)
				return
			}
			if config.MaxAge > 0 {
				c.Writer.Header().Set("Cache-Control", "max-age=86400, public")
			}
			if config.File == "" {
				c.Writer.WriteHeader(http.StatusNoContent)
				return
			}
			if cache != nil {
				cache.mu.RLock()
				contentType := cache.contentType
				data := cache.data
				cache.mu.RUnlock()
				c.Writer.Header().Set("Content-Type", contentType)
				if _, err := c.Writer.Write(data); err != nil {
					logger := config.Logger
					if logger == nil {
						logger = base.DefaultLogger("Favicon")
					}
					logger.Printf("could not write favicon data: %v", err)
				}
			} else {
				http.ServeFile(c.Writer, c.Request, config.File)
			}
		}
	}
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{})
}
func WithFile(file string) func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{
		File:      file,
		CacheFile: true,
	})
}
func loadFaviconFile(filePath string, cache *faviconCache) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	contentType := mime.TypeByExtension(filepath.Ext(filePath))
	if contentType == "" {
		contentType = "image/x-icon"
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.data = data
	cache.contentType = contentType
	return nil
}