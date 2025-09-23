package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arthurlch/goryu/context"
)

func TestBasicRouting(t *testing.T) {
	r := New()

	// Test basic routes
	r.GET("/", func(c *context.Context) {
		c.Text(200, "home")
	})

	r.GET("/users", func(c *context.Context) {
		c.Text(200, "users")
	})

	r.POST("/users", func(c *context.Context) {
		c.Text(200, "create user")
	})

	tests := []struct {
		method   string
		path     string
		expected string
		status   int
	}{
		{"GET", "/", "home", 200},
		{"GET", "/users", "users", 200},
		{"POST", "/users", "create user", 200},
		{"GET", "/notfound", "", 404},
		{"DELETE", "/users", "", 405}, // Method not allowed since GET/POST exist
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

func TestParameterRoutes(t *testing.T) {
	r := New()

	r.GET("/users/:id", func(c *context.Context) {
		c.Text(200, "user:"+c.Param("id"))
	})

	r.GET("/posts/:year/:month/:day", func(c *context.Context) {
		c.Text(200, c.Param("year")+"-"+c.Param("month")+"-"+c.Param("day"))
	})

	tests := []struct {
		path     string
		expected string
	}{
		{"/users/123", "user:123"},
		{"/users/john", "user:john"},
		{"/posts/2024/01/15", "2024-01-15"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Body.String() != tt.expected {
			t.Errorf("Expected %q, got %q for %s", tt.expected, w.Body.String(), tt.path)
		}
	}
}

func TestRouteGroups(t *testing.T) {
	r := New()

	// API v1 group
	v1 := r.Group("/api/v1")
	v1.GET("/users", func(c *context.Context) {
		c.Text(200, "v1 users")
	})

	// Test group routing
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Body.String() != "v1 users" {
		t.Errorf("Expected 'v1 users', got %q", w.Body.String())
	}
}

func TestGlobalMiddleware(t *testing.T) {
	r := New()

	callCount := 0
	middleware := func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			callCount++
			c.SetHeader("X-Test", "true")
			next(c)
		}
	}

	r.Use(middleware)

	r.GET("/test", func(c *context.Context) {
		c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if callCount != 1 {
		t.Errorf("Expected middleware to be called once, got %d", callCount)
	}

	if w.Header().Get("X-Test") != "true" {
		t.Error("Expected X-Test header to be set")
	}
}

func TestMethodNotAllowed(t *testing.T) {
	r := New()
	r.HandleMethodNotAllowed = true

	r.GET("/users", func(c *context.Context) {
		c.Text(200, "get users")
	})
	r.POST("/users", func(c *context.Context) {
		c.Text(200, "create user")
	})

	// Try DELETE which is not allowed
	req := httptest.NewRequest("DELETE", "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", w.Code)
	}

	allow := w.Header().Get("Allow")
	if allow == "" {
		t.Error("Expected Allow header to be set")
	}
}

func BenchmarkRouter(b *testing.B) {
	r := New()

	// Add various routes
	r.GET("/", func(c *context.Context) {})
	r.GET("/users", func(c *context.Context) {})
	r.GET("/users/:id", func(c *context.Context) {})
	r.GET("/posts/:year/:month/:day/:slug", func(c *context.Context) {})

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
