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

type Config struct {
	// AppName is the name of the application.
	// Default: ""
	AppName string
	// ServerHeader is the value of the "Server" header.
	// Default: ""
	ServerHeader string
	// StrictRouting enables strict routing.
	// Default: false
	StrictRouting bool
	// CaseSensitive enables case-sensitive routing.
	// Default: false
	CaseSensitive bool
	// DisableStartupMessage disables the startup message.
	// Default: false
	DisableStartupMessage bool
}

type App struct {
	Router      *router.Router
	middlewares []Middleware
	Config      Config
}

func New(config ...Config) *App {
	cfg := Config{
		AppName:      "",
		ServerHeader: "",
	}

	if len(config) > 0 {
		cfg = config[0]
	}

	return &App{
		Router:      router.New(),
		middlewares: make([]Middleware, 0),
		Config:      cfg,
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
	if app.Config.ServerHeader != "" {
		w.Header().Set("Server", app.Config.ServerHeader)
	}
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
	if !app.Config.DisableStartupMessage {
		appName := "Goryu"
		if app.Config.AppName != "" {
			appName = app.Config.AppName
		}
		fmt.Printf("%s is running on %s\n", appName, addr)
	}
	return http.ListenAndServe(addr, app)
}
