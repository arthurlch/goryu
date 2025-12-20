package tracing_test
import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	goryucontext "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
	"github.com/arthurlch/goryu/middleware/tracing"
)
func newTestContext(req *http.Request) (*goryucontext.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return goryucontext.NewContext(rr, req), rr
}
func TestTracingMiddleware(t *testing.T) {
	t.Run("BasicTracing", func(t *testing.T) {
		tracer := tracing.NewSimpleTracer()
		config := tracing.Config{
			Tracer: tracer,
		}
		middleware := tracing.New(config)
		handler := func(c *goryucontext.Context) {
			c.Text(http.StatusOK, "Hello, World!")
		}
		req := httptest.NewRequest("GET", "/test", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		spans := tracer.GetSpans()
		if len(spans) != 1 {
			t.Errorf("Expected 1 span, got %d", len(spans))
		}
		span := spans[0]
		if span.Name != "HTTP GET /test" {
			t.Errorf("Expected span name 'HTTP GET /test', got '%s'", span.Name)
		}
		if span.Tags["http.method"] != "GET" {
			t.Errorf("Expected http.method=GET, got %v", span.Tags["http.method"])
		}
		if span.Tags["http.status_code"] != 200 {
			t.Errorf("Expected http.status_code=200, got %v", span.Tags["http.status_code"])
		}
	})
	t.Run("CustomSpanName", func(t *testing.T) {
		tracer := tracing.NewSimpleTracer()
		config := tracing.Config{
			Tracer: tracer,
			SpanNameGenerator: func(c *goryucontext.Context) string {
				return "custom_span_name"
			},
		}
		middleware := tracing.New(config)
		handler := func(c *goryucontext.Context) {
			c.Text(http.StatusOK, "Success")
		}
		req := httptest.NewRequest("POST", "/custom", nil)
		ctx, _ := newTestContext(req)
		middleware(handler)(ctx)
		spans := tracer.GetSpans()
		if len(spans) != 1 {
			t.Errorf("Expected 1 span, got %d", len(spans))
		}
		if spans[0].Name != "custom_span_name" {
			t.Errorf("Expected span name 'custom_span_name', got '%s'", spans[0].Name)
		}
	})
	t.Run("CustomTags", func(t *testing.T) {
		tracer := tracing.NewSimpleTracer()
		config := tracing.Config{
			Tracer: tracer,
			CustomTags: func(c *goryucontext.Context) map[string]interface{} {
				return map[string]interface{}{
					"service":    "api",
					"version":    "v1.0",
					"custom.tag": "custom_value",
				}
			},
		}
		middleware := tracing.New(config)
		handler := func(c *goryucontext.Context) {
			c.Text(http.StatusOK, "Success")
		}
		req := httptest.NewRequest("GET", "/api/users", nil)
		ctx, _ := newTestContext(req)
		middleware(handler)(ctx)
		spans := tracer.GetSpans()
		if len(spans) != 1 {
			t.Errorf("Expected 1 span, got %d", len(spans))
		}
		span := spans[0]
		if span.Tags["service"] != "api" {
			t.Errorf("Expected service=api, got %v", span.Tags["service"])
		}
		if span.Tags["version"] != "v1.0" {
			t.Errorf("Expected version=v1.0, got %v", span.Tags["version"])
		}
		if span.Tags["custom.tag"] != "custom_value" {
			t.Errorf("Expected custom.tag=custom_value, got %v", span.Tags["custom.tag"])
		}
	})
	t.Run("ErrorStatus", func(t *testing.T) {
		tracer := tracing.NewSimpleTracer()
		config := tracing.Config{
			Tracer: tracer,
		}
		middleware := tracing.New(config)
		handler := func(c *goryucontext.Context) {
			c.Text(http.StatusInternalServerError, "Server Error")
		}
		req := httptest.NewRequest("GET", "/error", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", rr.Code)
		}
		spans := tracer.GetSpans()
		if len(spans) != 1 {
			t.Errorf("Expected 1 span, got %d", len(spans))
		}
		span := spans[0]
		if span.Status.Code != tracing.StatusCodeError {
			t.Errorf("Expected error status, got %v", span.Status.Code)
		}
		if span.Status.Message != "Internal Server Error" {
			t.Errorf("Expected error message 'Internal Server Error', got '%s'", span.Status.Message)
		}
	})
	t.Run("SkipMiddleware", func(t *testing.T) {
		tracer := tracing.NewSimpleTracer()
		config := tracing.Config{
			Tracer: tracer,
			BaseConfig: base.BaseConfig{
				Skip: func(c *goryucontext.Context) bool {
					return c.Request.URL.Path == "/health"
				},
			},
		}
		middleware := tracing.New(config)
		handler := func(c *goryucontext.Context) {
			c.Text(http.StatusOK, "Healthy")
		}
		req := httptest.NewRequest("GET", "/health", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		spans := tracer.GetSpans()
		if len(spans) != 0 {
			t.Errorf("Expected 0 spans for skipped request, got %d", len(spans))
		}
	})
	t.Run("SpanEvents", func(t *testing.T) {
		tracer := tracing.NewSimpleTracer()
		config := tracing.Config{
			Tracer: tracer,
		}
		middleware := tracing.New(config)
		handler := func(c *goryucontext.Context) {
			time.Sleep(1 * time.Millisecond)
			c.Text(http.StatusOK, "Success")
		}
		req := httptest.NewRequest("GET", "/test", nil)
		ctx, _ := newTestContext(req)
		middleware(handler)(ctx)
		spans := tracer.GetSpans()
		if len(spans) != 1 {
			t.Errorf("Expected 1 span, got %d", len(spans))
		}
		span := spans[0]
		if len(span.Events) < 2 {
			t.Errorf("Expected at least 2 events (start/end), got %d", len(span.Events))
		}
		hasStart := false
		hasEnd := false
		for _, event := range span.Events {
			if event.Name == "request.start" {
				hasStart = true
			}
			if event.Name == "request.end" {
				hasEnd = true
			}
		}
		if !hasStart {
			t.Error("Expected request.start event")
		}
		if !hasEnd {
			t.Error("Expected request.end event")
		}
	})
	t.Run("ParentSpanContext", func(t *testing.T) {
		tracer := tracing.NewSimpleTracer()
		config := tracing.Config{
			Tracer: tracer,
		}
		middleware := tracing.New(config)
		handler := func(c *goryucontext.Context) {
			c.Text(http.StatusOK, "Success")
		}
		req := httptest.NewRequest("GET", "/child", nil)
		req.Header.Set("X-Trace-Id", "parent-trace-123")
		req.Header.Set("X-Span-Id", "parent-span-456")
		req.Header.Set("X-Sampled", "1")
		ctx, _ := newTestContext(req)
		middleware(handler)(ctx)
		spans := tracer.GetSpans()
		if len(spans) != 1 {
			t.Errorf("Expected 1 span, got %d", len(spans))
		}
		span := spans[0]
		if span.TraceID != "parent-trace-123" {
			t.Errorf("Expected child span to inherit trace ID, got %s", span.TraceID)
		}
		if span.ParentSpanID != "parent-span-456" {
			t.Errorf("Expected parent span ID to be set, got %s", span.ParentSpanID)
		}
	})
	t.Run("ContextExtraction", func(t *testing.T) {
		tracer := tracing.NewSimpleTracer()
		config := tracing.Config{
			Tracer: tracer,
		}
		middleware := tracing.New(config)
		var extractedSpan tracing.Span
		handler := func(c *goryucontext.Context) {
			span, exists := tracing.GetSpan(c)
			if exists {
				extractedSpan = span
			}
			c.Text(http.StatusOK, "Success")
		}
		req := httptest.NewRequest("GET", "/test", nil)
		ctx, _ := newTestContext(req)
		middleware(handler)(ctx)
		if extractedSpan == nil {
			t.Error("Expected to extract span from context")
		}
		spans := tracer.GetSpans()
		if len(spans) != 1 {
			t.Errorf("Expected 1 span, got %d", len(spans))
		}
		if extractedSpan.Context().SpanID() != spans[0].SpanID {
			t.Error("Extracted span does not match created span")
		}
	})
}
func TestSimpleTracer(t *testing.T) {
	t.Run("TracerOperations", func(t *testing.T) {
		tracer := tracing.NewSimpleTracer()
		headers := http.Header{}
		headers.Set("X-Trace-Id", "test-trace-123")
		headers.Set("X-Span-Id", "test-span-456")
		headers.Set("X-Sampled", "1")
		spanCtx, err := tracer.Extract(headers)
		if err != nil {
			t.Fatalf("Error creating span context: %v", err)
		}
		headers = http.Header{}
		err = tracer.Inject(spanCtx, headers)
		if err != nil {
			t.Errorf("Error injecting headers: %v", err)
		}
		if headers.Get("X-Trace-Id") != "test-trace-123" {
			t.Errorf("Expected X-Trace-Id=test-trace-123, got %s", headers.Get("X-Trace-Id"))
		}
		if headers.Get("X-Span-Id") != "test-span-456" {
			t.Errorf("Expected X-Span-Id=test-span-456, got %s", headers.Get("X-Span-Id"))
		}
		if headers.Get("X-Sampled") != "1" {
			t.Errorf("Expected X-Sampled=1, got %s", headers.Get("X-Sampled"))
		}
		extractedCtx, err := tracer.Extract(headers)
		if err != nil {
			t.Errorf("Error extracting headers: %v", err)
		}
		if extractedCtx.TraceID() != "test-trace-123" {
			t.Errorf("Expected extracted trace ID=test-trace-123, got %s", extractedCtx.TraceID())
		}
		if extractedCtx.SpanID() != "test-span-456" {
			t.Errorf("Expected extracted span ID=test-span-456, got %s", extractedCtx.SpanID())
		}
		if !extractedCtx.IsSampled() {
			t.Error("Expected extracted context to be sampled")
		}
	})
	t.Run("SpanOperations", func(t *testing.T) {
		tracer := tracing.NewSimpleTracer()
		span, _ := tracer.StartSpan(context.Background(), "test-span")
		span.SetTag("key1", "value1")
		span.SetTag("key2", 123)
		span.SetStatus(tracing.StatusCodeOk, "All good")
		span.AddEvent("test.event", map[string]interface{}{
			"attribute": "value",
		})
		span.End()
		spans := tracer.GetSpans()
		if len(spans) != 1 {
			t.Errorf("Expected 1 span, got %d", len(spans))
		}
		createdSpan := spans[0]
		if createdSpan.Tags["key1"] != "value1" {
			t.Errorf("Expected tag key1=value1, got %v", createdSpan.Tags["key1"])
		}
		if createdSpan.Tags["key2"] != 123 {
			t.Errorf("Expected tag key2=123, got %v", createdSpan.Tags["key2"])
		}
		if len(createdSpan.Events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(createdSpan.Events))
		}
		if createdSpan.Events[0].Name != "test.event" {
			t.Errorf("Expected event name=test.event, got %s", createdSpan.Events[0].Name)
		}
	})
}
func TestNoopTracer(t *testing.T) {
	t.Run("DefaultNoopTracer", func(t *testing.T) {
		config := tracing.Config{}
		middleware := tracing.New(config)
		handler := func(c *goryucontext.Context) {
			c.Text(http.StatusOK, "Success")
		}
		req := httptest.NewRequest("GET", "/test", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})
}
func BenchmarkTracingMiddleware(b *testing.B) {
	tracer := tracing.NewSimpleTracer()
	config := tracing.Config{
		Tracer: tracer,
	}
	middleware := tracing.New(config)
	handler := func(c *goryucontext.Context) {
		c.Text(http.StatusOK, "Benchmark")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/bench", nil)
		ctx, _ := newTestContext(req)
		middleware(handler)(ctx)
	}
}
func BenchmarkTracingWithParent(b *testing.B) {
	tracer := tracing.NewSimpleTracer()
	config := tracing.Config{
		Tracer: tracer,
	}
	middleware := tracing.New(config)
	handler := func(c *goryucontext.Context) {
		c.Text(http.StatusOK, "Benchmark")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/bench", nil)
		req.Header.Set("X-Trace-Id", "parent-trace")
		req.Header.Set("X-Span-Id", "parent-span")
		req.Header.Set("X-Sampled", "1")
		ctx, _ := newTestContext(req)
		middleware(handler)(ctx)
	}
}
