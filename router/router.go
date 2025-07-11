package router

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/arthurlch/goryu/context"
)

type Route struct {
	Method  string
	Path    string
	Handler context.HandlerFunc
	Name    string
}

func (r *Route) SetName(name string) *Route {
	r.Name = name
	return r
}

type Group struct {
	prefix      string
	middlewares []context.Middleware
	router      *Router
}

type Router struct {
	trees       map[string]*node
	namedRoutes map[string]*Route
}

func New() *Router {
	return &Router{
		trees:       make(map[string]*node),
		namedRoutes: make(map[string]*Route),
	}
}

func (router *Router) Add(method, path string, handler context.HandlerFunc) *Route {
	if _, ok := router.trees[method]; !ok {
		router.trees[method] = &node{}
	}
	parts := parsePath(path)
	route := &Route{Method: method, Path: path, Handler: handler}
	router.trees[method].insert(path, parts, 0, route)
	return route
}

func (router *Router) GET(path string, handler context.HandlerFunc) *Route {
	return router.Add("GET", path, handler)
}

func (router *Router) POST(path string, handler context.HandlerFunc) *Route {
	return router.Add("POST", path, handler)
}

func (router *Router) PUT(path string, handler context.HandlerFunc) *Route {
	return router.Add("PUT", path, handler)
}

func (router *Router) DELETE(path string, handler context.HandlerFunc) *Route {
	return router.Add("DELETE", path, handler)
}

func (router *Router) PATCH(path string, handler context.HandlerFunc) *Route {
	return router.Add("PATCH", path, handler)
}

func (router *Router) HEAD(path string, handler context.HandlerFunc) *Route {
	return router.Add("HEAD", path, handler)
}

func (router *Router) OPTIONS(path string, handler context.HandlerFunc) *Route {
	return router.Add("OPTIONS", path, handler)
}

func (router *Router) ALL(path string, handler context.HandlerFunc) *Route {
	return router.Add("ALL", path, handler)
}

func (router *Router) Group(prefix string, middlewares ...context.Middleware) *Group {
	return &Group{
		prefix:      prefix,
		router:      router,
		middlewares: middlewares,
	}
}

func (router *Router) Reverse(name string, params ...interface{}) string {
	route, ok := router.namedRoutes[name]
	if !ok {
		return ""
	}

	path := route.Path
	paramIndex := 0
	for _, part := range parsePath(path) {
		if strings.HasPrefix(part, ":") && paramIndex < len(params) {
			path = strings.Replace(path, part, fmt.Sprint(params[paramIndex]), 1)
			paramIndex++
		}
	}
	return path
}

func (router *Router) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	ctx := context.NewContext(writer, request)

	tree, ok := router.trees[request.Method]
	var node *node
	var params map[string]string

	if ok {
		parts := parsePath(request.URL.Path)
		node, params = tree.find(parts, 0)
	}

	if node == nil {
		tree, ok = router.trees["ALL"]
		if ok {
			parts := parsePath(request.URL.Path)
			node, params = tree.find(parts, 0)
		}
	}

	// If still no handler, return 404 ...
	if node == nil {
		http.NotFound(writer, request)
		return
	}

	ctx.Params = params
	if node.route.Name != "" {
		router.namedRoutes[node.route.Name] = node.route
	}
	node.handler(ctx)
}

func (g *Group) Group(prefix string, middlewares ...context.Middleware) *Group {
	return &Group{
		prefix:      g.prefix + prefix,
		router:      g.router,
		middlewares: append(g.middlewares, middlewares...),
	}
}

func (g *Group) GET(path string, handler context.HandlerFunc) *Route {
	fullPath := g.prefix + path
	return g.router.GET(fullPath, g.wrapWithMiddleware(handler))
}

func (g *Group) POST(path string, handler context.HandlerFunc) *Route {
	fullPath := g.prefix + path
	return g.router.POST(fullPath, g.wrapWithMiddleware(handler))
}

func (g *Group) wrapWithMiddleware(handler context.HandlerFunc) context.HandlerFunc {
	currentHandler := handler
	for i := len(g.middlewares) - 1; i >= 0; i-- {
		currentHandler = g.middlewares[i](currentHandler)
	}
	return currentHandler
}

// helperss might move to utils
func parsePath(path string) []string {
	parts := strings.Split(path, "/")
	cleanedParts := make([]string, 0)
	for _, p := range parts {
		if p != "" {
			cleanedParts = append(cleanedParts, p)
		}
	}
	return cleanedParts
}
