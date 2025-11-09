package limiter
import (
	"container/heap"
	"net/http"
	"sync"
	"time"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/base"
)
type Config struct {
	base.BaseConfig
	Max int
	Expiration time.Duration
	MaxClients int
	CleanupInterval time.Duration
	KeyGenerator func(c *context.Context) string
	LimitReached func(c *context.Context)
}
type secureLimiter struct {
	mu              sync.RWMutex
	clients         map[string]*client
	maxClients      int
	expiration      time.Duration
	lastCleanup     time.Time
	cleanupInterval time.Duration
}
type client struct {
	count      int
	lastAccess time.Time
}
func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if c.Max <= 0 {
		c.Max = 60
	}
	if c.Expiration <= 0 {
		c.Expiration = 1 * time.Minute
	}
	if c.MaxClients <= 0 {
		c.MaxClients = 10000 
	}
	if c.CleanupInterval <= 0 {
		c.CleanupInterval = 1 * time.Minute 
	}
	if c.MaxClients > 100000 {
		return base.NewConfigError("MaxClients", "cannot exceed 100,000 entries")
	}
	if c.KeyGenerator == nil {
		c.KeyGenerator = func(ctx *context.Context) string {
			return ctx.RemoteIP()
		}
	}
	if c.LimitReached == nil {
		c.LimitReached = func(ctx *context.Context) {
			ctx.Status(http.StatusTooManyRequests).Text(http.StatusTooManyRequests, "Too Many Requests")
		}
	}
	return nil
}
func newSecureLimiter(maxClients int, expiration time.Duration, cleanupInterval time.Duration) *secureLimiter {
	return &secureLimiter{
		clients:         make(map[string]*client, maxClients),
		maxClients:      maxClients,
		expiration:      expiration,
		lastCleanup:     time.Now(),
		cleanupInterval: cleanupInterval,
	}
}
func (sl *secureLimiter) checkRate(key string, max int) (bool, int) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	now := time.Now()
	if now.Sub(sl.lastCleanup) > sl.cleanupInterval {
		sl.cleanupExpired(now)
	}
	if clientEntry, exists := sl.clients[key]; exists {
		if now.Sub(clientEntry.lastAccess) <= sl.expiration {
			clientEntry.count++
			clientEntry.lastAccess = now
			return clientEntry.count <= max, clientEntry.count
		}
		delete(sl.clients, key)
	}
	if len(sl.clients) >= sl.maxClients {
		sl.evictOldestClients(sl.maxClients / 10) 
	}
	sl.clients[key] = &client{count: 1, lastAccess: now}
	return true, 1 
}
func (sl *secureLimiter) cleanupExpired(now time.Time) {
	expiredKeys := make([]string, 0)
	for key, clientEntry := range sl.clients {
		if now.Sub(clientEntry.lastAccess) > sl.expiration {
			expiredKeys = append(expiredKeys, key)
		}
	}
	for _, key := range expiredKeys {
		delete(sl.clients, key)
	}
	sl.lastCleanup = now
}
type clientHeap []clientHeapItem
type clientHeapItem struct {
	key        string
	lastAccess time.Time
}
func (h clientHeap) Len() int           { return len(h) }
func (h clientHeap) Less(i, j int) bool { return h[i].lastAccess.Before(h[j].lastAccess) }
func (h clientHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *clientHeap) Push(x interface{}) {
	*h = append(*h, x.(clientHeapItem))
}
func (h *clientHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
func (sl *secureLimiter) evictOldestClients(count int) {
	if count <= 0 || len(sl.clients) == 0 {
		return
	}
	if count >= len(sl.clients)/2 {
		h := &clientHeap{}
		heap.Init(h)
		for key, clientEntry := range sl.clients {
			heap.Push(h, clientHeapItem{
				key:        key,
				lastAccess: clientEntry.lastAccess,
			})
		}
		for i := 0; i < count && h.Len() > 0; i++ {
			item := heap.Pop(h).(clientHeapItem)
			delete(sl.clients, item.key)
		}
	} else {
		h := &clientHeap{}
		heap.Init(h)
		for key, clientEntry := range sl.clients {
			item := clientHeapItem{
				key:        key,
				lastAccess: clientEntry.lastAccess,
			}
			if h.Len() < count {
				heap.Push(h, item)
			} else if item.lastAccess.Before((*h)[0].lastAccess) {
				(*h)[0] = item
				heap.Fix(h, 0)
			}
		}
		for h.Len() > 0 {
			item := heap.Pop(h).(clientHeapItem)
			delete(sl.clients, item.key)
		}
	}
}
func New(config Config) func(next context.HandlerFunc) context.HandlerFunc {
	if err := config.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "Limiter")
			}
		}
	}
	limiter := newSecureLimiter(config.MaxClients, config.Expiration, config.CleanupInterval)
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if config.Skip != nil && config.Skip(c) {
				next(c)
				return
			}
			key := config.KeyGenerator(c)
			allowed, _ := limiter.checkRate(key, config.Max)
			if !allowed {
				config.LimitReached(c)
				return
			}
			next(c)
		}
	}
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{})
}
func WithMax(max int) func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{Max: max})
}
