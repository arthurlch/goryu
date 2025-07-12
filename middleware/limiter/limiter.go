package limiter

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/arthurlch/goryu"
)

type Config struct {
	Next func(c *goryu.Context) bool

	Max int

	Expiration   time.Duration
	KeyGenerator func(c *goryu.Context) string

	LimitReached func(c *goryu.Context)
}

type client struct {
	count      int
	lastAccess time.Time
}

func New(config ...Config) goryu.Middleware {
	var cfg Config
	if len(config) > 0 {
		cfg = config[0]
	}

	if cfg.Max <= 0 {
		cfg.Max = 60
	}
	if cfg.Expiration <= 0 {
		cfg.Expiration = 1 * time.Minute
	}
	if cfg.KeyGenerator == nil {
		cfg.KeyGenerator = func(c *goryu.Context) string {
			return c.RemoteIP()
		}
	}
	if cfg.LimitReached == nil {
		cfg.LimitReached = func(c *goryu.Context) {
			if err := c.Status(http.StatusTooManyRequests).Text(http.StatusTooManyRequests, "Too Many Requests"); err != nil {
				log.Printf("limiter: could not send error response: %v", err)
			}
		}
	}

	clients := make(map[string]*client)
	var mu sync.Mutex

	return func(next goryu.HandlerFunc) goryu.HandlerFunc {
		return func(c *goryu.Context) {
			if cfg.Next != nil && cfg.Next(c) {
				next(c)
				return
			}

			key := cfg.KeyGenerator(c)
			mu.Lock()
			defer mu.Unlock()

			if _, exists := clients[key]; !exists || time.Since(clients[key].lastAccess) > cfg.Expiration {
				clients[key] = &client{count: 1, lastAccess: time.Now()}
				next(c)
				return
			}

			clients[key].count++

			if clients[key].count > cfg.Max {
				cfg.LimitReached(c)
				return
			}

			next(c)
		}
	}
}
