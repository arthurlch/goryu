package favicon

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
)

type Config struct {
	base.BaseConfig
	File      string
	URL       string
	CacheFile bool
	MaxAge    int
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
func New(config ...Config) func(next context.HandlerFunc) context.HandlerFunc {
	cfg := Config{}
	if len(config) > 0 {
		cfg = config[0]
	}

	if err := cfg.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "Favicon")
			}
		}
	}
	var cache *faviconCache
	if cfg.File != "" && cfg.CacheFile {
		cache = &faviconCache{}
		if err := loadFaviconFile(cfg.File, cache); err != nil {
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
			if cfg.Skip != nil && cfg.Skip(c) {
				next(c)
				return
			}
			if c.Request.URL.Path != cfg.URL {
				next(c)
				return
			}
			if cfg.MaxAge > 0 {
				c.Writer.Header().Set("Cache-Control", "max-age=86400, public")
			}
			if cfg.File == "" {
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
					logger := cfg.Logger
					if logger == nil {
						logger = base.DefaultLogger("Favicon")
					}
					logger.Printf("could not write favicon data: %v", err)
				}
			} else {
				http.ServeFile(c.Writer, c.Request, cfg.File)
			}
		}
	}
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New()
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
	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)

	// Normalize .ico MIME type for consistency
	if ext == ".ico" && (contentType == "image/vnd.microsoft.icon" || contentType == "image/x-icon" || contentType == "") {
		contentType = "image/x-icon"
	} else if contentType == "" {
		contentType = "application/octet-stream"
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.data = data
	cache.contentType = contentType
	return nil
}
