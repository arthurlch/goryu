# Goryu Monitoring System

A comprehensive monitoring and health check system for Goryu applications with automatic integration.

## Features

- **Automatic Integration**: Built into every Goryu app by default
- **Event Monitoring**: Track requests, errors, and custom events with correlation IDs
- **Health Checks**: Monitor application components with panic recovery
- **Advanced Metrics**: Route-level, middleware performance, and system metrics
- **Request Tracing**: Correlation ID generation and propagation
- **Built-in Endpoints**: Ready-to-use HTTP endpoints for monitoring data
- **Event Handlers**: Custom event processing and alerting

## Quick Start

```go
package main

import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/monitoring"
)

func main() {
    // goryu.Default() enables monitoring automatically!
    app := goryu.Default()
    
    // Add a health check
    app.AddHealthCheck("database", &monitoring.HealthCheck{
        Check: func() (monitoring.HealthStatus, error) {
            return monitoring.StatusHealthy, nil
        },
        Critical: true,
    })
    
    // Emit custom events
    app.EmitEvent(monitoring.EventCustom, "User login", map[string]interface{}{
        "user_id": "123",
    })
    
    app.Listen(":8080")
}
```

## Built-in Endpoints

Monitoring endpoints are automatically registered:

- `GET /_health` - Health check status
- `GET /_metrics` - Application metrics
- `GET /_events` - Recent events (with ?limit=N parameter)

## Health Checks

Health checks allow you to monitor the status of various application components:

```go
app.AddHealthCheck("database", &monitoring.HealthCheck{
    Check: func() (monitoring.HealthStatus, error) {
        if err := db.Ping(); err != nil {
            return monitoring.StatusUnhealthy, err
        }
        return monitoring.StatusHealthy, nil
    },
    Timeout:  5 * time.Second,
    Interval: 30 * time.Second,
    Critical: true, // Will mark overall status as unhealthy if this fails
})
```

### Health Status Levels

- `StatusHealthy` - Component is working normally
- `StatusDegraded` - Component has issues but is still functional
- `StatusUnhealthy` - Component is not working

### Overall Health Logic

- If any **critical** health check is `StatusUnhealthy` → Overall status is `StatusUnhealthy`
- If any health check is `StatusUnhealthy` (non-critical) → Overall status is `StatusDegraded`
- If any health check is `StatusDegraded` → Overall status is `StatusDegraded`
- Otherwise → Overall status is `StatusHealthy`

## Event Monitoring

The system automatically tracks HTTP requests and errors. You can also emit custom events:

```go
// Emit a custom event
app.EmitEvent(monitoring.EventCustom, "User registered", map[string]interface{}{
    "user_id": "123",
    "email":   "user@example.com",
})
```

### Event Types

- `EventRequest` - HTTP requests (automatic)
- `EventError` - HTTP errors (automatic)
- `EventHealthy` - Health check passed (automatic)
- `EventUnhealthy` - Health check failed (automatic)
- `EventStartup` - Application startup (automatic)
- `EventShutdown` - Application shutdown
- `EventCustom` - Your custom events

## Enhanced Metrics

The system automatically collects comprehensive metrics:

```go
{
    "request_count": 1523,
    "error_count": 12,
    "avg_response_time": "45ms",
    "active_requests": 5,
    "uptime": "2h30m15s",
    "memory_usage_bytes": 67108864,
    "goroutines": 15,
    "status_code_counts": {
        "200": 1400,
        "404": 15,
        "500": 8
    },
    "route_metrics": {
        "GET:/users": {
            "request_count": 800,
            "error_count": 2,
            "avg_response_time": "20ms",
            "status_codes": {"200": 798, "500": 2}
        },
        "POST:/users": {
            "request_count": 100,
            "error_count": 1,
            "avg_response_time": "45ms"
        }
    },
    "middleware_metrics": {
        "auth": {
            "execution_count": 1500,
            "avg_execution_time": "5ms",
            "error_count": 0
        },
        "cors": {
            "execution_count": 1500,
            "avg_execution_time": "1ms"
        }
    }
}
```

## Request Tracing

Every request automatically gets a correlation ID for tracing:

```go
app.GET("/api/data", func(c *context.Context) {
    // Get the correlation ID
    correlationID, _ := c.Get("correlation_id")
    
    // The correlation ID is also available in response headers
    // X-Correlation-ID: abc123def456
    
    c.JSON(200, map[string]interface{}{
        "correlation_id": correlationID,
        "data": "response",
    })
})
```

All events include correlation IDs for request tracing across logs and monitoring systems.

## Middleware Performance Tracking

Monitor individual middleware performance:

```go
// Wrap any middleware to track its performance
wrappedAuth := app.Monitor.MiddlewareWrapper("auth", authMiddleware)
app.Use(wrappedAuth)

// Performance data will be available in middleware_metrics
```

## Event Handlers

Add custom event handlers for logging, alerting, or integration with external systems:

```go
app.Monitor.AddEventHandler(func(event monitoring.Event) {
    if event.Type == monitoring.EventError {
        // Events now include correlation_id for tracing
        log.Printf("Error %s: %s (correlation: %s)", 
            event.ID, event.Message, event.Data["correlation_id"])
        sendAlert(event)
    }
})
```

## Configuration

Configure the monitoring system:

```go
monitor := monitoring.New(monitoring.Config{
    Enabled:        true,
    MaxEvents:      1000,           // Maximum events to keep in memory
    HealthInterval: 30*time.Second, // How often to run health checks
    MetricsEnabled: true,
})
```

## Health Check Examples

### Database Health Check

```go
app.AddHealthCheck("database", &monitoring.HealthCheck{
    Check: func() (monitoring.HealthStatus, error) {
        ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
        defer cancel()
        
        if err := db.PingContext(ctx); err != nil {
            return monitoring.StatusUnhealthy, err
        }
        return monitoring.StatusHealthy, nil
    },
    Critical: true,
})
```

### External Service Health Check

```go
app.AddHealthCheck("payment_service", &monitoring.HealthCheck{
    Check: func() (monitoring.HealthStatus, error) {
        resp, err := http.Get("https://api.payments.com/health")
        if err != nil {
            return monitoring.StatusUnhealthy, err
        }
        defer resp.Body.Close()
        
        if resp.StatusCode == 200 {
            return monitoring.StatusHealthy, nil
        }
        return monitoring.StatusDegraded, fmt.Errorf("service returned %d", resp.StatusCode)
    },
    Critical: false, // Non-critical service
})
```

### Memory Usage Health Check

```go
app.AddHealthCheck("memory", &monitoring.HealthCheck{
    Check: func() (monitoring.HealthStatus, error) {
        var m runtime.MemStats
        runtime.ReadMemStats(&m)
        
        memMB := m.Alloc / 1024 / 1024
        if memMB > 500 {
            return monitoring.StatusUnhealthy, fmt.Errorf("memory too high: %dMB", memMB)
        } else if memMB > 200 {
            return monitoring.StatusDegraded, fmt.Errorf("memory elevated: %dMB", memMB)
        }
        return monitoring.StatusHealthy, nil
    },
})
```

## Integration with External Systems

### Prometheus Integration

```go
app.Monitor.AddEventHandler(func(event monitoring.Event) {
    // Export metrics to Prometheus
    prometheusCounter.Inc()
})
```

### Logging Integration

```go
app.Monitor.AddEventHandler(func(event monitoring.Event) {
    logger.Info("monitoring_event",
        "type", event.Type,
        "message", event.Message,
        "data", event.Data,
    )
})
```

## Best Practices

1. **Use Critical Health Checks Sparingly** - Only mark health checks as critical if their failure should make the entire service unavailable

2. **Set Appropriate Timeouts** - Health checks should have reasonable timeouts to avoid blocking

3. **Monitor Key Dependencies** - Add health checks for databases, external APIs, and critical services

4. **Use Custom Events Meaningfully** - Emit custom events for business-critical operations

5. **Set Up Alerting** - Use event handlers to integrate with alerting systems

6. **Monitor Resource Usage** - Add health checks for memory, disk space, and other resources

7. **Test Health Checks** - Ensure your health checks actually detect problems

## Example Response Formats

### Health Endpoint Response

```json
{
  "status": "healthy",
  "timestamp": "2023-12-07T10:30:00Z",
  "checks": {
    "database": {
      "name": "database",
      "status": "healthy",
      "timestamp": "2023-12-07T10:30:00Z",
      "duration": "2ms",
      "critical": true
    }
  }
}
```

### Metrics Endpoint Response

```json
{
  "request_count": 1523,
  "error_count": 12,
  "avg_response_time": "45ms",
  "uptime": "2h30m15s",
  "memory_usage_bytes": 67108864,
  "goroutines": 15,
  "start_time": "2023-12-07T08:00:00Z"
}
```

### Events Endpoint Response

```json
{
  "events": [
    {
      "id": "1701936600000000000",
      "type": "request",
      "timestamp": "2023-12-07T10:30:00Z",
      "message": "GET /api/users",
      "data": {
        "method": "GET",
        "path": "/api/users",
        "status_code": 200,
        "duration_ms": 45
      }
    }
  ],
  "total": 1523
}
```