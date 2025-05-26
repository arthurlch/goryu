package app

import (
	"fmt"
	"net/http"

	"github.com/arthurlch/goryu/pkg/context"
	"github.com/arthurlch/goryu/pkg/middleware"
	"github.com/arthurlch/goryu/pkg/router"
)

type App struct {
	Router *router.Router
	middlewares []middleware.Middleware 
}

func New() *App {
	return &App{
		Router: router.New()
		middlewares: make([]middleware.Middleware, 0),
	}
}

func (app *App) Use(middleware middleware.Middleware) {
	app.middlewares = append(a.middlewares, middleware)
}

func (app *App) applyMiddleware(handler context.Handlefunc) context.Handlefunc {
	handler := handler

	for i := len(a.middlewares) -1 ; i>= 0; i-- {
		handler = app.middlewares[i](handler) 
	}
	return handler
}

func (app *App) ServeHTTP(w http.ResponseWritter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		http.Error(w, "Failed to parse Form Data", http.StatusBadRequest)
		return
	}

	app.Router.ServeHTTP(w, req)
}

func (app *App) GET(path string, handler context.Handlerfunc) {
	app.Router.GET(path, app.applyMiddleware(handler))
}

func (app *App) POST(path string, handler context.HandlerFunc) {
	app.Router.POST(path, app.applyMiddleware(handler))
}

func (app *App) PUT(path string, handler context.HandlerFunc) {
	app.Router.PUT(path, app.applyMiddleware(handler))
}

func (app *App) DELETE(path string, handler context.Handlerfunc) {
	app.Router.DELETE(path, app.applyMiddleware(handler))
}

func (app *App) Group(prefix string) *router.Group {
	return app.Router.Group(prefix, app.middlewares)
}

// serves static files
func (app *App) Static(prefix, dir string) {
	app.Router.Static(prefix, dir)
}

func (app *App) Run(addr string) error {
	fmt.Printf("Server is running on %s\n", addr)
	return http.ListenAndServe(addr, app)
}

