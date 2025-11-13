package builder

import (
	"time"

	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/router"
)

// SimpleApp interface is defined in interfaces.go

// SimpleRouteBuilder provides simplified fluent route registration
type SimpleRouteBuilder struct {
	app SimpleApp
}

// NewSimpleRouteBuilder creates a new simple route builder
func NewSimpleRouteBuilder(app SimpleApp) *SimpleRouteBuilder {
	return &SimpleRouteBuilder{app: app}
}

// Group creates a route group with a callback function
func (srb *SimpleRouteBuilder) Group(prefix string, callback func(*SimpleGroupBuilder)) *SimpleRouteBuilder {
	routerInterface := srb.app.GetRouter()
	router := routerInterface.(*router.Router) // Type assertion
	group := router.Group(prefix)
	sgb := &SimpleGroupBuilder{
		app:         srb.app,
		routerGroup: group,
		middlewares: make([]context.Middleware, 0),
	}
	callback(sgb)
	return srb
}

// SimpleGroupBuilder provides group-level route registration
type SimpleGroupBuilder struct {
	app         SimpleApp
	routerGroup *router.Group
	middlewares []context.Middleware
}

// Middleware adds middleware to all routes in this group
func (sgb *SimpleGroupBuilder) Middleware(middlewares ...context.Middleware) *SimpleGroupBuilder {
	sgb.middlewares = append(sgb.middlewares, middlewares...)
	return sgb
}

// GET registers a GET route
func (sgb *SimpleGroupBuilder) GET(path string, handler context.HandlerFunc) *SimpleRouteConfig {
	finalHandler := sgb.wrapHandler(handler)
	route := sgb.routerGroup.GET(path, finalHandler)
	return &SimpleRouteConfig{route: route}
}

// POST registers a POST route
func (sgb *SimpleGroupBuilder) POST(path string, handler context.HandlerFunc) *SimpleRouteConfig {
	finalHandler := sgb.wrapHandler(handler)
	route := sgb.routerGroup.POST(path, finalHandler)
	return &SimpleRouteConfig{route: route}
}

// PUT registers a PUT route
func (sgb *SimpleGroupBuilder) PUT(path string, handler context.HandlerFunc) *SimpleRouteConfig {
	finalHandler := sgb.wrapHandler(handler)
	route := sgb.routerGroup.PUT(path, finalHandler)
	return &SimpleRouteConfig{route: route}
}

// DELETE registers a DELETE route
func (sgb *SimpleGroupBuilder) DELETE(path string, handler context.HandlerFunc) *SimpleRouteConfig {
	finalHandler := sgb.wrapHandler(handler)
	route := sgb.routerGroup.DELETE(path, finalHandler)
	return &SimpleRouteConfig{route: route}
}

// PATCH registers a PATCH route
func (sgb *SimpleGroupBuilder) PATCH(path string, handler context.HandlerFunc) *SimpleRouteConfig {
	finalHandler := sgb.wrapHandler(handler)
	route := sgb.routerGroup.PATCH(path, finalHandler)
	return &SimpleRouteConfig{route: route}
}

// Group creates a nested group
func (sgb *SimpleGroupBuilder) Group(prefix string, callback func(*SimpleGroupBuilder)) *SimpleGroupBuilder {
	nestedGroup := sgb.routerGroup.Group(prefix)
	nestedSGB := &SimpleGroupBuilder{
		app:         sgb.app,
		routerGroup: nestedGroup,
		middlewares: append([]context.Middleware{}, sgb.middlewares...),
	}
	callback(nestedSGB)
	return sgb
}

// Resource creates RESTful routes for a resource
func (sgb *SimpleGroupBuilder) Resource(path string, controller interface{}) *SimpleResourceBuilder {
	return &SimpleResourceBuilder{
		group:      sgb,
		path:       path,
		controller: controller,
	}
}

// wrapHandler applies group middleware and app middleware
func (sgb *SimpleGroupBuilder) wrapHandler(handler context.HandlerFunc) context.HandlerFunc {
	// Apply group middleware
	for i := len(sgb.middlewares) - 1; i >= 0; i-- {
		handler = sgb.middlewares[i](handler)
	}
	
	// Apply app middleware
	return sgb.app.ApplyMiddleware(handler)
}

// SimpleRouteConfig provides fluent configuration for routes
type SimpleRouteConfig struct {
	route    *router.Route
	cacheTTL time.Duration
}

// Name sets the route name
func (src *SimpleRouteConfig) Name(name string) *SimpleRouteConfig {
	if src.route != nil {
		src.route.SetName(name)
	}
	return src
}

// Description sets the route description
func (src *SimpleRouteConfig) Description(desc string) *SimpleRouteConfig {
	if src.route != nil {
		src.route.Description = desc
	}
	return src
}

// Cache sets caching TTL for documentation purposes
func (src *SimpleRouteConfig) Cache(ttl time.Duration) *SimpleRouteConfig {
	src.cacheTTL = ttl
	// In a real implementation, cache middleware would be applied here
	return src
}

// SimpleResourceBuilder builds RESTful routes
type SimpleResourceBuilder struct {
	group       *SimpleGroupBuilder
	path        string
	controller  interface{}
	middlewares []context.Middleware
	name        string
	only        []string
	except      []string
}

// Middleware adds middleware to resource routes
func (srb *SimpleResourceBuilder) Middleware(middlewares ...context.Middleware) *SimpleResourceBuilder {
	srb.middlewares = append(srb.middlewares, middlewares...)
	return srb
}

// Name sets the resource name prefix
func (srb *SimpleResourceBuilder) Name(name string) *SimpleResourceBuilder {
	srb.name = name
	return srb
}

// Only includes only specified actions
func (srb *SimpleResourceBuilder) Only(actions ...string) *SimpleResourceBuilder {
	srb.only = actions
	return srb
}

// Except excludes specified actions
func (srb *SimpleResourceBuilder) Except(actions ...string) *SimpleResourceBuilder {
	srb.except = actions
	return srb
}

// Build creates the resource routes
func (srb *SimpleResourceBuilder) Build() *SimpleGroupBuilder {
	resourceController := NewResourceController(srb.controller)
	
	actions := map[string]struct {
		method string
		path   string
	}{
		"index":   {"GET", srb.path},
		"create":  {"POST", srb.path},
		"show":    {"GET", srb.path + "/:id"},
		"update":  {"PUT", srb.path + "/:id"},
		"destroy": {"DELETE", srb.path + "/:id"},
	}
	
	for action, config := range actions {
		if !srb.shouldIncludeAction(action) {
			continue
		}
		
		handler := resourceController.GetHandler(action)
		if handler == nil {
			continue
		}
		
		// Apply resource middleware
		for i := len(srb.middlewares) - 1; i >= 0; i-- {
			handler = srb.middlewares[i](handler)
		}
		
		// Register the route
		var routeConfig *SimpleRouteConfig
		switch config.method {
		case "GET":
			routeConfig = srb.group.GET(config.path, handler)
		case "POST":
			routeConfig = srb.group.POST(config.path, handler)
		case "PUT":
			routeConfig = srb.group.PUT(config.path, handler)
		case "DELETE":
			routeConfig = srb.group.DELETE(config.path, handler)
		}
		
		// Set route name if provided
		if srb.name != "" && routeConfig != nil {
			routeConfig.Name(srb.name + "." + action)
		}
	}
	
	return srb.group
}

// shouldIncludeAction checks if an action should be included
func (srb *SimpleResourceBuilder) shouldIncludeAction(action string) bool {
	if len(srb.only) > 0 {
		found := false
		for _, a := range srb.only {
			if a == action {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	if len(srb.except) > 0 {
		for _, a := range srb.except {
			if a == action {
				return false
			}
		}
	}
	
	return true
}