package router

import (
	"net/http"
	"strings"

	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/internal/utils"
)

type Route struct {
	Method  string
	Path    string
	Handler context.HandlerFunc
}

type Group struct {
	prefix      string
	middlewares []context.Middleware
	router      *Router
}

type Router struct {
	routes []Route
	groups []*Group
}

func New() *Router {
	return &Router{
		routes: make([]Route, 0),
		groups: make([]*Group, 0),
	}
}

func (router *Router) Add(method, path string, handler context.HandlerFunc) {
	router.routes = append(router.routes, Route{
		Method:  strings.ToUpper(method),
		Path:    path,
		Handler: handler,
	})
}

func (router *Router) GET(path string, handler context.HandlerFunc) {
	router.Add("GET", path, handler)
}

func (router *Router) POST(path string, handler context.HandlerFunc) {
	router.Add("POST", path, handler)
}

func (router *Router) Group(prefix string, middlewares []context.Middleware) *Group {
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
	ctx := context.NewContext(writer, request)
	for _, route := range router.routes {
		if route.Method == request.Method && utils.MatchPath(route.Path, request.URL.Path) {
			params := utils.ExtractParams(route.Path, request.URL.Path)
			ctx.Params = params
			route.Handler(ctx)
			return
		}
	}
	http.NotFound(writer, request)
}

func (group *Group) GET(path string, handler context.HandlerFunc) {
	fullPath := group.prefix + path
	wrappedHandler := group.wrapWithMiddleware(handler)
	group.router.GET(fullPath, wrappedHandler)
}

func (group *Group) POST(path string, handler context.HandlerFunc) {
	fullPath := group.prefix + path
	wrappedHandler := group.wrapWithMiddleware(handler)
	group.router.POST(fullPath, wrappedHandler)
}

func (group *Group) wrapWithMiddleware(handler context.HandlerFunc) context.HandlerFunc {
	currentHandler := handler
	for i := len(group.middlewares) - 1; i >= 0; i-- {
		currentHandler = group.middlewares[i](currentHandler)
	}
	return currentHandler
}
