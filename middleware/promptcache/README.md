# PromptCache Middleware

A response cache keyed on the **request body** (the "prompt") rather than the
URL, so it can cache `POST` responses — the common shape for LLM endpoints where
the same prompt should return the same answer.

## Features

- **Body-keyed**: SHA-256 of method + path + (optional vary headers) + body.
- **POST-friendly**: caches methods you choose (default `POST`).
- **Streaming-safe**: `text/event-stream` and `application/x-ndjson` responses
  are passed through, never cached.
- **Bounded**: size cap with expiry-based eviction; per-body size limit.
- **Correct by default**: never caches non-2xx, `Set-Cookie`, `no-store`, or
  `private` responses. Adds `X-Cache: HIT|MISS`.

## Usage

```go
import (
    "time"

    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/middleware/promptcache"
)

app.Use(promptcache.New(promptcache.Config{
    Expiration:  10 * time.Minute,
    VaryHeaders: []string{"X-Model"}, // fold model selector into the key
}))
```

Custom key (e.g. normalize the prompt before hashing):

```go
app.Use(promptcache.New(promptcache.Config{
    KeyGenerator: func(c *goryuctx.Context, body []byte) string {
        return normalizePrompt(body)
    },
}))
```

## Configuration Options

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| Expiration | `time.Duration` | Freshness window | `5m` |
| MaxSize | `int` | Max cached entries | `1000` |
| MaxBodyBytes | `int64` | Max request/response bytes cached | `1 MiB` |
| Methods | `[]string` | Cacheable methods | `["POST"]` |
| VaryHeaders | `[]string` | Request headers folded into the key | none |
| KeyGenerator | `func(c, body) string` | Custom key derivation | body hash |

## Security Note

Never put authentication headers in `VaryHeaders`, and be careful caching
per-user responses: a shared cache can leak one caller's answer to another.
Include a user/tenant discriminator in the key when responses are user-specific.
