package requestid

import (
	"crypto/rand"
	"encoding/hex"
	"io"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
)
const DefaultRequestIDHeader = "X-Request-ID"
type Config struct {
	base.BaseConfig
	Header string
	Generator func() string
	ContextKey string
}
func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if c.Header == "" {
		c.Header = DefaultRequestIDHeader
	}
	if c.Generator == nil {
		c.Generator = defaultGenerator
	}
	if c.ContextKey == "" {
		c.ContextKey = "requestid"
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
				base.DefaultErrorHandler(c, err, "RequestID")
			}
		}
	}
	handler := func(c *context.Context) error {
		rid := c.Request.Header.Get(cfg.Header)
		if rid == "" {
			rid = cfg.Generator()
		}
		c.Set(cfg.ContextKey, rid)
		c.Writer.Header().Set(cfg.Header, rid)
		return nil
	}
	return base.StandardMiddleware("RequestID", cfg.BaseConfig, handler)
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New()
}
func defaultGenerator() string {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "could-not-generate-random-string"
	}
	return hex.EncodeToString(b)
}
