package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arthurlch/goryu/pkg/context"
	"github.com/arthurlch/goryu/pkg/middleware"
)

func TestRecoveryMiddleware(t *testing.T) {
	recoveryMiddleware := middleware.Recovery()

	panicHandler := func(c *context.Context) {
		panic("test panic")
	}

	nextHandlerCalled := false
	okHandler := func(c *context.Context) {
		nextHandlerCalled = true
		c.Text(http.StatusOK, "OK")
	}

	t.Run("panic is recovered", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/panic", nil)
		rr := httptest.NewRecorder()
		ctx := context.New(rr, req)

		handlerToTest := recoveryMiddleware(panicHandler)
		handlerToTest(ctx)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status %d after panic, got %d", http.StatusInternalServerError, rr.Code)
		}
		expectedBody := "Internal Server Error"
		if rr.Body.String() != expectedBody {
			t.Errorf("Expected body '%s' after panic, got '%s'", expectedBody, rr.Body.String())
		}
	})

	t.Run("no panic proceeds normally", func(t *testing.T) {
		nextHandlerCalled = false // reset for this sub-test
		req, _ := http.NewRequest("GET", "/ok", nil)
		rr := httptest.NewRecorder()
		ctx := context.New(rr, req)

		handlerToTest := recoveryMiddleware(okHandler)
		handlerToTest(ctx)

		if !nextHandlerCalled {
			t.Error("Next handler was not called when no panic occurred")
		}
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d for normal flow, got %d", http.StatusOK, rr.Code)
		}
		if rr.Body.String() != "OK" {
			t.Errorf("Expected body 'OK' for normal flow, got '%s'", rr.Body.String())
		}
	})
}