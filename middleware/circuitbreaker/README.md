# CircuitBreaker Middleware

A middleware that implements the Circuit Breaker pattern for Goryu. It prevents cascading failures by stopping requests to a failing service and giving it time to recover.

## Features

- **State Management**: Handles `Closed`, `Open`, and `Half-Open` states.
- **Configurable Thresholds**: Customize failure ratios, request volume, and timeouts.
- **Custom Failure Logic**: Define exactly what constitutes a failure (e.g., specific errors or status codes).
- **Event Hooks**: Callbacks for state changes (e.g., for logging or metrics).
- **Monitoring**: Exposes internal state and metrics.

## Usage

### Basic Setup

Protect a route with default settings:

```go
package main

import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/goryuctx"
    "github.com/arthurlch/goryu/middleware/circuitbreaker"
    "time"
)

func main() {
    app := goryu.New()

    // Create a circuit breaker
    cb, middleware := circuitbreaker.WithCircuitBreaker(circuitbreaker.Config{
        FailureRatio: 0.5, // Open if 50% of requests fail
        Timeout:      60 * time.Second, // Wait 60s before trying again
    })

    // Apply to a specific route
    app.GET("/unreliable-service", middleware(func(c *goryuctx.Context) {
        // ... call external service ...
    }))

    app.Listen(":8080")
}
```

### Global Middleware

Apply to the entire application (careful: this shares state across all routes):

```go
app.Use(circuitbreaker.New(circuitbreaker.Config{
    MaxRequests: 5, // Allow 5 requests in Half-Open state
}))
```

### Advanced Configuration

```go
app.Use(circuitbreaker.New(circuitbreaker.Config{
    MinRequests:  10,   // Need at least 10 requests before calculating ratio
    FailureRatio: 0.6,  // Break if > 60% fail
    Interval:     1 * time.Minute, // Reset counts every minute
    Timeout:      30 * time.Second, // Open state duration
    IsFailure: func(err error) bool {
        // Only treat specific errors as failures
        return err != nil && err.Error() == "connection refused"
    },
    OnStateChange: func(state circuitbreaker.State) {
        log.Printf("Circuit breaker changed state to: %s", state)
    },
}))
```

## How It Works

1.  **Closed**: Requests flow normally. Failures are counted.
2.  **Open**: If the failure ratio exceeds the threshold, the breaker opens. All requests fail immediately with `503 Service Unavailable`.
3.  **Half-Open**: After the `Timeout` period, the breaker allows a limited number of requests (`MaxRequests`) to test if the service has recovered.
    - If these requests succeed, the breaker closes.
    - If any fail, it re-opens.
