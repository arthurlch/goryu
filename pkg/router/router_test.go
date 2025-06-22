package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arthurlch/goryu/pkg/context"
	"github.com/arthurlch/goryu/pkg/middleware"
	"github.com/arthurlch/goryu/pkg/router"
)

func TestRouter_SimpleRoute(t *testing.T) {
	r := router.New()
	var handlerCalled bool
	r.GET("/test", func(c *context.Context) {
		handlerCalled = true
		c.Text(http.StatusOK, "test_route_ok")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Error("Expected handler to be called for GET /test, but it wasn't")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d for GET /test, got %d", http.StatusOK, rr.Code)
	}
	if rr.Body.String() != "test_route_ok" {
		t.Errorf("Expected body 'test_route_ok' for GET /test, got '%s'", rr.Body.String())
	}
}

func TestRouter_RouteWithParams(t *testing.T) {
	r := router.New()
	var userID string
	r.GET("/users/:id", func(c *context.Context) {
		userID = c.Params["id"]
		c.Text(http.StatusOK, "user_id_"+userID)
	})

	req, _ := http.NewRequest("GET", "/users/123", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if userID != "123" {
		t.Errorf("Expected userID param to be '123', got '%s'", userID)
	}
	if rr.Body.String() != "user_id_123" {
		t.Errorf("Expected body 'user_id_123', got '%s'", rr.Body.String())
	}
}

func TestRouter_NotFound(t *testing.T) {
	r := router.New()
	r.GET("/exists", func(c *context.Context) { /* ... */ })

	req, _ := http.NewRequest("GET", "/doesnotexist", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status %d for non-existent route, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestRouter_Group(t *testing.T) {
	r := router.New()
	var groupHandlerCalled bool

	var groupMiddlewareCalled bool
	testGroupMiddleware := func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			groupMiddlewareCalled = true
			next(c)
		}
	}

	group := r.Group("/api", []middleware.Middleware{testGroupMiddleware}) // <--- CHANGE HERE
	group.GET("/info", func(c *context.Context) {
		groupHandlerCalled = true
		c.Text(http.StatusOK, "api_info")
	})

	req, _ := http.NewRequest("GET", "/api/info", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if !groupHandlerCalled {
		t.Error("Group handler was not called")
	}
	if !groupMiddlewareCalled {
		t.Error("Group middleware was not called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if rr.Body.String() != "api_info" {
		t.Errorf("Expected body 'api_info', got '%s'", rr.Body.String())
	}
}
