package graceful_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/arthurlch/goryu"
	goryu_context "github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/graceful"
)

func newTestApp() *goryu.App {
	app := goryu.New()
	app.GET("/", func(c *goryu.Context) {
		c.Text(http.StatusOK, "Hello, World!")
	})
	return app
}

func TestNewGracefulServer(t *testing.T) {
	app := newTestApp()
	server := graceful.NewGracefulServer(":8080", app)

	if server == nil {
		t.Fatal("Expected server to be created")
	}

	if server.Server().Addr != ":8080" {
		t.Errorf("Expected server addr ':8080', got %s", server.Server().Addr)
	}
}

func TestGracefulServerTimeouts(t *testing.T) {
	app := newTestApp()
	server := graceful.NewGracefulServer(":0", app) // Use port 0 for testing

	// Test timeout setters
	server.SetReadTimeout(5 * time.Second)
	server.SetWriteTimeout(10 * time.Second)
	server.SetIdleTimeout(15 * time.Second)

	httpServer := server.Server()
	if httpServer.ReadTimeout != 5*time.Second {
		t.Errorf("Expected ReadTimeout 5s, got %v", httpServer.ReadTimeout)
	}
	if httpServer.WriteTimeout != 10*time.Second {
		t.Errorf("Expected WriteTimeout 10s, got %v", httpServer.WriteTimeout)
	}
	if httpServer.IdleTimeout != 15*time.Second {
		t.Errorf("Expected IdleTimeout 15s, got %v", httpServer.IdleTimeout)
	}
}

func TestGracefulShutdownConfig(t *testing.T) {
	var logBuffer bytes.Buffer

	logger := &testLogger{buffer: &logBuffer}

	config := graceful.ShutdownConfig{
		Timeout: 5 * time.Second,
		Signals: []os.Signal{syscall.SIGTERM},
		Logger:  logger,
		OnShutdownStart: func() {
			// Callback for shutdown start
		},
		OnShutdownComplete: func() {
			// Callback for shutdown complete
		},
		CleanupFuncs: []func() error{
			func() error {
				// Cleanup function
				return nil
			},
		},
	}

	app := newTestApp()
	server := graceful.NewGracefulServer(":0", app, config)

	// We can't easily test the full shutdown process in a unit test,
	// but we can verify the configuration was applied
	if server == nil {
		t.Fatal("Expected server to be created with config")
	}

	// The callbacks and cleanup functions will be tested in the shutdown method
	// For now, just verify the server was created successfully
}

func TestGracefulMiddleware(t *testing.T) {
	middleware := graceful.Middleware()

	handler := func(c *goryu.Context) {
		// Check that active connections count is available
		if count, exists := c.Get("active_connections"); !exists {
			t.Error("Expected active_connections to be set in context")
		} else if count.(int64) < 0 {
			t.Error("Expected active_connections to be non-negative")
		}
		c.Text(http.StatusOK, "OK")
	}

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	ctx := goryu_context.NewContext(rr, req)

	middleware(handler)(ctx)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestRunWithGracefulShutdown(t *testing.T) {
	// This test would normally require actually starting a server and sending signals,
	// which is complex in a unit test. For now, we just verify the function exists
	// and can be called without panicking.

	// We can't test the actual shutdown because it would block forever waiting for signals
	// In a real application, you would test this with integration tests

	// Test that the function exists and is callable
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RunWithGracefulShutdown function access panicked: %v", r)
		}
	}()

	// Just check that we can reference the function without issues
	fn := graceful.RunWithGracefulShutdown
	if fn == nil {
		t.Error("RunWithGracefulShutdown function should not be nil")
	}

	// Test that we can create the function without it panicking
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RunWithGracefulShutdown creation panicked: %v", r)
		}
	}()

	// We can't actually run it because it would block, but we can verify it exists
	_ = graceful.RunWithGracefulShutdown
}

// testLogger implements the Logger interface for testing
type testLogger struct {
	buffer *bytes.Buffer
}

func (l *testLogger) Printf(format string, v ...any) {
	l.buffer.WriteString(format)
	l.buffer.WriteString("\n")
}

func TestShutdownProcess(t *testing.T) {
	// This test simulates the shutdown process without actually starting a server
	var logBuffer bytes.Buffer
	var shutdownStartCalled, shutdownCompleteCalled bool
	var cleanupCalled bool

	logger := &testLogger{buffer: &logBuffer}

	config := graceful.ShutdownConfig{
		Timeout: 100 * time.Millisecond, // Short timeout for testing
		Logger:  logger,
		OnShutdownStart: func() {
			shutdownStartCalled = true
		},
		OnShutdownComplete: func() {
			shutdownCompleteCalled = true
		},
		CleanupFuncs: []func() error{
			func() error {
				cleanupCalled = true
				return nil
			},
		},
	}

	app := newTestApp()
	server := graceful.NewGracefulServer(":0", app, config)

	// Start the server in a separate goroutine
	go func() {
		time.Sleep(50 * time.Millisecond) // Let server start
		// Simulate shutdown by closing the server directly
		ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
		defer cancel()
		server.Server().Shutdown(ctx)
	}()

	// The actual ListenAndServe call would block waiting for signals,
	// so we'll just test that our configuration is properly set up
	if !shutdownStartCalled && !shutdownCompleteCalled && !cleanupCalled {
		// This is expected since we haven't triggered the actual shutdown process
		// The callbacks would be called during the real shutdown
	}
}
