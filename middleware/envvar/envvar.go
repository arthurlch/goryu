package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
)

// NOTE:: EnvvarConfig defines the configuration for the EnvvarMiddleware
type EnvvarConfig struct {
	// Path is the endpoint to serve the env vars from.
	Path string
	// Expose is a list of specific environment variable keys to expose.
	// If empty, all variables are exposed (respecting the Exclude list)
	Expose []string
	// Exclude is a list of specific environment variable keys to hide
	// This is useful when exposing all variables except a few secrets.
	Exclude []string
}

func EnvvarMiddleware(config EnvvarConfig) func(http.Handler) http.Handler {
	envMap := make(map[string]string)

	exposeMap := make(map[string]bool)
	for _, key := range config.Expose {
		exposeMap[key] = true
	}
	excludeMap := make(map[string]bool)
	for _, key := range config.Exclude {
		excludeMap[key] = true
	}

	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		key, value := pair[0], pair[1]

		if excludeMap[key] {
			continue
		}

		if len(exposeMap) > 0 && !exposeMap[key] {
			continue
		}

		envMap[key] = value
	}

	jsonResponse, err := json.Marshal(envMap)
	if err != nil {
		panic(err)
	}

	cleanPath := "/" + strings.Trim(config.Path, "/")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == cleanPath {
				w.Header().Set("Content-Type", "application/json")
				if _, err := w.Write(jsonResponse); err != nil {
					log.Printf("Error writing envvar response: %v", err)
				}
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
