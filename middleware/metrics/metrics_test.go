package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/metrics"
)

type testMetrics struct {
	counters   map[string]float64
	histograms map[string][]float64
	gauges     map[string]float64
}

func newTestMetrics() *testMetrics {
	return &testMetrics{
		counters:   make(map[string]float64),
		histograms: make(map[string][]float64),
		gauges:     make(map[string]float64),
	}
}

func (m *testMetrics) IncrementCounter(name string, tags map[string]string) {
	m.AddToCounter(name, 1, tags)
}

func (m *testMetrics) AddToCounter(name string, value float64, tags map[string]string) {
	key := buildTestKey(name, tags)
	m.counters[key] += value
}

func (m *testMetrics) RecordHistogram(name string, value float64, tags map[string]string) {
	key := buildTestKey(name, tags)
	m.histograms[key] = append(m.histograms[key], value)
}

func (m *testMetrics) SetGauge(name string, value float64, tags map[string]string) {
	key := buildTestKey(name, tags)
	m.gauges[key] = value
}

func (m *testMetrics) AddToGauge(name string, value float64, tags map[string]string) {
	key := buildTestKey(name, tags)
	m.gauges[key] += value
}

func buildTestKey(name string, tags map[string]string) string {
	key := name
	// Sort keys to ensure consistent key generation
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}

	// Use a simple sort
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

func getKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func newTestContext(req *http.Request) (*goryu.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}

func TestMetricsMiddleware(t *testing.T) {
	t.Run("BasicMetrics", func(t *testing.T) {
		m := newTestMetrics()
		config := metrics.Config{
			Metrics: m,
			Prefix:  "test",
		}
		middleware := metrics.New(config)

		handler := func(c *goryu.Context) {
			c.Text(http.StatusOK, "Hello, World!")
		}

		req := httptest.NewRequest("GET", "/test", nil)
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		// Check counter was incremented
		counterKey := "test_requests_total:method=GET:path=/test:status=200"
		if m.counters[counterKey] != 1 {
			t.Errorf("Expected counter to be 1, got %f", m.counters[counterKey])
		}

		// Check histogram was recorded
		histogramKey := "test_request_duration_seconds:method=GET:path=/test:status=200"
		if len(m.histograms[histogramKey]) != 1 {
			t.Errorf("Expected 1 histogram entry, got %d", len(m.histograms[histogramKey]))
			return
		}
		if m.histograms[histogramKey][0] <= 0 {
			t.Error("Expected positive duration")
		}
	})

	t.Run("GroupStatusCode", func(t *testing.T) {
		m := newTestMetrics()
		config := metrics.Config{
			Metrics:         m,
			Prefix:          "test",
			GroupStatusCode: true,
		}
		middleware := metrics.New(config)

		handler := func(c *goryu.Context) {
			c.Text(http.StatusNotFound, "Not Found")
		}

		req := httptest.NewRequest("GET", "/missing", nil)
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rr.Code)
		}

		// Check counter uses status class
		counterKey := "test_requests_total:method=GET:path=/missing:status_class=4xx"
		if m.counters[counterKey] != 1 {
			t.Errorf("Expected counter to be 1, got %f", m.counters[counterKey])
		}
	})

	t.Run("RecordBodySizes", func(t *testing.T) {
		m := newTestMetrics()
		config := metrics.Config{
			Metrics:    m,
			Prefix:     "test",
			RecordBody: true,
		}
		middleware := metrics.New(config)

		handler := func(c *goryu.Context) {
			c.Text(http.StatusOK, "Hello, World!")
		}

		body := strings.NewReader("test body")
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Length", "9")
		ctx, _ := newTestContext(req)

		middleware(handler)(ctx)

		// Check request size histogram
		reqSizeKey := "test_request_size_bytes:method=POST:path=/test:status=200"
		if len(m.histograms[reqSizeKey]) != 1 {
			t.Errorf("Expected 1 request size entry, got %d", len(m.histograms[reqSizeKey]))
		}
		if m.histograms[reqSizeKey][0] != 9 {
			t.Errorf("Expected request size 9, got %f", m.histograms[reqSizeKey][0])
		}

		// Check response size histogram
		respSizeKey := "test_response_size_bytes:method=POST:path=/test:status=200"
		if len(m.histograms[respSizeKey]) != 1 {
			t.Errorf("Expected 1 response size entry, got %d", len(m.histograms[respSizeKey]))
		}
		expectedSize := float64(len("Hello, World!"))
		if m.histograms[respSizeKey][0] != expectedSize {
			t.Errorf("Expected response size %f, got %f", expectedSize, m.histograms[respSizeKey][0])
		}
	})

	t.Run("CustomTags", func(t *testing.T) {
		m := newTestMetrics()
		config := metrics.Config{
			Metrics: m,
			Prefix:  "test",
			CustomTags: func(c *goryu.Context) map[string]string {
				return map[string]string{
					"service": "api",
					"version": "v1",
				}
			},
		}
		middleware := metrics.New(config)

		handler := func(c *goryu.Context) {
			c.Text(http.StatusOK, "Success")
		}

		req := httptest.NewRequest("GET", "/api/users", nil)
		ctx, _ := newTestContext(req)

		middleware(handler)(ctx)

		// Check custom tags are included
		counterKey := "test_requests_total:method=GET:path=/api/users:service=api:status=200:version=v1"
		if m.counters[counterKey] != 1 {
			t.Errorf("Expected counter with custom tags to be 1, got %f", m.counters[counterKey])
		}
	})

	t.Run("SkipMiddleware", func(t *testing.T) {
		m := newTestMetrics()
		config := metrics.Config{
			Metrics: m,
			Prefix:  "test",
			Skip: func(c *goryu.Context) bool {
				return c.Request.URL.Path == "/health"
			},
		}
		middleware := metrics.New(config)

		handler := func(c *goryu.Context) {
			c.Text(http.StatusOK, "OK")
		}

		req := httptest.NewRequest("GET", "/health", nil)
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		// Check no metrics were recorded
		if len(m.counters) != 0 {
			t.Error("Expected no metrics to be recorded for skipped requests")
		}
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		m := newTestMetrics()
		config := metrics.Config{
			Metrics: m,
			Prefix:  "test",
		}
		middleware := metrics.New(config)

		handler := func(c *goryu.Context) {
			c.Text(http.StatusInternalServerError, "Server Error")
		}

		req := httptest.NewRequest("POST", "/error", nil)
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", rr.Code)
		}

		// Check error status is recorded
		counterKey := "test_requests_total:method=POST:path=/error:status=500"
		if m.counters[counterKey] != 1 {
			t.Errorf("Expected error counter to be 1, got %f", m.counters[counterKey])
		}
	})

	t.Run("MultipleRequests", func(t *testing.T) {
		m := newTestMetrics()
		config := metrics.Config{
			Metrics: m,
			Prefix:  "test",
		}
		middleware := metrics.New(config)

		handler := func(c *goryu.Context) {
			c.Text(http.StatusOK, "Success")
		}

		// Send multiple requests
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			ctx, _ := newTestContext(req)
			middleware(handler)(ctx)
		}

		// Check counter accumulated correctly
		counterKey := "test_requests_total:method=GET:path=/test:status=200"
		if m.counters[counterKey] != 5 {
			t.Errorf("Expected counter to be 5, got %f", m.counters[counterKey])
		}

		// Check histogram recorded all requests
		histogramKey := "test_request_duration_seconds:method=GET:path=/test:status=200"
		if len(m.histograms[histogramKey]) != 5 {
			t.Errorf("Expected 5 histogram entries, got %d", len(m.histograms[histogramKey]))
		}
	})

	t.Run("DefaultNoopMetrics", func(t *testing.T) {
		// Test with no metrics implementation
		config := metrics.Config{
			Prefix: "test",
		}
		middleware := metrics.New(config)

		handler := func(c *goryu.Context) {
			c.Text(http.StatusOK, "Success")
		}

		req := httptest.NewRequest("GET", "/test", nil)
		ctx, rr := newTestContext(req)

		// Should not panic with noop metrics
		middleware(handler)(ctx)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})
}

func TestPrometheusMetrics(t *testing.T) {
	t.Run("PrometheusImplementation", func(t *testing.T) {
		m := metrics.NewPrometheusMetrics()

		// Test counter operations
		m.IncrementCounter("test_counter", map[string]string{"label": "value"})
		m.AddToCounter("test_counter", 5, map[string]string{"label": "value"})

		// Test histogram operations
		m.RecordHistogram("test_histogram", 0.5, map[string]string{"path": "/test"})
		m.RecordHistogram("test_histogram", 1.2, map[string]string{"path": "/test"})

		// Test gauge operations
		m.SetGauge("test_gauge", 10, map[string]string{"type": "active"})
		m.AddToGauge("test_gauge", 5, map[string]string{"type": "active"})

		// Verify operations work (in real implementation, you'd check Prometheus registry)
		// This is a basic test to ensure the interface is implemented correctly
		if m == nil {
			t.Error("Expected metrics implementation to be created")
		}
	})
}

func BenchmarkMetricsMiddleware(b *testing.B) {
	m := newTestMetrics()
	config := metrics.Config{
		Metrics: m,
		Prefix:  "bench",
	}
	middleware := metrics.New(config)

	handler := func(c *goryu.Context) {
		c.Text(http.StatusOK, "Benchmark")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/bench", nil)
		ctx, _ := newTestContext(req)
		middleware(handler)(ctx)
	}
}

func BenchmarkMetricsWithBody(b *testing.B) {
	m := newTestMetrics()
	config := metrics.Config{
		Metrics:    m,
		Prefix:     "bench",
		RecordBody: true,
	}
	middleware := metrics.New(config)

	handler := func(c *goryu.Context) {
		c.Text(http.StatusOK, "Benchmark with body tracking")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/bench", strings.NewReader("test body"))
		ctx, _ := newTestContext(req)
		middleware(handler)(ctx)
	}
}
