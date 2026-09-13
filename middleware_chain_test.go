package goryu

import (
	"net/http/httptest"
	"testing"

	"github.com/arthurlch/goryu/middleware/cors"
)

func TestGlobalMiddlewareRunsOnPreflightAndGroups(t *testing.T) {
	app := New(Config{EnableMonitoring: boolPtr(false)})
	app.Use(cors.New(cors.Config{AllowOrigins: []string{"https://app.example"}}))
	app.POST("/api", func(c *Ctx) { c.Text(200, "ok") })

	t.Run("preflight gets CORS headers", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api", nil)
		req.Header.Set("Origin", "https://app.example")
		req.Header.Set("Access-Control-Request-Method", "POST")
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)
		if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example" {
			t.Fatalf("preflight missing CORS header, got %q", got)
		}
	})

	t.Run("group routes get global middleware", func(t *testing.T) {
		g := app.Group("/v1")
		g.GET("/ping", func(c *Ctx) { c.Text(200, "pong") })
		req := httptest.NewRequest("GET", "/v1/ping", nil)
		req.Header.Set("Origin", "https://app.example")
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)
		if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example" {
			t.Fatalf("group route did not run global CORS middleware, got %q", got)
		}
	})
}

func boolPtr(b bool) *bool { return &b }
