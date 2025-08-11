package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFilesystem(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "index.html"), []byte("<html></html>"), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tempDir, "css"), 0700); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "css", "style.css"), []byte("body {}"), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound) // Simulate a final 404 handler
	})

	t.Run("serves existing file", func(t *testing.T) {
		t.Parallel()
		// FIX: Use the tempDir string directly.
		config := FilesystemConfig{Root: tempDir}
		handler := Filesystem(config)(nextHandler)

		req := httptest.NewRequest("GET", "/index.html", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		if body := rr.Body.String(); body != "<html></html>" {
			t.Errorf("handler returned wrong body: got %q want %q", body, "<html></html>")
		}
	})

	t.Run("serves file with path prefix", func(t *testing.T) {
		t.Parallel()
		// FIX: Use the tempDir string directly.
		config := FilesystemConfig{Root: tempDir, PathPrefix: "/static"}
		handler := Filesystem(config)(nextHandler)

		req := httptest.NewRequest("GET", "/static/css/style.css", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		if body := rr.Body.String(); body != "body {}" {
			t.Errorf("handler returned wrong body: got %q want %q", body, "body {}")
		}
	})

	t.Run("passthrough for non-existent file", func(t *testing.T) {
		t.Parallel()
		// FIX: Use the tempDir string directly.
		config := FilesystemConfig{Root: tempDir}
		handler := Filesystem(config)(nextHandler)

		req := httptest.NewRequest("GET", "/not-found.txt", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler did not pass through: got %v want %v", status, http.StatusNotFound)
		}
	})

	t.Run("passthrough for directory request", func(t *testing.T) {
		t.Parallel()
		// FIX: Use the tempDir string directly.
		config := FilesystemConfig{Root: tempDir}
		handler := Filesystem(config)(nextHandler)

		req := httptest.NewRequest("GET", "/css/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler served directory or did not pass through: got %v want %v", status, http.StatusNotFound)
		}
	})
}
