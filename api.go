package goryu

import (
	"fmt"
	"net/http"
	"time"

	"github.com/arthurlch/goryu/config/builder"
	"github.com/arthurlch/goryu/plugins"
	routebuilder "github.com/arthurlch/goryu/router/builder"
)

func (app *App) Route() *routebuilder.SimpleRouteBuilder {
	return routebuilder.NewSimpleRouteBuilder(&appAdapter{app})
}

type appAdapter struct {
	*App
}

func (a *appAdapter) GetRouter() interface{} {
	return a.Router
}

func (a *appAdapter) ApplyMiddleware(handler Handler) Handler {
	return a.applyMiddleware(handler)
}

func Configuration() *builder.Builder {
	return builder.New()
}

func (app *App) ConfigureWith(config *builder.Config) *App {
	// Apply server configuration
	if config.Server.Port != 0 {
		app.Config.ServerPort = config.Server.Port
	}

	if config.Server.Host != "" {
		app.Config.ServerHost = config.Server.Host
	}

	if config.App.Name != "" {
		app.Config.AppName = config.App.Name
	}

	if config.App.ServerHeader != "" {
		app.Config.ServerHeader = config.App.ServerHeader
	}

	app.Config.DisableStartupMessage = config.App.DisableStartupMessage

	redirectTrailingSlash := config.Router.RedirectTrailingSlash
	app.Config.RedirectTrailingSlash = &redirectTrailingSlash

	enableHEADFallback := config.Router.EnableHEADFallback
	app.Config.EnableHEADFallback = &enableHEADFallback

	app.Config.StrictRouting = config.Router.StrictRouting
	app.Config.CaseSensitive = config.Router.CaseSensitive

	// Update router configuration directly
	app.Router.Config.StrictRouting = config.Router.StrictRouting
	app.Router.Config.RedirectTrailingSlash = config.Router.RedirectTrailingSlash
	app.Router.Config.HandleMethodNotAllowed = config.Router.HandleMethodNotAllowed
	app.Router.Config.HandleOPTIONS = config.Router.HandleOPTIONS
	app.Router.Config.EnableHEADFallback = config.Router.EnableHEADFallback
	app.Router.Config.MaxRouteDepth = config.Limits.MaxRouteDepth
	app.Router.Config.MaxTotalRoutes = config.Limits.MaxTotalRoutes
	app.Router.Config.MaxParametersPerRoute = config.Limits.MaxParametersPerRoute

	app.config = config

	return app
}

func NewWithConfig(builderFn func(*builder.Builder) *builder.Builder) *App {
	config, err := builderFn(builder.New()).Build()
	if err != nil {
		panic("Invalid configuration: " + err.Error())
	}

	app := New()
	return app.ConfigureWith(config)
}

func (app *App) GetConfig() *builder.Config {
	return app.config
}

func (app *App) Start() error {
	addr := app.getListenAddress()
	return app.Run(addr)
}

func (app *App) StartTLS(certFile, keyFile string) error {
	addr := app.getListenAddress()

	if !app.Config.DisableStartupMessage {
		appName := "Goryu"
		if app.Config.AppName != "" {
			appName = app.Config.AppName
		}
		fmt.Printf("🔒 %s is running on https://%s\n", appName, addr)
	}

	app.server = &http.Server{Addr: addr, Handler: app}
	return app.server.ListenAndServeTLS(certFile, keyFile)
}

func (app *App) getListenAddress() string {
	host := app.Config.ServerHost
	port := app.Config.ServerPort

	if port == 0 {
		port = 3000
	}

	if host == "" {
		return fmt.Sprintf(":%d", port)
	}

	return fmt.Sprintf("%s:%d", host, port)
}

func Logger() *plugins.LoggerBuilder {
	return plugins.NewLoggerBuilder()
}

func Recovery() *plugins.RecoveryBuilder {
	return plugins.NewRecoveryBuilder()
}

func CORS() *plugins.CORSBuilder {
	return plugins.NewCORSBuilder()
}

func RateLimit(max int, duration ...interface{}) *plugins.RateLimitBuilder {
	builder := plugins.NewRateLimitBuilder()

	// Handle different parameter patterns:
	// RateLimit(100) - 100 per minute (default)
	// RateLimit(100, time.Minute) - 100 per minute
	// RateLimit(100, "1m") - 100 per minute (string duration)
	// RateLimit(100, 60) - 100 per 60 seconds

	builder.Max(max)

	if len(duration) > 0 {
		switch d := duration[0].(type) {
		case string:
			if parsedDuration, err := time.ParseDuration(d); err == nil {
				builder.Duration(parsedDuration)
			}
		case int:
			builder.Duration(time.Duration(d) * time.Second)
		case time.Duration:
			builder.Duration(d)
		}
	}

	return builder
}

func Plugin(name string) (plugins.Builder, bool) {
	return plugins.Get(name)
}

func MustPlugin(name string) plugins.Builder {
	plugin, exists := plugins.Get(name)
	if !exists {
		panic("middleware plugin not found: " + name)
	}
	return plugin
}

// ListPlugins returns all registered plugin names
func ListPlugins() []string {
	return plugins.List()
}

// RegisterPlugin registers a new middleware plugin
func RegisterPlugin(name string, factory func() plugins.Builder) {
	plugins.Register(name, factory)
}
