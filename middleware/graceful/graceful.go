package graceful

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/arthurlch/goryu"
)

// ShutdownConfig defines the configuration for graceful shutdown
type ShutdownConfig struct {
	// Timeout for graceful shutdown. Default: 30 seconds
	Timeout time.Duration
	// Signals to listen for. Default: SIGINT, SIGTERM
	Signals []os.Signal
	// Logger for shutdown messages. Default: standard logger
	Logger Logger
	// OnShutdownStart is called when shutdown begins
	OnShutdownStart func()
	// OnShutdownComplete is called when shutdown completes
	OnShutdownComplete func()
	// CleanupFuncs are functions to run during shutdown
	CleanupFuncs []func() error
}

// Logger interface for shutdown logging
type Logger interface {
	Printf(format string, v ...any)
}

// defaultLogger implements Logger using standard log
type defaultLogger struct{}

func (d defaultLogger) Printf(format string, v ...any) {
	log.Printf(format, v...)
}

// GracefulServer wraps an HTTP server with graceful shutdown capabilities
type GracefulServer struct {
	server *http.Server
	config ShutdownConfig
	logger Logger
}

// NewGracefulServer creates a new server with graceful shutdown
func NewGracefulServer(addr string, handler http.Handler, config ...ShutdownConfig) *GracefulServer {
	cfg := ShutdownConfig{
		Timeout: 30 * time.Second,
		Signals: []os.Signal{syscall.SIGINT, syscall.SIGTERM},
		Logger:  defaultLogger{},
	}

	if len(config) > 0 {
		provided := config[0]
		if provided.Timeout > 0 {
			cfg.Timeout = provided.Timeout
		}
		if len(provided.Signals) > 0 {
			cfg.Signals = provided.Signals
		}
		if provided.Logger != nil {
			cfg.Logger = provided.Logger
		}
		cfg.OnShutdownStart = provided.OnShutdownStart
		cfg.OnShutdownComplete = provided.OnShutdownComplete
		cfg.CleanupFuncs = provided.CleanupFuncs
	}

	return &GracefulServer{
		server: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
		config: cfg,
		logger: cfg.Logger,
	}
}

// ListenAndServe starts the server and handles graceful shutdown
func (gs *GracefulServer) ListenAndServe() error {
	// Channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, gs.config.Signals...)

	// Channel to receive server errors
	errChan := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		gs.logger.Printf("Server starting on %s", gs.server.Addr)
		if err := gs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case sig := <-sigChan:
		gs.logger.Printf("Received signal: %v. Starting graceful shutdown...", sig)
		return gs.shutdown()
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	}
}

// shutdown performs the graceful shutdown
func (gs *GracefulServer) shutdown() error {
	if gs.config.OnShutdownStart != nil {
		gs.config.OnShutdownStart()
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), gs.config.Timeout)
	defer cancel()

	gs.logger.Printf("Shutting down server with timeout: %v", gs.config.Timeout)

	// Shutdown the server
	if err := gs.server.Shutdown(ctx); err != nil {
		gs.logger.Printf("Error during server shutdown: %v", err)
		return err
	}

	// Run cleanup functions
	for i, cleanup := range gs.config.CleanupFuncs {
		gs.logger.Printf("Running cleanup function %d/%d", i+1, len(gs.config.CleanupFuncs))
		if err := cleanup(); err != nil {
			gs.logger.Printf("Cleanup function %d failed: %v", i+1, err)
			// Continue with other cleanup functions
		}
	}

	gs.logger.Printf("Server shutdown completed successfully")

	if gs.config.OnShutdownComplete != nil {
		gs.config.OnShutdownComplete()
	}

	return nil
}

// SetReadTimeout sets the read timeout for the underlying server
func (gs *GracefulServer) SetReadTimeout(timeout time.Duration) {
	gs.server.ReadTimeout = timeout
}

// SetWriteTimeout sets the write timeout for the underlying server
func (gs *GracefulServer) SetWriteTimeout(timeout time.Duration) {
	gs.server.WriteTimeout = timeout
}

// SetIdleTimeout sets the idle timeout for the underlying server
func (gs *GracefulServer) SetIdleTimeout(timeout time.Duration) {
	gs.server.IdleTimeout = timeout
}

// Server returns the underlying HTTP server
func (gs *GracefulServer) Server() *http.Server {
	return gs.server
}

// Middleware creates a middleware that tracks active connections
func Middleware() goryu.Middleware {
	activeConnections := &connectionCounter{}

	return func(next goryu.HandlerFunc) goryu.HandlerFunc {
		return func(c *goryu.Context) {
			activeConnections.increment()
			defer activeConnections.decrement()

			// Add connection count to context for monitoring
			c.Set("active_connections", activeConnections.count())

			next(c)
		}
	}
}

// connectionCounter tracks active connections safely using atomic operations
type connectionCounter struct {
	value int64
}

func (cc *connectionCounter) increment() {
	atomic.AddInt64(&cc.value, 1)
}

func (cc *connectionCounter) decrement() {
	atomic.AddInt64(&cc.value, -1)
}

func (cc *connectionCounter) count() int64 {
	return atomic.LoadInt64(&cc.value)
}

// RunWithGracefulShutdown is a helper function to run a Goryu app with graceful shutdown
func RunWithGracefulShutdown(app *goryu.App, addr string, config ...ShutdownConfig) error {
	server := NewGracefulServer(addr, app, config...)
	return server.ListenAndServe()
}
