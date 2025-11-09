package builder
import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"github.com/arthurlch/goryu/context"
)
func TestMiddlewareBuilder_Before(t *testing.T) {
	beforeCalled := false
	middleware := New("Test").
		Before(func(c *context.Context) error {
			beforeCalled = true
			c.Set("test_value", "before_called")
			return nil
		}).
		Build()
	app := newTestApp()
	app.Use(middleware)
	app.GET("/test", func(c *context.Context) {
		value, exists := c.Get("test_value")
		if !exists || value != "before_called" {
			t.Error("Before function not called or value not set")
		}
		c.Status(200).FluentText(200, "OK")
	})
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if !beforeCalled {
		t.Error("Before function was not called")
	}
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
func TestMiddlewareBuilder_After(t *testing.T) {
	afterCalled := false
	middleware := New("Test").
		After(func(c *context.Context) error {
			afterCalled = true
			c.Writer.Header().Set("X-After-Called", "true")
			return nil
		}).
		Build()
	app := newTestApp()
	app.Use(middleware)
	app.GET("/test", func(c *context.Context) {
		c.Status(200).FluentText(200, "OK")
	})
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if !afterCalled {
		t.Error("After function was not called")
	}
	if w.Header().Get("X-After-Called") != "true" {
		t.Error("After function did not set header")
	}
}
func TestMiddlewareBuilder_BeforeAndAfter(t *testing.T) {
	var calls []string
	middleware := New("Test").
		Before(func(c *context.Context) error {
			calls = append(calls, "before")
			c.Set("start_time", time.Now())
			return nil
		}).
		After(func(c *context.Context) error {
			calls = append(calls, "after")
			return nil
		}).
		Build()
	app := newTestApp()
	app.Use(middleware)
	app.GET("/test", func(c *context.Context) {
		calls = append(calls, "handler")
		c.Status(200).FluentText(200, "OK")
	})
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	expectedCalls := []string{"before", "handler", "after"}
	if len(calls) != len(expectedCalls) {
		t.Errorf("Expected %d calls, got %d: %v", len(expectedCalls), len(calls), calls)
		return
	}
	for i, expected := range expectedCalls {
		if calls[i] != expected {
			t.Errorf("Call %d: expected %s, got %s", i, expected, calls[i])
		}
	}
}
func TestMiddlewareBuilder_Skip(t *testing.T) {
	beforeCalled := false
	middleware := New("Test").
		Skip(func(c *context.Context) bool {
			return strings.HasPrefix(c.Request.URL.Path, "/skip")
		}).
		Before(func(c *context.Context) error {
			beforeCalled = true
			return nil
		}).
		Build()
	app := newTestApp()
	app.Use(middleware)
	app.GET("/skip/test", func(c *context.Context) {
		c.Status(200).FluentText(200, "Skipped")
	})
	app.GET("/normal/test", func(c *context.Context) {
		c.Status(200).FluentText(200, "Normal")
	})
	req := httptest.NewRequest("GET", "/skip/test", nil)
	w := httptest.NewRecorder()
	beforeCalled = false
	app.ServeHTTP(w, req)
	if beforeCalled {
		t.Error("Before function should have been skipped")
	}
	req = httptest.NewRequest("GET", "/normal/test", nil)
	w = httptest.NewRecorder()
	beforeCalled = false
	app.ServeHTTP(w, req)
	if !beforeCalled {
		t.Error("Before function should have been called for normal path")
	}
}
func TestMiddlewareBuilder_OnError(t *testing.T) {
	errorHandlerCalled := false
	var capturedError error
	middleware := New("Test").
		Before(func(c *context.Context) error {
			return errors.New("test error")
		}).
		OnError(func(c *context.Context, err error) {
			errorHandlerCalled = true
			capturedError = err
			c.Status(500).FluentJSON(500, map[string]string{
				"error": "Custom error: " + err.Error(),
			})
		}).
		Build()
	app := newTestApp()
	app.Use(middleware)
	app.GET("/test", func(c *context.Context) {
		c.Status(200).FluentText(200, "Should not reach here")
	})
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if !errorHandlerCalled {
		t.Error("Error handler was not called")
	}
	if capturedError == nil || capturedError.Error() != "test error" {
		t.Errorf("Expected 'test error', got %v", capturedError)
	}
	if w.Code != 500 {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Custom error: test error") {
		t.Errorf("Response body should contain custom error message, got: %s", body)
	}
}
func TestMiddlewareBuilder_BuildSimple(t *testing.T) {
	simpleCalled := false
	middleware := New("Simple").
		BuildSimple(func(c *context.Context) {
			simpleCalled = true
			c.Set("simple_called", true)
		})
	app := newTestApp()
	app.Use(middleware)
	app.GET("/test", func(c *context.Context) {
		value, exists := c.Get("simple_called")
		if !exists || value != true {
			t.Error("Simple function not called or value not set")
		}
		c.Status(200).FluentText(200, "OK")
	})
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if !simpleCalled {
		t.Error("Simple function was not called")
	}
}
func TestConvenienceMiddleware_Security(t *testing.T) {
	middleware := NewSecurity().Build()
	app := newTestApp()
	app.Use(middleware)
	app.GET("/test", func(c *context.Context) {
		c.Status(200).FluentText(200, "OK")
	})
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	expectedHeaders := map[string]string{
		"X-Content-Type-Options":      "nosniff",
		"X-Frame-Options":             "DENY",
		"X-XSS-Protection":            "1; mode=block",
		"Strict-Transport-Security":   "max-age=31536000; includeSubDomains",
		"Referrer-Policy":             "strict-origin-when-cross-origin",
	}
	for header, expectedValue := range expectedHeaders {
		if w.Header().Get(header) != expectedValue {
			t.Errorf("Header %s: expected %s, got %s", header, expectedValue, w.Header().Get(header))
		}
	}
}
func TestConvenienceMiddleware_CORS(t *testing.T) {
	middleware := NewCORS("https://example.com").Build()
	app := newTestApp()
	app.Use(middleware)
	app.GET("/test", func(c *context.Context) {
		c.Status(200).FluentText(200, "OK")
	})
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Error("CORS origin header not set correctly")
	}
	req = httptest.NewRequest("OPTIONS", "/test", nil)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != 204 {
		t.Errorf("OPTIONS request should return 204, got %d", w.Code)
	}
}
func TestConvenienceMiddleware_Timing(t *testing.T) {
	middleware := NewTiming().Build()
	app := newTestApp()
	app.Use(middleware)
	app.GET("/test", func(c *context.Context) {
		time.Sleep(10 * time.Millisecond)
		c.Status(200).FluentText(200, "OK")
	})
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	responseTime := w.Header().Get("X-Response-Time")
	if responseTime == "" {
		t.Error("X-Response-Time header not set")
	}
	if _, err := time.ParseDuration(responseTime); err != nil {
		t.Errorf("Invalid duration format: %s", responseTime)
	}
}