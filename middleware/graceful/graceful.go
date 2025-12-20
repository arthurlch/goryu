package graceful

import (
	stdContext "context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/arthurlch/goryu"
	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
)
type ShutdownConfig struct {
	Timeout time.Duration
	Signals []os.Signal
	Logger Logger
	OnShutdownStart func()
	OnShutdownComplete func()
	CleanupFuncs []func() error
}
type Logger interface {
	Printf(format string, v ...any)
}
type defaultLogger struct{}
func (d defaultLogger) Printf(format string, v ...any) {
	log.Printf(format, v...)
}
type GracefulServer struct {
	server *http.Server
	config ShutdownConfig
	logger Logger
}
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
func (gs *GracefulServer) ListenAndServe() error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, gs.config.Signals...)
	errChan := make(chan error, 1)
	go func() {
		gs.logger.Printf("Server starting on %s", gs.server.Addr)
		if err := gs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()
	select {
	case sig := <-sigChan:
		gs.logger.Printf("Received signal: %v. Starting graceful shutdown...", sig)
		return gs.shutdown()
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	}
}
func (gs *GracefulServer) shutdown() error {
	if gs.config.OnShutdownStart != nil {
		gs.config.OnShutdownStart()
	}
	ctx, cancel := stdContext.WithTimeout(stdContext.Background(), gs.config.Timeout)
	defer cancel()
	gs.logger.Printf("Shutting down server with timeout: %v", gs.config.Timeout)
	if err := gs.server.Shutdown(ctx); err != nil {
		gs.logger.Printf("Error during server shutdown: %v", err)
		return err
	}
	for i, cleanup := range gs.config.CleanupFuncs {
		gs.logger.Printf("Running cleanup function %d/%d", i+1, len(gs.config.CleanupFuncs))
		if err := cleanup(); err != nil {
			gs.logger.Printf("Cleanup function %d failed: %v", i+1, err)
		}
	}
	gs.logger.Printf("Server shutdown completed successfully")
	if gs.config.OnShutdownComplete != nil {
		gs.config.OnShutdownComplete()
	}
	return nil
}
func (gs *GracefulServer) SetReadTimeout(timeout time.Duration) {
	gs.server.ReadTimeout = timeout
}
func (gs *GracefulServer) SetWriteTimeout(timeout time.Duration) {
	gs.server.WriteTimeout = timeout
}
func (gs *GracefulServer) SetIdleTimeout(timeout time.Duration) {
	gs.server.IdleTimeout = timeout
}
func (gs *GracefulServer) Server() *http.Server {
	return gs.server
}
type Config struct {
	base.BaseConfig
	ContextKey string
}
func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if c.ContextKey == "" {
		c.ContextKey = "active_connections"
	}
	return nil
}
func New(config ...Config) func(next context.HandlerFunc) context.HandlerFunc {
	cfg := Config{}
	if len(config) > 0 {
		cfg = config[0]
	}

	if err := cfg.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "Graceful")
			}
		}
	}
	activeConnections := &connectionCounter{}
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if cfg.Skip != nil && cfg.Skip(c) {
				next(c)
				return
			}
			activeConnections.increment()
			defer activeConnections.decrement()
			c.Set(cfg.ContextKey, activeConnections.count())
			next(c)
		}
	}
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New()
}
func Middleware() goryu.Middleware {
	activeConnections := &connectionCounter{}
	return func(next goryu.Handler) goryu.Handler {
		return func(c *goryu.Ctx) {
			activeConnections.increment()
			defer activeConnections.decrement()
			c.Set("active_connections", activeConnections.count())
			next(c)
		}
	}
}
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
func RunWithGracefulShutdown(app *goryu.App, addr string, config ...ShutdownConfig) error {
	server := NewGracefulServer(addr, app, config...)
	return server.ListenAndServe()
}
