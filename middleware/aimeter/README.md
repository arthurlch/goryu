# AIMeter Middleware

Token/cost metering and per-key rate limiting for AI/LLM backends. Handlers
report provider token usage; the middleware accounts cost, enforces per-key
request and token budgets within a fixed window (counters reset at the end of
each window), and surfaces the result via response headers and an `OnResult`
callback.

## Features

- **Token accounting**: prompt/completion/total tokens per request.
- **Cost metering**: USD cost from per-1K-token pricing.
- **Per-key limits**: request count *and* token budget within a fixed window.
- **Per-key isolation**: each API key (or IP) has its own window.
- **Reliable hook**: `OnResult` for logging/metrics/billing.

## Usage

```go
import (
    "time"

    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/middleware/aimeter"
)

app.Use(aimeter.New(aimeter.Config{
    PromptCostPer1K:     0.003,
    CompletionCostPer1K: 0.015,
    MaxRequests:         100,
    MaxTokens:           200_000,
    Window:              time.Minute,
    OnResult: func(c *goryuctx.Context, r aimeter.Result) {
        log.Printf("key=%s tokens=%d cost=$%.4f", r.Key, r.Usage.Total(), r.CostUSD)
    },
}))
```

Inside a handler, after calling the provider, report usage:

```go
func chat(c *goryu.Ctx) {
    resp := callProvider(...)
    aimeter.SetUsage(c, aimeter.Usage{
        PromptTokens:     resp.Usage.PromptTokens,
        CompletionTokens: resp.Usage.CompletionTokens,
    })
    c.JSON(200, resp)
}
```

## Configuration Options

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| KeyGenerator | `func(*goryuctx.Context) string` | Caller identity | `X-API-Key` → remote IP |
| PromptCostPer1K | `float64` | USD per 1K prompt tokens | `0` |
| CompletionCostPer1K | `float64` | USD per 1K completion tokens | `0` |
| MaxRequests | `int` | Max requests per window (0 = off) | `0` |
| MaxTokens | `int` | Max tokens per window (0 = off) | `0` |
| Window | `time.Duration` | Fixed window length | `1m` |
| MaxClients | `int` | Max tracked keys | `10000` |
| OnResult | `func(c, Result)` | Post-request accounting hook | none |
| LimitReached | `func(c, reason)` | Custom rejection response | 429 |

## Notes

- **Token budgets are enforced from prior consumption.** Because token counts
  are only known after the provider responds, a request is rejected when the
  key's accumulated tokens for the window already exceed `MaxTokens`.
- **Response headers are best-effort.** `X-Tokens-*` / `X-Cost-USD` are set only
  when the handler has not already flushed the response (i.e. non-streaming). For
  streaming endpoints, use `OnResult` as the reliable accounting path.
