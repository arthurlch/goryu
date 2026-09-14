package cli

import (
	"strings"

	"github.com/arthurlch/goryu/internal/utils"
)

// generateAIHandlerContent generates a streaming AI/LLM chat handler that shows
// off goryu's AI batteries: typed Bind[T] + Validate, JSON Schema generation,
// SSE token streaming (SSEJSON), and token/cost accounting via aimeter.
func generateAIHandlerContent(name string) string {
	handlerName := utils.ToGoIdentifier(name)

	tmpl := `package handlers

import (
	"errors"
	"strings"
	"time"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/middleware/aimeter"
)

// __NAME__Message is one turn in the conversation.
type __NAME__Message struct {
	Role    string ` + "`json:\"role\" jsonschema:\"description=Speaker role,enum=system|user|assistant\"`" + `
	Content string ` + "`json:\"content\" jsonschema:\"description=Message text,minLength=1\"`" + `
}

// __NAME__Request is the typed, validated request body for the chat endpoint.
type __NAME__Request struct {
	Model    string          ` + "`json:\"model\" jsonschema:\"description=Model to use,enum=fast|smart\"`" + `
	Messages []__NAME__Message ` + "`json:\"messages\" jsonschema:\"description=Conversation so far\"`" + `
}

// Validate is the goryu.Bind hook: it runs automatically during Bind.
func (r __NAME__Request) Validate() error {
	if len(r.Messages) == 0 {
		return errors.New("messages: at least one message is required")
	}
	return nil
}

// __NAME__Token is a single streamed chunk of the reply.
type __NAME__Token struct {
	Index   int    ` + "`json:\"index\"`" + `
	Content string ` + "`json:\"content,omitempty\"`" + `
	Done    bool   ` + "`json:\"done,omitempty\"`" + `
}

// __NAME__Stream streams the reply token-by-token over Server-Sent Events, then
// reports token usage so aimeter can meter cost and enforce budgets.
//
// Register it with:  app.POST("/__LOWER__/stream", handlers.__NAME__Stream)
func __NAME__Stream(c *goryu.Context) {
	req, err := goryu.Bind[__NAME__Request](c)
	if err != nil {
		_ = c.JSON(400, goryu.Map{"error": err.Error()})
		return
	}

	// TODO: replace this stub with a real provider call and stream its tokens.
	words := strings.Fields("This is a streamed reply from goryu. Replace me with a real model call.")
	completion := 0

	_ = goryu.SSEJSON(c, "token", func(send func(__NAME__Token) error) error {
		for i, w := range words {
			if sendErr := send(__NAME__Token{Index: i, Content: w + " "}); sendErr != nil {
				return sendErr
			}
			completion++
			time.Sleep(30 * time.Millisecond) // simulate generation latency
		}
		return send(__NAME__Token{Index: len(words), Done: true})
	})

	aimeter.SetUsage(c, aimeter.Usage{
		PromptTokens:     promptTokens__NAME__(req),
		CompletionTokens: completion,
	})
}

// __NAME__Schema serves the JSON Schema for the request type — hand it to an LLM
// as a response schema, or to clients for validation.
//
// Register it with:  app.GET("/__LOWER__/schema", handlers.__NAME__Schema)
func __NAME__Schema(c *goryu.Context) {
	_ = c.JSON(200, goryu.Schema[__NAME__Request]())
}

// promptTokens__NAME__ is a crude token estimate over the conversation.
func promptTokens__NAME__(req __NAME__Request) int {
	n := 0
	for _, m := range req.Messages {
		n += len(strings.Fields(m.Content))
	}
	return n
}
`

	tmpl = strings.ReplaceAll(tmpl, "__NAME__", handlerName)
	tmpl = strings.ReplaceAll(tmpl, "__LOWER__", strings.ToLower(name))
	return tmpl
}
