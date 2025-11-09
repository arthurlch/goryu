package tracing
import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
	goryuContext "github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/base"
)
type Tracer interface {
	StartSpan(ctx context.Context, name string) (Span, context.Context)
	Extract(headers http.Header) (SpanContext, error)
	Inject(spanContext SpanContext, headers http.Header) error
}
type Span interface {
	SetTag(key string, value interface{})
	SetStatus(code StatusCode, message string)
	AddEvent(name string, attributes map[string]interface{})
	End()
	Context() SpanContext
}
type SpanContext interface {
	TraceID() string
	SpanID() string
	IsSampled() bool
}
type StatusCode int
const (
	StatusCodeUnset StatusCode = iota
	StatusCodeOk
	StatusCodeError
)
type Config struct {
	base.BaseConfig
	Tracer Tracer
	SpanNameGenerator func(c *goryuContext.Context) string
	CustomTags func(c *goryuContext.Context) map[string]interface{}
	SampleRate float64
}
type contextKey string
const traceContextKey contextKey = "trace_context"
func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if c.Tracer == nil {
		c.Tracer = &noopTracer{}
	}
	if c.SpanNameGenerator == nil {
		c.SpanNameGenerator = func(ctx *goryuContext.Context) string {
			return fmt.Sprintf("HTTP %s %s", ctx.Request.Method, ctx.Request.URL.Path)
		}
	}
	if c.SampleRate <= 0 {
		c.SampleRate = 1.0 
	}
	return nil
}
func New(config Config) func(next goryuContext.HandlerFunc) goryuContext.HandlerFunc {
	if err := config.Validate(); err != nil {
		return func(next goryuContext.HandlerFunc) goryuContext.HandlerFunc {
			return func(c *goryuContext.Context) {
				base.DefaultErrorHandler(c, err, "Tracing")
			}
		}
	}
	return func(next goryuContext.HandlerFunc) goryuContext.HandlerFunc {
		return func(c *goryuContext.Context) {
			if config.Skip != nil && config.Skip(c) {
				next(c)
				return
			}
			parentSpanCtx, _ := config.Tracer.Extract(c.Request.Header)
			ctx := context.Background()
			if parentSpanCtx != nil {
				ctx = context.WithValue(ctx, traceContextKey, parentSpanCtx)
			}
			spanName := config.SpanNameGenerator(c)
			span, spanCtx := config.Tracer.StartSpan(ctx, spanName)
			defer span.End()
			c.Set("trace_context", spanCtx)
			c.Set("span", span)
			span.SetTag("http.method", c.Request.Method)
			span.SetTag("http.url", c.Request.URL.String())
			span.SetTag("http.scheme", c.Request.URL.Scheme)
			span.SetTag("http.host", c.Request.Host)
			span.SetTag("http.user_agent", c.Request.UserAgent())
			span.SetTag("http.remote_addr", c.Request.RemoteAddr)
			if config.CustomTags != nil {
				for k, v := range config.CustomTags(c) {
					span.SetTag(k, v)
				}
			}
			rw := &tracingResponseWriter{
				ResponseWriter: c.Writer,
				span:           span,
			}
			c.Writer = rw
			span.AddEvent("request.start", map[string]interface{}{
				"timestamp": time.Now().Unix(),
			})
			next(c)
			span.SetTag("http.status_code", rw.statusCode)
			span.SetTag("http.response_size", rw.responseSize)
			if rw.statusCode >= 400 {
				span.SetStatus(StatusCodeError, http.StatusText(rw.statusCode))
			} else {
				span.SetStatus(StatusCodeOk, "")
			}
			span.AddEvent("request.end", map[string]interface{}{
				"timestamp":   time.Now().Unix(),
				"status_code": rw.statusCode,
			})
		}
	}
}
func Default() func(next goryuContext.HandlerFunc) goryuContext.HandlerFunc {
	return New(Config{})
}
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
	n, err := w.ResponseWriter.Write(data)
	w.responseSize += n
	return n, err
}
func GetTraceContext(c *goryuContext.Context) (context.Context, bool) {
	ctx, exists := c.Get("trace_context")
	if !exists {
		return nil, false
	}
	if traceCtx, ok := ctx.(context.Context); ok {
		return traceCtx, true
	}
	return nil, false
}
func GetSpan(c *goryuContext.Context) (Span, bool) {
	span, exists := c.Get("span")
	if !exists {
		return nil, false
	}
	if s, ok := span.(Span); ok {
		return s, true
	}
	return nil, false
}
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
func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
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
