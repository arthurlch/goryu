package cache

import (
	"bytes"
	"container/list"
	"net/http"
	"sync"
	"time"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
)

type cacheEntry struct {
	statusCode int
	headers    http.Header
	body       []byte
	createdAt  time.Time
	lruElement *list.Element
}
type lruItem struct {
	key   string
	entry *cacheEntry
}
type secureCache struct {
	mu            sync.RWMutex
	entries       map[string]*cacheEntry
	lruList       *list.List
	maxSize       int
	maxMemory     int64
	currentMemory int64
	expiration    time.Duration
	lastCleanup   time.Time
}
type Config struct {
	base.BaseConfig
	Expiration      time.Duration
	MaxSize         int
	MaxMemory       int64
	CleanupInterval time.Duration
	KeyGenerator    func(c *context.Context) string
}
type cacheWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (cw *cacheWriter) WriteHeader(code int) {
	cw.statusCode = code
}
func (cw *cacheWriter) Write(b []byte) (int, error) {
	return cw.body.Write(b)
}
func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if c.Expiration <= 0 {
		c.Expiration = 5 * time.Minute
	}
	if c.KeyGenerator == nil {
		c.KeyGenerator = func(ctx *context.Context) string {
			return ctx.Request.Method + ctx.Request.URL.Path
		}
	}
	if c.MaxSize <= 0 {
		c.MaxSize = 1000
	}
	if c.MaxMemory <= 0 {
		c.MaxMemory = 50 << 20
	}
	if c.CleanupInterval <= 0 {
		c.CleanupInterval = 5 * time.Minute
	}
	if c.MaxSize > 10000 {
		return base.NewConfigError("MaxSize", "cannot exceed 10,000 entries")
	}
	if c.MaxMemory > 500<<20 {
		return base.NewConfigError("MaxMemory", "cannot exceed 500MB")
	}
	return nil
}
func newSecureCache(maxSize int, maxMemory int64, expiration time.Duration, cleanupInterval time.Duration) *secureCache {
	cache := &secureCache{
		entries:       make(map[string]*cacheEntry, maxSize),
		lruList:       list.New(),
		maxSize:       maxSize,
		maxMemory:     maxMemory,
		currentMemory: 0,
		expiration:    expiration,
		lastCleanup:   time.Now(),
	}
	return cache
}
func (sc *secureCache) get(key string) (*cacheEntry, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	entry, found := sc.entries[key]
	if !found {
		return nil, false
	}
	if time.Since(entry.createdAt) >= sc.expiration {
		return nil, false
	}
	sc.lruList.MoveToFront(entry.lruElement)
	return entry, true
}
func (sc *secureCache) put(key string, entry *cacheEntry) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	entrySize := int64(len(entry.body))
	for k, v := range entry.headers {
		entrySize += int64(len(k))
		for _, val := range v {
			entrySize += int64(len(val))
		}
	}
	entrySize += int64(len(key)) + 64
	if entrySize > sc.maxMemory/10 {
		return
	}
	if existingEntry, found := sc.entries[key]; found {
		oldSize := sc.calculateEntrySize(key, existingEntry)
		sc.currentMemory = sc.currentMemory - oldSize + entrySize
		existingEntry.statusCode = entry.statusCode
		existingEntry.headers = entry.headers
		existingEntry.body = entry.body
		existingEntry.createdAt = entry.createdAt
		sc.lruList.MoveToFront(existingEntry.lruElement)
		return
	}
	for (sc.currentMemory+entrySize > sc.maxMemory || len(sc.entries) >= sc.maxSize) && sc.lruList.Len() > 0 {
		sc.evictLRU()
	}
	lruItem := &lruItem{key: key, entry: entry}
	element := sc.lruList.PushFront(lruItem)
	entry.lruElement = element
	sc.entries[key] = entry
	sc.currentMemory += entrySize
}
func (sc *secureCache) evictLRU() {
	if sc.lruList.Len() == 0 {
		return
	}
	element := sc.lruList.Back()
	if element != nil {
		sc.lruList.Remove(element)
		lruItem := element.Value.(*lruItem)
		if entry, found := sc.entries[lruItem.key]; found {
			entrySize := sc.calculateEntrySize(lruItem.key, entry)
			sc.currentMemory -= entrySize
			delete(sc.entries, lruItem.key)
		}
	}
}
func (sc *secureCache) calculateEntrySize(key string, entry *cacheEntry) int64 {
	size := int64(len(entry.body) + len(key) + 64)
	for k, v := range entry.headers {
		size += int64(len(k))
		for _, val := range v {
			size += int64(len(val))
		}
	}
	return size
}
func (sc *secureCache) cleanup() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	now := time.Now()
	if now.Sub(sc.lastCleanup) < time.Minute {
		return
	}
	expiredKeys := make([]string, 0)
	for key, entry := range sc.entries {
		if now.Sub(entry.createdAt) >= sc.expiration {
			expiredKeys = append(expiredKeys, key)
		}
	}
	for _, key := range expiredKeys {
		if entry, found := sc.entries[key]; found {
			sc.lruList.Remove(entry.lruElement)
			entrySize := sc.calculateEntrySize(key, entry)
			sc.currentMemory -= entrySize
			delete(sc.entries, key)
		}
	}
	sc.lastCleanup = now
}
func New(config ...Config) func(next context.HandlerFunc) context.HandlerFunc {
	cfg := Config{}
	if len(config) > 0 {
		cfg = config[0]
	}

	if err := cfg.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "Cache")
			}
		}
	}
	cacheStore := newSecureCache(cfg.MaxSize, cfg.MaxMemory, cfg.Expiration, cfg.CleanupInterval)
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if cfg.Skip != nil && cfg.Skip(c) {
				next(c)
				return
			}
			if c.Request.Method != http.MethodGet {
				next(c)
				return
			}
			if time.Since(cacheStore.lastCleanup) > cfg.CleanupInterval {
				go cacheStore.cleanup()
			}
			key := cfg.KeyGenerator(c)
			entry, found := cacheStore.get(key)
			if found {
				for k, v := range entry.headers {
					c.Writer.Header()[k] = v
				}
				c.Writer.WriteHeader(entry.statusCode)
				if _, err := c.Writer.Write(entry.body); err != nil {
					logger := cfg.Logger
					if logger == nil {
						logger = base.DefaultLogger("Cache")
					}
					logger.Printf("could not write cached response body: %v", err)
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
			if writer.statusCode < 200 || writer.statusCode >= 400 {
				originalWriter := writer.ResponseWriter
				c.Writer = originalWriter
				for k, v := range writer.Header() {
					c.Writer.Header()[k] = v
				}
				c.Writer.WriteHeader(writer.statusCode)
				c.Writer.Write(writer.body.Bytes())
				return
			}
			newEntry := &cacheEntry{
				statusCode: writer.statusCode,
				headers:    writer.Header(),
				body:       writer.body.Bytes(),
				createdAt:  time.Now(),
			}
			cacheStore.put(key, newEntry)
			originalWriter := writer.ResponseWriter
			c.Writer = originalWriter
			for k, v := range newEntry.headers {
				c.Writer.Header()[k] = v
			}
			c.Writer.WriteHeader(newEntry.statusCode)
			if _, err := c.Writer.Write(newEntry.body); err != nil {
				logger := cfg.Logger
				if logger == nil {
					logger = base.DefaultLogger("Cache")
				}
				logger.Printf("could not write captured body to response: %v", err)
			}
		}
	}
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New()
}
