package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	context "github.com/arthurlch/goryu/goryuctx"
)

// Test basic router creation and default handlers
func TestNew(t *testing.T) {
	r := New()
	
	if r == nil {
		t.Fatal("Expected router to be created")
	}
	
	if r.trees == nil {
		t.Error("Expected trees map to be initialized")
	}
	
	if r.namedRoutes == nil {
		t.Error("Expected namedRoutes map to be initialized")
	}
	
	if r.NotFound == nil {
		t.Error("Expected NotFound handler to be set")
	}
	
	if r.MethodNotAllowed == nil {
		t.Error("Expected MethodNotAllowed handler to be set")
	}
	
	if r.PanicHandler == nil {
		t.Error("Expected PanicHandler to be set")
	}
}

// Test basic routing with different HTTP methods
func TestBasicRouting(t *testing.T) {
	r := New()
	
	r.GET("/", func(c *context.Context) {
		c.Text(200, "home")
	})
	
	r.GET("/users", func(c *context.Context) {
		c.Text(200, "users")
	})
	
	r.POST("/users", func(c *context.Context) {
		c.Text(201, "create user")
	})
	
	r.PUT("/users/123", func(c *context.Context) {
		c.Text(200, "update user")
	})
	
	r.DELETE("/users/123", func(c *context.Context) {
		c.Text(200, "delete user")
	})

	tests := []struct {
		method   string
		path     string
		expected string
		status   int
	}{
		{"GET", "/", "home", 200},
		{"GET", "/users", "users", 200},
		{"POST", "/users", "create user", 201},
		{"PUT", "/users/123", "update user", 200},
		{"DELETE", "/users/123", "delete user", 200},
		{"GET", "/notfound", "", 404},
		{"PATCH", "/users", "", 405}, // Method not allowed
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != tt.status {
			t.Errorf("Expected status %d, got %d for %s %s", tt.status, w.Code, tt.method, tt.path)
		}

		if tt.expected != "" && w.Body.String() != tt.expected {
			t.Errorf("Expected body %q, got %q for %s %s", tt.expected, w.Body.String(), tt.method, tt.path)
		}
	}
}

// Test ALL method that registers all HTTP methods
func TestALLMethod(t *testing.T) {
	r := New()
	
	r.ALL("/api", func(c *context.Context) {
		c.Text(200, "all methods: "+c.Request.Method)
	})

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	
	for _, method := range methods {
		req := httptest.NewRequest(method, "/api", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200 for %s, got %d", method, w.Code)
		}
		
		if method != "HEAD" { // HEAD requests don't return body
			expected := "all methods: " + method
			if w.Body.String() != expected {
				t.Errorf("Expected body %q for %s, got %q", expected, method, w.Body.String())
			}
		}
	}
}

// Test parameter routes
func TestParameterRoutes(t *testing.T) {
	r := New()

	r.GET("/users/:id", func(c *context.Context) {
		c.Text(200, "user:"+c.Param("id"))
	})

	r.GET("/posts/:year/:month/:day", func(c *context.Context) {
		year := c.Param("year")
		month := c.Param("month")
		day := c.Param("day")
		c.Text(200, year+"-"+month+"-"+day)
	})

	tests := []struct {
		path     string
		expected string
	}{
		{"/users/123", "user:123"},
		{"/users/john", "user:john"},
		{"/posts/2024/01/15", "2024-01-15"},
		{"/posts/2023/12/31", "2023-12-31"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200 for %s, got %d", tt.path, w.Code)
		}

		if w.Body.String() != tt.expected {
			t.Errorf("Expected %q, got %q for %s", tt.expected, w.Body.String(), tt.path)
		}
	}
}

// Test wildcard routes
func TestWildcardRoutes(t *testing.T) {
	r := New()

	r.GET("/static/*filepath", func(c *context.Context) {
		filepath := c.Param("filepath")
		c.Text(200, "static:"+filepath)
	})

	tests := []struct {
		path     string
		expected string
	}{
		{"/static/", "static:"},                                    // Empty wildcard match
		{"/static/css/style.css", "static:css/style.css"},
		{"/static/js/app.js", "static:js/app.js"},
		{"/static/images/logo.png", "static:images/logo.png"},
		{"/static/deep/nested/path/file.txt", "static:deep/nested/path/file.txt"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200 for %s, got %d", tt.path, w.Code)
		}

		if w.Body.String() != tt.expected {
			t.Errorf("Expected %q, got %q for %s", tt.expected, w.Body.String(), tt.path)
		}
	}
}

// Test wildcard routes with empty path matching
func TestWildcardEmptyPath(t *testing.T) {
	r := New()

	// Test multiple wildcard routes
	r.GET("/files/*filepath", func(c *context.Context) {
		filepath := c.Param("filepath")
		c.Text(200, "files:"+filepath)
	})

	r.GET("/assets/*path", func(c *context.Context) {
		path := c.Param("path")
		c.Text(200, "assets:"+path)
	})

	tests := []struct {
		path     string
		expected string
	}{
		{"/files/", "files:"},           // Empty wildcard match
		{"/assets/", "assets:"},         // Empty wildcard match for different route
		{"/files/doc.pdf", "files:doc.pdf"},
		{"/assets/image.jpg", "assets:image.jpg"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200 for %s, got %d", tt.path, w.Code)
		}

		if w.Body.String() != tt.expected {
			t.Errorf("Expected %q, got %q for %s", tt.expected, w.Body.String(), tt.path)
		}
	}
}

// Test route groups
func TestRouteGroups(t *testing.T) {
	r := New()

	// API v1 group
	v1 := r.Group("/api/v1")
	v1.GET("/users", func(c *context.Context) {
		c.Text(200, "v1 users")
	})
	v1.POST("/users", func(c *context.Context) {
		c.Text(201, "v1 create user")
	})

	// API v2 group
	v2 := r.Group("/api/v2")
	v2.GET("/users", func(c *context.Context) {
		c.Text(200, "v2 users")
	})

	tests := []struct {
		method   string
		path     string
		expected string
		status   int
	}{
		{"GET", "/api/v1/users", "v1 users", 200},
		{"POST", "/api/v1/users", "v1 create user", 201},
		{"GET", "/api/v2/users", "v2 users", 200},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != tt.status {
			t.Errorf("Expected status %d for %s %s, got %d", tt.status, tt.method, tt.path, w.Code)
		}

		if w.Body.String() != tt.expected {
			t.Errorf("Expected %q for %s %s, got %q", tt.expected, tt.method, tt.path, w.Body.String())
		}
	}
}

// Test nested groups
func TestNestedGroups(t *testing.T) {
	r := New()

	api := r.Group("/api")
	v1 := api.Group("/v1")
	users := v1.Group("/users")
	
	users.GET("/:id", func(c *context.Context) {
		c.Text(200, "nested user:"+c.Param("id"))
	})

	req := httptest.NewRequest("GET", "/api/v1/users/123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	expected := "nested user:123"
	if w.Body.String() != expected {
		t.Errorf("Expected %q, got %q", expected, w.Body.String())
	}
}

// Test route naming and URL reversal
func TestRouteNaming(t *testing.T) {
	r := New()

	route := r.GET("/users/:id", func(c *context.Context) {
		c.Text(200, "user")
	}).SetName("user.show")

	if route.GetName() != "user.show" {
		t.Errorf("Expected route name 'user.show', got %q", route.GetName())
	}

	if route.GetPath() != "/users/:id" {
		t.Errorf("Expected route path '/users/:id', got %q", route.GetPath())
	}

	// Test URL reversal
	url := r.Reverse("user.show", 123)
	expected := "/users/123"
	if url != expected {
		t.Errorf("Expected URL %q, got %q", expected, url)
	}

	// Test non-existent route
	url = r.Reverse("nonexistent")
	if url != "" {
		t.Errorf("Expected empty string for non-existent route, got %q", url)
	}
}

// Test duplicate route name panic
func TestDuplicateRouteName(t *testing.T) {
	r := New()
	
	r.GET("/users", func(c *context.Context) {}).SetName("users")
	
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for duplicate route name")
		}
	}()
	
	r.GET("/posts", func(c *context.Context) {}).SetName("users")
}

// Test method not allowed with OPTIONS handling
func TestMethodNotAllowed(t *testing.T) {
	r := New()
	
	r.GET("/users", func(c *context.Context) {
		c.Text(200, "get users")
	})
	r.POST("/users", func(c *context.Context) {
		c.Text(201, "create user")
	})

	// Test unsupported method
	req := httptest.NewRequest("DELETE", "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}

	// Test OPTIONS method
	req = httptest.NewRequest("OPTIONS", "/users", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d for OPTIONS, got %d", http.StatusNoContent, w.Code)
	}

	allow := w.Header().Get("Allow")
	if allow == "" {
		t.Error("Expected Allow header to be set for OPTIONS")
	}

	if !strings.Contains(allow, "GET") || !strings.Contains(allow, "POST") {
		t.Errorf("Expected Allow header to contain GET and POST, got %q", allow)
	}
}

// Test HEAD method handling with configurable fallback
func TestHEADMethod(t *testing.T) {
	t.Run("EnableHEADFallback true (default)", func(t *testing.T) {
		r := New()
		
		r.GET("/users", func(c *context.Context) {
			c.Text(200, "users data")
		})

		// Test HEAD request when GET exists
		req := httptest.NewRequest("HEAD", "/users", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200 for HEAD with fallback, got %d", w.Code)
		}
	})

	t.Run("EnableHEADFallback false", func(t *testing.T) {
		r := New(RouterConfig{
			EnableHEADFallback: false,
		})
		
		r.GET("/users", func(c *context.Context) {
			c.Text(200, "users data")
		})

		// Test HEAD request when GET exists but fallback disabled
		req := httptest.NewRequest("HEAD", "/users", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for HEAD without fallback, got %d", w.Code)
		}
	})

	t.Run("Explicit HEAD route", func(t *testing.T) {
		r := New()
		
		r.HEAD("/users", func(c *context.Context) {
			c.Writer.Header().Set("X-Custom", "head-specific")
			c.Writer.WriteHeader(200)
		})
		r.GET("/users", func(c *context.Context) {
			c.Text(200, "users data")
		})

		// Test HEAD request with explicit HEAD route
		req := httptest.NewRequest("HEAD", "/users", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200 for explicit HEAD, got %d", w.Code)
		}
		if w.Header().Get("X-Custom") != "head-specific" {
			t.Error("Expected explicit HEAD handler to be called")
		}
	})
}

// Test trailing slash handling
func TestTrailingSlashHandling(t *testing.T) {
	t.Run("RedirectTrailingSlash enabled (default)", func(t *testing.T) {
		r := New(RouterConfig{
			StrictRouting: true,  // Enable strict routing so /users and /users/ are different
			RedirectTrailingSlash: true,
		})
		
		r.GET("/users", func(c *context.Context) {
			c.Text(200, "users")
		})

		// Test request with trailing slash should redirect
		req := httptest.NewRequest("GET", "/users/", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusMovedPermanently {
			t.Errorf("Expected status 301 for trailing slash redirect, got %d", w.Code)
		}

		location := w.Header().Get("Location")
		if location != "/users" {
			t.Errorf("Expected redirect to /users, got %s", location)
		}
	})

	t.Run("RedirectTrailingSlash with query params", func(t *testing.T) {
		r := New(RouterConfig{
			StrictRouting: true,
			RedirectTrailingSlash: true,
		})
		
		r.GET("/search", func(c *context.Context) {
			c.Text(200, "search results")
		})

		// Test request with trailing slash and query params
		req := httptest.NewRequest("GET", "/search/?q=test&page=2", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusMovedPermanently {
			t.Errorf("Expected status 301, got %d", w.Code)
		}

		location := w.Header().Get("Location")
		if location != "/search?q=test&page=2" {
			t.Errorf("Expected redirect with query params preserved, got %s", location)
		}
	})

	t.Run("RedirectTrailingSlash adds slash", func(t *testing.T) {
		r := New(RouterConfig{
			StrictRouting: true,
			RedirectTrailingSlash: true,
		})
		
		r.GET("/users/", func(c *context.Context) {
			c.Text(200, "users with slash")
		})

		// Test request without trailing slash should redirect
		req := httptest.NewRequest("GET", "/users", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusMovedPermanently {
			t.Errorf("Expected status 301 for adding slash, got %d", w.Code)
		}

		location := w.Header().Get("Location")
		if location != "/users/" {
			t.Errorf("Expected redirect to /users/, got %s", location)
		}
	})

	t.Run("RedirectTrailingSlash disabled", func(t *testing.T) {
		r := New(RouterConfig{
			RedirectTrailingSlash: false,
		})
		
		r.GET("/users", func(c *context.Context) {
			c.Text(200, "users")
		})

		// Test request with trailing slash should 404
		req := httptest.NewRequest("GET", "/users/", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// When RedirectTrailingSlash is disabled, /users/ won't redirect
		// but since parsePath normalizes the path, it will still match
		// This is actually the current behavior - both /users and /users/ match the same route
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 (both paths match same route), got %d", w.Code)
		}
	})

	t.Run("POST method redirect uses 308", func(t *testing.T) {
		r := New(RouterConfig{
			StrictRouting: true,
			RedirectTrailingSlash: true,
		})
		
		r.POST("/users", func(c *context.Context) {
			c.Text(201, "created")
		})

		// Test POST with trailing slash
		req := httptest.NewRequest("POST", "/users/", strings.NewReader("data"))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusPermanentRedirect {
			t.Errorf("Expected status 308 for POST redirect, got %d", w.Code)
		}
	})
}

// Test ALL method returns RouteCollection
func TestALLMethodReturnsCollection(t *testing.T) {
	r := New()
	
	collection := r.ALL("/api", func(c *context.Context) {
		c.Text(200, "all methods")
	})

	if collection == nil {
		t.Fatal("Expected ALL to return RouteCollection")
	}

	if len(collection.Routes) != 7 {
		t.Errorf("Expected 7 routes in collection, got %d", len(collection.Routes))
	}

	// Test SetName on collection
	collection.SetName("api_all")

	// Verify each route has unique name
	expectedMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	for i, route := range collection.Routes {
		expectedName := fmt.Sprintf("api_all_%s", strings.ToLower(expectedMethods[i]))
		if route.Name != expectedName {
			t.Errorf("Expected route name %s, got %s", expectedName, route.Name)
		}
	}

	// Test that all methods work
	for _, method := range expectedMethods {
		req := httptest.NewRequest(method, "/api", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200 for %s /api, got %d", method, w.Code)
		}
	}
}

// Test panic recovery
func TestPanicRecovery(t *testing.T) {
	r := New()
	
	r.GET("/panic", func(c *context.Context) {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d for panic, got %d", http.StatusInternalServerError, w.Code)
	}

	if !strings.Contains(w.Body.String(), "Internal Server Error") {
		t.Error("Expected panic to be handled with Internal Server Error")
	}
}

// Test group middleware functionality
func TestGroupMiddleware(t *testing.T) {
	r := New()
	
	middleware1Called := false
	middleware2Called := false
	
	middleware1 := func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			middleware1Called = true
			c.SetHeader("X-Middleware-1", "true")
			next(c)
		}
	}
	
	middleware2 := func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			middleware2Called = true
			c.SetHeader("X-Middleware-2", "true")
			next(c)
		}
	}

	group := r.Group("/api", middleware1, middleware2)
	group.GET("/test", func(c *context.Context) {
		c.Text(200, "test")
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !middleware1Called {
		t.Error("Expected middleware1 to be called")
	}

	if !middleware2Called {
		t.Error("Expected middleware2 to be called")
	}

	if w.Header().Get("X-Middleware-1") != "true" {
		t.Error("Expected X-Middleware-1 header to be set")
	}

	if w.Header().Get("X-Middleware-2") != "true" {
		t.Error("Expected X-Middleware-2 header to be set")
	}
}

// Test conflicting route registration panics
func TestConflictingRoutes(t *testing.T) {
	r := New()
	
	// Test duplicate exact route
	r.GET("/users", func(c *context.Context) {})
	
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for duplicate route")
		}
	}()
	
	r.GET("/users", func(c *context.Context) {})
}

// Test invalid route patterns
func TestInvalidRoutes(t *testing.T) {
	r := New()
	
	// Test route without leading slash
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for route without leading slash")
		}
	}()
	
	r.GET("users", func(c *context.Context) {})
}

// Test wildcard position validation
func TestWildcardPosition(t *testing.T) {
	r := New()
	
	// Wildcard not at end should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for wildcard not at end")
		}
	}()
	
	r.GET("/files/*path/more", func(c *context.Context) {})
}

// Test route priority: static > param > wildcard
func TestRoutePriority(t *testing.T) {
	r := New()
	
	// Register routes in order to test priority
	r.GET("/users/new", func(c *context.Context) {
		c.Text(200, "new user form")
	})
	
	r.GET("/users/:id", func(c *context.Context) {
		c.Text(200, "user:"+c.Param("id"))
	})
	
	// Use a different route for wildcard to avoid conflicts
	r.GET("/files/*filepath", func(c *context.Context) {
		c.Text(200, "wildcard:"+c.Param("filepath"))
	})

	tests := []struct {
		path     string
		expected string
	}{
		{"/users/new", "new user form"},        // Static route wins
		{"/users/123", "user:123"},             // Param route
		{"/files/css/style.css", "wildcard:css/style.css"}, // Wildcard route
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200 for %s, got %d", tt.path, w.Code)
		}

		if w.Body.String() != tt.expected {
			t.Errorf("Expected %q for %s, got %q", tt.expected, tt.path, w.Body.String())
		}
	}
}

// Test edge cases with special characters in paths
func TestSpecialCharacterPaths(t *testing.T) {
	r := New()
	
	// Routes with special characters (URL encoded)
	r.GET("/users/:name", func(c *context.Context) {
		name := c.Param("name")
		c.Text(200, "user:"+name)
	})
	
	r.GET("/search/*query", func(c *context.Context) {
		query := c.Param("query")
		c.Text(200, "search:"+query)
	})

	tests := []struct {
		path     string
		expected string
		status   int
	}{
		// URL encoded characters (will be decoded by HTTP request parsing)
		{"/users/john%20doe", "user:john doe", 200},
		{"/users/caf%C3%A9", "user:café", 200},
		{"/users/user%40example.com", "user:user@example.com", 200},
		
		// Special characters in wildcard (will be decoded by HTTP request parsing)
		{"/search/hello%20world", "search:hello world", 200},
		{"/search/test%3Dvalue%26more", "search:test=value&more", 200},
		{"/search/unicode%20%E2%9C%93", "search:unicode ✓", 200},
		
		// Raw special characters (not URL encoded but valid in URLs)
		{"/users/user-name", "user:user-name", 200},
		{"/users/user_underscore", "user:user_underscore", 200},
		{"/users/user.email", "user:user.email", 200},
		{"/search/query-term", "search:query-term", 200},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != tt.status {
			t.Errorf("Expected status %d for %s, got %d", tt.status, tt.path, w.Code)
		}

		if w.Body.String() != tt.expected {
			t.Errorf("Expected %q for %s, got %q", tt.expected, tt.path, w.Body.String())
		}
	}
}

// Test edge cases with empty and root paths
func TestEmptyAndRootPaths(t *testing.T) {
	r := New()
	
	// Root path
	r.GET("/", func(c *context.Context) {
		c.Text(200, "root")
	})
	
	// Empty wildcard match
	r.GET("/files/*path", func(c *context.Context) {
		path := c.Param("path")
		c.Text(200, "files:"+path)
	})
	
	// Parameter at root level
	r.GET("/:slug", func(c *context.Context) {
		slug := c.Param("slug")
		c.Text(200, "slug:"+slug)
	})

	tests := []struct {
		path     string
		expected string
		status   int
	}{
		// Root path variations
		{"/", "root", 200},
		
		// Empty wildcard should match
		{"/files/", "files:", 200},
		
		// Single character paths
		{"/a", "slug:a", 200},
		{"/1", "slug:1", 200},
		{"/x", "slug:x", 200},
		
		// Very long path segments
		{"/verylongslugnamethatshouldstillwork", "slug:verylongslugnamethatshouldstillwork", 200},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != tt.status {
			t.Errorf("Expected status %d for %s, got %d", tt.status, tt.path, w.Code)
		}

		if w.Body.String() != tt.expected {
			t.Errorf("Expected %q for %s, got %q", tt.expected, tt.path, w.Body.String())
		}
	}
}

// Test route matching edge cases with multiple parameters
func TestMultipleParameterEdgeCases(t *testing.T) {
	r := New()
	
	// Route with many parameters
	r.GET("/api/:v1/:v2/:v3/:v4/:action", func(c *context.Context) {
		result := fmt.Sprintf("%s-%s-%s-%s-%s", 
			c.Param("v1"), c.Param("v2"), c.Param("v3"), c.Param("v4"), c.Param("action"))
		c.Text(200, result)
	})
	
	// Route with parameters and static segments mixed
	r.GET("/users/:id/profile/:section/edit", func(c *context.Context) {
		result := fmt.Sprintf("user:%s-section:%s", c.Param("id"), c.Param("section"))
		c.Text(200, result)
	})

	tests := []struct {
		path     string
		expected string
		status   int
	}{
		// Many parameters
		{"/api/1/2/3/4/delete", "1-2-3-4-delete", 200},
		{"/api/a/b/c/d/create", "a-b-c-d-create", 200},
		
		// Mixed static and parameters
		{"/users/123/profile/settings/edit", "user:123-section:settings", 200},
		{"/users/john/profile/privacy/edit", "user:john-section:privacy", 200},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != tt.status {
			t.Errorf("Expected status %d for %s, got %d", tt.status, tt.path, w.Code)
		}

		if w.Body.String() != tt.expected {
			t.Errorf("Expected %q for %s, got %q", tt.expected, tt.path, w.Body.String())
		}
	}
}

// Test case sensitivity in routing
func TestRouteCaseSensitivity(t *testing.T) {
	r := New()
	
	r.GET("/Users", func(c *context.Context) {
		c.Text(200, "uppercase users")
	})
	
	r.GET("/users", func(c *context.Context) {
		c.Text(200, "lowercase users")
	})
	
	r.GET("/API/:version", func(c *context.Context) {
		c.Text(200, "API:"+c.Param("version"))
	})

	tests := []struct {
		path     string
		expected string
		status   int
	}{
		// Case sensitive matching
		{"/Users", "uppercase users", 200},
		{"/users", "lowercase users", 200},
		{"/USERS", "", 404}, // Should not match either
		
		// Case sensitive parameters
		{"/API/v1", "API:v1", 200},
		{"/api/v1", "", 404}, // Should not match
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != tt.status {
			t.Errorf("Expected status %d for %s, got %d", tt.status, tt.path, w.Code)
		}

		if tt.expected != "" && w.Body.String() != tt.expected {
			t.Errorf("Expected %q for %s, got %q", tt.expected, tt.path, w.Body.String())
		}
	}
}

// Benchmark router performance
func BenchmarkRouter(b *testing.B) {
	r := New()

	// Add various routes
	r.GET("/", func(c *context.Context) {})
	r.GET("/users", func(c *context.Context) {})
	r.GET("/users/:id", func(c *context.Context) {})
	r.GET("/posts/:year/:month/:day/:slug", func(c *context.Context) {})
	r.GET("/static/*filepath", func(c *context.Context) {})

	// API routes
	api := r.Group("/api/v1")
	api.GET("/users", func(c *context.Context) {})
	api.GET("/posts", func(c *context.Context) {})
	api.GET("/comments", func(c *context.Context) {})

	requests := []string{
		"/",
		"/users",
		"/users/123",
		"/posts/2024/01/15/hello-world",
		"/static/css/style.css",
		"/api/v1/users",
		"/notfound",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			path := requests[i%len(requests)]
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			i++
		}
	})
}