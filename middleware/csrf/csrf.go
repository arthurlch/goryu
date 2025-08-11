package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"time"
)

const (
	csrfTokenHeader = "X-CSRF-Token"
	csrfTokenCookie = "csrf-token"
	tokenByteLength = 32
)

var (
	errNoCSRFToken = errors.New("missing CSRF token in request header")
)

func generateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isSafeMethod := r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS"

		if isSafeMethod {
			token, err := generateSecureToken(tokenByteLength)
			if err != nil {
				log.Printf("Error generating CSRF token: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			cookie := http.Cookie{
				Name:     csrfTokenCookie,
				Value:    token,
				Expires:  time.Now().Add(12 * time.Hour),
				Secure:   true, // Set to true in production !!!
				HttpOnly: true,
				Path:     "/",
				SameSite: http.SameSiteStrictMode,
			}
			http.SetCookie(w, &cookie)

			w.Header().Set(csrfTokenHeader, token)
		} else {
			tokenFromHeader := r.Header.Get(csrfTokenHeader)
			if tokenFromHeader == "" {
				log.Println("CSRF validation failed: missing token in header")
				http.Error(w, errNoCSRFToken.Error(), http.StatusForbidden)
				return
			}

			cookie, err := r.Cookie(csrfTokenCookie)
			if err != nil {
				log.Println("CSRF validation failed: missing token cookie")
				http.Error(w, "Missing CSRF cookie", http.StatusForbidden)
				return
			}

			if cookie.Value != tokenFromHeader {
				log.Printf("CSRF validation failed: token mismatch. Header: %s, Cookie: %s", tokenFromHeader, cookie.Value)
				http.Error(w, "Invalid CSRF token", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
