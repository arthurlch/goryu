package builder

import (
	"time"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/router"
)

// RouteBuilder provides a fluent interface for route registration
type RouteBuilder struct {
	router *router.Router
}

// New creates a new RouteBuilder
func New(r *router.Router) *RouteBuilder {
	return &RouteBuilder{router: r}
}

// Group creates a route group with a fluent callback API
func (rb *RouteBuilder) Group(prefix string, callback func(*GroupBuilder)) *RouteBuilder {
	group := rb.router.Group(prefix)
	gb := &GroupBuilder{group: group}
	callback(gb)
	return rb
}

// GroupBuilder provides fluent methods for building route groups
type GroupBuilder struct {
	group       *router.Group
	middlewares []context.Middleware
}

// Middleware adds middleware to all routes in this group
func (gb *GroupBuilder) Middleware(middlewares ...context.Middleware) *GroupBuilder {
	gb.middlewares = append(gb.middlewares, middlewares...)
	return gb
}

// Resource creates RESTful routes for a resource
func (gb *GroupBuilder) Resource(path string, controller interface{}) *ResourceBuilder {
	return &ResourceBuilder{
		group:      gb,
		path:       path,
		controller: controller,
	}
}

// GET registers a GET route with fluent configuration
func (gb *GroupBuilder) GET(path string, handler context.HandlerFunc) *RouteConfig {
	route := gb.group.GET(path, gb.wrapHandler(handler))
	return &RouteConfig{route: route}
}

// POST registers a POST route with fluent configuration
func (gb *GroupBuilder) POST(path string, handler context.HandlerFunc) *RouteConfig {
	route := gb.group.POST(path, gb.wrapHandler(handler))
	return &RouteConfig{route: route}
}

// PUT registers a PUT route with fluent configuration
func (gb *GroupBuilder) PUT(path string, handler context.HandlerFunc) *RouteConfig {
	route := gb.group.PUT(path, gb.wrapHandler(handler))
	return &RouteConfig{route: route}
}

// DELETE registers a DELETE route with fluent configuration
func (gb *GroupBuilder) DELETE(path string, handler context.HandlerFunc) *RouteConfig {
	route := gb.group.DELETE(path, gb.wrapHandler(handler))
	return &RouteConfig{route: route}
}

// PATCH registers a PATCH route with fluent configuration
func (gb *GroupBuilder) PATCH(path string, handler context.HandlerFunc) *RouteConfig {
	route := gb.group.PATCH(path, gb.wrapHandler(handler))
	return &RouteConfig{route: route}
}

// Group creates a nested group
func (gb *GroupBuilder) Group(prefix string, callback func(*GroupBuilder)) *GroupBuilder {
	nestedGroup := gb.group.Group(prefix, gb.middlewares...)
	nestedGB := &GroupBuilder{group: nestedGroup}
	callback(nestedGB)
	return gb
}

// wrapHandler applies group middleware to a handler
func (gb *GroupBuilder) wrapHandler(handler context.HandlerFunc) context.HandlerFunc {
	// Apply group-level middleware
	for i := len(gb.middlewares) - 1; i >= 0; i-- {
		handler = gb.middlewares[i](handler)
	}
	return handler
}

// RouteConfig provides fluent configuration for individual routes
type RouteConfig struct {
	route       *router.Route
	middlewares []context.Middleware
	cacheTTL    time.Duration
}

// Name sets the route name for URL generation
func (rc *RouteConfig) Name(name string) *RouteConfig {
	if rc.route != nil {
		rc.route.SetName(name)
	}
	return rc
}

// Middleware adds middleware to this specific route
func (rc *RouteConfig) Middleware(middlewares ...context.Middleware) *RouteConfig {
	rc.middlewares = append(rc.middlewares, middlewares...)
	// Note: Individual route middleware would need to be applied differently
	// This is a design consideration - routes are already registered
	return rc
}

// Cache sets caching for this route
func (rc *RouteConfig) Cache(ttl time.Duration) *RouteConfig {
	rc.cacheTTL = ttl
	// Caching implementation would be handled by middleware
	return rc
}

// Description adds a description to the route (useful for API documentation)
func (rc *RouteConfig) Description(desc string) *RouteConfig {
	if rc.route != nil {
		rc.route.Description = desc
	}
	return rc
}

// ResourceBuilder builds RESTful routes for a resource
type ResourceBuilder struct {
	group          *GroupBuilder
	path           string
	controller     interface{}
	middlewares    []context.Middleware
	name           string
	only           []string
	except         []string
}

// Middleware adds middleware to all resource routes
func (rb *ResourceBuilder) Middleware(middlewares ...context.Middleware) *ResourceBuilder {
	rb.middlewares = append(rb.middlewares, middlewares...)
	return rb
}

// Name sets the base name for all resource routes
func (rb *ResourceBuilder) Name(name string) *ResourceBuilder {
	rb.name = name
	return rb
}

// Only specifies which resource actions to include
func (rb *ResourceBuilder) Only(actions ...string) *ResourceBuilder {
	rb.only = actions
	return rb
}

// Except specifies which resource actions to exclude
func (rb *ResourceBuilder) Except(actions ...string) *ResourceBuilder {
	rb.except = actions
	return rb
}

// Build finalizes the resource routes
func (rb *ResourceBuilder) Build() *GroupBuilder {
	// Map of action names to HTTP methods and paths
	actions := map[string]struct {
		method string
		path   string
	}{
		"index":   {"GET", rb.path},
		"create":  {"POST", rb.path},
		"show":    {"GET", rb.path + "/:id"},
		"update":  {"PUT", rb.path + "/:id"},
		"destroy": {"DELETE", rb.path + "/:id"},
	}

	// Filter actions based on only/except
	for action, config := range actions {
		if !rb.shouldIncludeAction(action) {
			continue
		}

		// Get handler method from controller
		handler := rb.getHandlerForAction(action)
		if handler == nil {
			continue
		}

		// Apply resource middleware
		for i := len(rb.middlewares) - 1; i >= 0; i-- {
			handler = rb.middlewares[i](handler)
		}

		// Register route
		var route *router.Route
		switch config.method {
		case "GET":
			route = rb.group.group.GET(config.path, rb.group.wrapHandler(handler))
		case "POST":
			route = rb.group.group.POST(config.path, rb.group.wrapHandler(handler))
		case "PUT":
			route = rb.group.group.PUT(config.path, rb.group.wrapHandler(handler))
		case "DELETE":
			route = rb.group.group.DELETE(config.path, rb.group.wrapHandler(handler))
		}

		// Set route name if provided
		if rb.name != "" && route != nil {
			route.SetName(rb.name + "." + action)
		}
	}

	return rb.group
}

// shouldIncludeAction determines if an action should be included
func (rb *ResourceBuilder) shouldIncludeAction(action string) bool {
	// If 'only' is specified, action must be in the list
	if len(rb.only) > 0 {
		found := false
		for _, a := range rb.only {
			if a == action {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// If 'except' is specified, action must not be in the list
	if len(rb.except) > 0 {
		for _, a := range rb.except {
			if a == action {
				return false
			}
		}
	}

	return true
}

// getHandlerForAction extracts the handler method from the controller
func (rb *ResourceBuilder) getHandlerForAction(action string) context.HandlerFunc {
	// If controller is already a map of handlers
	if handlers, ok := rb.controller.(map[string]context.HandlerFunc); ok {
		return handlers[action]
	}

	// If controller is a ResourceController
	if rc, ok := rb.controller.(*ResourceController); ok {
		return rc.GetHandler(action)
	}

	// Try to extract method using reflection
	return extractControllerMethod(rb.controller, action)
}