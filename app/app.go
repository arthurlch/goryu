package app

import (
	"context"
	"fmt"
	"net/http"

	goryu_context "github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/router"
)

type App struct {
	Router      *router.Router
	server      *http.Server
	middlewares []goryu_context.Middleware
	Config      Config
	mountPath   string
}

type Config struct {
	AppName               string
	ServerHeader          string
	StrictRouting         bool
	CaseSensitive         bool
	DisableStartupMessage bool
}

func New(config ...Config) *App {
	cfg := Config{} // Default config
	if len(config) > 0 {
		cfg = config[0]
	}

	app := &App{
		Router:      router.New(),
		middlewares: make([]goryu_context.Middleware, 0),
		Config:      cfg,
	}

	return app
}

func (app *App) Listen(addr string) error {
	if !app.Config.DisableStartupMessage {
		appName := "Goryu"
		if app.Config.AppName != "" {
			appName = app.Config.AppName
		}
		fmt.Printf("🚀 %s is running on %s\n", appName, addr)
	}

	app.server = &http.Server{Addr: addr, Handler: app}
	return app.server.ListenAndServe()
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if app.Config.ServerHeader != "" {
		w.Header().Set("Server", app.Config.ServerHeader)
	}
	app.Router.ServeHTTP(w, r)
}

func (app *App) Shutdown() error {
	if app.server == nil {
		return fmt.Errorf("server is not running")
	}
	return app.server.Shutdown(context.Background())
}

func (app *App) ShutdownWithContext(ctx context.Context) error {
	if app.server == nil {
		return fmt.Errorf("server is not running")
	}
	return app.server.Shutdown(ctx)
}

func (app *App) Server() *http.Server {
	return app.server
}

func (app *App) Handler() http.Handler {
	return app
}

func (app *App) MountPath() string {
	return app.mountPath
}
