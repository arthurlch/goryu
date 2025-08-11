package middleware

import (
	"expvar"
	"net/http"
	"strings"
)

// If the path is "/debug/vars", this provides a simple way to wrap
// the default expvar handler with other middleware like authentication
func ExpvarMiddleware(path string) func(http.Handler) http.Handler {
	cleanPath := "/" + strings.Trim(path, "/")

	return func(next http.Handler) http.Handler {
		expvarHandler := expvar.Handler()

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == cleanPath {
				expvarHandler.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
