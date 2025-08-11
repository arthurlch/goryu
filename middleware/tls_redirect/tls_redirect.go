package middleware

import (
	"net/http"
)

// RequireTLSMiddleware redirects insecure HTTP requests to HTTPS !
// Wew checks the 'X-Forwarded-Proto' header, which is standard for reverse proxies.
// This is useful if your server is behind a reverse proxy (like Nginx or a cloud load balancer)
// that terminates TLS and forwards traffic to your app over HTTP
func RequireTLSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proto := r.Header.Get("X-Forwarded-Proto")

		if proto == "http" || (proto == "" && r.TLS == nil) {
			targetURL := "https://" + r.Host + r.URL.RequestURI()
			http.Redirect(w, r, targetURL, http.StatusMovedPermanently)
			return
		}

		next.ServeHTTP(w, r)
	})
}
