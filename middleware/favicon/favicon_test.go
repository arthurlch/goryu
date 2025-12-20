package favicon_test
import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/favicon"
)
func newTestContext(req *http.Request) (*context.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}
func TestFaviconMiddleware(t *testing.T) {
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
		if err := os.WriteFile(faviconFile, faviconData, 0644); err != nil {
			t.Fatalf("Could not create test favicon file: %v", err)
		}
		middleware := favicon.New(favicon.Config{
			File: faviconFile,
			CacheFile: true,
		})
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
		if string(body) != string(faviconData) {
			t.Errorf("Expected favicon data, got %s", string(body))
		}
		contentType := rr.Header().Get("Content-Type")
		if contentType != "image/x-icon" {
			t.Errorf("Expected content-type 'image/x-icon', got %s", contentType)
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