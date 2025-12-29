package favicon_test

import (
	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/favicon"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func newTestContext(req *http.Request) (*context.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}
func TestFaviconMiddleware(t *testing.T) {
	// Helper function to check if middleware creation failed
	isErrorMiddleware := func(middleware func(next context.HandlerFunc) context.HandlerFunc) bool {
		testCalled := false
		handler := func(c *context.Context) {
			testCalled = true
			c.Text(http.StatusOK, "test")
		}
		req := httptest.NewRequest("GET", "/favicon.ico", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		// If status is 500 and handler wasn't called, it's the error middleware
		return rr.Code >= 500 && !testCalled
	}
	t.Run("DefaultConfigNoFile", func(t *testing.T) {
		middleware := favicon.New(favicon.Config{})
		handler := func(c *context.Context) {
			c.Text(http.StatusOK, "OK")
		}
		req := httptest.NewRequest("GET", "/favicon.ico", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", rr.Code)
		}
	})
	t.Run("NonFaviconRequest", func(t *testing.T) {
		middleware := favicon.New(favicon.Config{})
		handler := func(c *context.Context) {
			c.Text(http.StatusOK, "Homepage")
		}
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		if rr.Body.String() != "Homepage" {
			t.Errorf("Expected 'Homepage', got %s", rr.Body.String())
		}
	})
	t.Run("WithFaviconFile", func(t *testing.T) {
		tempDir := t.TempDir()
		faviconFile := filepath.Join(tempDir, "favicon.ico")
		faviconData := []byte("fake favicon data")

		// Write file with explicit close and sync
		f, err := os.Create(faviconFile)
		if err != nil {
			t.Fatalf("Could not create test favicon file: %v", err)
		}
		if _, err := f.Write(faviconData); err != nil {
			f.Close()
			t.Fatalf("Could not write to test favicon file: %v", err)
		}
		if err := f.Sync(); err != nil {
			f.Close()
			t.Fatalf("Could not sync test favicon file: %v", err)
		}
		if err := f.Close(); err != nil {
			t.Fatalf("Could not close test favicon file: %v", err)
		}

		// Verify file was created and is readable
		if data, err := os.ReadFile(faviconFile); err != nil {
			t.Fatalf("Cannot read favicon file after creation: %v", err)
		} else if string(data) != string(faviconData) {
			t.Fatalf("Favicon file content mismatch: expected %q, got %q", faviconData, data)
		}

		// Create middleware after file is confirmed to exist and readable
		middleware := favicon.New(favicon.Config{
			File:      faviconFile,
			CacheFile: true,
		})

		// Check if middleware creation failed
		if isErrorMiddleware(middleware) {
			// Try to understand why it failed
			if _, err := os.Stat(faviconFile); err != nil {
				t.Fatalf("File disappeared after middleware creation: %v", err)
			}
			if data, err := os.ReadFile(faviconFile); err != nil {
				t.Fatalf("Cannot read file after middleware creation: %v", err)
			} else {
				t.Fatalf("Middleware creation failed despite file being readable. File content: %q", data)
			}
		}

		handler := func(c *context.Context) {
			c.Text(http.StatusOK, "Should not reach here")
		}
		req := httptest.NewRequest("GET", "/favicon.ico", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response body: %q, Headers: %v", rr.Code, rr.Body.String(), rr.Header())
		}

		body := rr.Body.Bytes()
		if string(body) != string(faviconData) {
			t.Errorf("Expected favicon data %q, got %q", faviconData, body)
		}

		contentType := rr.Header().Get("Content-Type")
		if contentType != "image/x-icon" {
			t.Errorf("Expected content-type 'image/x-icon', got %q", contentType)
		}
	})
	t.Run("WithFaviconFileNoCache", func(t *testing.T) {
		tempDir := t.TempDir()
		faviconFile := filepath.Join(tempDir, "favicon.ico")
		faviconData := []byte("fake favicon data without cache")
		if err := os.WriteFile(faviconFile, faviconData, 0644); err != nil {
			t.Fatalf("Could not create test favicon file: %v", err)
		}

		// Create middleware without caching
		middleware := favicon.New(favicon.Config{
			File:      faviconFile,
			CacheFile: false, // Don't cache, serve directly
		})

		handler := func(c *context.Context) {
			c.Text(http.StatusOK, "Should not reach here")
		}
		req := httptest.NewRequest("GET", "/favicon.ico", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response body: %q", rr.Code, rr.Body.String())
		}

		body := rr.Body.Bytes()
		if string(body) != string(faviconData) {
			t.Errorf("Expected favicon data %q, got %q", faviconData, body)
		}
	})
	t.Run("CustomURL", func(t *testing.T) {
		middleware := favicon.New(favicon.Config{
			URL: "/custom-favicon.ico",
		})
		handler := func(c *context.Context) {
			c.Text(http.StatusOK, "OK")
		}
		req := httptest.NewRequest("GET", "/custom-favicon.ico", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", rr.Code)
		}
		req2 := httptest.NewRequest("GET", "/favicon.ico", nil)
		ctx2, rr2 := newTestContext(req2)
		middleware(handler)(ctx2)
		if rr2.Code != http.StatusOK {
			t.Errorf("Expected status 200 for regular request, got %d", rr2.Code)
		}
	})
	t.Run("WithFile Helper", func(t *testing.T) {
		tempDir := t.TempDir()
		pngFile := filepath.Join(tempDir, "favicon.png")
		pngData := []byte("fake png data")
		if err := os.WriteFile(pngFile, pngData, 0644); err != nil {
			t.Fatalf("Could not create test PNG file: %v", err)
		}
		middleware := favicon.WithFile(pngFile)
		handler := func(c *context.Context) {
			c.Text(http.StatusOK, "Should not reach here")
		}
		req := httptest.NewRequest("GET", "/favicon.ico", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		body := rr.Body.Bytes()
		if string(body) != string(pngData) {
			t.Errorf("Expected PNG data, got %s", string(body))
		}
		contentType := rr.Header().Get("Content-Type")
		if contentType != "image/png" {
			t.Errorf("Expected content-type 'image/png', got %s", contentType)
		}
	})
}
