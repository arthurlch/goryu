package app

import (
	"strings"

	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/router"
)

func (app *App) Use(middleware context.Middleware) {
	app.middlewares = append(app.middlewares, middleware)
}

func (app *App) applyMiddleware(handler context.HandlerFunc) context.HandlerFunc {
	for i := len(app.middlewares) - 1; i >= 0; i-- {
		handler = app.middlewares[i](handler)
	}
	return handler
}

func (app *App) GET(path string, handler context.HandlerFunc) *router.Route {
	return app.Router.GET(path, app.applyMiddleware(handler))
}

func (app *App) POST(path string, handler context.HandlerFunc) *router.Route {
	return app.Router.POST(path, app.applyMiddleware(handler))
}

func (app *App) PUT(path string, handler context.HandlerFunc) *router.Route {
	return app.Router.PUT(path, app.applyMiddleware(handler))
}

func (app *App) DELETE(path string, handler context.HandlerFunc) *router.Route {
	return app.Router.DELETE(path, app.applyMiddleware(handler))
}

func (app *App) PATCH(path string, handler context.HandlerFunc) *router.Route {
	return app.Router.PATCH(path, app.applyMiddleware(handler))
}

func (app *App) HEAD(path string, handler context.HandlerFunc) *router.Route {
	return app.Router.HEAD(path, app.applyMiddleware(handler))
}

func (app *App) OPTIONS(path string, handler context.HandlerFunc) *router.Route {
	return app.Router.OPTIONS(path, app.applyMiddleware(handler))
}

func (app *App) Group(prefix string, middlewares ...context.Middleware) *router.Group {
	return app.Router.Group(prefix, middlewares...)
}
func (app *App) All(path string, handler context.HandlerFunc) *router.Route {
	return app.Router.ALL(path, app.applyMiddleware(handler))
}

func (app *App) Mount(prefix string, subApp *App) {
	subApp.mountPath = app.mountPath + prefix

	mountHandler := func(c *context.Context) {
		originalPath := c.Request.URL.Path
		c.Request.URL.Path = strings.TrimPrefix(originalPath, prefix)

		if c.Request.URL.Path == "" {
			c.Request.URL.Path = "/"
		}

		subApp.ServeHTTP(c.Writer, c.Request)

		c.Request.URL.Path = originalPath
	}

	routePath := prefix
	if !strings.HasSuffix(routePath, "/") {
		routePath += "/"
	}
	routePath += "*subpath"

	app.All(routePath, mountHandler)
}
