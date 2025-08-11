package middleware

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type FaviconConfig struct {
	File  string
	Cache bool
}

func Favicon(config FaviconConfig) func(http.Handler) http.Handler {
	if config.File == "" {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/favicon.ico" {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				next.ServeHTTP(w, r)
			})
		}
	}

	if config.Cache {
		fileBytes, err := os.ReadFile(config.File)
		if err != nil {
			panic("favicon: could not read file specified in config: " + err.Error())
		}
		// check the mime type
		contentType := "image/x-icon"
		switch filepath.Ext(config.File) {
		case ".png":
			contentType = "image/png"
		case ".svg":
			contentType = "image/svg+xml"
		}

		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/favicon.ico" {
					w.Header().Set("Content-Type", contentType)
					if _, err := w.Write(fileBytes); err != nil {
						log.Printf("Error writing cached favicon: %v", err)
					}
					return
				}
				next.ServeHTTP(w, r)
			})
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/favicon.ico" {
				http.ServeFile(w, r, config.File)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
