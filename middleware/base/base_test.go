package base

import (
	"bytes"
	"errors"
	context "github.com/arthurlch/goryu/goryuctx"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStandardMiddleware(t *testing.T) {
	t.Run("Normal execution", func(t *testing.T) {
		called := false
		middleware := StandardMiddleware("Test", BaseConfig{}, func(c *context.Context) error {
			called = true
			c.SetHeader("X-Test", "true")
			return nil
		})
		handler := middleware(func(c *context.Context) {
			c.Text(200, "success")
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		c := context.NewContext(w, req)
		handler(c)
		if !called {
			t.Error("Expected middleware handler to be called")
		}
		if w.Header().Get("X-Test") != "true" {
			t.Error("Expected X-Test header to be set")
		}
		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
	t.Run("Skip middleware", func(t *testing.T) {
		called := false
		middleware := StandardMiddleware("Test", BaseConfig{
			Skip: func(c *context.Context) bool {
				return c.Request.Header.Get("Skip") == "true"
			},
		}, func(c *context.Context) error {
			called = true
			return nil
		})
		handler := middleware(func(c *context.Context) {
			c.Text(200, "success")
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Skip", "true")
		c := context.NewContext(w, req)
		handler(c)
		if called {
			t.Error("Expected middleware handler to be skipped")
		}
		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
	t.Run("Error handling", func(t *testing.T) {
		var logBuf bytes.Buffer
		logger := log.New(&logBuf, "", 0)
		middleware := StandardMiddleware("Test", BaseConfig{
			Logger: logger,
			ErrorHandler: func(c *context.Context, err error, middlewareName string) {
				logger.Printf("[MIDDLEWARE:%s] Error: %v", middlewareName, err)
				DefaultErrorHandler(c, err, middlewareName)
			},
		}, func(c *context.Context) error {
			return MiddlewareError{
				Middleware: "Test",
				Err:        errors.New("test error"),
				StatusCode: http.StatusBadRequest,
			}
		})
		handler := middleware(func(c *context.Context) {
			c.Text(200, "should not reach here")
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		c := context.NewContext(w, req)
		handler(c)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
		if !strings.Contains(logBuf.String(), "test error") {
			t.Errorf("Expected error to be logged, got: %s", logBuf.String())
		}
	})
}
func TestPostProcessMiddleware(t *testing.T) {
	t.Run("Pre and post processing", func(t *testing.T) {
		preProcessed := false
		postProcessed := false
		preHandler := func(c *context.Context) error {
			preProcessed = true
			c.SetHeader("X-Pre", "true")
			return nil
		}
		postHandler := func(c *context.Context) error {
			postProcessed = true
			c.SetHeader("X-Post", "true")
			return nil
		}
		middleware := PostProcessMiddleware("Test", BaseConfig{}, preHandler, postHandler)
		handler := middleware(func(c *context.Context) {
			c.Text(200, "success")
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		c := context.NewContext(w, req)
		handler(c)
		if !preProcessed {
			t.Error("Expected pre-handler to be called")
		}
		if !postProcessed {
			t.Error("Expected post-handler to be called")
		}
		if w.Header().Get("X-Pre") != "true" {
			t.Error("Expected X-Pre header to be set")
		}
		if w.Header().Get("X-Post") != "true" {
			t.Error("Expected X-Post header to be set")
		}
	})
	t.Run("Post processing continues even if main handler has errors", func(t *testing.T) {
		postProcessed := false
		preHandler := func(c *context.Context) error {
			return nil
		}
		postHandler := func(c *context.Context) error {
			postProcessed = true
			return nil
		}
		middleware := PostProcessMiddleware("Test", BaseConfig{}, preHandler, postHandler)
		handler := middleware(func(c *context.Context) {
			c.Error(errors.New("handler error"), 500)
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		c := context.NewContext(w, req)
		handler(c)
		if !postProcessed {
			t.Error("Expected post-handler to be called even when main handler has errors")
		}
	})
}
func TestStandardResponseWriter(t *testing.T) {
	t.Run("Status and size tracking", func(t *testing.T) {
		w := httptest.NewRecorder()
		srw := NewStandardResponseWriter(w)
		if srw.Status() != http.StatusOK {
			t.Errorf("Expected default status 200, got %d", srw.Status())
		}
		if srw.Size() != 0 {
			t.Errorf("Expected initial size 0, got %d", srw.Size())
		}
		if srw.Written() {
			t.Error("Expected Written() to be false initially")
		}
		srw.WriteHeader(404)
		if srw.Status() != 404 {
			t.Errorf("Expected status 404, got %d", srw.Status())
		}
		if !srw.Written() {
			t.Error("Expected Written() to be true after WriteHeader")
		}
		n, err := srw.Write([]byte("test"))
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if n != 4 {
			t.Errorf("Expected to write 4 bytes, got %d", n)
		}
		if srw.Size() != 4 {
			t.Errorf("Expected size 4, got %d", srw.Size())
		}
	})
	t.Run("Write without explicit WriteHeader", func(t *testing.T) {
		w := httptest.NewRecorder()
		srw := NewStandardResponseWriter(w)
		srw.Write([]byte("test"))
		if srw.Status() != http.StatusOK {
			t.Errorf("Expected status 200, got %d", srw.Status())
		}
		if !srw.Written() {
			t.Error("Expected Written() to be true after Write")
		}
	})
}
func TestDefaultErrorHandler(t *testing.T) {
	t.Run("Server error", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		c := context.NewContext(w, req)
		err := MiddlewareError{
			Middleware: "Test",
			Err:        errors.New("internal error"),
			StatusCode: http.StatusInternalServerError,
		}
		DefaultErrorHandler(c, err, "Test")
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})
	t.Run("Client error", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		c := context.NewContext(w, req)
		err := MiddlewareError{
			Middleware: "Test",
			Err:        errors.New("bad request"),
			StatusCode: http.StatusBadRequest,
		}
		DefaultErrorHandler(c, err, "Test")
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}
func TestExampleMiddleware(t *testing.T) {
	t.Run("Valid configuration", func(t *testing.T) {
		middleware := NewExampleMiddleware(ExampleConfig{
			Timeout:        time.Second,
			CustomHeader:   "X-Custom",
			AllowedMethods: []string{"GET", "POST"},
		})
		handler := middleware(func(c *context.Context) {
			c.Text(200, "success")
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		c := context.NewContext(w, req)
		handler(c)
		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if w.Header().Get("X-Custom") != "processed" {
			t.Error("Expected X-Custom header to be set to 'processed'")
		}
	})
	t.Run("Invalid method", func(t *testing.T) {
		middleware := NewExampleMiddleware(ExampleConfig{
			Timeout:        time.Second,
			CustomHeader:   "X-Custom",
			AllowedMethods: []string{"GET", "POST"},
		})
		handler := middleware(func(c *context.Context) {
			c.Text(200, "should not reach here")
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("DELETE", "/", nil)
		c := context.NewContext(w, req)
		handler(c)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})
	t.Run("Invalid configuration", func(t *testing.T) {
		middleware := NewExampleMiddleware(ExampleConfig{
			Timeout:      0,
			CustomHeader: "",
		})
		handler := middleware(func(c *context.Context) {
			c.Text(200, "should not reach here")
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		c := context.NewContext(w, req)
		handler(c)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})
}
func TestExamplePostProcessMiddleware(t *testing.T) {
	t.Run("Pre and post processing headers", func(t *testing.T) {
		middleware := NewExamplePostProcessMiddleware(ExampleConfig{
			Timeout:      time.Second,
			CustomHeader: "X-Custom",
		})
		handler := middleware(func(c *context.Context) {
			time.Sleep(10 * time.Millisecond)
			c.Text(200, "success")
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		c := context.NewContext(w, req)
		handler(c)
		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if w.Header().Get("X-Pre-Process") != "true" {
			t.Error("Expected X-Pre-Process header")
		}
		if w.Header().Get("X-Post-Process") != "true" {
			t.Error("Expected X-Post-Process header")
		}
		if w.Header().Get("X-Processing-Time") == "" {
			t.Error("Expected X-Processing-Time header")
		}
	})
}
