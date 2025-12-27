package monitoring

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	context "github.com/arthurlch/goryu/goryuctx"
)

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

type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Duration  time.Duration          `json:"duration,omitempty"`
}

type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusDegraded  HealthStatus = "degraded"
)

type HealthCheck struct {
	Name     string                                  `json:"name"`
	Check    func() (status HealthStatus, err error) `json:"-"`
	Timeout  time.Duration                           `json:"timeout"`
	Interval time.Duration                           `json:"interval"`
	Critical bool                                    `json:"critical"`
}

type HealthResult struct {
	Name      string        `json:"name"`
	Status    HealthStatus  `json:"status"`
	Message   string        `json:"message,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration"`
	Critical  bool          `json:"critical"`
}

type Metrics struct {
	RequestCount      int64                    `json:"request_count"`
	ErrorCount        int64                    `json:"error_count"`
	AvgResponseTime   time.Duration            `json:"avg_response_time"`
	Uptime            time.Duration            `json:"uptime"`
	MemoryUsage       uint64                   `json:"memory_usage_bytes"`
	GoRoutines        int                      `json:"goroutines"`
	StartTime         time.Time                `json:"start_time"`
	RouteMetrics      map[string]*RouteMetric  `json:"route_metrics"`
	MiddlewareMetrics map[string]*MiddlewareMetric `json:"middleware_metrics"`
	StatusCodeCounts  map[int]int64            `json:"status_code_counts"`
	ActiveRequests    int64                    `json:"active_requests"`
	TotalResponseTime int64                    `json:"total_response_time_ms"`
}

type RouteMetric struct {
	Pattern         string        `json:"pattern"`
	Method          string        `json:"method"`
	RequestCount    int64         `json:"request_count"`
	ErrorCount      int64         `json:"error_count"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
	StatusCodes     map[int]int64 `json:"status_codes"`
}

type MiddlewareMetric struct {
	Name            string        `json:"name"`
	ExecutionCount  int64         `json:"execution_count"`
	AvgExecutionTime time.Duration `json:"avg_execution_time"`
	ErrorCount      int64         `json:"error_count"`
}

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
	safeExecute   bool
	
	// Optimization: Async event processing
	eventsChan chan Event
	done       chan struct{}
}

type Config struct {
	Enabled        bool          `json:"enabled"`
	MaxEvents      int           `json:"max_events"`
	HealthInterval time.Duration `json:"health_interval"`
	MetricsEnabled bool          `json:"metrics_enabled"`
	SafeExecute    bool          `json:"safe_execute"`
	// Optimization: Buffer size for processing channel
	EventBufferSize int           `json:"event_buffer_size"`
}

func New(config ...Config) *Monitor {
	cfg := Config{
		Enabled:         true,
		MaxEvents:       1000,
		HealthInterval:  30 * time.Second,
		MetricsEnabled:  true,
		SafeExecute:     true,
		EventBufferSize: 1000, // Default buffer size
	}
	if len(config) > 0 {
		cfg = config[0]
		if cfg.MaxEvents == 0 {
			cfg.MaxEvents = 1000
		}
		if cfg.HealthInterval == 0 {
			cfg.HealthInterval = 30 * time.Second
		}
		if cfg.EventBufferSize == 0 {
			cfg.EventBufferSize = 1000
		}
		if !cfg.Enabled {
			if cfg.MaxEvents != 0 || cfg.HealthInterval != 0 || cfg.MetricsEnabled || cfg.SafeExecute {
			} else {
				cfg.Enabled = true
			}
		}
	}

	m := &Monitor{
		events:        make([]Event, 0, cfg.MaxEvents),
		maxEvents:     cfg.MaxEvents,
		healthChecks:  make(map[string]*HealthCheck),
		healthResults: make(map[string]*HealthResult),
		startTime:     time.Now(),
		enabled:       cfg.Enabled,
		safeExecute:   cfg.SafeExecute,
		metrics: &Metrics{
			StartTime:         time.Now(),
			RouteMetrics:      make(map[string]*RouteMetric),
			MiddlewareMetrics: make(map[string]*MiddlewareMetric),
			StatusCodeCounts:  make(map[int]int64),
		},
		// Initialize async channels
		eventsChan: make(chan Event, cfg.EventBufferSize),
		done:       make(chan struct{}),
	}

	// Start processing worker
	go m.processEventsWorker()

	if cfg.MetricsEnabled {
		go m.updateMetrics()
	}

	if cfg.HealthInterval > 0 {
		go m.runHealthChecks(cfg.HealthInterval)
	}

	m.EmitEvent(EventStartup, "Monitoring system started", nil)
	return m
}

func (m *Monitor) processEventsWorker() {
	defer close(m.done) // Signal completion when channel is closed and drained
	for event := range m.eventsChan {
		m.mu.Lock()
		if len(m.events) >= m.maxEvents {
			m.events = m.events[1:]
		}
		m.events = append(m.events, event)
		m.mu.Unlock()

		m.mu.RLock()
		handlers := make([]func(Event), len(m.eventHandlers))
		copy(handlers, m.eventHandlers)
		m.mu.RUnlock()

		// Executing handlers is also moved off the request path
		for _, handler := range handlers {
			// Still launch goroutines for handlers to prevent one blocking the worker
			go m.safeExecuteEventHandler(handler, event)
		}
	}
}

// Close gracefully shuts down the monitor
func (m *Monitor) Close() {
	close(m.eventsChan)
	// Do not close done here, let worker close it
}

// Wait blocks until the monitor worker has finished processing events
func (m *Monitor) Wait() {
	<-m.done
}

func (m *Monitor) safeExecuteEventHandler(handler func(Event), event Event) {
	if !m.safeExecute {
		handler(event)
		return
	}
	
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Event handler panicked: %v", r)
			
			// Restore functionality: Emit error event
			data := map[string]interface{}{
				"panic_value":   fmt.Sprintf("%v", r),
				"handler_error": true,
			}
			m.EmitEvent(EventError, "Event handler panicked", data)
		}
	}()
	
	handler(event)
}

func (m *Monitor) safeExecuteHealthCheck(name string, check *HealthCheck) {
	if !m.safeExecute {
		m.executeHealthCheck(name, check)
		return
	}
	
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Health check '%s' panicked: %v", name, r)
			
			result := &HealthResult{
				Name:      name,
				Status:    StatusUnhealthy,
				Message:   fmt.Sprintf("Health check panicked: %v", r),
				Timestamp: time.Now(),
				Duration:  0,
				Critical:  check.Critical,
			}
			
			m.mu.Lock()
			m.healthResults[name] = result
			m.mu.Unlock()
			
			data := map[string]interface{}{
				"check_name":    name,
				"status":        string(StatusUnhealthy),
				"critical":      check.Critical,
				"panic_value":   fmt.Sprintf("%v", r),
				"health_error":  true,
			}
			m.EmitEvent(EventUnhealthy, fmt.Sprintf("Health check '%s' panicked: %v", name, r), data)
		}
	}()
	
	m.executeHealthCheck(name, check)
}

func (m *Monitor) executeHealthCheck(name string, check *HealthCheck) {
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
}

func (m *Monitor) EmitEvent(eventType EventType, message string, data map[string]interface{}) {
	if !m.enabled {
		return
	}

	event := Event{
		// Optimization: Use strconv instead of fmt.Sprintf
		ID:        strconv.FormatInt(time.Now().UnixNano(), 10),
		Type:      eventType,
		Timestamp: time.Now(),
		Message:   message,
		Data:      data,
	}

	// Non-blocking send (optional: drop event if channel full to preserve performance)
	// Non-blocking send (optional: drop event if channel full to preserve performance)
	// Also protect against send on closed channel
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Failed to emit event (monitor closed): %s", message)
		}
	}()
	
	select {
	case m.eventsChan <- event:
		// Sent
	default:
		// Channel full, drop event to prevent blocking application
		log.Printf("Monitor event buffer full, dropping event: %s", message)
	}
}

func (m *Monitor) AddHealthCheck(name string, check *HealthCheck) {
	m.mu.Lock()
	defer m.mu.Unlock()
	check.Name = name
	m.healthChecks[name] = check
}

func (m *Monitor) RemoveHealthCheck(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.healthChecks, name)
	delete(m.healthResults, name)
}

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

func (m *Monitor) GetHealthResults() map[string]*HealthResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make(map[string]*HealthResult)
	for k, v := range m.healthResults {
		results[k] = v
	}
	return results
}

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

func (m *Monitor) GetMetrics() *Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := *m.metrics
	metrics.Uptime = time.Since(m.startTime)
	if m.metrics.RequestCount > 0 {
		metrics.AvgResponseTime = time.Duration(m.metrics.TotalResponseTime/m.metrics.RequestCount) * time.Millisecond
	}
	return &metrics
}

func (m *Monitor) AddEventHandler(handler func(Event)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventHandlers = append(m.eventHandlers, handler)
}

func (m *Monitor) MiddlewareWrapper(name string, middleware context.Middleware) context.Middleware {
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if !m.enabled {
				middleware(next)(c)
				return
			}
			
			start := time.Now()
			
			middleware(next)(c)
			
			duration := time.Since(start)
			
			m.mu.Lock()
			if metric, exists := m.metrics.MiddlewareMetrics[name]; exists {
				metric.ExecutionCount++
				totalTime := metric.AvgExecutionTime.Nanoseconds()*metric.ExecutionCount + duration.Nanoseconds()
				metric.AvgExecutionTime = time.Duration(totalTime / (metric.ExecutionCount + 1))
			} else {
				m.metrics.MiddlewareMetrics[name] = &MiddlewareMetric{
					Name:             name,
					ExecutionCount:   1,
					AvgExecutionTime: duration,
					ErrorCount:       0,
				}
			}
			m.mu.Unlock()
			
			if duration > 100*time.Millisecond {
				data := map[string]interface{}{
					"middleware_name": name,
					"duration_ms":     duration.Milliseconds(),
					"slow_middleware": true,
				}
				m.EmitEvent(EventCustom, fmt.Sprintf("Slow middleware '%s': %v", name, duration), data)
			}
		}
	}
}

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
	return rw.ResponseWriter.Write(b)
}

func (m *Monitor) Middleware() context.Middleware {
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if !m.enabled {
				next(c)
				return
			}

			start := time.Now()
			atomic.AddInt64(&m.metrics.RequestCount, 1)
			atomic.AddInt64(&m.metrics.ActiveRequests, 1)

			correlationID := c.GetHeader("X-Correlation-ID")
			if correlationID == "" {
				correlationID = m.generateCorrelationID()
				c.SetHeader("X-Correlation-ID", correlationID)
			}
			c.Set("correlation_id", correlationID)

			wrapper := &responseWrapper{
				ResponseWriter: c.Writer,
				statusCode:     200,
			}
			c.Writer = wrapper

			next(c)

			duration := time.Since(start)
			atomic.AddInt64(&m.metrics.ActiveRequests, -1)
			atomic.AddInt64(&m.metrics.TotalResponseTime, duration.Milliseconds())
			
			m.mu.Lock()
			m.metrics.StatusCodeCounts[wrapper.statusCode]++
			
			route, _ := c.Get("route.pattern")
			if route != nil {
				routeKey := fmt.Sprintf("%s:%s", c.Request.Method, route)
				if metric, exists := m.metrics.RouteMetrics[routeKey]; exists {
					metric.RequestCount++
					if wrapper.statusCode >= 400 {
						metric.ErrorCount++
					}
					if metric.StatusCodes == nil {
						metric.StatusCodes = make(map[int]int64)
					}
					metric.StatusCodes[wrapper.statusCode]++
				} else {
					m.metrics.RouteMetrics[routeKey] = &RouteMetric{
						Pattern:      fmt.Sprintf("%v", route),
						Method:       c.Request.Method,
						RequestCount: 1,
						ErrorCount:   func() int64 { if wrapper.statusCode >= 400 { return 1 }; return 0 }(),
						StatusCodes:  map[int]int64{wrapper.statusCode: 1},
					}
				}
			}
			m.mu.Unlock()
			
			data := map[string]interface{}{
				"method":         c.Request.Method,
				"path":           c.Request.URL.Path,
				"status_code":    wrapper.statusCode,
				"duration_ms":    duration.Milliseconds(),
				"user_agent":     c.GetHeader("User-Agent"),
				"remote_addr":    c.Request.RemoteAddr,
				"correlation_id": correlationID,
			}
			
			if route != nil {
				data["route_pattern"] = route
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

func (m *Monitor) runHealthChecks(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		m.executeHealthChecks()
	}
}

func (m *Monitor) executeHealthChecks() {
	m.mu.RLock()
	checks := make(map[string]*HealthCheck)
	for k, v := range m.healthChecks {
		checks[k] = v
	}
	m.mu.RUnlock()

	for name, check := range checks {
		go m.safeExecuteHealthCheck(name, check)
	}
}

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

func (m *Monitor) MetricsHandler() context.HandlerFunc {
	return func(c *context.Context) {
		metrics := m.GetMetrics()
		_ = c.JSON(http.StatusOK, metrics)
	}
}

func (m *Monitor) EventsHandler() context.HandlerFunc {
	return func(c *context.Context) {
		limit := 100
		if l := c.Query("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				limit = parsed
			}
		}

		events := m.GetEvents(limit)
		_ = c.JSON(http.StatusOK, map[string]interface{}{
			"events": events,
			"total":  len(m.events),
		})
	}
}

func (m *Monitor) generateCorrelationID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}
