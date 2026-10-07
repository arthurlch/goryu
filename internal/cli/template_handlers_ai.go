package cli

import (
	"strings"

	"github.com/arthurlch/goryu/internal/utils"
)

// generateAIHandlerContent builds an AI handler: chat, rag, or agent.
func generateAIHandlerContent(name, kind string) string {
	var tmpl string
	switch kind {
	case "rag":
		tmpl = ragTemplate
	case "agent":
		tmpl = agentTemplate
	default:
		tmpl = chatTemplate
	}
	tmpl = strings.ReplaceAll(tmpl, "__NAME__", utils.ToGoIdentifier(name))
	tmpl = strings.ReplaceAll(tmpl, "__LOWER__", strings.ToLower(name))
	return tmpl
}

const chatTemplate = `package handlers

import (
	"errors"
	"strings"
	"time"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/middleware/aimeter"
)

type __NAME__Message struct {
	Role    string ` + "`json:\"role\" jsonschema:\"description=Speaker role,enum=system|user|assistant\"`" + `
	Content string ` + "`json:\"content\" jsonschema:\"description=Message text,minLength=1\"`" + `
}

type __NAME__Request struct {
	Model    string            ` + "`json:\"model\" jsonschema:\"description=Model to use,enum=fast|smart\"`" + `
	Messages []__NAME__Message ` + "`json:\"messages\" jsonschema:\"description=Conversation so far\"`" + `
}

func (r __NAME__Request) Validate() error {
	if len(r.Messages) == 0 {
		return errors.New("messages: at least one message is required")
	}
	return nil
}

type __NAME__Token struct {
	Index   int    ` + "`json:\"index\"`" + `
	Content string ` + "`json:\"content,omitempty\"`" + `
	Done    bool   ` + "`json:\"done,omitempty\"`" + `
}

// __NAME__Stream streams the reply token-by-token over SSE, then reports usage.
// Register with: app.POST("/__LOWER__/stream", handlers.__NAME__Stream)
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
			time.Sleep(30 * time.Millisecond)
		}
		return send(__NAME__Token{Index: len(words), Done: true})
	})

	prompt := 0
	for _, m := range req.Messages {
		prompt += len(strings.Fields(m.Content))
	}
	aimeter.SetUsage(c, aimeter.Usage{PromptTokens: prompt, CompletionTokens: completion})
}

// __NAME__Schema serves the JSON Schema for the request type.
// Register with: app.GET("/__LOWER__/schema", handlers.__NAME__Schema)
func __NAME__Schema(c *goryu.Context) {
	_ = c.JSON(200, goryu.Schema[__NAME__Request]())
}
`

const ragTemplate = `package handlers

import (
	"errors"
	"strconv"
	"strings"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/middleware/aimeter"
)

type __NAME__Query struct {
	Question string ` + "`json:\"question\" jsonschema:\"description=User question,minLength=1\"`" + `
	TopK     int    ` + "`json:\"top_k,omitempty\" jsonschema:\"description=Documents to retrieve,minimum=1,maximum=20\"`" + `
}

func (q __NAME__Query) Validate() error {
	if q.Question == "" {
		return errors.New("question: is required")
	}
	return nil
}

type __NAME__Source struct {
	ID    string  ` + "`json:\"id\"`" + `
	Score float64 ` + "`json:\"score\"`" + `
	Text  string  ` + "`json:\"text\"`" + `
}

type __NAME__Answer struct {
	Answer  string           ` + "`json:\"answer\"`" + `
	Sources []__NAME__Source ` + "`json:\"sources\"`" + `
}

// __NAME__Ask retrieves context for the question and returns an answer with sources.
// Register with: app.POST("/__LOWER__", handlers.__NAME__Ask)
func __NAME__Ask(c *goryu.Context) {
	q, err := goryu.Bind[__NAME__Query](c)
	if err != nil {
		_ = c.JSON(400, goryu.Map{"error": err.Error()})
		return
	}

	topK := q.TopK
	if topK == 0 {
		topK = 3
	}
	sources := retrieve__NAME__(q.Question, topK)

	// TODO: build a prompt from sources and call your LLM.
	answer := "TODO: answer using the retrieved sources."

	contextTokens := 0
	for _, s := range sources {
		contextTokens += len(strings.Fields(s.Text))
	}
	aimeter.SetUsage(c, aimeter.Usage{
		PromptTokens:     len(strings.Fields(q.Question)) + contextTokens,
		CompletionTokens: len(strings.Fields(answer)),
	})

	_ = c.JSON(200, __NAME__Answer{Answer: answer, Sources: sources})
}

// retrieve__NAME__ is a stub retriever — replace it with a vector-store query.
func retrieve__NAME__(question string, topK int) []__NAME__Source {
	_ = question
	out := make([]__NAME__Source, 0, topK)
	for i := 0; i < topK; i++ {
		out = append(out, __NAME__Source{
			ID:    "doc-" + strconv.Itoa(i+1),
			Score: 0,
			Text:  "TODO: retrieved passage " + strconv.Itoa(i+1),
		})
	}
	return out
}
`

const agentTemplate = `package handlers

import (
	"errors"

	"github.com/arthurlch/goryu"
)

type __NAME__Input struct {
	Query string ` + "`json:\"query\" jsonschema:\"description=What the tool should act on,minLength=1\"`" + `
}

func (i __NAME__Input) Validate() error {
	if i.Query == "" {
		return errors.New("query: is required")
	}
	return nil
}

type __NAME__Result struct {
	Output string ` + "`json:\"output\"`" + `
}

// __NAME__Tool runs the tool and returns its result.
// Register with: app.POST("/tools/__LOWER__", handlers.__NAME__Tool)
func __NAME__Tool(c *goryu.Context) {
	in, err := goryu.Bind[__NAME__Input](c)
	if err != nil {
		_ = c.JSON(400, goryu.Map{"error": err.Error()})
		return
	}

	// TODO: execute the tool for in.Query.
	_ = c.JSON(200, __NAME__Result{Output: "TODO: result for " + in.Query})
}

// __NAME__ToolSchema serves an OpenAI-style tool definition for function calling.
// Register with: app.GET("/tools/__LOWER__/schema", handlers.__NAME__ToolSchema)
func __NAME__ToolSchema(c *goryu.Context) {
	_ = c.JSON(200, goryu.Map{
		"name":        "__LOWER__",
		"description": "TODO: describe what this tool does",
		"parameters":  goryu.Schema[__NAME__Input](),
	})
}
`
