package tracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/arthurlch/goryu"
)

// Tracer interface defines the operations for distributed tracing
type Tracer interface {
	// StartSpan creates a new span with the given name and parent context
	StartSpan(ctx context.Context, name string) (Span, context.Context)
	// Extract extracts span context from HTTP headers
	Extract(headers http.Header) (SpanContext, error)
	// Inject injects span context into HTTP headers
	Inject(spanContext SpanContext, headers http.Header) error
}

// Span represents a single span in a trace
type Span interface {
	// SetTag sets a key-value pair tag on the span
	SetTag(key string, value interface{})
	// SetStatus sets the status of the span
	SetStatus(code StatusCode, message string)
	// AddEvent adds a timed event to the span
	AddEvent(name string, attributes map[string]interface{})
	// End finishes the span
	End()
	// Context returns the span's context
	Context() SpanContext
}

// SpanContext represents the span context for propagation
type SpanContext interface {
	// TraceID returns the trace ID
	TraceID() string
	// SpanID returns the span ID
	SpanID() string
	// IsSampled returns whether this trace is sampled
	IsSampled() bool
}

// StatusCode represents the status of a span
type StatusCode int

const (
	StatusCodeUnset StatusCode = iota
	StatusCodeOk
	StatusCodeError
)

// Config defines the configuration for tracing middleware
type Config struct {
	// Tracer implementation to use
	Tracer Tracer
	// Skip defines when to skip tracing
	Skip func(c *goryu.Context) bool
	// SpanNameGenerator generates span names from context
	SpanNameGenerator func(c *goryu.Context) string
	// CustomTags allows adding custom tags to spans
	CustomTags func(c *goryu.Context) map[string]interface{}
	// SampleRate determines what percentage of traces to sample (0.0 to 1.0)
	SampleRate float64
}

// contextKey is used for storing trace context in goryu.Context
type contextKey string

const traceContextKey contextKey = "trace_context"

// New creates a new tracing middleware
func New(config Config) goryu.Middleware {
	if config.Tracer == nil {
		config.Tracer = &noopTracer{}
	}
	if config.SpanNameGenerator == nil {
		config.SpanNameGenerator = func(c *goryu.Context) string {
			return fmt.Sprintf("HTTP %s %s", c.Request.Method, c.Request.URL.Path)
		}
	}
	if config.SampleRate <= 0 {
		config.SampleRate = 1.0 // Default to 100% sampling
	}

	return func(next goryu.HandlerFunc) goryu.HandlerFunc {
		return func(c *goryu.Context) {
			if config.Skip != nil && config.Skip(c) {
				next(c)
				return
			}

			// Extract parent span context from headers
			parentSpanCtx, _ := config.Tracer.Extract(c.Request.Header)

			// Create base context
			ctx := context.Background()
			if parentSpanCtx != nil {
				ctx = context.WithValue(ctx, traceContextKey, parentSpanCtx)
			}

			// Start new span
			spanName := config.SpanNameGenerator(c)
			span, spanCtx := config.Tracer.StartSpan(ctx, spanName)
			defer span.End()

			// Store span context in goryu context for downstream middleware/handlers
			c.Set("trace_context", spanCtx)
			c.Set("span", span)

			// Set standard tags
			span.SetTag("http.method", c.Request.Method)
			span.SetTag("http.url", c.Request.URL.String())
			span.SetTag("http.scheme", c.Request.URL.Scheme)
			span.SetTag("http.host", c.Request.Host)
			span.SetTag("http.user_agent", c.Request.UserAgent())
			span.SetTag("http.remote_addr", c.Request.RemoteAddr)

			// Add custom tags
			if config.CustomTags != nil {
				for k, v := range config.CustomTags(c) {
					span.SetTag(k, v)
				}
			}

			// Wrap response writer to capture status code
			rw := &tracingResponseWriter{
				ResponseWriter: c.Writer,
				span:           span,
			}
			c.Writer = rw

			// Add request start event
			span.AddEvent("request.start", map[string]interface{}{
				"timestamp": time.Now().Unix(),
			})

			// Execute request
			next(c)

			// Set final span attributes
			span.SetTag("http.status_code", rw.statusCode)
			span.SetTag("http.response_size", rw.responseSize)

			// Set span status based on HTTP status code
			if rw.statusCode >= 400 {
				span.SetStatus(StatusCodeError, http.StatusText(rw.statusCode))
			} else {
				span.SetStatus(StatusCodeOk, "")
			}

			// Add request end event
			span.AddEvent("request.end", map[string]interface{}{
				"timestamp":   time.Now().Unix(),
				"status_code": rw.statusCode,
			})
		}
	}
}

// tracingResponseWriter wraps ResponseWriter to capture response data
type tracingResponseWriter struct {
	http.ResponseWriter
	span         Span
	statusCode   int
	responseSize int
}

func (w *tracingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *tracingResponseWriter) Write(data []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = 200
	}
	n, err := w._, _ = ResponseWriter.Write(data)
	w.responseSize += n
	return n, err
}

// GetTraceContext extracts the trace context from goryu.Context
func GetTraceContext(c *goryu.Context) (context.Context, bool) {
	ctx, exists := c.Get("trace_context")
	if !exists {
		return nil, false
	}
	if traceCtx, ok := ctx.(context.Context); ok {
		return traceCtx, true
	}
	return nil, false
}

// GetSpan extracts the current span from goryu.Context
func GetSpan(c *goryu.Context) (Span, bool) {
	span, exists := c.Get("span")
	if !exists {
		return nil, false
	}
	if s, ok := span.(Span); ok {
		return s, true
	}
	return nil, false
}

// Simple in-memory tracer implementation for testing/basic usage
type SimpleTracer struct {
	spans []*SimpleSpan
}

func NewSimpleTracer() *SimpleTracer {
	return &SimpleTracer{
		spans: make([]*SimpleSpan, 0),
	}
}

func (t *SimpleTracer) StartSpan(ctx context.Context, name string) (Span, context.Context) {
	span := &SimpleSpan{
		Name:      name,
		TraceID:   generateID(),
		SpanID:    generateID(),
		StartTime: time.Now(),
		Tags:      make(map[string]interface{}),
		Events:    make([]SpanEvent, 0),
		IsSampled: true,
	}

	// Check for parent span context
	if parentCtx, ok := ctx.Value(traceContextKey).(SpanContext); ok {
		span.TraceID = parentCtx.TraceID()
		span.ParentSpanID = parentCtx.SpanID()
	}

	t.spans = append(t.spans, span)
	newCtx := context.WithValue(ctx, traceContextKey, span)

	return span, newCtx
}

func (t *SimpleTracer) Extract(headers http.Header) (SpanContext, error) {
	traceID := headers.Get("X-Trace-Id")
	spanID := headers.Get("X-Span-Id")
	sampled := headers.Get("X-Sampled") == "1"

	if traceID == "" || spanID == "" {
		return nil, fmt.Errorf("missing trace headers")
	}

	return &SimpleSpanContext{
		traceID:   traceID,
		spanID:    spanID,
		isSampled: sampled,
	}, nil
}

func (t *SimpleTracer) Inject(spanContext SpanContext, headers http.Header) error {
	headers.Set("X-Trace-Id", spanContext.TraceID())
	headers.Set("X-Span-Id", spanContext.SpanID())
	if spanContext.IsSampled() {
		headers.Set("X-Sampled", "1")
	} else {
		headers.Set("X-Sampled", "0")
	}
	return nil
}

func (t *SimpleTracer) GetSpans() []*SimpleSpan {
	return t.spans
}

type SimpleSpan struct {
	Name         string
	TraceID      string
	SpanID       string
	ParentSpanID string
	StartTime    time.Time
	EndTime      time.Time
	Tags         map[string]interface{}
	Events       []SpanEvent
	Status       SpanStatus
	IsSampled    bool
}

type SpanEvent struct {
	Name       string
	Attributes map[string]interface{}
	Timestamp  time.Time
}

type SpanStatus struct {
	Code    StatusCode
	Message string
}

func (s *SimpleSpan) SetTag(key string, value interface{}) {
	s.Tags[key] = value
}

func (s *SimpleSpan) SetStatus(code StatusCode, message string) {
	s.Status = SpanStatus{
		Code:    code,
		Message: message,
	}
}

func (s *SimpleSpan) AddEvent(name string, attributes map[string]interface{}) {
	s.Events = append(s.Events, SpanEvent{
		Name:       name,
		Attributes: attributes,
		Timestamp:  time.Now(),
	})
}

func (s *SimpleSpan) End() {
	s.EndTime = time.Now()
}

func (s *SimpleSpan) Context() SpanContext {
	return &SimpleSpanContext{
		traceID:   s.TraceID,
		spanID:    s.SpanID,
		isSampled: s.IsSampled,
	}
}

type SimpleSpanContext struct {
	traceID   string
	spanID    string
	isSampled bool
}

func (c *SimpleSpanContext) TraceID() string {
	return c.traceID
}

func (c *SimpleSpanContext) SpanID() string {
	return c.spanID
}

func (c *SimpleSpanContext) IsSampled() bool {
	return c.isSampled
}

// generateID generates a random hex string for trace/span IDs
func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// noopTracer is a no-op implementation
type noopTracer struct{}

func (n *noopTracer) StartSpan(ctx context.Context, name string) (Span, context.Context) {
	return &noopSpan{}, ctx
}

func (n *noopTracer) Extract(headers http.Header) (SpanContext, error) {
	return nil, fmt.Errorf("noop tracer")
}

func (n *noopTracer) Inject(spanContext SpanContext, headers http.Header) error {
	return nil
}

type noopSpan struct{}

func (n *noopSpan) SetTag(key string, value interface{})                    {}
func (n *noopSpan) SetStatus(code StatusCode, message string)               {}
func (n *noopSpan) AddEvent(name string, attributes map[string]interface{}) {}
func (n *noopSpan) End()                                                    {}
func (n *noopSpan) Context() SpanContext                                    { return &noopSpanContext{} }

type noopSpanContext struct{}

func (n *noopSpanContext) TraceID() string { return "" }
func (n *noopSpanContext) SpanID() string  { return "" }
func (n *noopSpanContext) IsSampled() bool { return false }
