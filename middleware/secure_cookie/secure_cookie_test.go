// middleware/securecookie_test.go
package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// Test key for AES-256
// This is a dummy key for testing purpose, do not reuse it !
const testHexKey = "a1b2c3d4e5f6a7b8c9d0a1b2c3d4e5f6a7b8c9d0a1b2c3d4e5f6a7b8c9d0a1b2"

func TestSecureCookieRoundTrip(t *testing.T) {
	sc, err := NewSecureCookie(testHexKey, "test-cookie")
	if err != nil {
		t.Fatalf("Failed to create SecureCookie: %v", err)
	}

	originalValue := map[string]string{"user": "gopher", "id": "42"}

	encoded, err := sc.encrypt(originalValue)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	if encoded == "" {
		t.Fatal("Encoded value is empty")
	}

	decoded, err := sc.decrypt(encoded)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if !reflect.DeepEqual(originalValue, decoded) {
		t.Errorf("Decoded value does not match original. got %v, want %v", decoded, originalValue)
	}
}

func TestTamperedCookie(t *testing.T) {
	sc, err := NewSecureCookie(testHexKey, "test-cookie")
	if err != nil {
		t.Fatalf("Failed to create SecureCookie: %v", err)
	}

	originalValue := map[string]string{"access": "granted"}
	encoded, err := sc.encrypt(originalValue)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	tampered := encoded + "tamper"

	_, err = sc.decrypt(tampered)
	if err == nil {
		t.Error("Expected an error when decrypting tampered cookie, but got nil")
	}
	if !errors.Is(err, ErrInvalidValue) {
		t.Errorf("Expected ErrInvalidValue, got %v", err)
	}
}

func TestMiddlewareIntegration(t *testing.T) {
	sc, err := NewSecureCookie(testHexKey, "integration-test")
	if err != nil {
		t.Fatalf("Failed to create SecureCookie: %v", err)
	}

	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := Get(r)
		if err == nil {
			expected := map[string]string{"hello": "world"}
			if !reflect.DeepEqual(data, expected) {
				t.Errorf("Got unexpected data from cookie: got %v, want %v", data, expected)
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		// Data not found, set it for the next request
		if errors.Is(err, ErrValueNotFound) {
			if err := Set(r, w, map[string]string{"hello": "world"}); err != nil {
				t.Errorf("Failed to set cookie: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
			return
		}

		http.Error(w, "Unexpected error getting cookie", http.StatusInternalServerError)
	})

	wrappedHandler := sc.Middleware(handler)

	req1 := httptest.NewRequest("GET", "/", nil)
	rr1 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusCreated {
		t.Fatalf("First request: expected status %d, got %d", http.StatusCreated, rr1.Code)
	}

	cookieHeader := rr1.Header().Get("Set-Cookie")
	if !strings.Contains(cookieHeader, "integration-test=") {
		t.Fatal("First request: cookie was not set in response")
	}

	req2 := httptest.NewRequest("GET", "/", nil)
	// Add the cookie from the first response to the second request
	req2.Header.Set("Cookie", cookieHeader)

	rr2 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("Second request: expected status %d, got %d", http.StatusOK, rr2.Code)
	}
}
