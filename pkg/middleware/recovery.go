package middleware

import (
	"log"
	"net/http"

	"github.com/arthurlch/goryu/pkg/context"
)

// when panic occurs we need recovery
func Recovery() Middleware {
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			defer func() {
				if err := recover(); err != nil {
					log.Printf("Panic: %v", err)
					c.Text(http.StatusInternalServerError, "Internal Server Error")
				}
			}()

			next(c)
		}
	}
}
