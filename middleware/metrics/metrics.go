package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/arthurlch/goryu"
)

// Metrics interface defines the operations for collecting metrics
type Metrics interface {
	// Counter operations
	IncrementCounter(name string, tags map[string]string)
	AddToCounter(name string, value float64, tags map[string]string)

	// Histogram operations (for timing and distributions)
	RecordHistogram(name string, value float64, tags map[string]string)

	// Gauge operations (for current values)
	SetGauge(name string, value float64, tags map[string]string)
	AddToGauge(name string, value float64, tags map[string]string)
}

// Config defines the configuration for metrics middleware
type Config struct {
	// Metrics implementation to use
	Metrics Metrics
	// Skip defines when to skip metrics collection
	Skip func(c *goryu.Context) bool
	// CustomTags allows adding custom tags to all metrics
	CustomTags func(c *goryu.Context) map[string]string
	// RecordBody determines if request/response body sizes should be recorded
	RecordBody bool
	// GroupStatusCode groups similar status codes (e.g., 2xx, 4xx, 5xx)
	GroupStatusCode bool
	// Prefix for all metric names
	Prefix string
}

// responseWriter wraps http.ResponseWriter to capture metrics
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = 200
	}
	n, err := rw._, _ = ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// New creates a new metrics middleware
func New(config Config) goryu.Middleware {
	if config.Metrics == nil {
		config.Metrics = &noopMetrics{}
	}
	if config.Prefix == "" {
		config.Prefix = "http"
	}

	return func(next goryu.HandlerFunc) goryu.HandlerFunc {
		return func(c *goryu.Context) {
			if config.Skip != nil && config.Skip(c) {
				next(c)
				return
			}

			start := time.Now()

			// Wrap response writer
			rw := &responseWriter{
				ResponseWriter: c.Writer,
				statusCode:     200,
			}
			c.Writer = rw

			// Track active requests (increment at start)
			config.Metrics.AddToGauge(config.Prefix+"_requests_active", 1, map[string]string{})

			// Execute request
			next(c)

			// Calculate metrics
			duration := time.Since(start)

			// Build tags
			tags := map[string]string{
				"method": c.Request.Method,
				"path":   c.Request.URL.Path,
			}

			// Add status code tag
			if config.GroupStatusCode {
				tags["status_class"] = getStatusClass(rw.statusCode)
			} else {
				tags["status"] = strconv.Itoa(rw.statusCode)
			}

			// Add custom tags
			if config.CustomTags != nil {
				for k, v := range config.CustomTags(c) {
					tags[k] = v
				}
			}

			// Record metrics
			config.Metrics.IncrementCounter(config.Prefix+"_requests_total", tags)
			config.Metrics.RecordHistogram(config.Prefix+"_request_duration_seconds", duration.Seconds(), tags)

			// Record body sizes if enabled
			if config.RecordBody {
				config.Metrics.RecordHistogram(config.Prefix+"_request_size_bytes", float64(c.Request.ContentLength), tags)
				config.Metrics.RecordHistogram(config.Prefix+"_response_size_bytes", float64(rw.size), tags)
			}

			// Track active requests (decrement at end)
			config.Metrics.AddToGauge(config.Prefix+"_requests_active", -1, map[string]string{})
		}
	}
}

// getStatusClass returns the status code class (2xx, 4xx, etc.)
func getStatusClass(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return "2xx"
	case statusCode >= 300 && statusCode < 400:
		return "3xx"
	case statusCode >= 400 && statusCode < 500:
		return "4xx"
	case statusCode >= 500:
		return "5xx"
	default:
		return "1xx"
	}
}

// noopMetrics is a no-op implementation of Metrics
type noopMetrics struct{}

func (n *noopMetrics) IncrementCounter(name string, tags map[string]string)               {}
func (n *noopMetrics) AddToCounter(name string, value float64, tags map[string]string)    {}
func (n *noopMetrics) RecordHistogram(name string, value float64, tags map[string]string) {}
func (n *noopMetrics) SetGauge(name string, value float64, tags map[string]string)        {}
func (n *noopMetrics) AddToGauge(name string, value float64, tags map[string]string)      {}

// PrometheusMetrics implements Metrics interface for Prometheus
// This is a basic implementation - in real usage you'd use prometheus client
type PrometheusMetrics struct {
	// In a real implementation, this would contain prometheus metric objects
	counters   map[string]float64
	histograms map[string][]float64
	gauges     map[string]float64
}

// NewPrometheusMetrics creates a new Prometheus metrics implementation
func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		counters:   make(map[string]float64),
		histograms: make(map[string][]float64),
		gauges:     make(map[string]float64),
	}
}

func (p *PrometheusMetrics) IncrementCounter(name string, tags map[string]string) {
	p.AddToCounter(name, 1, tags)
}

func (p *PrometheusMetrics) AddToCounter(name string, value float64, tags map[string]string) {
	key := buildKey(name, tags)
	p.counters[key] += value
}

func (p *PrometheusMetrics) RecordHistogram(name string, value float64, tags map[string]string) {
	key := buildKey(name, tags)
	p.histograms[key] = append(p.histograms[key], value)
}

func (p *PrometheusMetrics) SetGauge(name string, value float64, tags map[string]string) {
	key := buildKey(name, tags)
	p.gauges[key] = value
}

func (p *PrometheusMetrics) AddToGauge(name string, value float64, tags map[string]string) {
	key := buildKey(name, tags)
	p.gauges[key] += value
}

// buildKey creates a unique key from metric name and tags
func buildKey(name string, tags map[string]string) string {
	key := name
	// Sort keys to ensure consistent key generation
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}

	// Use a simple sort (since we don't have access to sort package)
	for i := 0; i < len(keys)-1; i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	for _, k := range keys {
		key += ":" + k + "=" + tags[k]
	}
	return key
}
