package circuitbreaker

import (
	"net/http"
	"sync"
	"time"

	"github.com/arthurlch/goryu"
)

// State represents the circuit breaker state
type State int32

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half-open"
	case StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

// Config defines the configuration for circuit breaker middleware
type Config struct {
	// Timeout for requests in open state before transitioning to half-open
	Timeout time.Duration
	// MaxRequests is the maximum number of requests allowed to pass through
	// when the CircuitBreaker is half-open. Default: 1
	MaxRequests uint32
	// Interval is the cyclic period in Closed state to clear counts. Default: 0
	Interval time.Duration
	// ReadyToTrip is called with a copy of Counts whenever a request fails in Closed state.
	// If ReadyToTrip returns true, the CircuitBreaker will be placed into Open state.
	// Default: fails >= 5
	ReadyToTrip func(counts Counts) bool
	// OnStateChange is called whenever the state of the CircuitBreaker changes.
	OnStateChange func(name string, from State, to State)
	// IsSuccessful determines whether the request was successful.
	// Default: status code < 500
	IsSuccessful func(c *goryu.Context, statusCode int) bool
	// FallbackHandler is called when the circuit is open. If nil, returns 503 Service Unavailable
	FallbackHandler goryu.HandlerFunc
	// Name is used to identify this circuit breaker in callbacks
	Name string
	// Skip defines when to skip circuit breaker middleware
	Skip func(c *goryu.Context) bool
}

// Counts holds the numbers of requests and their successes/failures
type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

// CircuitBreaker represents the circuit breaker
type CircuitBreaker struct {
	config Config
	mutex  sync.RWMutex
	state  State
	counts Counts
	expiry time.Time
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	return rw._, _ = ResponseWriter.Write(b)
}

// New creates a new circuit breaker middleware
func New(config Config) goryu.Middleware {
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	if config.MaxRequests == 0 {
		config.MaxRequests = 1
	}
	if config.ReadyToTrip == nil {
		config.ReadyToTrip = func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 5
		}
	}
	if config.IsSuccessful == nil {
		config.IsSuccessful = func(c *goryu.Context, statusCode int) bool {
			return statusCode < 500
		}
	}
	if config.FallbackHandler == nil {
		config.FallbackHandler = func(c *goryu.Context) {
			_ = c.Status(http.StatusServiceUnavailable).Text(http.StatusServiceUnavailable, "Service Temporarily Unavailable")
		}
	}
	if config.Name == "" {
		config.Name = "default"
	}

	cb := &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}

	// Set up interval clearing if configured
	if config.Interval > 0 {
		go cb.intervalClearing()
	}

	return func(next goryu.HandlerFunc) goryu.HandlerFunc {
		return func(c *goryu.Context) {
			if cb.config.Skip != nil && cb.config.Skip(c) {
				next(c)
				return
			}

			// Check if we can execute the request
			if !cb.allowRequest() {
				cb.config.FallbackHandler(c)
				return
			}

			// Wrap response writer to capture status code
			rw := &responseWriter{
				ResponseWriter: c.Writer,
				statusCode:     http.StatusOK,
			}
			c.Writer = rw

			// Execute the request
			next(c)

			// Record the result
			cb.recordResult(c, rw.statusCode)
		}
	}
}

// allowRequest checks if the request should be allowed
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		if cb.config.Interval > 0 && cb.expiry.Before(now) {
			cb.counts = Counts{}
			cb.expiry = now.Add(cb.config.Interval)
		}
		return true

	case StateOpen:
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen)
			cb.counts = Counts{}
			return true
		}
		return false

	case StateHalfOpen:
		return cb.counts.Requests < cb.config.MaxRequests

	default:
		return false
	}
}

// recordResult records the result of a request
func (cb *CircuitBreaker) recordResult(c *goryu.Context, statusCode int) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.counts.Requests++

	if cb.config.IsSuccessful(c, statusCode) {
		cb.counts.TotalSuccesses++
		cb.counts.ConsecutiveSuccesses++
		cb.counts.ConsecutiveFailures = 0

		if cb.state == StateHalfOpen {
			cb.setState(StateClosed)
			cb.counts = Counts{}
		}
	} else {
		cb.counts.TotalFailures++
		cb.counts.ConsecutiveFailures++
		cb.counts.ConsecutiveSuccesses = 0

		if cb.state == StateClosed && cb.config.ReadyToTrip(cb.counts) {
			cb.setState(StateOpen)
			cb.expiry = time.Now().Add(cb.config.Timeout)
		} else if cb.state == StateHalfOpen {
			cb.setState(StateOpen)
			cb.expiry = time.Now().Add(cb.config.Timeout)
		}
	}
}

// setState changes the state and calls the callback if configured
func (cb *CircuitBreaker) setState(newState State) {
	if cb.config.OnStateChange != nil {
		cb.config.OnStateChange(cb.config.Name, cb.state, newState)
	}
	cb.state = newState
}

// intervalClearing runs the interval clearing goroutine
func (cb *CircuitBreaker) intervalClearing() {
	ticker := time.NewTicker(cb.config.Interval)
	defer ticker.Stop()

	for range ticker.C {
		cb.mutex.Lock()
		if cb.state == StateClosed {
			cb.counts = Counts{}
		}
		cb.mutex.Unlock()
	}
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() State {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// Counts returns a copy of the current counts
func (cb *CircuitBreaker) Counts() Counts {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.counts
}
