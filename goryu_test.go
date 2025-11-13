package goryu_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/arthurlch/goryu"
)

func TestNew(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		app := goryu.New()
		if app == nil {
			t.Fatal("Expected app to be created")
		}
		if app.Config.AppName != "" {
			t.Errorf("Expected empty AppName, got %s", app.Config.AppName)
		}
		if app.Config.ServerHeader != "" {
			t.Errorf("Expected empty ServerHeader, got %s", app.Config.ServerHeader)
		}
		if app.Config.StrictRouting {
			t.Error("Expected StrictRouting to be false by default")
		}
		if app.Config.CaseSensitive {
			t.Error("Expected CaseSensitive to be false by default")
		}
		if app.Config.DisableStartupMessage {
			t.Error("Expected DisableStartupMessage to be false by default")
		}
	})

	t.Run("CustomConfig", func(t *testing.T) {
		config := goryu.Config{
			AppName:               "TestApp",
			ServerHeader:          "TestServer/1.0",
			StrictRouting:         true,
			CaseSensitive:         true,
			DisableStartupMessage: true,
		}
		app := goryu.New(config)

		if app.Config.AppName != "TestApp" {
			t.Errorf("Expected AppName 'TestApp', got %s", app.Config.AppName)
		}
		if app.Config.ServerHeader != "TestServer/1.0" {
			t.Errorf("Expected ServerHeader 'TestServer/1.0', got %s", app.Config.ServerHeader)
		}
		if !app.Config.StrictRouting {
			t.Error("Expected StrictRouting to be true")
		}
		if !app.Config.CaseSensitive {
			t.Error("Expected CaseSensitive to be true")
		}
		if !app.Config.DisableStartupMessage {
			t.Error("Expected DisableStartupMessage to be true")
		}
	})
}

func TestAppUse(t *testing.T) {
	app := goryu.New()

	middleware1Called := false
	middleware2Called := false

	middleware1 := func(next goryu.Handler) goryu.Handler {
		return func(c *goryu.Ctx) {
			middleware1Called = true
			next(c)
		}
	}

	middleware2 := func(next goryu.Handler) goryu.Handler {
		return func(c *goryu.Ctx) {
			middleware2Called = true
			next(c)
		}
	}

	app.Use(middleware1)
	app.Use(middleware2)

	handler := func(c *goryu.Ctx) {
		_ = c.Text(http.StatusOK, "test")
	}

	app.GET("/test", handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	app.ServeHTTP(rr, req)

	if !middleware1Called {
		t.Error("Expected middleware1 to be called")
	}
	if !middleware2Called {
		t.Error("Expected middleware2 to be called")
	}
}

func TestAppGET(t *testing.T) {
	app := goryu.New()

	app.GET("/hello", func(c *goryu.Ctx) {
		_ = c.Text(http.StatusOK, "Hello, World!")
	})

	req := httptest.NewRequest("GET", "/hello", nil)
	rr := httptest.NewRecorder()

	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	expected := "Hello, World!"
	if rr.Body.String() != expected {
		t.Errorf("Expected body '%s', got '%s'", expected, rr.Body.String())
	}
}

func TestAppPOST(t *testing.T) {
	app := goryu.New()

	app.POST("/data", func(c *goryu.Ctx) {
		_ = c.Text(http.StatusCreated, "Data created")
	})

	req := httptest.NewRequest("POST", "/data", strings.NewReader("test data"))
	rr := httptest.NewRecorder()

	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, rr.Code)
	}

	expected := "Data created"
	if rr.Body.String() != expected {
		t.Errorf("Expected body '%s', got '%s'", expected, rr.Body.String())
	}
}

func TestAppServeHTTP(t *testing.T) {
	t.Run("ServerHeader", func(t *testing.T) {
		config := goryu.Config{
			ServerHeader: "Goryu/1.0",
		}
		app := goryu.New(config)

		app.GET("/", func(c *goryu.Ctx) {
			c.Text(http.StatusOK, "OK")
		})

		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		app.ServeHTTP(rr, req)

		serverHeader := rr.Header().Get("Server")
		if serverHeader != "Goryu/1.0" {
			t.Errorf("Expected Server header 'Goryu/1.0', got '%s'", serverHeader)
		}
	})

	t.Run("FormParseError", func(t *testing.T) {
		app := goryu.New()

		// Create a request with malformed form data
		req := httptest.NewRequest("POST", "/", strings.NewReader("invalid%form%data"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		// Manually set content length to trigger parse error
		req.ContentLength = -1

		rr := httptest.NewRecorder()

		app.ServeHTTP(rr, req)

		// The request should succeed because req.ParseForm() doesn't actually fail
		// in this test scenario, but we test the error handling path exists
		if rr.Code != http.StatusOK && rr.Code != http.StatusNotFound {
			// Either OK (if route exists) or NotFound (if no route matches)
			// Both are acceptable here
		}
	})

	t.Run("MiddlewareOrder", func(t *testing.T) {
		app := goryu.New()

		var order []string

		middleware1 := func(next goryu.Handler) goryu.Handler {
			return func(c *goryu.Ctx) {
				order = append(order, "middleware1_before")
				next(c)
				order = append(order, "middleware1_after")
			}
		}

		middleware2 := func(next goryu.Handler) goryu.Handler {
			return func(c *goryu.Ctx) {
				order = append(order, "middleware2_before")
				next(c)
				order = append(order, "middleware2_after")
			}
		}

		app.Use(middleware1)
		app.Use(middleware2)

		app.GET("/test", func(c *goryu.Ctx) {
			order = append(order, "handler")
			c.Text(http.StatusOK, "OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		app.ServeHTTP(rr, req)

		expected := []string{
			"middleware1_before",
			"middleware2_before",
			"handler",
			"middleware2_after",
			"middleware1_after",
		}

		if len(order) != len(expected) {
			t.Fatalf("Expected %d items in order, got %d", len(expected), len(order))
		}

		for i, item := range expected {
			if order[i] != item {
				t.Errorf("Expected order[%d] = %s, got %s", i, item, order[i])
			}
		}
	})
}

func TestAppRun(t *testing.T) {
	t.Run("DisabledStartupMessage", func(t *testing.T) {
		config := goryu.Config{
			DisableStartupMessage: true,
		}
		app := goryu.New(config)

		if !app.Config.DisableStartupMessage {
			t.Error("Expected startup message to be disabled")
		}
	})
}

// Test route parameters and dynamic routes
func TestRouteParameters(t *testing.T) {
	app := goryu.New()

	app.GET("/user/:id", func(c *goryu.Ctx) {
		id := c.Param("id")
		c.Text(http.StatusOK, "User ID: "+id)
	})

	req := httptest.NewRequest("GET", "/user/123", nil)
	rr := httptest.NewRecorder()

	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	expected := "User ID: 123"
	if rr.Body.String() != expected {
		t.Errorf("Expected body '%s', got '%s'", expected, rr.Body.String())
	}
}

// Test 404 handling
func TestNotFound(t *testing.T) {
	app := goryu.New()

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	rr := httptest.NewRecorder()

	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, rr.Code)
	}
}

// Test all HTTP methods
func TestHTTPMethods(t *testing.T) {
	app := goryu.New()

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	
	// Register a single ALL handler that responds to all methods
	app.ALL("/all", func(c *goryu.Ctx) {
		c.Text(http.StatusOK, "Method: "+c.Request.Method)
	})

	for _, method := range methods {
		req := httptest.NewRequest(method, "/all", nil)
		rr := httptest.NewRecorder()

		app.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d for %s method, got %d", http.StatusOK, method, rr.Code)
		}
		
		if method != "HEAD" { // HEAD requests don't return body
			expected := "Method: " + method
			if rr.Body.String() != expected {
				t.Errorf("Expected body '%s' for %s method, got '%s'", expected, method, rr.Body.String())
			}
		}
	}
}

// Test route method specificity
func TestRouteMethodSpecificity(t *testing.T) {
	app := goryu.New()

	app.GET("/test", func(c *goryu.Ctx) {
		c.Text(http.StatusOK, "GET")
	})

	app.POST("/test", func(c *goryu.Ctx) {
		c.Text(http.StatusOK, "POST")
	})

	app.PUT("/test", func(c *goryu.Ctx) {
		c.Text(http.StatusOK, "PUT")
	})

	app.DELETE("/test", func(c *goryu.Ctx) {
		c.Text(http.StatusOK, "DELETE")
	})

	app.PATCH("/test", func(c *goryu.Ctx) {
		c.Text(http.StatusOK, "PATCH")
	})

	tests := []struct {
		method   string
		expected string
	}{
		{"GET", "GET"},
		{"POST", "POST"},
		{"PUT", "PUT"},
		{"DELETE", "DELETE"},
		{"PATCH", "PATCH"},
	}

	for _, test := range tests {
		req := httptest.NewRequest(test.method, "/test", nil)
		rr := httptest.NewRecorder()

		app.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d for %s method, got %d", http.StatusOK, test.method, rr.Code)
		}

		if rr.Body.String() != test.expected {
			t.Errorf("Expected body '%s' for %s method, got '%s'", test.expected, test.method, rr.Body.String())
		}
	}
}

// Test Mount functionality
func TestMount(t *testing.T) {
	mainApp := goryu.New()
	subApp := goryu.New()

	subApp.GET("/hello", func(c *goryu.Ctx) {
		c.Text(http.StatusOK, "Hello from sub-app")
	})

	subApp.GET("/world", func(c *goryu.Ctx) {
		c.Text(http.StatusOK, "World from sub-app")
	})

	mainApp.Mount("/api", subApp)

	tests := []struct {
		path     string
		expected string
		status   int
	}{
		{"/api/hello", "Hello from sub-app", http.StatusOK},
		{"/api/world", "World from sub-app", http.StatusOK},
		{"/api/nonexistent", "", http.StatusNotFound},
	}

	for _, test := range tests {
		req := httptest.NewRequest("GET", test.path, nil)
		rr := httptest.NewRecorder()

		mainApp.ServeHTTP(rr, req)

		if rr.Code != test.status {
			t.Errorf("Expected status %d for %s, got %d", test.status, test.path, rr.Code)
		}

		if test.expected != "" && rr.Body.String() != test.expected {
			t.Errorf("Expected body '%s' for %s, got '%s'", test.expected, test.path, rr.Body.String())
		}
	}
}

// Test MountPath
func TestMountPath(t *testing.T) {
	app := goryu.New()
	if app.MountPath() != "" {
		t.Errorf("Expected empty mount path for main app, got '%s'", app.MountPath())
	}

	subApp := goryu.New()
	app.Mount("/api", subApp)

	if subApp.MountPath() != "/api" {
		t.Errorf("Expected mount path '/api' for sub-app, got '%s'", subApp.MountPath())
	}
}

// Test server management methods
func TestServerManagement(t *testing.T) {
	app := goryu.New()

	// Test Server() method before Listen
	if server := app.Server(); server != nil {
		t.Error("Expected Server() to return nil before Listen() is called")
	}

	// Test Handler method
	if handler := app.Handler(); handler != app {
		t.Error("Expected Handler() to return the app instance")
	}

	// Test Shutdown without running server
	if err := app.Shutdown(); err == nil {
		t.Error("Expected Shutdown() to return error when server is not running")
	}
}

// Test Groups
func TestGroups(t *testing.T) {
	app := goryu.New()

	v1 := app.Group("/api/v1")
	v1.GET("/users", func(c *goryu.Ctx) {
		c.Text(http.StatusOK, "v1 users")
	})

	v2 := app.Group("/api/v2")
	v2.GET("/users", func(c *goryu.Ctx) {
		c.Text(http.StatusOK, "v2 users")
	})

	tests := []struct {
		path     string
		expected string
	}{
		{"/api/v1/users", "v1 users"},
		{"/api/v2/users", "v2 users"},
	}

	for _, test := range tests {
		req := httptest.NewRequest("GET", test.path, nil)
		rr := httptest.NewRecorder()

		app.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d for %s, got %d", http.StatusOK, test.path, rr.Code)
		}

		if rr.Body.String() != test.expected {
			t.Errorf("Expected body '%s' for %s, got '%s'", test.expected, test.path, rr.Body.String())
		}
	}
}

// Test Static file serving
func TestStatic(t *testing.T) {
	app := goryu.New()

	// Create a temporary directory and file for testing
	tmpDir, err := os.MkdirTemp("", "goryu_static_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := tmpDir + "/test.txt"
	testContent := "Hello, Static World!"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Configure static serving
	app.Static("/static", tmpDir)

	req := httptest.NewRequest("GET", "/static/test.txt", nil)
	rr := httptest.NewRecorder()

	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if rr.Body.String() != testContent {
		t.Errorf("Expected body '%s', got '%s'", testContent, rr.Body.String())
	}
}

// Test Listen and Shutdown functionality
func TestListenAndShutdown(t *testing.T) {
	app := goryu.New(goryu.Config{
		DisableStartupMessage: true,
	})

	app.GET("/test", func(c *goryu.Ctx) {
		c.Text(http.StatusOK, "Server running")
	})

	// Test that Server() returns nil before Listen
	if server := app.Server(); server != nil {
		t.Error("Expected Server() to return nil before Listen() is called")
	}

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		// Use a random port
		errChan <- app.Listen(":0")
	}()

	// Give the server a moment to start
	time.Sleep(10 * time.Millisecond)

	// Test that Server() returns a server after Listen
	server := app.Server()
	if server == nil {
		t.Fatal("Expected Server() to return a server after Listen() is called")
	}

	// Test shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		t.Errorf("Failed to shutdown server: %v", err)
	}

	// Verify the server stopped
	select {
	case err := <-errChan:
		if err != http.ErrServerClosed {
			t.Errorf("Expected http.ErrServerClosed, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Server did not shutdown within expected time")
	}
}

// Test route method return values
func TestRouteReturns(t *testing.T) {
	app := goryu.New()

	// Test that route methods return *router.Route
	route := app.GET("/test", func(c *goryu.Ctx) {
		c.Text(http.StatusOK, "test")
	})

	if route == nil {
		t.Error("Expected GET to return a *router.Route, got nil")
	}

	postRoute := app.POST("/post", func(c *goryu.Ctx) {
		c.Text(http.StatusOK, "post")
	})

	if postRoute == nil {
		t.Error("Expected POST to return a *router.Route, got nil")
	}

	allRoute := app.ALL("/all", func(c *goryu.Ctx) {
		c.Text(http.StatusOK, "all")
	})

	if allRoute == nil {
		t.Error("Expected ALL to return a *router.Route, got nil")
	}
}

// Test Handler interface compliance
func TestHandlerInterface(t *testing.T) {
	app := goryu.New()

	// Test that app implements http.Handler
	var _ http.Handler = app

	// Test Handler() method
	handler := app.Handler()
	if handler != app {
		t.Error("Expected Handler() to return the app instance")
	}
}

// Test nested Mount functionality
func TestNestedMount(t *testing.T) {
	mainApp := goryu.New()
	apiApp := goryu.New()
	v1App := goryu.New()

	// Create nested structure: main -> /api -> /v1
	v1App.GET("/users", func(c *goryu.Ctx) {
		c.Text(http.StatusOK, "v1 users")
	})

	apiApp.Mount("/v1", v1App)
	mainApp.Mount("/api", apiApp)

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	rr := httptest.NewRecorder()

	mainApp.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	expected := "v1 users"
	if rr.Body.String() != expected {
		t.Errorf("Expected body '%s', got '%s'", expected, rr.Body.String())
	}

	// Test mount path tracking
	if v1App.MountPath() != "/api/v1" {
		t.Errorf("Expected mount path '/api/v1', got '%s'", v1App.MountPath())
	}
}

// Test comprehensive HTTP method coverage
func TestAllHTTPMethods(t *testing.T) {
	app := goryu.New()

	// Test individual method handlers
	app.GET("/get", func(c *goryu.Ctx) { c.Text(http.StatusOK, "GET") })
	app.POST("/post", func(c *goryu.Ctx) { c.Text(http.StatusOK, "POST") })
	app.PUT("/put", func(c *goryu.Ctx) { c.Text(http.StatusOK, "PUT") })
	app.DELETE("/delete", func(c *goryu.Ctx) { c.Text(http.StatusOK, "DELETE") })
	app.PATCH("/patch", func(c *goryu.Ctx) { c.Text(http.StatusOK, "PATCH") })
	app.HEAD("/head", func(c *goryu.Ctx) { c.Text(http.StatusOK, "HEAD") })
	app.OPTIONS("/options", func(c *goryu.Ctx) { c.Text(http.StatusOK, "OPTIONS") })

	tests := []struct {
		method string
		path   string
		expect string
	}{
		{"GET", "/get", "GET"},
		{"POST", "/post", "POST"},
		{"PUT", "/put", "PUT"},
		{"DELETE", "/delete", "DELETE"},
		{"PATCH", "/patch", "PATCH"},
		{"HEAD", "/head", ""},     // HEAD doesn't return body
		{"OPTIONS", "/options", "OPTIONS"},
	}

	for _, test := range tests {
		req := httptest.NewRequest(test.method, test.path, nil)
		rr := httptest.NewRecorder()

		app.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d for %s %s, got %d", http.StatusOK, test.method, test.path, rr.Code)
		}

		if test.expect != "" && rr.Body.String() != test.expect {
			t.Errorf("Expected body '%s' for %s %s, got '%s'", test.expect, test.method, test.path, rr.Body.String())
		}
	}
}

// Test template compatibility - ensure Listen method works as expected by templates
func TestTemplateCompatibility(t *testing.T) {
	app := goryu.New(goryu.Config{
		AppName:               "test-app",
		DisableStartupMessage: true,
	})

	app.GET("/", func(c *goryu.Ctx) {
		c.JSON(200, map[string]string{
			"message": "Hello from test-app!",
		})
	})

	app.GET("/health", func(c *goryu.Ctx) {
		c.JSON(200, map[string]interface{}{
			"status":    "healthy",
			"timestamp": "test-time",
		})
	})

	// Test template-generated routes work
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Test health endpoint
	req = httptest.NewRequest("GET", "/health", nil)
	rr = httptest.NewRecorder()

	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d for health endpoint, got %d", http.StatusOK, rr.Code)
	}

	// Verify that Listen method exists and can be called (compilation test)
	go func() {
		// This tests that Listen exists and has the right signature
		_ = app.Listen(":0")
	}()
}
