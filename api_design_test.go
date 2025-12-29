package goryu

// honestly it is too embeded inside main pack and api related test and design will be stay inside main folder

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTypeAliases(t *testing.T) {
	stdCtx := context.Background()
	if stdCtx == nil {
		t.Error("Standard context should work")
	}

	app := New()

	var handler Handler = func(c *Ctx) {
		c.Text(200, "test")
	}

	var middleware Middleware = func(next Handler) Handler {
		return func(c *Ctx) {
			c.SetHeader("X-Test", "middleware")
			next(c)
		}
	}

	app.Use(middleware)
	app.GET("/test", handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	app.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	if rr.Body.String() != "test" {
		t.Errorf("Expected 'test', got '%s'", rr.Body.String())
	}

	if rr.Header().Get("X-Test") != "middleware" {
		t.Error("Middleware should have set X-Test header")
	}
}

func TestStaticFileConfiguration(t *testing.T) {
	// Create a temporary directory with test files
	tempDir, err := os.MkdirTemp("", "goryu_static_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	indexContent := `<!DOCTYPE html><html><head><title>Test Index</title></head><body><h1>Index Page</h1></body></html>`
	err = os.WriteFile(filepath.Join(tempDir, "index.html"), []byte(indexContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	testContent := "This is a test file"
	err = os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(subDir, "sub.txt"), []byte("sub file"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("DefaultConfiguration", func(t *testing.T) {
		app := New()
		app.Static("/static", tempDir)

		// Test serving index file
		req := httptest.NewRequest("GET", "/static/", nil)
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)

		if rr.Code != 200 {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		if !strings.Contains(rr.Body.String(), "Index Page") {
			t.Error("Should serve index.html by default")
		}

		// Check cache headers are set
		if rr.Header().Get("Cache-Control") == "" {
			t.Error("Should set Cache-Control header by default")
		}
	})

	t.Run("CustomIndexFile", func(t *testing.T) {
		// Create custom index
		customIndex := `<h1>Custom Index</h1>`
		err = os.WriteFile(filepath.Join(tempDir, "home.html"), []byte(customIndex), 0644)
		if err != nil {
			t.Fatal(err)
		}

		app := New()
		app.Static("/static", tempDir, StaticConfig{
			Index: "home.html",
		})

		req := httptest.NewRequest("GET", "/static/", nil)
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)

		if rr.Code != 200 {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		if !strings.Contains(rr.Body.String(), "Custom Index") {
			t.Error("Should serve custom index file")
		}
	})

	t.Run("DirectoryBrowsing", func(t *testing.T) {
		app := New()
		app.Static("/browse", tempDir, StaticConfig{
			Browse: true,
		})

		req := httptest.NewRequest("GET", "/browse/", nil)
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)

		if rr.Code != 200 {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		body := rr.Body.String()
		if !strings.Contains(body, "Directory listing") {
			t.Error("Should show directory listing")
		}

		if !strings.Contains(body, "test.txt") {
			t.Error("Should list test.txt file")
		}

		if !strings.Contains(body, "subdir/") {
			t.Error("Should list subdir directory")
		}
	})

	t.Run("DisableBrowsingWithoutIndex", func(t *testing.T) {
		// Create directory without index
		noIndexDir := filepath.Join(tempDir, "noindex")
		err = os.Mkdir(noIndexDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		app := New()
		app.Static("/noindex", noIndexDir, StaticConfig{
			Browse: false, // Browsing disabled
		})

		req := httptest.NewRequest("GET", "/noindex/", nil)
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)

		if rr.Code != 403 {
			t.Errorf("Expected status 403, got %d", rr.Code)
		}

		if !strings.Contains(rr.Body.String(), "Directory browsing is disabled") {
			t.Error("Should show browsing disabled message")
		}
	})

	t.Run("CacheConfiguration", func(t *testing.T) {
		app := New()
		app.Static("/cached", tempDir, StaticConfig{
			CacheDuration: 2 * time.Hour,
			MaxAge:        7200, // 2 hours
		})

		req := httptest.NewRequest("GET", "/cached/test.txt", nil)
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)

		if rr.Code != 200 {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		cacheControl := rr.Header().Get("Cache-Control")
		if !strings.Contains(cacheControl, "max-age=7200") {
			t.Errorf("Expected max-age=7200 in Cache-Control, got: %s", cacheControl)
		}

		if rr.Header().Get("Expires") == "" {
			t.Error("Should set Expires header")
		}
	})

	t.Run("FileNotFound", func(t *testing.T) {
		app := New()
		app.Static("/static", tempDir)

		req := httptest.NewRequest("GET", "/static/nonexistent.txt", nil)
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)

		if rr.Code != 404 {
			t.Errorf("Expected status 404, got %d", rr.Code)
		}

		if !strings.Contains(rr.Body.String(), "File Not Found") {
			t.Error("Should show file not found message")
		}
	})
}

func TestMountFunctionality(t *testing.T) {
	t.Run("BasicMounting", func(t *testing.T) {
		mainApp := New()
		subApp := New()

		subApp.GET("/hello", func(c *Ctx) {
			c.Text(200, "Hello from sub-app")
		})

		mainApp.Mount("/api", subApp)

		req := httptest.NewRequest("GET", "/api/hello", nil)
		rr := httptest.NewRecorder()

		mainApp.ServeHTTP(rr, req)

		if rr.Code != 200 {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		if rr.Body.String() != "Hello from sub-app" {
			t.Errorf("Expected 'Hello from sub-app', got '%s'", rr.Body.String())
		}
	})

	t.Run("MountWithMiddleware", func(t *testing.T) {
		mainApp := New()
		subApp := New()

		// Main app middleware
		mainApp.Use(func(next Handler) Handler {
			return func(c *Ctx) {
				c.SetHeader("X-Main", "main-middleware")
				next(c)
			}
		})

		// Sub app middleware
		subApp.Use(func(next Handler) Handler {
			return func(c *Ctx) {
				c.SetHeader("X-Sub", "sub-middleware")
				next(c)
			}
		})

		subApp.GET("/test", func(c *Ctx) {
			// Access mount information
			originalPath, prefix, subPath, isMounted := GetMountInfo(c)
			if !isMounted {
				c.Text(200, "Mount info not available")
				return
			}

			c.Text(200, "Mount info available")
			c.SetHeader("X-Original-Path", originalPath)
			c.SetHeader("X-Prefix", prefix)
			c.SetHeader("X-Sub-Path", subPath)
		})

		mainApp.Mount("/mounted", subApp)

		req := httptest.NewRequest("GET", "/mounted/test", nil)
		rr := httptest.NewRecorder()

		mainApp.ServeHTTP(rr, req)

		if rr.Code != 200 {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		// Check that main app middleware was applied
		if rr.Header().Get("X-Main") != "main-middleware" {
			t.Error("Main app middleware should be applied")
		}

		// Check that mount info is NOT available in main context
		// (because we create a new context for sub-app)
		if strings.Contains(rr.Body.String(), "Mount info available") {
			t.Error("Mount info should not be available in sub-app context")
		}
	})

	t.Run("NestedMounting", func(t *testing.T) {
		mainApp := New()
		apiApp := New()
		v1App := New()

		v1App.GET("/users", func(c *Ctx) {
			c.Text(200, "API v1 users")
		})

		apiApp.Mount("/v1", v1App)
		mainApp.Mount("/api", apiApp)

		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		rr := httptest.NewRecorder()

		mainApp.ServeHTTP(rr, req)

		if rr.Code != 200 {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		if rr.Body.String() != "API v1 users" {
			t.Errorf("Expected 'API v1 users', got '%s'", rr.Body.String())
		}
	})

	t.Run("MountPathTracking", func(t *testing.T) {
		mainApp := New()
		subApp := New()

		mainApp.Mount("/sub", subApp)

		if subApp.MountPath() != "/sub" {
			t.Errorf("Expected mount path '/sub', got '%s'", subApp.MountPath())
		}
	})
}

func TestMountPathPreservation(t *testing.T) {
	// Test that mounting doesn't interfere with middleware that depends on original path
	mainApp := New()
	subApp := New()

	var capturedPath string

	// Middleware that captures the request path
	mainApp.Use(func(next Handler) Handler {
		return func(c *Ctx) {
			capturedPath = c.Request.URL.Path
			next(c)
		}
	})

	subApp.GET("/endpoint", func(c *Ctx) {
		c.Text(200, "sub endpoint")
	})

	mainApp.Mount("/api", subApp)

	req := httptest.NewRequest("GET", "/api/endpoint", nil)
	rr := httptest.NewRecorder()

	mainApp.ServeHTTP(rr, req)

	// The middleware should see the original path
	if capturedPath != "/api/endpoint" {
		t.Errorf("Middleware should see original path '/api/endpoint', got '%s'", capturedPath)
	}

	if rr.Code != 200 {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}
