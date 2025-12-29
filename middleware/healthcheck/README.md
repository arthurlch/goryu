# HealthCheck Middleware

A middleware for adding health check endpoints to your Goryu application. It supports Liveness and Readiness probes, making it ideal for Kubernetes and other container orchestration platforms.

## Features

- **Standard Endpoints**: Configurable paths for Liveness (`/health/live`), Readiness (`/health/ready`), and general Health (`/health`).
- **Custom Probes**: Register custom checks for databases, external services, or internal state.
- **Timeout Support**: Enforces timeouts on health checks to ensure responsiveness.
- **JSON Output**: Returns detailed status and error messages in JSON format.

## Usage

### Basic Setup

Enable default health check endpoints:

```go
app.Use(healthcheck.Default())
```

This exposes:
- `/health/live`: Returns 200 OK if the server is running.
- `/health/ready`: Returns 200 OK if the server is running.
- `/health`: Returns 200 OK if the server is running.

### Custom Probes

Add checks for your database and external dependencies:

```go
app.Use(healthcheck.New(healthcheck.Config{
    LivenessProbes: map[string]healthcheck.Probe{
        "goroutine": healthcheck.AlwaysUpProbe(),
    },
    ReadinessProbes: map[string]healthcheck.Probe{
        "database": healthcheck.DatabaseProbe(db.Ping),
        "redis": func(ctx context.Context) error {
            return redisClient.Ping(ctx).Err()
        },
    },
}))
```

### Helper Probes

The package includes helper functions for common checks:

- `healthcheck.DatabaseProbe(pingFunc)`: Checks database connectivity.
- `healthcheck.HTTPProbe(url, client)`: Checks an external HTTP service.
- `healthcheck.AlwaysUpProbe()`: Always returns success.
- `healthcheck.AlwaysDownProbe(msg)`: Always returns failure (useful for testing).

## Response Format

**Success (200 OK):**

```json
{
  "status": "UP",
  "checks": {
    "database": { "status": "UP" }
  }
}
```

**Failure (503 Service Unavailable):**

```json
{
  "status": "DOWN",
  "errors": {
    "database": "connection refused"
  },
  "checks": {
    "database": { "status": "DOWN", "error": "connection refused" }
  }
}
```
