package builder

import (
	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/router"
	"net/http"
)

type testApp struct {
	router      *router.Router
	middlewares []context.Middleware
}

func newTestApp() *testApp {
	return &testApp{
		router:      router.New(),
		middlewares: make([]context.Middleware, 0),
	}
}
func (app *testApp) Use(middleware context.Middleware) {
	app.middlewares = append(app.middlewares, middleware)
}
func (app *testApp) GET(path string, handler context.HandlerFunc) {
	finalHandler := handler
	for i := len(app.middlewares) - 1; i >= 0; i-- {
		finalHandler = app.middlewares[i](finalHandler)
	}
	app.router.GET(path, finalHandler)
}
func (app *testApp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.router.ServeHTTP(w, r)
}
