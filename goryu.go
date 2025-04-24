package goryu

import (
	"github.com/yourusername/goframe/pkg/app"
	"github.com/yourusername/goframe/pkg/context"
	"github.com/yourusername/goframe/pkg/middleware"
)

type App = app.App 
type Context = context.Context
type HandlerFunc = context.HandlerFunc
type Middleware = middleware.Middleware 

func newApp() *App {
	return app.New()
}

func Default() *App {
	app := app.New()
	app.Use(middleware.Logger())
	app.Use(middleware.Recovery())
	return a 
}

func Logger() Middleware {
	return middleware.Logger() 
}

func Recovery() Middleware {
	return middleware.Recovery()
}

