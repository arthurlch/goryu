# aiproxy

A small streaming reverse-proxy helper for LLM/AI providers. It forwards the
current request to an upstream provider and streams the response back to the
client, flushing as data arrives so SSE and chunked token streams pass
straight through.

## Features

- **Streaming passthrough**: `Stream` flushes upstream bytes as they arrive
  (SSE / chunked token streams).
- **Buffered forward**: `Forward` for non-streaming provider calls.
- **Auth injection**: attaches `Authorization: Bearer <key>` (configurable).
- **Header hygiene**: drops hop-by-hop headers both ways.
- **Context-aware**: honors the request context for cancellation; no client
  timeout on streams.

## Usage

```go
import (
    "os"

    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/aiproxy"
)

px := aiproxy.New(aiproxy.Config{
    BaseURL: "https://api.openai.com",
    APIKey:  os.Getenv("OPENAI_API_KEY"),
})

app.POST("/v1/chat/completions", func(c *goryu.Ctx) {
    if err := px.Stream(c, "/v1/chat/completions"); err != nil {
        // upstream/network error; response may be partially written
        log.Printf("proxy error: %v", err)
    }
})
```

## Configuration Options

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| BaseURL | `string` | Upstream origin (required) | none |
| APIKey | `string` | Injected credential | none |
| AuthHeader | `string` | Header for the credential | `Authorization` |
| AuthScheme | `string` | Scheme prefix | `Bearer` |
| Header | `http.Header` | Extra headers per request | none |
| Client | `*http.Client` | Custom HTTP client | see below |
| Timeout | `time.Duration` | Timeout for `Forward` | `60s` |

## Notes

- `Stream` uses a client with **no timeout** (streams can be long-lived) and
  relies on the request context for cancellation. `Forward` uses `Timeout`.
- Supplying your own `Client` overrides both; set its `Timeout` to `0` if you
  reuse it for streaming.

## Security

- Client headers are forwarded except hop-by-hop headers **and the caller's
  own secrets** (`Authorization`, `Cookie`, `X-Api-Key`) — those are stripped so
  they never leak to the third-party provider. The proxy adds its own credential.
- `path` is joined onto `BaseURL`; do **not** pass a user-controlled value, or a
  caller could reach unintended upstream paths.
