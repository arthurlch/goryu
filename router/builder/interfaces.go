package builder

import context "github.com/arthurlch/goryu/goryuctx"

// SimpleApp interface defines the methods needed from the main App struct
type SimpleApp interface {
	GetRouter() interface{} // Returns *router.Router but using interface{} to avoid circular imports
	ApplyMiddleware(handler context.HandlerFunc) context.HandlerFunc
}

// Middleware type alias for cleaner code
type Middleware = context.Middleware

// Handler type alias
type Handler = context.HandlerFunc