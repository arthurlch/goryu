package plugins

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/limiter"
)

// RateLimitBuilder provides fluent configuration for rate limiting middleware
type RateLimitBuilder struct {
	*BaseBuilder
	config limiter.Config
}

func NewRateLimitBuilder() *RateLimitBuilder {
	return &RateLimitBuilder{
		BaseBuilder: NewBaseBuilder("ratelimit"),
		config: limiter.Config{
			Max:             100,
			Expiration:      time.Minute,
			MaxClients:      10000,
			CleanupInterval: time.Minute,
			KeyGenerator: func(c *context.Context) string {
				return getClientIP(c)
			},
			LimitReached: func(c *context.Context) {
				c.Status(http.StatusTooManyRequests)
				c.Text(http.StatusTooManyRequests, "Rate limit exceeded")
			},
		},
	}
}

func (b *RateLimitBuilder) Rate(max int, duration time.Duration) *RateLimitBuilder {
	b.config.Max = max
	b.config.Expiration = duration
	return b
}

func (b *RateLimitBuilder) Max(max int) *RateLimitBuilder {
	b.config.Max = max
	return b
}

func (b *RateLimitBuilder) Duration(duration time.Duration) *RateLimitBuilder {
	b.config.Expiration = duration
	return b
}

func (b *RateLimitBuilder) PerSecond(max int) *RateLimitBuilder {
	b.config.Max = max
	b.config.Expiration = time.Second
	return b
}

func (b *RateLimitBuilder) PerMinute(max int) *RateLimitBuilder {
	b.config.Max = max
	b.config.Expiration = time.Minute
	return b
}

func (b *RateLimitBuilder) PerHour(max int) *RateLimitBuilder {
	b.config.Max = max
	b.config.Expiration = time.Hour
	return b
}

func (b *RateLimitBuilder) PerDay(max int) *RateLimitBuilder {
	b.config.Max = max
	b.config.Expiration = 24 * time.Hour
	return b
}

func (b *RateLimitBuilder) KeyGenerator(keyGen func(c *context.Context) string) *RateLimitBuilder {
	b.config.KeyGenerator = keyGen
	return b
}

func (b *RateLimitBuilder) ByIP() *RateLimitBuilder {
	b.config.KeyGenerator = func(c *context.Context) string {
		return getClientIP(c)
	}
	return b
}

func (b *RateLimitBuilder) ByHeader(header string) *RateLimitBuilder {
	b.config.KeyGenerator = func(c *context.Context) string {
		return c.GetHeader(header)
	}
	return b
}

func (b *RateLimitBuilder) ByUserID() *RateLimitBuilder {
	b.config.KeyGenerator = func(c *context.Context) string {
		if userID, exists := c.Get("user_id"); exists {
			return fmt.Sprintf("user:%v", userID)
		}
		return getClientIP(c) // Fallback to IP
	}
	return b
}

func (b *RateLimitBuilder) ByAPIKey(headerName string) *RateLimitBuilder {
	if headerName == "" {
		headerName = "X-API-Key"
	}
	b.config.KeyGenerator = func(c *context.Context) string {
		apiKey := c.Request.Header.Get(headerName)
		if apiKey == "" {
			return getClientIP(c) // Fallback to IP
		}
		return fmt.Sprintf("api:%s", apiKey)
	}
	return b
}

func (b *RateLimitBuilder) Handler(handler func(c *context.Context)) *RateLimitBuilder {
	b.config.LimitReached = handler
	return b
}

func (b *RateLimitBuilder) JSONResponse() *RateLimitBuilder {
	b.config.LimitReached = func(c *context.Context) {
		c.JSON(http.StatusTooManyRequests, map[string]interface{}{
			"error":   "Rate limit exceeded",
			"message": "Too many requests, please try again later",
			"retry_after": int(b.config.Expiration.Seconds()),
		})
	}
	return b
}

func (b *RateLimitBuilder) CustomMessage(message string) *RateLimitBuilder {
	b.config.LimitReached = func(c *context.Context) {
		c.Status(http.StatusTooManyRequests)
		c.Text(http.StatusTooManyRequests, message)
	}
	return b
}

func (b *RateLimitBuilder) MaxClients(max int) *RateLimitBuilder {
	b.config.MaxClients = max
	return b
}

func (b *RateLimitBuilder) CleanupInterval(interval time.Duration) *RateLimitBuilder {
	b.config.CleanupInterval = interval
	return b
}

func (b *RateLimitBuilder) Burst(burstMax int, burstDuration time.Duration) *RateLimitBuilder {
	b.SetMetadata("burst_max", burstMax)
	b.SetMetadata("burst_duration", burstDuration)
	
	// For now, just use the burst max as the regular max
	// A more sophisticated implementation could use a token bucket algorithm
	b.config.Max = burstMax
	b.config.Expiration = burstDuration
	return b
}

func (b *RateLimitBuilder) Conservative() *RateLimitBuilder {
	return b.PerMinute(60).ByIP()
}

func (b *RateLimitBuilder) Moderate() *RateLimitBuilder {
	return b.PerMinute(200).ByIP()
}

func (b *RateLimitBuilder) Generous() *RateLimitBuilder {
	return b.PerMinute(1000).ByIP()
}

func (b *RateLimitBuilder) VeryStrict() *RateLimitBuilder {
	return b.PerMinute(10).ByIP()
}

func (b *RateLimitBuilder) Build() context.Middleware {
	if err := b.Validate(); err != nil {
		panic(fmt.Sprintf("RateLimit configuration invalid: %v", err))
	}
	return limiter.New(b.config)
}

func (b *RateLimitBuilder) Validate() error {
	b.ClearErrors()
	
	if b.config.Max <= 0 {
		b.AddError(fmt.Errorf("max requests must be greater than 0, got %d", b.config.Max))
	}
	
	if b.config.Expiration <= 0 {
		b.AddError(fmt.Errorf("expiration must be greater than 0, got %v", b.config.Expiration))
	}
	
	if b.config.KeyGenerator == nil {
		b.AddError(fmt.Errorf("key generator function cannot be nil"))
	}
	
	if b.config.LimitReached == nil {
		b.AddError(fmt.Errorf("limit reached handler cannot be nil"))
	}
	
	if b.config.MaxClients <= 0 {
		b.AddError(fmt.Errorf("max clients must be greater than 0, got %d", b.config.MaxClients))
	}
	
	if b.config.CleanupInterval <= 0 {
		b.AddError(fmt.Errorf("cleanup interval must be greater than 0, got %v", b.config.CleanupInterval))
	}
	
	if b.config.Max > 10000 {
		// Not an error, but could be logged as a warning
		b.SetMetadata("warning", "Very high rate limit detected - ensure this is intentional")
	}
	
	if b.config.Max < 5 && b.config.Expiration <= time.Minute {
		b.SetMetadata("warning", "Very strict rate limit detected - may impact user experience")
	}
	
	return b.BaseBuilder.Validate()
}

func (b *RateLimitBuilder) Reset() Builder {
	return NewRateLimitBuilder()
}

func (b *RateLimitBuilder) Clone() Builder {
	clone := NewRateLimitBuilder()
	clone.config = b.config
	return clone
}

func getClientIP(c *context.Context) string {
	if xff := c.Request.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}
	
	if xri := c.Request.Header.Get("X-Real-IP"); xri != "" {
		if net.ParseIP(xri) != nil {
			return xri
		}
	}
	
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return host
}

func init() {
	Register("ratelimit", func() Builder {
		return NewRateLimitBuilder()
	})
}