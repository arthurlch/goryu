package monitoring

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/arthurlch/goryu/context"
)

// EventType represents the type of monitoring event
type EventType string

const (
	EventRequest   EventType = "request"
	EventError     EventType = "error"
	EventHealthy   EventType = "healthy"
	EventUnhealthy EventType = "unhealthy"
	EventStartup   EventType = "startup"
	EventShutdown  EventType = "shutdown"
	EventCustom    EventType = "custom"
)

// Event represents a monitoring event
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Duration  time.Duration          `json:"duration,omitempty"`
}

// HealthStatus represents the health status of the application
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusDegraded  HealthStatus = "degraded"
)

// HealthCheck represents a health check function
type HealthCheck struct {
	Name     string                                  `json:"name"`
	Check    func() (status HealthStatus, err error) `json:"-"`
	Timeout  time.Duration                           `json:"timeout"`
	Interval time.Duration                           `json:"interval"`
	Critical bool                                    `json:"critical"`
}

// HealthResult represents the result of a health check
type HealthResult struct {
	Name      string        `json:"name"`
	Status    HealthStatus  `json:"status"`
	Message   string        `json:"message,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration"`
	Critical  bool          `json:"critical"`
}

// Metrics holds application metrics
type Metrics struct {
	RequestCount    int64         `json:"request_count"`
	ErrorCount      int64         `json:"error_count"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
	Uptime          time.Duration `json:"uptime"`
	MemoryUsage     uint64        `json:"memory_usage_bytes"`
	GoRoutines      int           `json:"goroutines"`
	StartTime       time.Time     `json:"start_time"`
}

// Monitor is the main monitoring system
type Monitor struct {
	mu            sync.RWMutex
	events        []Event
	maxEvents     int
	healthChecks  map[string]*HealthCheck
	healthResults map[string]*HealthResult
	metrics       *Metrics
	startTime     time.Time
	eventHandlers []func(Event)
	enabled       bool
}

// Config holds monitoring configuration
type Config struct {
	Enabled        bool          `json:"enabled"`
	MaxEvents      int           `json:"max_events"`
	HealthInterval time.Duration `json:"health_interval"`
	MetricsEnabled bool          `json:"metrics_enabled"`
}

// New creates a new monitoring system
func New(config ...Config) *Monitor {
	cfg := Config{
		Enabled:        true,
		MaxEvents:      1000,
		HealthInterval: 30 * time.Second,
		MetricsEnabled: true,
	}
	if len(config) > 0 {
		cfg = config[0]
	}

	m := &Monitor{
		events:        make([]Event, 0, cfg.MaxEvents),
		maxEvents:     cfg.MaxEvents,
		healthChecks:  make(map[string]*HealthCheck),
		healthResults: make(map[string]*HealthResult),
		startTime:     time.Now(),
		enabled:       cfg.Enabled,
		metrics: &Metrics{
			StartTime: time.Now(),
		},
	}

	if cfg.MetricsEnabled {
		go m.updateMetrics()
	}

	if cfg.HealthInterval > 0 {
		go m.runHealthChecks(cfg.HealthInterval)
	}

	m.EmitEvent(EventStartup, "Monitoring system started", nil)
	return m
}

// EmitEvent emits a new monitoring event
func (m *Monitor) EmitEvent(eventType EventType, message string, data map[string]interface{}) {
	if !m.enabled {
		return
	}

	event := Event{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:      eventType,
		Timestamp: time.Now(),
		Message:   message,
		Data:      data,
	}

	m.mu.Lock()
	if len(m.events) >= m.maxEvents {
		m.events = m.events[1:]
	}
	m.events = append(m.events, event)
	m.mu.Unlock()

	// Call event handlers
	for _, handler := range m.eventHandlers {
		go handler(event)
	}
}

// AddHealthCheck adds a health check
func (m *Monitor) AddHealthCheck(name string, check *HealthCheck) {
	m.mu.Lock()
	defer m.mu.Unlock()
	check.Name = name
	m.healthChecks[name] = check
}

// RemoveHealthCheck removes a health check
func (m *Monitor) RemoveHealthCheck(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.healthChecks, name)
	delete(m.healthResults, name)
}

// GetHealthStatus returns the overall health status
func (m *Monitor) GetHealthStatus() HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.healthResults) == 0 {
		return StatusHealthy
	}

	hasUnhealthy := false
	hasDegraded := false

	for _, result := range m.healthResults {
		switch result.Status {
		case StatusUnhealthy:
			if result.Critical {
				return StatusUnhealthy
			}
			hasUnhealthy = true
		case StatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return StatusDegraded
	}
	if hasDegraded {
		return StatusDegraded
	}

	return StatusHealthy
}

// GetHealthResults returns all health check results
func (m *Monitor) GetHealthResults() map[string]*HealthResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make(map[string]*HealthResult)
	for k, v := range m.healthResults {
		results[k] = v
	}
	return results
}

// GetEvents returns recent events
func (m *Monitor) GetEvents(limit int) []Event {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || limit > len(m.events) {
		limit = len(m.events)
	}

	start := len(m.events) - limit
	events := make([]Event, limit)
	copy(events, m.events[start:])
	return events
}

// GetMetrics returns current metrics
func (m *Monitor) GetMetrics() *Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := *m.metrics
	metrics.Uptime = time.Since(m.startTime)
	return &metrics
}

// AddEventHandler adds an event handler
func (m *Monitor) AddEventHandler(handler func(Event)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventHandlers = append(m.eventHandlers, handler)
}

// responseWrapper wraps http.ResponseWriter to capture status code
type responseWrapper struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWrapper) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWrapper) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = 200
		rw.written = true
	}
	return rw._, _ = ResponseWriter.Write(b)
}

// Middleware returns a middleware function for request monitoring
func (m *Monitor) Middleware() context.Middleware {
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if !m.enabled {
				next(c)
				return
			}

			start := time.Now()
			atomic.AddInt64(&m.metrics.RequestCount, 1)

			// Wrap response writer to capture status code
			wrapper := &responseWrapper{
				ResponseWriter: c.Writer,
				statusCode:     200, // default status
			}
			c.Writer = wrapper

			// Execute the handler
			next(c)

			// Capture response info
			duration := time.Since(start)
			data := map[string]interface{}{
				"method":      c.Request.Method,
				"path":        c.Request.URL.Path,
				"status_code": wrapper.statusCode,
				"duration_ms": duration.Milliseconds(),
				"user_agent":  c.GetHeader("User-Agent"),
				"remote_addr": c.Request.RemoteAddr,
			}

			if wrapper.statusCode >= 400 {
				atomic.AddInt64(&m.metrics.ErrorCount, 1)
				m.EmitEvent(EventError, fmt.Sprintf("HTTP %d: %s %s", wrapper.statusCode, c.Request.Method, c.Request.URL.Path), data)
			} else {
				m.EmitEvent(EventRequest, fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path), data)
			}
		}
	}
}

// runHealthChecks runs health checks periodically
func (m *Monitor) runHealthChecks(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		m.executeHealthChecks()
	}
}

// executeHealthChecks executes all registered health checks
func (m *Monitor) executeHealthChecks() {
	m.mu.RLock()
	checks := make(map[string]*HealthCheck)
	for k, v := range m.healthChecks {
		checks[k] = v
	}
	m.mu.RUnlock()

	for name, check := range checks {
		go func(name string, check *HealthCheck) {
			start := time.Now()
			status, err := check.Check()
			duration := time.Since(start)

			result := &HealthResult{
				Name:      name,
				Status:    status,
				Timestamp: time.Now(),
				Duration:  duration,
				Critical:  check.Critical,
			}

			if err != nil {
				result.Message = err.Error()
				result.Status = StatusUnhealthy
			}

			m.mu.Lock()
			m.healthResults[name] = result
			m.mu.Unlock()

			// Emit event
			eventType := EventHealthy
			if status != StatusHealthy {
				eventType = EventUnhealthy
			}

			data := map[string]interface{}{
				"check_name": name,
				"status":     string(status),
				"duration":   duration.Milliseconds(),
				"critical":   check.Critical,
			}
			if err != nil {
				data["error"] = err.Error()
			}

			m.EmitEvent(eventType, fmt.Sprintf("Health check '%s': %s", name, status), data)
		}(name, check)
	}
}

// updateMetrics updates system metrics periodically
func (m *Monitor) updateMetrics() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		m.mu.Lock()
		m.metrics.MemoryUsage = memStats.Alloc
		m.metrics.GoRoutines = runtime.NumGoroutine()
		m.mu.Unlock()
	}
}

// HealthHandler returns an HTTP handler for health checks
func (m *Monitor) HealthHandler() context.HandlerFunc {
	return func(c *context.Context) {
		status := m.GetHealthStatus()
		results := m.GetHealthResults()

		response := map[string]interface{}{
			"status":    string(status),
			"timestamp": time.Now(),
			"checks":    results,
		}

		statusCode := http.StatusOK
		if status == StatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		} else if status == StatusDegraded {
			statusCode = http.StatusPartialContent
		}

		_ = c.Status(statusCode).JSON(statusCode, response)
	}
}

// MetricsHandler returns an HTTP handler for metrics
func (m *Monitor) MetricsHandler() context.HandlerFunc {
	return func(c *context.Context) {
		metrics := m.GetMetrics()
		_ = c.JSON(http.StatusOK, metrics)
	}
}

// EventsHandler returns an HTTP handler for events
func (m *Monitor) EventsHandler() context.HandlerFunc {
	return func(c *context.Context) {
		limit := 100
		if l := c.Query("limit"); l != "" {
			if parsed, err := json.Marshal(l); err == nil {
				limit = int(parsed[0])
			}
		}

		events := m.GetEvents(limit)
		_ = c.JSON(http.StatusOK, map[string]interface{}{
			"events": events,
			"total":  len(m.events),
		})
	}
}
