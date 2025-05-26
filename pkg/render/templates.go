package render

import (
	"html/template"
	"net/http"

	"github.com/arthurlch/goryu/pkg/context"
)

func Render(c *context.Context, code int, templateName string, data interface{}) {
	tmpl, err := template.ParseFiles(templateName)
	if err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
		return
	}
	
	c.Writer.Header().Set("Content-Type", "text/html")
	c.Writer.WriteHeader(code)
	
	// exe the template
	if err := tmpl.Execute(c.Writer, data); err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
	}
}