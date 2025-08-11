package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFavicon(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	t.Run("ignore mode", func(t *testing.T) {
		t.Parallel()
		config := FaviconConfig{File: ""}
		handler := Favicon(config)(nextHandler)

		req := httptest.NewRequest("GET", "/favicon.ico", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNoContent {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
		}
	})

	t.Run("cache mode", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		dummyIconPath := filepath.Join(tempDir, "test.ico")
		dummyIconData := []byte("dummy icon data")
		if err := os.WriteFile(dummyIconPath, dummyIconData, 0600); err != nil {
			t.Fatalf("Failed to write dummy icon file: %v", err)
		}

		config := FaviconConfig{File: dummyIconPath, Cache: true}
		handler := Favicon(config)(nextHandler)

		req := httptest.NewRequest("GET", "/favicon.ico", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		if contentType := rr.Header().Get("Content-Type"); contentType != "image/x-icon" {
			t.Errorf("wrong content type: got %q want %q", contentType, "image/x-icon")
		}
		if !bytes.Equal(rr.Body.Bytes(), dummyIconData) {
			t.Errorf("body does not match icon data: got %q want %q", rr.Body.String(), string(dummyIconData))
		}
	})

	t.Run("passthrough on other path", func(t *testing.T) {
		t.Parallel()
		config := FaviconConfig{File: ""}
		handler := Favicon(config)(nextHandler)

		req := httptest.NewRequest("GET", "/not-a-favicon", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusTeapot {
			t.Errorf("handler did not pass through request: got status %v want %v", status, http.StatusTeapot)
		}
	})
}
