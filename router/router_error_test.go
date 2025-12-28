package router

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"

	context "github.com/arthurlch/goryu/goryuctx"
)

func TestRouterErrorHandling(t *testing.T) {
	t.Run("RouterErrorModePanic", func(t *testing.T) {
		router := New(RouterConfig{ErrorMode: RouterErrorModePanic})

		// First route should work fine
		router.GET("/test", func(c *context.Context) {})

		// Duplicate route should panic
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for duplicate route in panic mode")
			} else {
				errStr := r.(string)
				if !strings.Contains(errStr, "route '/test' already exists") {
					t.Errorf("Expected specific error message, got: %s", errStr)
				}
			}
		}()

		router.GET("/test", func(c *context.Context) {})
	})

	t.Run("RouterErrorModeLog", func(t *testing.T) {
		// Capture log output
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(os.Stderr)

		router := New(RouterConfig{ErrorMode: RouterErrorModeLog})

		// First route should work fine
		router.GET("/test", func(c *context.Context) {})

		// Duplicate route should log but not panic
		router.GET("/test", func(c *context.Context) {})

		// Check that error was logged
		logOutput := buf.String()
		if !strings.Contains(logOutput, "Router error") {
			t.Errorf("Expected error to be logged, got: %s", logOutput)
		}
		if !strings.Contains(logOutput, "route '/test' already exists") {
			t.Errorf("Expected specific error message in log, got: %s", logOutput)
		}
	})

	t.Run("RouterErrorModeSilent", func(t *testing.T) {
		// Capture log output
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(os.Stderr)

		router := New(RouterConfig{ErrorMode: RouterErrorModeSilent})

		// First route should work fine
		router.GET("/test", func(c *context.Context) {})

		// Duplicate route should be silently ignored
		router.GET("/test", func(c *context.Context) {})

		// Check that nothing was logged
		logOutput := buf.String()
		if strings.Contains(logOutput, "Router error") {
			t.Errorf("Expected no logging in silent mode, got: %s", logOutput)
		}
	})
}

func TestRouterNamedRouteErrors(t *testing.T) {
	t.Run("DuplicateNamedRoute_Panic", func(t *testing.T) {
		router := New(RouterConfig{ErrorMode: RouterErrorModePanic})

		// First named route should work
		router.GET("/test1", func(c *context.Context) {}).SetName("test")

		// Duplicate named route should panic
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for duplicate named route in panic mode")
			} else {
				errStr := r.(string)
				if !strings.Contains(errStr, "Route with name 'test' already exists") {
					t.Errorf("Expected specific error message, got: %s", errStr)
				}
			}
		}()

		router.GET("/test2", func(c *context.Context) {}).SetName("test")
	})

	t.Run("DuplicateNamedRoute_Log", func(t *testing.T) {
		// Capture log output
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(os.Stderr)

		router := New(RouterConfig{ErrorMode: RouterErrorModeLog})

		// First named route should work
		router.GET("/test1", func(c *context.Context) {}).SetName("test")

		// Duplicate named route should log but not panic
		router.GET("/test2", func(c *context.Context) {}).SetName("test")

		// Check that error was logged
		logOutput := buf.String()
		if !strings.Contains(logOutput, "Router error") {
			t.Errorf("Expected error to be logged, got: %s", logOutput)
		}
		if !strings.Contains(logOutput, "Route with name 'test' already exists") {
			t.Errorf("Expected specific error message in log, got: %s", logOutput)
		}
	})
}

func TestRouterInvalidPathErrors(t *testing.T) {
	t.Run("InvalidPath_Panic", func(t *testing.T) {
		router := New(RouterConfig{ErrorMode: RouterErrorModePanic})

		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for invalid path in panic mode")
			} else {
				errStr := r.(string)
				if !strings.Contains(errStr, "path must begin with '/'") {
					t.Errorf("Expected specific error message, got: %s", errStr)
				}
			}
		}()

		router.GET("invalid", func(c *context.Context) {})
	})

	t.Run("InvalidPath_Log", func(t *testing.T) {
		// Capture log output
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(os.Stderr)

		router := New(RouterConfig{ErrorMode: RouterErrorModeLog})

		// Invalid path should log but not panic
		route := router.GET("invalid", func(c *context.Context) {})

		// Should return a dummy route
		if route == nil {
			t.Error("Expected dummy route to be returned in log mode")
		}

		// Check that error was logged
		logOutput := buf.String()
		if !strings.Contains(logOutput, "Router error") {
			t.Errorf("Expected error to be logged, got: %s", logOutput)
		}
		if !strings.Contains(logOutput, "path must begin with '/'") {
			t.Errorf("Expected specific error message in log, got: %s", logOutput)
		}
	})
}

func TestRouterWildcardErrors(t *testing.T) {
	t.Run("WildcardNotAtEnd_Panic", func(t *testing.T) {
		router := New(RouterConfig{ErrorMode: RouterErrorModePanic})

		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for wildcard not at end in panic mode")
			} else {
				errStr := r.(string)
				if !strings.Contains(errStr, "catch-all routes are only allowed at the end of the path") {
					t.Errorf("Expected specific error message, got: %s", errStr)
				}
			}
		}()

		router.GET("/files/*path/more", func(c *context.Context) {})
	})

	t.Run("WildcardNotAtEnd_Log", func(t *testing.T) {
		// Capture log output
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(os.Stderr)

		router := New(RouterConfig{ErrorMode: RouterErrorModeLog})

		// Invalid wildcard should log but not panic
		router.GET("/files/*path/more", func(c *context.Context) {})

		// Check that error was logged
		logOutput := buf.String()
		if !strings.Contains(logOutput, "Router error") {
			t.Errorf("Expected error to be logged, got: %s", logOutput)
		}
		if !strings.Contains(logOutput, "catch-all routes are only allowed at the end of the path") {
			t.Errorf("Expected specific error message in log, got: %s", logOutput)
		}
	})
}

func TestRouterErrorMethods(t *testing.T) {
	router := New()

	// Test default error mode
	if router.GetErrorHandlingMode() != RouterErrorModePanic {
		t.Error("Expected default error mode to be RouterErrorModePanic")
	}

	// Test setting error mode
	router.SetErrorHandlingMode(RouterErrorModeLog)
	if router.GetErrorHandlingMode() != RouterErrorModeLog {
		t.Error("Expected error mode to be RouterErrorModeLog after setting")
	}

	// Test RouterError type
	err := &RouterError{
		Operation: "TestOp",
		Message:   "test message",
	}

	expected := "router TestOp error: test message"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}
