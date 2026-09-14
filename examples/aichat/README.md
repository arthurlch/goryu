# aichat — streaming chat service example

A small end-to-end example wiring goryu's AI batteries together into one service.
It ships a fake local "LLM", so it runs with **no API keys**.

## What it demonstrates

| Piece | Used for |
|-------|----------|
| `goryu.Bind[T]` + `Validate()` | typed, validated request bodies |
| `goryu.Schema[T]` | JSON Schema endpoint for the request type |
| `goryu.SSEJSON[T]` | token streaming over Server-Sent Events |
| `middleware/aimeter` | token/cost metering + per-key rate limits |
| `middleware/promptcache` | cache identical prompts on the non-streaming route |
| `middleware/recorder` | append every exchange to `evals.jsonl` |
| `aiproxy` | optional streaming passthrough to a real provider |

## Run

```sh
go run ./examples/aichat
# listening on :3000, writing evals.jsonl in the working directory
```

## Endpoints

**`POST /chat/stream`** — streams the reply token-by-token over SSE:

```sh
curl -N -X POST localhost:3000/chat/stream \
     -H 'Content-Type: application/json' -H 'X-API-Key: demo' \
     -d '{"model":"fast","messages":[{"role":"user","content":"hello there"}]}'
```

**`POST /chat`** — non-streaming JSON, wrapped in the prompt cache. Send the same
body twice and watch the `X-Cache` response header go `MISS` → `HIT`:

```sh
curl -i -X POST localhost:3000/chat \
     -H 'Content-Type: application/json' -H 'X-API-Key: demo' \
     -d '{"model":"smart","messages":[{"role":"user","content":"ping"}]}'
```

**`GET /chat/schema`** — the JSON Schema for `ChatRequest` (hand this to an LLM as
a response schema, or to clients for validation):

```sh
curl localhost:3000/chat/schema
```

> Content-Type **must** be `application/json` — that is what selects typed
> binding. `curl -d` defaults to form encoding, so always pass the header.

## Metering, limits, and evals

- Each request logs a meter line: `meter key=demo tokens=13 cost=$0.00002`.
- Per key: 60 requests/min and 100k tokens/min (429 when exceeded).
- Every exchange is appended to `evals.jsonl` as NDJSON (method, path, status,
  latency, request/response bodies, key, token metadata).

## Optional: real provider passthrough

Set `PROVIDER_BASE_URL` (and optionally `PROVIDER_API_KEY`) to expose
`POST /v1/chat/completions`, which streams straight through to the upstream:

```sh
PROVIDER_BASE_URL=https://api.openai.com \
PROVIDER_API_KEY=sk-... \
go run ./examples/aichat
```
