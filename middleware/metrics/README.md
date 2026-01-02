# Metrics Middleware

A middleware for instrumenting Goryu applications with Prometheus-style metrics. It tracks request rates, latencies, and sizes, and provides an interface for custom metric recording.

## Features

- **Standard Metrics**: Automatically tracks:
    - `http_requests_total` (Counter)
    - `http_request_duration_seconds` (Histogram)
    - `http_requests_active` (Gauge)
    - `http_request_size_bytes` (Histogram, optional)
    - `http_response_size_bytes` (Histogram, optional)
- **Tagging**: Adds standard tags like `method`, `path`, and `status` (or `status_class`).
- **Custom Metrics**: Interface for recording application-specific metrics.
- **Backend Agnostic**: Defines a `Metrics` interface, allowing integration with any backend (Prometheus, StatsD, Datadog) by implementing the adapter.

## Usage

### Basic Setup

You need to provide an implementation of the `Metrics` interface. Here is a conceptual example using a hypothetical Prometheus client:

```go
package main

import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/goryuctx"
    "github.com/arthurlch/goryu/middleware/metrics"
    
    // Note: This example uses the official Prometheus client
    // go get github.com/prometheus/client_golang/prometheus
    "github.com/prometheus/client_golang/prometheus"
)

// Adapter implementation (simplified)
type PrometheusAdapter struct {
    // ... fields for prometheus collectors
}
// ... implement IncrementCounter, RecordHistogram, etc.

func main() {
    app := goryu.New()

    adapter := NewPrometheusAdapter()

    app.Use(metrics.New(metrics.Config{
        Metrics: adapter,
        Prefix:  "myapp", // Metric name prefix
    }))

    app.GET("/", func(c *goryuctx.Context) {
        c.Status(200).String("Hello Metrics")
    })

    app.Listen(":8080")
}
```

### Configuration

```go
app.Use(metrics.New(metrics.Config{
    Metrics:         adapter,
    Prefix:          "api",
    RecordBody:      true,  // Track request/response sizes
    GroupStatusCode: true,  // Use "2xx", "5xx" instead of "200", "500"
    CustomTags: func(c *goryuctx.Context) map[string]string {
        return map[string]string{
            "region": os.Getenv("REGION"),
        }
    },
}))
```

### Interface Definition

To integrate your preferred metrics backend, implement this interface:

```go
type Metrics interface {
    IncrementCounter(name string, tags map[string]string)
    AddToCounter(name string, value float64, tags map[string]string)
    RecordHistogram(name string, value float64, tags map[string]string)
    SetGauge(name string, value float64, tags map[string]string)
    AddToGauge(name string, value float64, tags map[string]string)
}
```
