package context

// core context

import (
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/arthurlch/goryu/route"
)

// I had a mental breakdown to think about context package name conflict, fluent API and all that stuff
// So I need to come back here !
// MEMO: I think I will rename this package to goryuctx or something later if it is too confusing
type Context struct {
	Writer  http.ResponseWriter
	Request *http.Request
	Params  map[string]string
	Keys    map[string]interface{}
	Route   *route.Route 
	
	// SECUCHECK: Sync for thread-safe operations
	mu           sync.RWMutex 
	responseSent int32        // SECUCHECK: Atomic flag to prevent response race conditions
	
	errors       []error                
	errorHandler func(c *Context, err error) 
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

// SECUCHECK: Thread-safe Set method
func (c *Context) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Keys[key] = value
}

// SECUCHECK: Thread-safe Get method
func (c *Context) Get(key string) (value interface{}, exists bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, exists = c.Keys[key]
	return
}

// SECUCHECK: Check if response has been sent (thread-safe using atomic operations)
func (c *Context) IsResponseSent() bool {
	return atomic.LoadInt32(&c.responseSent) == 1
}

// SECUCHECK: Mark response as sent (thread-safe using atomic CAS to prevent race conditions)
func (c *Context) markResponseSent() bool {
	return atomic.CompareAndSwapInt32(&c.responseSent, 0, 1)
}

func (c *Context) Param(key string) string {
	value, exists := c.Params[key]
	if exists {
		return value
	}
	return ""
}


func (c *Context) collectError(err error) *Context {
	if err != nil {
		c.mu.Lock()
		c.errors = append(c.errors, err)
		c.mu.Unlock()
		
		if c.errorHandler != nil {
			c.errorHandler(c, err)
		}
	}
	return c
}

func (c *Context) OnError(handler func(c *Context, err error)) *Context {
	c.errorHandler = handler
	return c
}

func (c *Context) HasErrors() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.errors) > 0
}

func (c *Context) Errors() []error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]error, len(c.errors))
	copy(result, c.errors)
	return result
}

func (c *Context) ClearErrors() *Context {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errors = nil
	return c
}

func (c *Context) FirstError() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.errors) > 0 {
		return c.errors[0]
	}
	return nil
}
