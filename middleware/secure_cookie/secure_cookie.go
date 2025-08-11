// middleware/securecookie.go
package middleware

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

type contextKey string

const secureCookieContextKey = contextKey("secure-cookie-data")
const secureCookieInstanceKey = contextKey("secure-cookie-instance")

var (
	ErrValueNotFound = errors.New("securecookie: value not found")
	ErrInvalidValue  = errors.New("securecookie: invalid value")
)

type SecureCookie struct {
	gcm        cipher.AEAD
	cookieName string
	cookiePath string
	cookieTTL  time.Duration
}

func NewSecureCookie(hexKey, cookieName string) (*SecureCookie, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex key: %w", err)
	}
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes (AES-256)")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &SecureCookie{
		gcm:        gcm,
		cookieName: cookieName,
		cookiePath: "/",
		cookieTTL:  12 * time.Hour,
	}, nil
}

func (sc *SecureCookie) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sc.cookieName)
		var value map[string]string

		if err == nil {
			value, err = sc.decrypt(cookie.Value)
			if err != nil {
				log.Printf("WARN: Failed to decrypt cookie '%s': %v", sc.cookieName, err)
				http.SetCookie(w, sc.createExpiredCookie())
			}
		}

		ctx := context.WithValue(r.Context(), secureCookieContextKey, value)
		ctx = context.WithValue(ctx, secureCookieInstanceKey, sc)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (sc *SecureCookie) encrypt(value map[string]string) (string, error) {
	plaintext, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, sc.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := sc.gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

func (sc *SecureCookie) decrypt(encodedValue string) (map[string]string, error) {
	ciphertext, err := base64.URLEncoding.DecodeString(encodedValue)
	if err != nil {
		return nil, ErrInvalidValue
	}

	nonceSize := sc.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidValue
	}

	nonce, encryptedMessage := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := sc.gcm.Open(nil, nonce, encryptedMessage, nil)
	if err != nil {
		// the error indicates the cookie was tampered with or used a different key.
		return nil, ErrInvalidValue
	}

	var value map[string]string
	if err := json.Unmarshal(plaintext, &value); err != nil {
		return nil, ErrInvalidValue
	}

	return value, nil
}

func (sc *SecureCookie) createCookie(encodedValue string) *http.Cookie {
	return &http.Cookie{
		Name:     sc.cookieName,
		Value:    encodedValue,
		Path:     sc.cookiePath,
		Expires:  time.Now().Add(sc.cookieTTL),
		HttpOnly: true,
		Secure:   true, // Always true for secure cookies !!!
		SameSite: http.SameSiteLaxMode,
	}
}

func (sc *SecureCookie) createExpiredCookie() *http.Cookie {
	return &http.Cookie{
		Name:     sc.cookieName,
		Value:    "",
		Path:     sc.cookiePath,
		MaxAge:   -1, // We shgal intruct the browser to delete immediately !!!!!
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}

// Helpers
func Set(r *http.Request, w http.ResponseWriter, value map[string]string) error {
	sc, ok := r.Context().Value(secureCookieInstanceKey).(*SecureCookie)
	if !ok {
		return errors.New("securecookie: middleware not installed")
	}

	encoded, err := sc.encrypt(value)
	if err != nil {
		return err
	}
	http.SetCookie(w, sc.createCookie(encoded))
	return nil
}

func Get(r *http.Request) (map[string]string, error) {
	value, ok := r.Context().Value(secureCookieContextKey).(map[string]string)
	if !ok || value == nil {
		return nil, ErrValueNotFound
	}
	return value, nil
}

func Clear(r *http.Request, w http.ResponseWriter) error {
	sc, ok := r.Context().Value(secureCookieInstanceKey).(*SecureCookie)
	if !ok {
		return errors.New("securecookie: middleware not installed")
	}
	http.SetCookie(w, sc.createExpiredCookie())
	return nil
}
