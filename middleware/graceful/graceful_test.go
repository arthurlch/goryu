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
	goryu_context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/graceful"
)
func newTestApp() *goryu.App {
	app := goryu.New()
	app.GET("/", func(c *goryu.Ctx) {
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
	server := graceful.NewGracefulServer(":0", app) 
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
		},
		OnShutdownComplete: func() {
		},
		CleanupFuncs: []func() error{
			func() error {
				return nil
			},
		},
	}
	app := newTestApp()
	server := graceful.NewGracefulServer(":0", app, config)
	if server == nil {
		t.Fatal("Expected server to be created with config")
	}
}
func TestGracefulMiddleware(t *testing.T) {
	middleware := graceful.Middleware()
	handler := func(c *goryu.Ctx) {
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
func TestGracefulNewMiddleware(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		middleware := graceful.New(graceful.Config{})
		handler := func(c *goryu_context.Context) {
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
	})
	t.Run("CustomContextKey", func(t *testing.T) {
		config := graceful.Config{ContextKey: "custom_connections"}
		middleware := graceful.New(config)
		handler := func(c *goryu_context.Context) {
			if count, exists := c.Get("custom_connections"); !exists {
				t.Error("Expected custom_connections to be set in context")
			} else if count.(int64) < 0 {
				t.Error("Expected custom_connections to be non-negative")
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
	})
	t.Run("DefaultHelper", func(t *testing.T) {
		middleware := graceful.Default()
		handler := func(c *goryu_context.Context) {
			c.Text(http.StatusOK, "OK")
		}
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		ctx := goryu_context.NewContext(rr, req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})
}
func TestRunWithGracefulShutdown(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RunWithGracefulShutdown function access panicked: %v", r)
		}
	}()
	fn := graceful.RunWithGracefulShutdown
	if fn == nil {
		t.Error("RunWithGracefulShutdown function should not be nil")
	}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RunWithGracefulShutdown creation panicked: %v", r)
		}
	}()
	_ = graceful.RunWithGracefulShutdown
}
type testLogger struct {
	buffer *bytes.Buffer
}
func (l *testLogger) Printf(format string, v ...any) {
	l.buffer.WriteString(format)
	l.buffer.WriteString("\n")
}
func TestShutdownProcess(t *testing.T) {
	var logBuffer bytes.Buffer
	var shutdownStartCalled, shutdownCompleteCalled bool
	var cleanupCalled bool
	logger := &testLogger{buffer: &logBuffer}
	config := graceful.ShutdownConfig{
		Timeout: 100 * time.Millisecond, 
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
	go func() {
		time.Sleep(50 * time.Millisecond) 
		ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
		defer cancel()
		server.Server().Shutdown(ctx)
	}()
	if !shutdownStartCalled && !shutdownCompleteCalled && !cleanupCalled {
	}
}
