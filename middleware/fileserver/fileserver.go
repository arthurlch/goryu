package middleware

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type FilesystemConfig struct {
	Root       string
	PathPrefix string
}

func Filesystem(config FilesystemConfig) func(http.Handler) http.Handler {
	cleanRoot := filepath.Clean(config.Root)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if config.PathPrefix != "" && !strings.HasPrefix(r.URL.Path, config.PathPrefix) {
				next.ServeHTTP(w, r)
				return
			}

			path := strings.TrimPrefix(r.URL.Path, config.PathPrefix)
			path = strings.TrimPrefix(path, "/")

			fullPath := filepath.Join(cleanRoot, path)

			file, err := os.Open(fullPath)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			defer func() {
				if cerr := file.Close(); cerr != nil {
					log.Printf("error closing file %s: %v", fullPath, cerr)
				}
			}()

			info, err := file.Stat()
			if err != nil || info.IsDir() {
				next.ServeHTTP(w, r)
				return
			}

			http.ServeContent(w, r, info.Name(), info.ModTime(), file)
		})
	}
}
