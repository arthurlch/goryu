package router

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/route"
)

// Route type alias to the shared route package
type Route = route.Route

// SetRouteName gives a name to the route, allowing for URL reversal.
// This method is called by Route.SetName() to properly register named routes.
func (router *Router) SetRouteName(r *Route, name string) *Route {
	if _, exists := router.namedRoutes[name]; exists {
		router.handleRouterError("SetName", fmt.Sprintf("Route with name '%s' already exists", name))
		return r
	}
	r.Name = name
	router.namedRoutes[name] = r
	return r
}

// Group allows for grouping routes with a common prefix and middlewares.
type Group struct {
	prefix      string
	middlewares []goryuctx.Middleware
	router      *Router
}

// RouterConfig defines configuration options for the router
type RouterConfig struct {
	// StrictRouting when true, treats /foo and /foo/ as different paths
	// Default: false
	StrictRouting bool
	// RedirectTrailingSlash enables automatic redirection of trailing slashes.
	// For example, /foo/ redirects to /foo (or vice versa based on route definition)
	// Default: true
	RedirectTrailingSlash bool
	// RedirectFixedPath attempts to fix the case and trailing slashes of the request path
	// Default: false
	RedirectFixedPath bool
	// HandleMethodNotAllowed responds with 405 when a route exists but not for the requested method
	// Default: true
	HandleMethodNotAllowed bool
	// HandleOPTIONS automatically handles OPTIONS requests
	// Default: true
	HandleOPTIONS bool
	// EnableHEADFallback allows HEAD requests to fall back to GET handlers
	// Default: true
	EnableHEADFallback bool
	// ErrorMode defines how to handle router registration errors
	// Default: RouterErrorModePanic (backward compatible)
	ErrorMode RouterErrorMode
	// SECURITY: MaxRouteDepth limits the maximum depth of route paths to prevent memory exhaustion
	// Default: 32
	MaxRouteDepth int
	// SECURITY: MaxTotalRoutes limits the total number of routes to prevent memory exhaustion
	// Default: 10000
	MaxTotalRoutes int
	// SECURITY: MaxParametersPerRoute limits the number of parameters in a single route
	// Default: 32
	MaxParametersPerRoute int
}

// Router is the main router struct. It holds the routing tree and configuration.
type Router struct {
	trees            map[string]*node
	namedRoutes      map[string]*Route
	NotFound         goryuctx.HandlerFunc
	MethodNotAllowed goryuctx.HandlerFunc
	PanicHandler     func(http.ResponseWriter, *http.Request, interface{})
	Config           RouterConfig
	totalRoutes      int // SECURITY: Track total routes for limits
}

// New creates a new Router instance with default handlers.
func New(config ...RouterConfig) *Router {
	cfg := RouterConfig{
		RedirectTrailingSlash:  true,
		RedirectFixedPath:      false,
		HandleMethodNotAllowed: true,
		HandleOPTIONS:          true,
		EnableHEADFallback:     true,
		ErrorMode:              RouterErrorModePanic, // Backward compatible default
		MaxRouteDepth:          32,                   // SECURITY: Reasonable default limit
		MaxTotalRoutes:         10000,                // SECURITY: Prevent memory exhaustion
		MaxParametersPerRoute:  32,                   // SECURITY: Prevent complex route attacks
	}

	if len(config) > 0 {
		// Merge the provided config with defaults
		userCfg := config[0]

		// Override only non-zero values to preserve defaults
		if userCfg.ErrorMode != 0 {
			cfg.ErrorMode = userCfg.ErrorMode
		}
		if userCfg.MaxRouteDepth != 0 {
			cfg.MaxRouteDepth = userCfg.MaxRouteDepth
		}
		if userCfg.MaxTotalRoutes != 0 {
			cfg.MaxTotalRoutes = userCfg.MaxTotalRoutes
		}
		if userCfg.MaxParametersPerRoute != 0 {
			cfg.MaxParametersPerRoute = userCfg.MaxParametersPerRoute
		}

		// Boolean fields need special handling since false is a valid value
		cfg.RedirectTrailingSlash = userCfg.RedirectTrailingSlash
		cfg.RedirectFixedPath = userCfg.RedirectFixedPath
		cfg.HandleMethodNotAllowed = userCfg.HandleMethodNotAllowed
		cfg.HandleOPTIONS = userCfg.HandleOPTIONS
		cfg.EnableHEADFallback = userCfg.EnableHEADFallback
		cfg.StrictRouting = userCfg.StrictRouting
	}

	r := &Router{
		trees:       make(map[string]*node),
		namedRoutes: make(map[string]*Route),
		Config:      cfg,
	}
	r.NotFound = func(ctx *goryuctx.Context) {
		http.NotFound(ctx.Writer, ctx.Request)
	}
	r.MethodNotAllowed = func(ctx *goryuctx.Context) {
		http.Error(ctx.Writer, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
	r.PanicHandler = func(w http.ResponseWriter, r *http.Request, err interface{}) {
		log.Printf("Handler panicked: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
	return r
}

// Add registers a new route with the given method, path, and handler.
func (router *Router) Add(method, path string, handler goryuctx.HandlerFunc) *Route {
	// SECURITY: Check total route limit
	if router.totalRoutes >= router.Config.MaxTotalRoutes {
		router.handleRouterError("Add", fmt.Sprintf("exceeded maximum number of routes (%d)", router.Config.MaxTotalRoutes))
		// Return dummy route
		return &Route{Method: method, Path: path}
	}

	if path == "" || path[0] != '/' {
		router.handleRouterError("Add", "path must begin with '/'")
		return &Route{Method: method, Path: path}
	}

	if _, ok := router.trees[method]; !ok {
		router.trees[method] = &node{}
	}

	route := &Route{Method: method, Path: path, Handler: handler}
	route.SetRouter(router)

	// Delegate insertion to Radix Tree node
	if err := router.trees[method].addRoute(path, handler, route); err != nil {
		router.handleRouterError("Add", err.Error())
		// If logging/silent, we must return something valid-ish or the dummy
		// Actually if error, route is not added.
	}

	// Parse parameters for the Route object
	// This allows Context.Param() to map values (collected during traversal) to names
	for i, l := 0, len(path); i < l; i++ {
		if path[i] == ':' {
			j := i + 1
			for j < l && path[j] != '/' {
				j++
			}
			route.ParamNames = append(route.ParamNames, path[i+1:j])
			i = j - 1 // -1 because loop increments
		} else if path[i] == '*' {
			j := i + 1
			for j < l && path[j] != '/' {
				j++
			}
			route.ParamNames = append(route.ParamNames, path[i+1:j])
			i = j - 1
		}
	}

	router.totalRoutes++
	return route
}

// --- HTTP Method Helpers ---
func (router *Router) GET(path string, handler goryuctx.HandlerFunc) *Route {
	return router.Add("GET", path, handler)
}
func (router *Router) POST(path string, handler goryuctx.HandlerFunc) *Route {
	return router.Add("POST", path, handler)
}
func (router *Router) PUT(path string, handler goryuctx.HandlerFunc) *Route {
	return router.Add("PUT", path, handler)
}
func (router *Router) DELETE(path string, handler goryuctx.HandlerFunc) *Route {
	return router.Add("DELETE", path, handler)
}
func (router *Router) PATCH(path string, handler goryuctx.HandlerFunc) *Route {
	return router.Add("PATCH", path, handler)
}
func (router *Router) HEAD(path string, handler goryuctx.HandlerFunc) *Route {
	return router.Add("HEAD", path, handler)
}
func (router *Router) OPTIONS(path string, handler goryuctx.HandlerFunc) *Route {
	return router.Add("OPTIONS", path, handler)
}

// RouteCollection represents a collection of routes (used by ALL method)
type RouteCollection struct {
	Routes []*Route
	Path   string
}

// SetName sets a name for all routes in the collection with method suffix
func (rc *RouteCollection) SetName(name string) *RouteCollection {
	for _, route := range rc.Routes {
		// Add method suffix to make names unique
		methodSuffix := strings.ToLower(route.Method)
		route.SetName(fmt.Sprintf("%s_%s", name, methodSuffix))
	}
	return rc
}

// ALL registers a handler for all HTTP methods
// Returns a RouteCollection that contains all created routes
func (router *Router) ALL(path string, handler goryuctx.HandlerFunc) *RouteCollection {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	routes := make([]*Route, 0, len(methods))

	for _, method := range methods {
		route := router.Add(method, path, handler)
		routes = append(routes, route)
	}

	return &RouteCollection{
		Routes: routes,
		Path:   path,
	}
}

// Group creates a new route group with a common prefix.
func (router *Router) Group(prefix string, middlewares ...goryuctx.Middleware) *Group {
	return &Group{
		prefix:      prefix,
		router:      router,
		middlewares: middlewares,
	}
}

// Reverse generates a URL for a named route.
func (router *Router) Reverse(name string, params ...interface{}) string {
	route, ok := router.namedRoutes[name]
	if !ok {
		return ""
	}

	path := route.Path
	paramIndex := 0
	// This is a simple replacement; more complex scenarios might need a more robust solution.
	for _, part := range strings.Split(path, "/") {
		if strings.HasPrefix(part, ":") && paramIndex < len(params) {
			path = strings.Replace(path, part, fmt.Sprint(params[paramIndex]), 1)
			paramIndex++
		}
	}
	return path
}

// ServeHTTP makes the router implement the http.Handler interface.
func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil && router.PanicHandler != nil {
			router.PanicHandler(w, r, err)
		}
	}()

	ctx := goryuctx.NewContext(w, r)
	defer ctx.Release()
	path := r.URL.Path

	if root := router.trees[r.Method]; root != nil {
		// RADIX OPTIMIZATION: Direct lookup without recursion or stack
		handler, route, params, tsr := root.getValue(path, ctx.ParamValues, false)

		if handler != nil {
			ctx.ParamValues = params
			ctx.Route = route
			handler(ctx)
			return
		} else if method := r.Method; method != "CONNECT" && path != "/" {
			// Redirect Trailing Slash
			if router.Config.RedirectTrailingSlash && tsr {
				code := 301 // Permanent redirect, request with GET method
				if method != "GET" {
					code = 308
				}

				if len(path) > 1 && path[len(path)-1] == '/' {
					r.URL.Path = path[:len(path)-1]
				} else {
					r.URL.Path = path + "/"
				}
				http.Redirect(w, r, r.URL.String(), code)
				return
			}
		}

		// Try to fix the request path (StrictRouting disabled)
		if !router.Config.StrictRouting && path != "/" {
			fixedPath := path
			if len(path) > 1 && path[len(path)-1] == '/' {
				fixedPath = path[:len(path)-1]
			} else {
				fixedPath = path + "/"
			}

			handler, route, params, _ := root.getValue(fixedPath, nil, false)
			if handler != nil {
				ctx.ParamValues = params
				ctx.Route = route
				handler(ctx)
				return
			}
		}
	}

	// Fallback for HEAD -> GET (outside of tree check loop)
	if r.Method == "HEAD" && router.Config.EnableHEADFallback {
		// If we haven't found a HEAD handler (handler is nil above), try GET
		// But note: if we found a HEAD tree but no handler, we are here.
		// If we didn't find a HEAD tree, we are here.
		// So checking GET tree is correct.
		if root := router.trees["GET"]; root != nil {
			handler, route, params, _ := root.getValue(path, ctx.ParamValues, false)
			if handler != nil {
				ctx.ParamValues = params
				ctx.Route = route
				handler(ctx)
				return
			}
		}
	}

	// Handle 405 Method Not Allowed
	if router.Config.HandleMethodNotAllowed {
		allowed := router.calculateAllowedMethods(path)
		if len(allowed) > 0 {
			w.Header().Set("Allow", strings.Join(allowed, ", "))
			if r.Method == "OPTIONS" && router.Config.HandleOPTIONS {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if router.MethodNotAllowed != nil {
				router.MethodNotAllowed(ctx)
			} else {
				http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			}
			return
		}
	}

	// 404 Not Found
	if router.NotFound != nil {
		router.NotFound(ctx)
	} else {
		http.NotFound(w, r)
	}
}

// parsePath splits a URL path into segments using the provided buffer
// Please don't mind the linter for unused ! 
// Used by Add() to parse structure (legacy helper, maybe unused now but keeping for safety)
func parsePath(path string, buf []string) []string {
	if path == "/" {
		return buf
	}
	path = strings.Trim(path, "/")
	if path == "" {
		return buf
	}
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			buf = append(buf, path[start:i])
			start = i + 1
		}
	}
	buf = append(buf, path[start:])
	return buf
}

// calculateAllowedMethods returns the allowed methods for a given path
func (router *Router) calculateAllowedMethods(path string) []string {
	allowed := make([]string, 0)

	for method, root := range router.trees {
		handler, _, _, _ := root.getValue(path, nil, false)
		if handler != nil {
			allowed = append(allowed, method)
		}
	}

	if len(allowed) > 0 {
		allowed = append(allowed, "OPTIONS")
	}

	sort.Strings(allowed)
	return allowed
}

// --- Group Methods ---

func (g *Group) Group(prefix string, middlewares ...goryuctx.Middleware) *Group {
	return &Group{
		prefix:      g.prefix + prefix,
		router:      g.router,
		middlewares: append(g.middlewares, middlewares...),
	}
}

func (g *Group) add(method, path string, handler goryuctx.HandlerFunc) *Route {
	fullPath := g.prefix + path
	wrappedHandler := g.wrapWithMiddleware(handler)
	return g.router.Add(method, fullPath, wrappedHandler)
}

func (g *Group) GET(path string, handler goryuctx.HandlerFunc) *Route {
	return g.add("GET", path, handler)
}
func (g *Group) POST(path string, handler goryuctx.HandlerFunc) *Route {
	return g.add("POST", path, handler)
}
func (g *Group) PUT(path string, handler goryuctx.HandlerFunc) *Route {
	return g.add("PUT", path, handler)
}
func (g *Group) DELETE(path string, handler goryuctx.HandlerFunc) *Route {
	return g.add("DELETE", path, handler)
}
func (g *Group) PATCH(path string, handler goryuctx.HandlerFunc) *Route {
	return g.add("PATCH", path, handler)
}
func (g *Group) HEAD(path string, handler goryuctx.HandlerFunc) *Route {
	return g.add("HEAD", path, handler)
}
func (g *Group) OPTIONS(path string, handler goryuctx.HandlerFunc) *Route {
	return g.add("OPTIONS", path, handler)
}

func (g *Group) wrapWithMiddleware(handler goryuctx.HandlerFunc) goryuctx.HandlerFunc {
	for i := len(g.middlewares) - 1; i >= 0; i-- {
		handler = g.middlewares[i](handler)
	}
	return handler
}

// Use adds middleware to the group
func (g *Group) Use(middlewares ...goryuctx.Middleware) {
	g.middlewares = append(g.middlewares, middlewares...)
}

// RouteInfo holds information about a registered route
type RouteInfo struct {
	Method string
	Path   string
	Name   string
}

// Routes returns a list of all registered routes
func (router *Router) Routes() []RouteInfo {
	var routes []RouteInfo

	// Walk through all trees
	for method, tree := range router.trees {
		tree.walk(func(path string, route *Route) {
			routes = append(routes, RouteInfo{
				Method: method,
				Path:   path,
				Name:   route.Name,
			})
		})
	}

	// Sort for consistent output
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path == routes[j].Path {
			return routes[i].Method < routes[j].Method
		}
		return routes[i].Path < routes[j].Path
	})

	return routes
}
