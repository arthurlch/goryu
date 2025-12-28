package goryuctx

// core context

import (
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/arthurlch/goryu/route"
)

var contextPool = sync.Pool{
	New: func() interface{} {
		return &Context{
			// Optimization: Maps are initialized lazily to save allocation
			// Params: nil
			// Keys:   nil
		}
	},
}

// I had a mental breakdown to think about context package name conflict, fluent API and all that stuff
// So I need to come back here !
// MEMO: I think I will rename this package to goryuctx or something later if it is too confusing
type Context struct {
	Writer  http.ResponseWriter
	Request *http.Request
	Params  map[string]string
	Keys    map[string]interface{}
	Route   *route.Route

	// Optimization: Reused buffer for path splitting to avoid allocations
	PathBuffer []string
	// Optimization: Reused buffer for param values to avoid map usage in traversal
	ParamValues []string

	// SECUCHECK: Sync for thread-safe operations
	mu           sync.RWMutex
	responseSent int32 // SECUCHECK: Atomic flag to prevent response race conditions

	errors       []error
	errorHandler func(c *Context, err error)

	// Performance optimization: cache error handling mode
	errorHandlingMode ErrorHandlingMode
	errorModeSet      bool
}

type HandlerFunc func(*Context)

type Middleware func(HandlerFunc) HandlerFunc

func NewContext(writer http.ResponseWriter, request *http.Request) *Context {
	c := contextPool.Get().(*Context)
	c.Reset(writer, request)
	return c
}

// Reset resets the context for a new request.
func (c *Context) Reset(writer http.ResponseWriter, request *http.Request) {
	c.Writer = writer
	c.Request = request
	c.Route = nil
	// Reset Params map if it exists
	if c.Params != nil {
		clear(c.Params)
	}
	// Reset Keys map if it exists
	if c.Keys != nil {
		clear(c.Keys)
	}

	// Optimization: Reset PathBuffer
	c.PathBuffer = c.PathBuffer[:0]
	// Optimization: Reset ParamValues
	c.ParamValues = c.ParamValues[:0]

	c.mu.Lock()
	c.responseSent = 0
	c.errors = c.errors[:0] // keep capacity
	c.errorHandler = nil
	c.mu.Unlock()

	// Reset cached error mode
	c.errorModeSet = false
	c.errorHandlingMode = ErrorModeReturn
}

// Release puts the context back into the pool.
func (c *Context) Release() {
	c.Writer = nil
	c.Request = nil
	c.Route = nil
	contextPool.Put(c)
}

// Set stores a key-value pair in the context.
func (c *Context) Set(key string, value interface{}) {
	c.mu.Lock()
	if c.Keys == nil {
		c.Keys = make(map[string]interface{})
	}
	c.Keys[key] = value
	c.mu.Unlock()
}

// Get gets a value from the context.
func (c *Context) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Keys == nil {
		return nil, false
	}
	value, exists := c.Keys[key]
	return value, exists
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
	// Optimization: Scan ParamValues using Route.ParamNames to avoid map access
	if c.Route != nil && len(c.Route.ParamNames) > 0 {
		for i, name := range c.Route.ParamNames {
			if name == key {
				if i < len(c.ParamValues) {
					return c.ParamValues[i]
				}
				break
			}
		}
	}
	// Fallback to map if populated (deprecated usage pattern but supported)
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

// Methods moved to response.go
