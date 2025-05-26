package middleware

import (
	"log"
	"time"

	"github.com/arthurlch/goryu/pkg/context"
)

func Logger() Middleware {
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			start := time.Now()
			next(c)
			log.Printf("%s %s %v", c.Request.Method, c.Request.URL.Path, time.Since(start))
		}
	}
}