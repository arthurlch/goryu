package goryu

import (
	"strconv"
	"strings"
)

// APIRoute is one endpoint in a machine-readable API reference.
type APIRoute struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Name   string `json:"name,omitempty"`
}

// APIReference is a machine-readable description of an app's HTTP surface,
// suitable for tooling and LLM consumption.
type APIReference struct {
	Name    string     `json:"name"`
	Summary string     `json:"summary"`
	Routes  []APIRoute `json:"routes"`
}

// APIReference builds a reference from the app's registered routes.
func (app *App) APIReference() APIReference {
	infos := app.Router.Routes()

	routes := make([]APIRoute, 0, len(infos))
	for _, r := range infos {
		routes = append(routes, APIRoute{Method: r.Method, Path: r.Path, Name: r.Name})
	}

	name := app.Config.AppName
	if name == "" {
		name = "API"
	}

	unit := "endpoints"
	if len(routes) == 1 {
		unit = "endpoint"
	}
	summary := name + " — HTTP API built with goryu, exposing " + strconv.Itoa(len(routes)) + " " + unit + "."

	return APIReference{Name: name, Summary: summary, Routes: routes}
}

// LLMsText renders the app's API as an llms.txt document (https://llmstxt.org):
// a title, a one-line summary, and the list of endpoints.
func (app *App) LLMsText() string {
	ref := app.APIReference()

	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(ref.Name)
	b.WriteString("\n\n> ")
	b.WriteString(ref.Summary)
	b.WriteString("\n\n## Endpoints\n\n")

	for _, r := range ref.Routes {
		b.WriteString("- `")
		b.WriteString(r.Method)
		b.WriteByte(' ')
		b.WriteString(r.Path)
		b.WriteByte('`')
		if r.Name != "" {
			b.WriteString(" — ")
			b.WriteString(r.Name)
		}
		b.WriteByte('\n')
	}

	return b.String()
}

// MountLLMs registers GET /llms.txt (the llms.txt document) and GET /llms.json
// (the machine-readable APIReference). Call it after your routes are defined.
func (app *App) MountLLMs() {
	app.GET("/llms.txt", func(c *Ctx) {
		_ = c.Data(200, "text/plain; charset=utf-8", []byte(app.LLMsText()))
	})
	app.GET("/llms.json", func(c *Ctx) {
		_ = c.JSON(200, app.APIReference())
	})
}
