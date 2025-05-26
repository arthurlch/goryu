// base for the routing
package router

import (
	"net/http"
	"strings"

	"github.com/arthurlch/goryu/internals/utils"
	"github.com/arthurlch/goryu/pkg/context"
	"github.com/arthurlch/goryu/pkg/middleware"
)

type Route struct {
	Method  string
	Path    string
	Handler context.HandlerFunc
}

type Group struct {
	prefix      string
	middlewares []middleware.Middleware
	router      *Router
}

type Router struct {
	routes []Route
	groups []*Group
	// Add a field for static file handling if needed, e.g.
	// staticHandlers map[string]http.Handler
}

// this create a new intance for the router
func New() *Router {
	return &Router{
		routes: make([]Route, 0),
		groups: make([]*Group, 0),
		// staticHandlers: make(map[string]http.Handler), // Initialize if added
	}
}

func (router *Router) Add(method, path string, handler context.HandlerFunc) {
	router.routes = append(router.routes, Route{
		Method:  strings.ToUpper(method),
		Path:    path,
		Handler: handler,
	})
}

// route method

func (router *Router) GET(path string, handler context.HandlerFunc) {
	router.Add("GET", path, handler)
}

func (router *Router) PUT(path string, handler context.HandlerFunc) {
	router.Add("PUT", path, handler)
}

func (router *Router) DELETE(path string, handler context.HandlerFunc) {
	router.Add("DELETE", path, handler)
}

func (router *Router) POST(path string, handler context.HandlerFunc) {
	router.Add("POST", path, handler)
}

// router group
// might want handle that in sepearates files later on
// method, router, group
func (router *Router) Group(prefix string, middlewares []middleware.Middleware) *Group {
	group := &Group{
		prefix:      prefix,
		router:      router,
		middlewares: middlewares,
	}
	router.groups = append(router.groups, group)
	return group
}

func (router *Router) Static(prefix string, dir string) {
	fileServer := http.FileServer(http.Dir(dir))
	routePattern := strings.TrimSuffix(prefix, "/") + "/*filepath"
	router.Add("GET", routePattern, func(ctx *context.Context) {
		http.StripPrefix(prefix, fileServer).ServeHTTP(ctx.Writer, ctx.Request)
	})
}



func (router *Router) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	ctx := context.New(writer, request) // Renamed to avoid conflict

	// checking the routes
	for _, route := range router.routes {
		if route.Method == request.Method && utils.MatchPath(route.Path, request.URL.Path) {
			// extract path parameters and we add context
			params := utils.ExtractParams(route.Path, request.URL.Path)
			ctx.Params = params
			// exe handler
			route.Handler(ctx)
			return
		}
	}
	http.NotFound(writer, request) // notfound
}

func (group *Group) GET(path string, handler context.HandlerFunc) {
	fullPath := group.prefix + path
	wrappedHandler := group.wrapWithMiddleware(handler)
	group.router.GET(fullPath, wrappedHandler) // Corrected to GET
}

func (group *Group) POST(path string, handler context.HandlerFunc) {
	fullPath := group.prefix + path
	wrappedHandler := group.wrapWithMiddleware(handler)
	group.router.POST(fullPath, wrappedHandler)
}

func (group *Group) wrapWithMiddleware(handler context.HandlerFunc) context.HandlerFunc {
	currentHandler := handler // Use a new variable
	// reverse order to apply handlers
	for i := len(group.middlewares) - 1; i >= 0; i-- {
		currentHandler = group.middlewares[i](currentHandler)
	}
	return currentHandler
}