package goryu

import (
	"github.com/arthurlch/goryu/pkg/app"
	"github.com/arthurlch/goryu/pkg/context"
	"github.com/arthurlch/goryu/pkg/middleware"
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
	return app
}

func Logger() Middleware {
	return middleware.Logger() 
}

func Recovery() Middleware {
	return middleware.Recovery()
}

