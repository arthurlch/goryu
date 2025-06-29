package context

// core context

import (
	"net/http"
)

type Context struct {
	Writer  http.ResponseWriter
	Request *http.Request
	Params  map[string]string
	Keys    map[string]interface{}
}

type HandlerFunc func(*Context)

type Middleware func(HandlerFunc) HandlerFunc

func NewContext(writer http.ResponseWriter, request *http.Request) *Context {
	return &Context{
		Writer:  writer,
		Request: request,
		Params:  make(map[string]string),
		Keys:    make(map[string]interface{}),
	}
}

func (c *Context) Set(key string, value interface{}) {
	c.Keys[key] = value
}

func (c *Context) Get(key string) (value interface{}, exists bool) {
	value, exists = c.Keys[key]
	return
}
