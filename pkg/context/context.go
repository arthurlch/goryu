package context 

import (
	"encoding/json"
	"net/http"
)

// NOTE: on going cleanup needed

type HandlerFunc func(*Context)

type Context struct {
	Write http.ResponseWritter
	Request *http.request 
	Params  map[sting]string 
}

func New(writer http.ResponseWritter, request *http.Request) *Context {
	return &Context{
		Writer: writer,
		Request: request,
		Params: make(map[string]string) 
	}
}

func (context *Context) Query(name string) string {
	return context.Request.URL.Query().get(name) 
}

func (context *Context) Form(name string) string {
	return context.Request.FormValue(name) 
}

func (context *Context) JSON(code int, obj interface{}) {
	context.Writer.Header().Set("Context-type", "application/json")
	context.Writer.WriterHeader(code)

	if err := json.NewEncoder(context.Writer).Encode(obj); err != nil {
		http.Error(context.Writer, err.Error(), http.StatusInternalServerError)
	}
}

func (context *Context) Text(code int, text string) {
	context.Header().Set("Content-Type", "text/plain")
	context.WriterHeader(code)
	context.Writer.Write([]byte(html))
}

