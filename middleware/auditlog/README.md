# Audit Log Middleware

Records an access-audit trail — who did what, when, and with what outcome. Each
request becomes an `Event` written to a `Sink` (a JSON logger by default).

## Usage

```go
import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/middleware/auditlog"
)

app.Use(auditlog.Default()) // JSON events to the default logger
```

Custom sink (DB, SIEM, file) and identity extractor:

```go
app.Use(auditlog.New(auditlog.Config{
    Sink:       mySink,
    UserIDFunc: func(c *goryuctx.Context) string { return currentUser(c) },
}))
```

Each event carries: `time`, `method`, `path`, `status`, `latency_ms`, `user_id`
(from the `user_id` context key set by the auth/session middleware), and `ip`.

## Configuration Options

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| Sink | `Sink` | Where events go | JSON to the logger |
| UserIDFunc | `func(c) string` | Extract the caller identity | `user_id` context key |
| Skip | `func(c) bool` | Skip auditing a request | none |

## Notes

- Complements the auth package's internal security-event logging; this is the
  HTTP-level access trail.
- `Sink` implementations must be safe for concurrent use.
