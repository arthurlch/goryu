package aimeter

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
)

const usageKey = "goryu.aimeter.usage"

type Usage struct {
	PromptTokens     int
	CompletionTokens int
}

func (u Usage) Total() int { return u.PromptTokens + u.CompletionTokens }

func SetUsage(c *context.Context, u Usage) {
	c.Set(usageKey, u)
}

func GetUsage(c *context.Context) (Usage, bool) {
	v, ok := c.Get(usageKey)
	if !ok {
		return Usage{}, false
	}
	u, ok := v.(Usage)
	return u, ok
}

type Result struct {
	Key     string
	Usage   Usage
	CostUSD float64
}

type Config struct {
	base.BaseConfig

	KeyGenerator func(c *context.Context) string

	PromptCostPer1K     float64
	CompletionCostPer1K float64

	MaxRequests int
	MaxTokens   int
	Window      time.Duration

	MaxClients int

	OnResult func(c *context.Context, r Result)

	LimitReached func(c *context.Context, reason string)
}

func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}

func (c *Config) Validate() error {
	if c.Window <= 0 {
		c.Window = time.Minute
	}
	if c.MaxClients <= 0 {
		c.MaxClients = 10000
	}
	if c.MaxClients > 100000 {
		return base.NewConfigError("MaxClients", "cannot exceed 100,000 entries")
	}
	if c.PromptCostPer1K < 0 || c.CompletionCostPer1K < 0 {
		return base.NewConfigError("Pricing", "cost per 1K tokens cannot be negative")
	}
	if c.KeyGenerator == nil {
		c.KeyGenerator = func(ctx *context.Context) string {
			if k := ctx.GetHeader("X-API-Key"); k != "" {
				return k
			}
			return ctx.RemoteIP()
		}
	}
	if c.LimitReached == nil {
		c.LimitReached = func(ctx *context.Context, reason string) {
			_ = ctx.SetHeader("X-RateLimit-Reason", reason)
			_ = ctx.Status(http.StatusTooManyRequests).Text(http.StatusTooManyRequests, "Too Many Requests")
		}
	}
	return nil
}

func (c *Config) cost(u Usage) float64 {
	return float64(u.PromptTokens)/1000*c.PromptCostPer1K +
		float64(u.CompletionTokens)/1000*c.CompletionCostPer1K
}

type window struct {
	requests  int
	tokens    int
	windowEnd time.Time
}

type meter struct {
	mu         sync.Mutex
	keys       map[string]*window
	maxClients int
	window     time.Duration
}

func newMeter(maxClients int, w time.Duration) *meter {
	return &meter{keys: make(map[string]*window, 0), maxClients: maxClients, window: w}
}

func (m *meter) allow(key string, maxReq, maxTok int) (bool, string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	w, ok := m.keys[key]
	if !ok || now.After(w.windowEnd) {
		if !ok {
			m.evictIfFull()
		}
		w = &window{windowEnd: now.Add(m.window)}
		m.keys[key] = w
	}
	if maxTok > 0 && w.tokens >= maxTok {
		return false, "token budget exceeded"
	}
	if maxReq > 0 && w.requests >= maxReq {
		return false, "request limit exceeded"
	}
	w.requests++
	return true, ""
}

func (m *meter) record(key string, tokens int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if w, ok := m.keys[key]; ok {
		w.tokens += tokens
	}
}

// Caller must hold m.mu.
func (m *meter) evictIfFull() {
	if len(m.keys) < m.maxClients {
		return
	}
	now := time.Now()
	for k, w := range m.keys {
		if now.After(w.windowEnd) {
			delete(m.keys, k)
		}
	}
	if len(m.keys) < m.maxClients {
		return
	}
	for k := range m.keys {
		delete(m.keys, k)
		break
	}
}

func New(config ...Config) func(next context.HandlerFunc) context.HandlerFunc {
	cfg := Config{}
	if len(config) > 0 {
		cfg = config[0]
	}
	if err := cfg.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "AIMeter")
			}
		}
	}
	m := newMeter(cfg.MaxClients, cfg.Window)

	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if cfg.Skip != nil && cfg.Skip(c) {
				next(c)
				return
			}
			key := cfg.KeyGenerator(c)

			if allowed, reason := m.allow(key, cfg.MaxRequests, cfg.MaxTokens); !allowed {
				cfg.LimitReached(c, reason)
				return
			}

			next(c)

			usage, ok := GetUsage(c)
			if !ok {
				return
			}
			m.record(key, usage.Total())
			cost := cfg.cost(usage)

			if !c.IsResponseSent() {
				_ = c.SetHeader("X-Tokens-Prompt", strconv.Itoa(usage.PromptTokens))
				_ = c.SetHeader("X-Tokens-Completion", strconv.Itoa(usage.CompletionTokens))
				_ = c.SetHeader("X-Tokens-Total", strconv.Itoa(usage.Total()))
				_ = c.SetHeader("X-Cost-USD", strconv.FormatFloat(cost, 'f', 6, 64))
			}
			if cfg.OnResult != nil {
				cfg.OnResult(c, Result{Key: key, Usage: usage, CostUSD: cost})
			}
		}
	}
}

func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New()
}
