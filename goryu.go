package goryu

import (
	"fmt"
	"net/http"

	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/router"
)

type Context = context.Context
type HandlerFunc = context.HandlerFunc
type Middleware = context.Middleware

type App struct {
	Router      *router.Router
	middlewares []Middleware
}

func New() *App {
	return &App{
		Router:      router.New(),
		middlewares: make([]Middleware, 0),
	}
}

func (app *App) Use(middleware Middleware) {
	app.middlewares = append(app.middlewares, middleware)
}

func (app *App) applyMiddleware(handler HandlerFunc) HandlerFunc {
	appliedHandler := handler
	for i := len(app.middlewares) - 1; i >= 0; i-- {
		appliedHandler = app.middlewares[i](appliedHandler)
	}
	return appliedHandler
}

func (app *App) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		http.Error(w, "Failed to parse Form Data", http.StatusBadRequest)
		return
	}
	app.Router.ServeHTTP(w, req)
}

func (app *App) GET(path string, handler HandlerFunc) {
	app.Router.GET(path, app.applyMiddleware(handler))
}

func (app *App) POST(path string, handler HandlerFunc) {
	app.Router.POST(path, app.applyMiddleware(handler))
}

func (app *App) Run(addr string) error {
	fmt.Printf("Server is running on %s\n", addr)
	return http.ListenAndServe(addr, app)
}
