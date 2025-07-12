package requestid

import (
	"crypto/rand"
	"encoding/hex"
	"io"

	"github.com/arthurlch/goryu"
)

const DefaultRequestIDHeader = "X-Request-ID"

type Config struct {
	Next func(c *goryu.Context) bool

	Header string

	Generator func() string

	ContextKey string
}

func New(config ...Config) goryu.Middleware {
	var cfg Config
	if len(config) > 0 {
		cfg = config[0]
	}

	if cfg.Header == "" {
		cfg.Header = DefaultRequestIDHeader
	}
	if cfg.Generator == nil {
		cfg.Generator = defaultGenerator
	}
	if cfg.ContextKey == "" {
		cfg.ContextKey = "requestid"
	}

	return func(next goryu.HandlerFunc) goryu.HandlerFunc {
		return func(c *goryu.Context) {
			if cfg.Next != nil && cfg.Next(c) {
				next(c)
				return
			}

			rid := c.GetHeader(cfg.Header)
			if rid == "" {
				rid = cfg.Generator()
			}

			c.Set(cfg.ContextKey, rid)
			c.Writer.Header().Set(cfg.Header, rid)

			next(c)
		}
	}
}

func defaultGenerator() string {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		// should not happens tho
		return "could-not-generate-random-string"
	}
	return hex.EncodeToString(b)
}
