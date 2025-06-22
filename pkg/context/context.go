package context

import (
	"encoding/json"
	"log"
	"net/http"
)

// NOTE: on going cleanup needed

type HandlerFunc func(*Context)

type Context struct {
	Writer  http.ResponseWriter
	Request *http.Request
	Params  map[string]string
}

func New(writer http.ResponseWriter, request *http.Request) *Context {
	return &Context{
		Writer:  writer,
		Request: request,
		Params:  make(map[string]string),
	}
}

func (context *Context) Query(name string) string {
	return context.Request.URL.Query().Get(name)
}

func (context *Context) Form(name string) string {
	return context.Request.FormValue(name)
}

func (context *Context) JSON(code int, obj interface{}) {
	context.Writer.Header().Set("Content-Type", "application/json")
	context.Writer.WriteHeader(code)

	if err := json.NewEncoder(context.Writer).Encode(obj); err != nil {
		http.Error(context.Writer, err.Error(), http.StatusInternalServerError)
	}
}

func (context *Context) Text(code int, text string) {
	context.Writer.Header().Set("Content-Type", "text/plain")
	context.Writer.WriteHeader(code)
	if _, err := context.Writer.Write([]byte(text)); err != nil {
		log.Printf("Error writing text response: %v", err)
	}
}

func (context *Context) HTML(code int, html string) {
	context.Writer.Header().Set("Content-Type", "text/html")
	context.Writer.WriteHeader(code)
	if _, err := context.Writer.Write([]byte(html)); err != nil {
		log.Printf("Error writing HTML response: %v", err)
	}
}

// work on going