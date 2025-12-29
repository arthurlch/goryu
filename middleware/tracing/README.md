# Tracing Middleware

A middleware for distributed tracing in Goryu applications. It provides a vendor-agnostic interface for starting spans, propagating context, and recording events.

## Features

- **Vendor Agnostic**: Defines a `Tracer` interface that can be implemented for any tracing backend (OpenTelemetry, Jaeger, Zipkin).
- **Context Propagation**: Automatically extracts and injects trace headers (`X-Trace-Id`, `X-Span-Id`).
- **Automatic Instrumentation**: Creates spans for every request with standard HTTP tags.
- **Event Recording**: Helper methods to add events and custom tags to spans.

## Usage

### Basic Setup

You need to provide an implementation of the `Tracer` interface. Here is a conceptual example:

```go
package main

import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/goryuctx"
    "github.com/arthurlch/goryu/middleware/tracing"
)

// Your tracer implementation (e.g., wrapping OpenTelemetry)
type MyTracer struct {}
// ... implement StartSpan, Extract, Inject

func main() {
    app := goryu.New()

    tracer := &MyTracer{}

    app.Use(tracing.New(tracing.Config{
        Tracer: tracer,
        SampleRate: 1.0, // Sample 100% of requests
    }))

    app.GET("/", func(c *goryuctx.Context) {
        // Get current span
        if span, ok := tracing.GetSpan(c); ok {
            span.AddEvent("processing_logic", map[string]interface{}{
                "user_id": "123",
            })
        }
        c.Status(200).String("Hello Tracing")
    })

    app.Listen(":8080")
}
```

### Interface Definition

To integrate your preferred tracing backend, implement this interface:

```go
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
```

### Simple Tracer

The package includes a `SimpleTracer` for testing or simple debugging purposes. It stores spans in memory.

```go
tracer := tracing.NewSimpleTracer()
app.Use(tracing.New(tracing.Config{Tracer: tracer}))
```
