// Command aichat is a small streaming-chat service that wires goryu's AI
// batteries together:
//
//   - goryu.Bind[T] + Validate()  typed, validated request bodies
//   - goryu.Schema[T]             JSON Schema endpoint for the request type
//   - goryu.SSEJSON[T]            token streaming over Server-Sent Events
//   - middleware/aimeter          token/cost metering + per-key rate limits
//   - middleware/promptcache      cache identical prompts (non-streaming route)
//   - middleware/recorder         record every exchange to evals.jsonl
//   - aiproxy                     optional passthrough to a real provider
//
// It ships a fake local "LLM" so it runs with no API keys. Set PROVIDER_BASE_URL
// (and optionally PROVIDER_API_KEY) to also expose a streaming passthrough route.
//
// Run:
//
//	go run ./examples/aichat
//
// Try it (Content-Type must be application/json for typed binding):
//
//	curl -N -X POST localhost:3000/chat/stream \
//	     -H 'Content-Type: application/json' -H 'X-API-Key: demo' \
//	     -d '{"model":"fast","messages":[{"role":"user","content":"hello there"}]}'
//	curl -X POST localhost:3000/chat \
//	     -H 'Content-Type: application/json' -H 'X-API-Key: demo' \
//	     -d '{"model":"smart","messages":[{"role":"user","content":"ping"}]}'
//	curl localhost:3000/chat/schema
package main

import (
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/aiproxy"
	"github.com/arthurlch/goryu/middleware/aimeter"
	"github.com/arthurlch/goryu/middleware/promptcache"
	"github.com/arthurlch/goryu/middleware/recorder"
)

// Message is one turn in the conversation.
type Message struct {
	Role    string `json:"role" jsonschema:"description=Speaker role,enum=system|user|assistant"`
	Content string `json:"content" jsonschema:"description=Message text,minLength=1"`
}

// ChatRequest is the typed request body for the chat endpoints.
type ChatRequest struct {
	Model    string    `json:"model" jsonschema:"description=Model to use,enum=fast|smart"`
	Messages []Message `json:"messages" jsonschema:"description=Conversation so far"`
	Stream   bool      `json:"stream,omitempty" jsonschema:"description=Whether to stream tokens"`
}

// Validate is the goryu.Bind hook: it runs automatically during Bind[ChatRequest].
func (r ChatRequest) Validate() error {
	if len(r.Messages) == 0 {
		return errors.New("messages: at least one message is required")
	}
	if r.Model != "fast" && r.Model != "smart" {
		return errors.New("model: must be 'fast' or 'smart'")
	}
	return nil
}

// Token is a single streamed chunk of the reply.
type Token struct {
	Index   int    `json:"index"`
	Content string `json:"content,omitempty"`
	Done    bool   `json:"done,omitempty"`
}

func main() {
	app := goryu.New()

	// --- Eval recording (outermost): capture every request/response pair. ---
	sink, err := recorder.NewFileSink("evals.jsonl")
	if err != nil {
		log.Fatalf("open eval sink: %v", err)
	}
	defer sink.Close()

	app.Use(recorder.New(recorder.Config{
		Sink:    sink,
		KeyFunc: apiKey,
		MetaFunc: func(c *goryu.Ctx) map[string]any {
			u, _ := aimeter.GetUsage(c)
			return map[string]any{"tokens": u.Total()}
		},
	}))

	// --- Metering + per-key limits: pricing + 60 req/min + 100k tokens/min. ---
	app.Use(aimeter.New(aimeter.Config{
		KeyGenerator:        apiKey,
		PromptCostPer1K:     0.0005,
		CompletionCostPer1K: 0.0015,
		MaxRequests:         60,
		MaxTokens:           100_000,
		Window:              time.Minute,
		OnResult: func(c *goryu.Ctx, r aimeter.Result) {
			log.Printf("meter key=%s tokens=%d cost=$%.5f", r.Key, r.Usage.Total(), r.CostUSD)
		},
	}))

	// Streaming endpoint: tokens over SSE. Not cached (streams are pass-through).
	app.POST("/chat/stream", chatStream)

	// Schema endpoint: hand this to an LLM as a response schema, or to clients.
	app.GET("/chat/schema", func(c *goryu.Ctx) {
		c.JSON(200, goryu.Schema[ChatRequest]())
	})

	// Non-streaming endpoint, wrapped in the prompt cache: identical bodies
	// return the cached answer (see the X-Cache header) without re-running.
	cached := app.Group("", promptcache.New(promptcache.Config{
		Expiration: 5 * time.Minute,
	}))
	cached.POST("/chat", chat)

	// Optional: stream straight through to a real provider when configured.
	if base := os.Getenv("PROVIDER_BASE_URL"); base != "" {
		px := aiproxy.New(aiproxy.Config{
			BaseURL: base,
			APIKey:  os.Getenv("PROVIDER_API_KEY"),
		})
		app.POST("/v1/chat/completions", func(c *goryu.Ctx) {
			if err := px.Stream(c, "/v1/chat/completions"); err != nil {
				log.Printf("proxy error: %v", err)
			}
		})
		log.Printf("provider passthrough enabled at POST /v1/chat/completions -> %s", base)
	}

	log.Println("aichat listening on :3000")
	if err := app.Run(":3000"); err != nil {
		log.Fatal(err)
	}
}

// chatStream streams the reply token-by-token over SSE, then reports usage.
func chatStream(c *goryu.Ctx) {
	req, err := goryu.Bind[ChatRequest](c)
	if err != nil {
		c.JSON(400, goryu.Map{"error": err.Error()})
		return
	}

	words := strings.Fields(fakeReply(req))
	completion := 0

	streamErr := goryu.SSEJSON(c, "token", func(send func(Token) error) error {
		for i, w := range words {
			if err := send(Token{Index: i, Content: w + " "}); err != nil {
				return err
			}
			completion++
			time.Sleep(40 * time.Millisecond) // simulate generation latency
		}
		return send(Token{Index: len(words), Done: true})
	})
	if streamErr != nil {
		log.Printf("stream aborted: %v", streamErr)
	}

	// Report usage so aimeter/recorder can account it. Headers won't apply on a
	// streamed response, but OnResult and the eval record still capture it.
	aimeter.SetUsage(c, aimeter.Usage{
		PromptTokens:     promptTokens(req),
		CompletionTokens: completion,
	})
}

// chat returns the whole reply as JSON; the prompt cache serves repeats.
func chat(c *goryu.Ctx) {
	req, err := goryu.Bind[ChatRequest](c)
	if err != nil {
		c.JSON(400, goryu.Map{"error": err.Error()})
		return
	}

	reply := fakeReply(req)
	aimeter.SetUsage(c, aimeter.Usage{
		PromptTokens:     promptTokens(req),
		CompletionTokens: len(strings.Fields(reply)),
	})
	c.JSON(200, goryu.Map{"model": req.Model, "reply": reply})
}

// apiKey identifies the caller by X-API-Key, falling back to the remote IP.
func apiKey(c *goryu.Ctx) string {
	if k := c.GetHeader("X-API-Key"); k != "" {
		return k
	}
	return c.RemoteIP()
}

// promptTokens is a crude token estimate over the conversation.
func promptTokens(req ChatRequest) int {
	n := 0
	for _, m := range req.Messages {
		n += len(strings.Fields(m.Content))
	}
	return n
}

// fakeReply is a stand-in for a real model call.
func fakeReply(req ChatRequest) string {
	last := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			last = req.Messages[i].Content
			break
		}
	}
	prefix := "You said"
	if req.Model == "smart" {
		prefix = "Thoughtfully, you said"
	}
	return prefix + ": " + last + ". Here is a streamed demonstration reply from goryu."
}
