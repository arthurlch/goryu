# Recorder Middleware

Captures request/response pairs for building **eval datasets**. Each completed
request is written to a `Sink` as a structured `Record`; the bundled `FileSink`
appends newline-delimited JSON (NDJSON), the usual input format for offline eval
pipelines.

## Features

- **Structured records**: method, path, status, latency, request/response bodies.
- **Pluggable sinks**: implement `Sink` (DB, queue, blob) or use `FileSink`.
- **Bounded capture**: per-body byte cap; toggle request/response capture.
- **Body-safe**: request body is read and restored so handlers still see it.
- **Enrichable**: `KeyFunc` and `MetaFunc` attach a caller key and metadata
  (model, token usage, etc.).

## Usage

```go
import (
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/middleware/recorder"
)

sink, err := recorder.NewFileSink("evals.jsonl")
if err != nil {
    log.Fatal(err)
}
defer sink.Close()

app.Use(recorder.New(recorder.Config{
    Sink:    sink,
    KeyFunc: func(c *goryuctx.Context) string { return c.GetHeader("X-API-Key") },
    MetaFunc: func(c *goryuctx.Context) map[string]any {
        u, _ := aimeter.GetUsage(c)
        return map[string]any{"model": c.GetHeader("X-Model"), "tokens": u.Total()}
    },
}))
```

Each line of `evals.jsonl` looks like:

```json
{"time":"2026-09-14T10:00:00Z","method":"POST","path":"/chat","status":200,"latency_ms":812,"key":"k-123","request_body":{"q":"life"},"response_body":{"answer":"42"},"meta":{"model":"gpt-x","tokens":1156}}
```

## Configuration Options

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| Sink | `Sink` | Where records go (required) | none (no-op) |
| MaxBodyBytes | `int64` | Max bytes captured per body | `64 KiB` |
| CaptureRequest | `*bool` | Capture request body | `true` |
| CaptureResponse | `*bool` | Capture response body | `true` |
| KeyFunc | `func(c) string` | Caller/session key | none |
| MetaFunc | `func(c) map[string]any` | Extra metadata | none |

## Notes

- Non-JSON bodies are stored as JSON strings so every record marshals cleanly.
- `FileSink` flushes after each record and is safe for concurrent use. For
  high-throughput recording, implement a buffered/async `Sink`.
- Bodies are truncated at `MaxBodyBytes`; raise it for large prompts.
