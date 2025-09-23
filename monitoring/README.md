# Goryu Monitoring System

A comprehensive monitoring and health check system for Goryu applications.

## Features

- **Event Monitoring**: Track application events (requests, errors, custom events)
- **Health Checks**: Monitor the health of your application components
- **Metrics Collection**: Collect and expose application metrics
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
    app := goryu.New()
    
    // Enable monitoring endpoints
    app.EnableMonitoring("/_monitor")
    
    // Add a health check
    app.AddHealthCheck("database", &monitoring.HealthCheck{
        Check: func() (monitoring.HealthStatus, error) {
            // Your health check logic here
            return monitoring.StatusHealthy, nil
        },
        Critical: true,
    })
    
    app.Listen(":8080")
}
```

## Built-in Endpoints

When you call `app.EnableMonitoring("/_monitor")`, the following endpoints become available:

- `GET /_monitor/health` - Health check status
- `GET /_monitor/metrics` - Application metrics
- `GET /_monitor/events` - Recent events
- `GET /_monitor/dashboard` - Complete monitoring overview

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

## Metrics

The system automatically collects metrics:

```go
{
    "request_count": 1523,
    "error_count": 12,
    "avg_response_time": "45ms",
    "uptime": "2h30m15s",
    "memory_usage_bytes": 67108864,
    "goroutines": 15,
    "start_time": "2023-12-07T10:30:00Z"
}
```

## Event Handlers

Add custom event handlers for logging, alerting, or integration with external systems:

```go
app.Monitor.AddEventHandler(func(event monitoring.Event) {
    if event.Type == monitoring.EventError {
        // Send alert to Slack, email, etc.
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