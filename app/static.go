package app

import (
	"net/http"
	"strings"
	"time"

	"github.com/arthurlch/goryu/context"
)

type Static struct {
	Browse        bool          `json:"browse"`
	Index         string        `json:"index"`
	CacheDuration time.Duration `json:"cache_duration"`
	MaxAge        int           `json:"max_age"`
}

func (app *App) Static(prefix, root string, config ...Static) {
	fs := http.FileServer(http.Dir(root))

	handler := func(c *context.Context) {
		http.StripPrefix(prefix, fs).ServeHTTP(c.Writer, c.Request)
	}

	routePath := prefix
	if !strings.HasSuffix(routePath, "/") {
		routePath += "/"
	}
	routePath += "*filepath"

	app.Router.GET(routePath, handler)
}
