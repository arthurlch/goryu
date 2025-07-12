package basicauth

import (
	"encoding/base64"
	"log"
	"net/http"
	"strings"

	"github.com/arthurlch/goryu"
)

type Config struct {
	Next func(c *goryu.Context) bool

	Users map[string]string

	Validator func(username, password string) bool

	// Realm is a string to be used in the WWW-Authenticate header.
	// Default: "Restricted"
	Realm string
}

func New(config Config) goryu.Middleware {
	if config.Realm == "" {
		config.Realm = "Restricted"
	}

	return func(next goryu.HandlerFunc) goryu.HandlerFunc {
		return func(c *goryu.Context) {
			if config.Next != nil && config.Next(c) {
				next(c)
				return
			}

			auth := c.GetHeader("Authorization")
			if auth == "" {
				unauthorized(c, config.Realm)
				return
			}

			const prefix = "Basic "
			if !strings.HasPrefix(auth, prefix) {
				unauthorized(c, config.Realm)
				return
			}

			encoded, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
			if err != nil {
				unauthorized(c, config.Realm)
				return
			}

			creds := string(encoded)
			parts := strings.SplitN(creds, ":", 2)
			if len(parts) != 2 {
				unauthorized(c, config.Realm)
				return
			}

			username, password := parts[0], parts[1]

			if config.Validator != nil {
				if config.Validator(username, password) {
					next(c)
					return
				}
				unauthorized(c, config.Realm)
				return
			}

			if storedPassword, ok := config.Users[username]; ok && storedPassword == password {
				next(c)
				return
			}

			unauthorized(c, config.Realm)
		}
	}
}

func unauthorized(c *goryu.Context, realm string) {
	c.Writer.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
	if err := c.Status(http.StatusUnauthorized).Text(http.StatusUnauthorized, "Unauthorized"); err != nil {
		log.Printf("basicauth: could not send error response: %v", err)
	}
}
