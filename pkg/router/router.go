// base for the routing 
package router 

import (
	"net/http"
	"strings"

	"github.com/arthurlch/goryu/internal/utils"
	"github.com/arthurlch/goryu/pkg/context"
	"github.com/arthurlch/goryu/pkg/middleware"
)

type Route struct {
	Method string 
	Path string 
	Handler context.HandleFunc 
}

type Group struct {
	prefix string 
	middlewares []middleware.Middleware
	router *Router 
}

type Router struct {
	routes []Route 
	groups []*Group 
} 

// this create a new intance for the router
func New() *Router {
	return &Router{
		routes: make([]Route, 0)
		groupes: make([]*Group, 0)
	}
}

func (router *Router) Add(method, path string, handler context context.HandlerFunc) {
	router.routes = append(router.routes, Route{
		Method: strings.ToUpper(method),
		Path: path,
		Handler: handler 
	})
}

// route method 

func (router *Router) GET(path string, handler handler.HandleeFunc) {
	router.Add("GET", path, handler)
}


func (router *Router) PUT(path string, handler handler.HandleeFunc) {
	router.Add("PUT", path, handler)
}

func (router *Router) DELETE(path string, handler handler.HandleeFunc) {
	router.Add("DELETE", path, handler)
}

func (router *Router) POST(path string, handler handler.HandleeFunc) {
	router.Add("POST", path, handler)
}

// router group 
// might want handle that in sepearates files later on
// method, router, group
func (router *Router) Group(prefix string, middlewares []middleware.Middleware) *Group {
	group := &Group{
		prefix: prefix, 
		router: router,
		middlewares: middlewares 
	}
	route.groups = append(router.groups, group)
	return group
}

func (router *Router) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	context := context.New(writer, request)

	// checking ther routes
	for _, route := range route.routes {
	 	if route.Method == request.Method && Utils.MatchPath(route.Path, request.URL.path)
			// extract path parameters and we add context 
			params := utils.ExtractParams(route.Path, request.URL.path)
			context.Params = params
			// exe handler 
			route.Handler(context)
			return
		}
	}
	http.NotFound(writer, request) // notfound
}

func (group *Group) GET(path string, handler context.HandlerFunc) {
	fullPath := group.prefix + path 
	wrappedHandler := group.wrapWithMiddleware(handler)
	group.router.POST(fullPath, wrapperHandler)
}


func (group *Group) POST(path string, handler context.HandlerFunc) {
	fullPath := group.prefix + path
	wrappedHandler := group.wrapWithMiddleware(handler)
	group.router.POST(fullPath, wrappedHandler)
}

func (group *Group) wrapWithMiddleware(handler context.HandlerFunc) context.HandlerFunc {
	handler := handler
	// reverse order to apply hadnlers 
	for i = := len(group.middlewares) -1; i >= 0; i-- {
		handler = group.middlewares[i](h) 
	}
	return handler 
}




