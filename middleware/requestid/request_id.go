package requestid
import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"github.com/arthurlch/goryu/context"
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
func New(config Config) func(next context.HandlerFunc) context.HandlerFunc {
	if err := config.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "RequestID")
			}
		}
	}
	handler := func(c *context.Context) error {
		rid := c.Request.Header.Get(config.Header)
		if rid == "" {
			rid = config.Generator()
		}
		c.Set(config.ContextKey, rid)
		c.Writer.Header().Set(config.Header, rid)
		return nil
	}
	return base.StandardMiddleware("RequestID", config.BaseConfig, handler)
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{})
}
func defaultGenerator() string {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "could-not-generate-random-string"
	}
	return hex.EncodeToString(b)
}
