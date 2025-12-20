package circuitbreaker

import (
	"errors"
	"sync"
	"time"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
)
type State int
const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)
type Config struct {
	base.BaseConfig
	MaxRequests    uint32        
	Interval       time.Duration 
	Timeout        time.Duration 
	FailureRatio   float64       
	MinRequests    uint32        
	OnStateChange  func(State)   
	IsFailure      func(error) bool 
}
type CircuitBreaker struct {
	config         Config
	mutex          sync.RWMutex
	state          State
	failures       uint32
	requests       uint32
	lastFailure    time.Time
	lastRequest    time.Time
	halfOpenCount  uint32
}
func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if c.MaxRequests == 0 {
		c.MaxRequests = 1
	}
	if c.Interval == 0 {
		c.Interval = 60 * time.Second
	}
	if c.Timeout == 0 {
		c.Timeout = 60 * time.Second
	}
	if c.FailureRatio == 0 {
		c.FailureRatio = 0.6
	}
	if c.MinRequests == 0 {
		c.MinRequests = 3
	}
	if c.IsFailure == nil {
		c.IsFailure = func(err error) bool {
			return err != nil
		}
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
				base.DefaultErrorHandler(c, err, "CircuitBreaker")
			}
		}
	}
	cb := &CircuitBreaker{
		config: cfg,
		state:  StateClosed,
	}
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if cfg.Skip != nil && cfg.Skip(c) {
				next(c)
				return
			}
			err := cb.Execute(func() error {
				next(c)
				return nil
			})
			if err != nil {
				if cfg.ErrorHandler != nil {
					cfg.ErrorHandler(c, base.MiddlewareError{
						Middleware: "CircuitBreaker",
						Err:        err,
						StatusCode: 503,
					}, "CircuitBreaker")
				} else {
					c.Writer.WriteHeader(503)
					c.Writer.Write([]byte("Service Unavailable - Circuit Breaker Open"))
				}
			}
		}
	}
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New()
}
func NewCircuitBreaker(config Config) *CircuitBreaker {
	config.Validate()
	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}
func (cb *CircuitBreaker) Middleware() context.Middleware {
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			err := cb.Execute(func() error {
				next(c)
				return nil
			})
			if err != nil {
				c.Writer.WriteHeader(503)
				_, _ = c.Writer.Write([]byte("Service Unavailable - Circuit Breaker Open"))
			}
		}
	}
}
func WithCircuitBreaker(config Config) (*CircuitBreaker, func(next context.HandlerFunc) context.HandlerFunc) {
	if err := config.Validate(); err != nil {
		return nil, func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "CircuitBreaker")
			}
		}
	}
	cb := &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
	middleware := func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if config.Skip != nil && config.Skip(c) {
				next(c)
				return
			}
			err := cb.Execute(func() error {
				next(c)
				return nil
			})
			if err != nil {
				if config.ErrorHandler != nil {
					config.ErrorHandler(c, base.MiddlewareError{
						Middleware: "CircuitBreaker",
						Err:        err,
						StatusCode: 503,
					}, "CircuitBreaker")
				} else {
					c.Writer.WriteHeader(503)
					c.Writer.Write([]byte("Service Unavailable - Circuit Breaker Open"))
				}
			}
		}
	}
	return cb, middleware
}
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.canRequest() {
		return errors.New("circuit breaker is open")
	}
	err := fn()
	cb.record(err)
	return err
}
func (cb *CircuitBreaker) canRequest() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	now := time.Now()
	cb.lastRequest = now
	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if now.Sub(cb.lastFailure) > cb.config.Timeout {
			cb.state = StateHalfOpen
			cb.halfOpenCount = 0
			cb.onStateChange()
			return true
		}
		return false
	case StateHalfOpen:
		return cb.halfOpenCount < cb.config.MaxRequests
	}
	return false
}
func (cb *CircuitBreaker) record(err error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.requests++
	if cb.state == StateHalfOpen {
		cb.halfOpenCount++
	}
	if cb.config.IsFailure(err) {
		cb.failures++
		cb.lastFailure = time.Now()
		if cb.state == StateHalfOpen {
			cb.state = StateOpen
			cb.onStateChange()
			return
		}
		if cb.shouldOpen() {
			cb.state = StateOpen
			cb.onStateChange()
		}
	} else {
		if cb.state == StateHalfOpen {
			cb.state = StateClosed
			cb.reset()
			cb.onStateChange()
		}
	}
	if cb.shouldReset() {
		cb.reset()
	}
}
func (cb *CircuitBreaker) shouldOpen() bool {
	if cb.requests < cb.config.MinRequests {
		return false
	}
	failureRatio := float64(cb.failures) / float64(cb.requests)
	return failureRatio > cb.config.FailureRatio
}
func (cb *CircuitBreaker) shouldReset() bool {
	return time.Since(cb.lastRequest) > cb.config.Interval
}
func (cb *CircuitBreaker) reset() {
	cb.failures = 0
	cb.requests = 0
}
func (cb *CircuitBreaker) onStateChange() {
	if cb.config.OnStateChange != nil {
		cb.config.OnStateChange(cb.state)
	}
}
func (cb *CircuitBreaker) State() State {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}
func (cb *CircuitBreaker) Metrics() map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return map[string]interface{}{
		"state":        cb.state.String(),
		"failures":     cb.failures,
		"requests":     cb.requests,
		"failure_rate": func() float64 {
			if cb.requests == 0 {
				return 0
			}
			return float64(cb.failures) / float64(cb.requests)
		}(),
	}
}
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}