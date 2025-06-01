package app_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arthurlch/goryu/pkg/app"
	"github.com/arthurlch/goryu/pkg/context"
)

func TestSimpleGETRequest(t *testing.T) {
	goryuApp := app.New()

	goryuApp.GET("/test", func(c *context.Context) {
		c.Text(http.StatusOK, "Hello, Goryu!")
	})

	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatalf("could not create request: %v", err)
	}

	rr := httptest.NewRecorder()

	goryuApp.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := "Hello, Goryu!"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestMiddlewareExecution(t *testing.T) {
	goryuApp := app.New()
	var middlewareExecuted bool

	testMiddleware := func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			middlewareExecuted = true
			next(c)
		}
	}

	goryuApp.Use(testMiddleware)

	goryuApp.GET("/middleware-test", func(c *context.Context) {
		c.Text(http.StatusOK, "Middleware Test Done")
	})

	req, _ := http.NewRequest("GET", "/middleware-test", nil)
	rr := httptest.NewRecorder()
	goryuApp.ServeHTTP(rr, req)

	if !middlewareExecuted {
		t.Errorf("middleware was not executed")
	}
}

func TestRouteNotFound(t *testing.T) {
	goryuApp := app.New()

	req, _ := http.NewRequest("GET", "/nonexistent-route", nil)
	rr := httptest.NewRecorder()
	goryuApp.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code for nonexistent route: got %v want %v",
			status, http.StatusNotFound)
	}
}