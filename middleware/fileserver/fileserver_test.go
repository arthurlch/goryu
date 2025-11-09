package fileserver_test
import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/fileserver"
)
func newTestContext(req *http.Request) (*context.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}
func TestFileserverMiddleware(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "index.html"), []byte("<html>Index</html>"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tempDir, "css"), 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "css", "style.css"), []byte("body {}"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	t.Run("ServesExistingFile", func(t *testing.T) {
		config := fileserver.Config{Root: tempDir}
		middleware := fileserver.New(config)
		handler := func(c *context.Context) {
			c.Status(http.StatusNotFound).Text(http.StatusNotFound, "Not found")
		}
		req := httptest.NewRequest("GET", "/test.txt", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		if rr.Body.String() != "test content" {
			t.Errorf("Expected 'test content', got %s", rr.Body.String())
		}
	})
	t.Run("WithPathPrefix", func(t *testing.T) {
		config := fileserver.Config{Root: tempDir, PathPrefix: "/static/"}
		middleware := fileserver.New(config)
		handler := func(c *context.Context) {
			c.Status(http.StatusNotFound).Text(http.StatusNotFound, "Not found")
		}
		req := httptest.NewRequest("GET", "/static/css/style.css", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		if rr.Body.String() != "body {}" {
			t.Errorf("Expected 'body {}', got %s", rr.Body.String())
		}
	})
	t.Run("NonExistentFile", func(t *testing.T) {
		config := fileserver.Config{Root: tempDir}
		middleware := fileserver.New(config)
		handler := func(c *context.Context) {
			c.Status(http.StatusNotFound).Text(http.StatusNotFound, "Not found")
		}
		req := httptest.NewRequest("GET", "/nonexistent.txt", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rr.Code)
		}
		if rr.Body.String() != "Not found" {
			t.Errorf("Expected 'Not found', got %s", rr.Body.String())
		}
	})
	t.Run("DirectoryWithIndex", func(t *testing.T) {
		config := fileserver.Config{Root: tempDir}
		middleware := fileserver.New(config)
		handler := func(c *context.Context) {
			c.Status(http.StatusNotFound).Text(http.StatusNotFound, "Not found")
		}
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 for index file, got %d", rr.Code)
		}
		if rr.Body.String() != "<html>Index</html>" {
			t.Errorf("Expected index content, got %s", rr.Body.String())
		}
	})
	t.Run("DirectoryWithoutBrowse", func(t *testing.T) {
		config := fileserver.Config{Root: tempDir, Browse: false}
		middleware := fileserver.New(config)
		handler := func(c *context.Context) {
			c.Status(http.StatusNotFound).Text(http.StatusNotFound, "Not found")
		}
		req := httptest.NewRequest("GET", "/css/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for directory without browse, got %d", rr.Code)
		}
	})
	t.Run("WithRootHelper", func(t *testing.T) {
		middleware := fileserver.WithRoot(tempDir)
		handler := func(c *context.Context) {
			c.Status(http.StatusNotFound).Text(http.StatusNotFound, "Not found")
		}
		req := httptest.NewRequest("GET", "/test.txt", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})
	t.Run("WithPrefixHelper", func(t *testing.T) {
		middleware := fileserver.WithPrefix("/assets/", tempDir)
		handler := func(c *context.Context) {
			c.Status(http.StatusNotFound).Text(http.StatusNotFound, "Not found")
		}
		req := httptest.NewRequest("GET", "/assets/test.txt", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})
	t.Run("SecurityPathTraversal", func(t *testing.T) {
		config := fileserver.Config{Root: tempDir}
		middleware := fileserver.New(config)
		handler := func(c *context.Context) {
			c.Status(http.StatusNotFound).Text(http.StatusNotFound, "Not found")
		}
		req := httptest.NewRequest("GET", "/../../../etc/passwd", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusForbidden {
			t.Errorf("Expected status 403 for path traversal, got %d", rr.Code)
		}
	})
	t.Run("NonMatchingPrefix", func(t *testing.T) {
		config := fileserver.Config{Root: tempDir, PathPrefix: "/static/"}
		middleware := fileserver.New(config)
		handler := func(c *context.Context) {
			c.Status(http.StatusNotFound).Text(http.StatusNotFound, "Not found")
		}
		req := httptest.NewRequest("GET", "/api/test", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for non-matching prefix, got %d", rr.Code)
		}
		if rr.Body.String() != "Not found" {
			t.Errorf("Expected 'Not found', got %s", rr.Body.String())
		}
	})
}