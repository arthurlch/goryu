package cache

import (
	"bytes"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/arthurlch/goryu"
)

type cacheEntry struct {
	statusCode int
	headers    http.Header
	body       []byte
	createdAt  time.Time
}

type Config struct {
	Next func(c *goryu.Context) bool

	Expiration time.Duration

	KeyGenerator func(c *goryu.Context) string
}

type cacheWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (cw *cacheWriter) WriteHeader(code int) {
	cw.statusCode = code
}

// Write captures the response body.
func (cw *cacheWriter) Write(b []byte) (int, error) {
	return cw.body.Write(b)
}

func New(config ...Config) goryu.Middleware {
	var cfg Config
	if len(config) > 0 {
		cfg = config[0]
	}

	if cfg.Expiration <= 0 {
		cfg.Expiration = 5 * time.Minute
	}
	if cfg.KeyGenerator == nil {
		cfg.KeyGenerator = func(c *goryu.Context) string {
			return c.Request.Method + c.Request.URL.Path
		}
	}

	cacheStore := make(map[string]*cacheEntry)
	var mu sync.Mutex

	return func(next goryu.HandlerFunc) goryu.HandlerFunc {
		return func(c *goryu.Context) {
			if cfg.Next != nil && cfg.Next(c) {
				next(c)
				return
			}

			if c.Request.Method != http.MethodGet {
				next(c)
				return
			}

			key := cfg.KeyGenerator(c)
			mu.Lock()
			entry, found := cacheStore[key]
			mu.Unlock()

			if found && time.Since(entry.createdAt) < cfg.Expiration {
				for k, v := range entry.headers {
					c.Writer.Header()[k] = v
				}
				c.Writer.WriteHeader(entry.statusCode)
				if _, err := c.Writer.Write(entry.body); err != nil {
					log.Printf("cache: could not write response body: %v", err)
				}
				return
			}

			writer := &cacheWriter{
				ResponseWriter: c.Writer,
				statusCode:     http.StatusOK,
				body:           bytes.NewBuffer(nil),
			}
			c.Writer = writer

			next(c)

			originalWriter := writer.ResponseWriter
			c.Writer = originalWriter

			newEntry := &cacheEntry{
				statusCode: writer.statusCode,
				headers:    writer.Header(),
				body:       writer.body.Bytes(),
				createdAt:  time.Now(),
			}

			mu.Lock()
			cacheStore[key] = newEntry
			mu.Unlock()

			for k, v := range newEntry.headers {
				c.Writer.Header()[k] = v
			}
			c.Writer.WriteHeader(newEntry.statusCode)
			if _, err := c.Writer.Write(newEntry.body); err != nil {
				log.Printf("cache: could not write captured body to response: %v", err)
			}
		}
	}
}
