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
		MaxRouteDepth:          32,    // SECURITY: Reasonable default limit
		MaxTotalRoutes:         10000, // SECURITY: Prevent memory exhaustion
		MaxParametersPerRoute:  32,    // SECURITY: Prevent complex route attacks
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
		// Return a dummy route to prevent nil pointer errors in non-panic modes
		dummyRoute := &Route{Method: method, Path: path, Handler: handler}
		dummyRoute.SetRouter(router)
		return dummyRoute
	}
	
	if path == "" || path[0] != '/' {
		router.handleRouterError("Add", "path must begin with '/'")
		// Return a dummy route to prevent nil pointer errors in non-panic modes
		dummyRoute := &Route{Method: method, Path: path, Handler: handler}
		dummyRoute.SetRouter(router)
		return dummyRoute
	}
	
	// SECURITY: Check route depth limit
	parts := parsePath(path)
	if len(parts) > router.Config.MaxRouteDepth {
		router.handleRouterError("Add", fmt.Sprintf("route depth (%d) exceeds maximum allowed (%d)", len(parts), router.Config.MaxRouteDepth))
		dummyRoute := &Route{Method: method, Path: path, Handler: handler}
		dummyRoute.SetRouter(router)
		return dummyRoute
	}
	
	// SECURITY: Check parameter count limit
	paramCount := 0
	for _, part := range parts {
		if strings.HasPrefix(part, ":") || strings.HasPrefix(part, "*") {
			paramCount++
		}
	}
	if paramCount > router.Config.MaxParametersPerRoute {
		router.handleRouterError("Add", fmt.Sprintf("route has too many parameters (%d) exceeds maximum allowed (%d)", paramCount, router.Config.MaxParametersPerRoute))
		dummyRoute := &Route{Method: method, Path: path, Handler: handler}
		dummyRoute.SetRouter(router)
		return dummyRoute
	}
	
	if _, ok := router.trees[method]; !ok {
		router.trees[method] = &node{}
	}
	
	// Store whether path has trailing slash
	hasTrailingSlash := len(path) > 1 && path[len(path)-1] == '/'
	
	// For StrictRouting mode, preserve the trailing slash distinction
	// Otherwise, normalize the path
	normalizedPath := path
	if !router.Config.StrictRouting && hasTrailingSlash {
		normalizedPath = path[:len(path)-1]
	}
	
	route := &Route{Method: method, Path: path, Handler: handler}
	route.SetRouter(router)
	router.trees[method].insert(normalizedPath, parts, 0, route, hasTrailingSlash, router)
	
	// SECURITY: Increment route counter
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
	method := r.Method

	// Store whether path has trailing slash before parsing
	hasTrailingSlash := len(path) > 1 && path[len(path)-1] == '/'

	// Try to find exact match first
	if tree, ok := router.trees[method]; ok {
		parts := parsePath(path)
		if foundNode, params, foundRoute := tree.find(parts, 0, hasTrailingSlash, router.Config.StrictRouting); foundNode != nil && foundRoute != nil {
			ctx.Params = params
			ctx.Route = foundRoute
			if handler, ok := foundRoute.Handler.(goryuctx.HandlerFunc); ok {
				handler(ctx)
			}
			return
		}
	}

	// Handle HEAD method fallback to GET if enabled
	if method == http.MethodHead && router.Config.EnableHEADFallback {
		if tree, ok := router.trees[http.MethodGet]; ok {
			parts := parsePath(path)
			if foundNode, params, foundRoute := tree.find(parts, 0, hasTrailingSlash, router.Config.StrictRouting); foundNode != nil && foundRoute != nil {
				ctx.Params = params
				ctx.Route = foundRoute
				if handler, ok := foundRoute.Handler.(goryuctx.HandlerFunc); ok {
					handler(ctx)
				}
				return
			}
		}
	}

	// Handle trailing slash redirect if enabled
	if router.Config.RedirectTrailingSlash {
		// Try alternate path (with or without trailing slash)
		var alternatePath string
		if hasTrailingSlash {
			alternatePath = path[:len(path)-1]
		} else if path != "/" {
			alternatePath = path + "/"
		}
		
		if alternatePath != "" {
			if tree, ok := router.trees[method]; ok {
				parts := parsePath(alternatePath)
				// Use opposite trailing slash for alternate path
				altHasSlash := !hasTrailingSlash
				if _, _, foundRoute := tree.find(parts, 0, altHasSlash, router.Config.StrictRouting); foundRoute != nil {
					// Redirect to alternate path
					code := http.StatusMovedPermanently // 301
					if method != http.MethodGet {
						code = http.StatusPermanentRedirect // 308
					}
					
					redirectURL := alternatePath
					if r.URL.RawQuery != "" {
						redirectURL += "?" + r.URL.RawQuery
					}
					
					http.Redirect(w, r, redirectURL, code)
					return
				}
			}
		}
	}

	// Handle automatic OPTIONS response
	if method == http.MethodOptions && router.Config.HandleOPTIONS {
		allowed := router.calculateAllowedMethods(path)
		if len(allowed) > 0 {
			w.Header().Set("Allow", strings.Join(allowed, ", "))
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	// Check if method not allowed
	if router.Config.HandleMethodNotAllowed {
		allowed := router.calculateAllowedMethods(path)
		if len(allowed) > 0 {
			router.MethodNotAllowed(ctx)
			return
		}
	}

	router.NotFound(ctx)
}


func (router *Router) calculateAllowedMethods(path string) []string {
	allowed := make([]string, 0)
	parts := parsePath(path)
	hasTrailingSlash := len(path) > 1 && path[len(path)-1] == '/'
	
	// Check exact path
	for method, tree := range router.trees {
		if _, _, foundRoute := tree.find(parts, 0, hasTrailingSlash, router.Config.StrictRouting); foundRoute != nil {
			allowed = append(allowed, method)
		}
	}
	
	// If RedirectTrailingSlash is enabled, also check alternate path
	if router.Config.RedirectTrailingSlash && len(allowed) == 0 {
		altHasSlash := !hasTrailingSlash
		for method, tree := range router.trees {
			if _, _, foundRoute := tree.find(parts, 0, altHasSlash, router.Config.StrictRouting); foundRoute != nil {
				allowed = append(allowed, method)
			}
		}
	}
	
	// Add HEAD if GET exists and EnableHEADFallback is true
	if router.Config.EnableHEADFallback {
		hasGet := false
		hasHead := false
		for _, m := range allowed {
			if m == "GET" {
				hasGet = true
			}
			if m == "HEAD" {
				hasHead = true
			}
		}
		if hasGet && !hasHead {
			allowed = append(allowed, "HEAD")
		}
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

func (g *Group) GET(path string, handler goryuctx.HandlerFunc) *Route     { return g.add("GET", path, handler) }
func (g *Group) POST(path string, handler goryuctx.HandlerFunc) *Route    { return g.add("POST", path, handler) }
func (g *Group) PUT(path string, handler goryuctx.HandlerFunc) *Route     { return g.add("PUT", path, handler) }
func (g *Group) DELETE(path string, handler goryuctx.HandlerFunc) *Route  { return g.add("DELETE", path, handler) }
func (g *Group) PATCH(path string, handler goryuctx.HandlerFunc) *Route   { return g.add("PATCH", path, handler) }
func (g *Group) HEAD(path string, handler goryuctx.HandlerFunc) *Route    { return g.add("HEAD", path, handler) }
func (g *Group) OPTIONS(path string, handler goryuctx.HandlerFunc) *Route { return g.add("OPTIONS", path, handler) }

func (g *Group) wrapWithMiddleware(handler goryuctx.HandlerFunc) goryuctx.HandlerFunc {
	for i := len(g.middlewares) - 1; i >= 0; i-- {
		handler = g.middlewares[i](handler)
	}
	return handler
}

