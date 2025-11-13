package builder

import (
	"reflect"
	"strings"

	"github.com/arthurlch/goryu/context"
)

// Controller interface that resources can implement
type Controller interface {
	// Index handles GET /resource
	Index(c *context.Context)
	// Create handles POST /resource
	Create(c *context.Context)
	// Show handles GET /resource/:id
	Show(c *context.Context)
	// Update handles PUT /resource/:id
	Update(c *context.Context)
	// Destroy handles DELETE /resource/:id
	Destroy(c *context.Context)
}

// PartialController allows implementing only some resource methods
type PartialController interface{}

// extractControllerMethod uses reflection to find and return controller methods
func extractControllerMethod(controller interface{}, methodName string) context.HandlerFunc {
	controllerValue := reflect.ValueOf(controller)

	// Look for the method
	method := controllerValue.MethodByName(strings.Title(methodName))
	if !method.IsValid() {
		return nil
	}

	// Check if the method has the right signature
	methodType := method.Type()
	if methodType.NumIn() != 1 || methodType.NumOut() != 0 {
		return nil
	}

	// Check if the parameter is *context.Context
	contextType := reflect.TypeOf((*context.Context)(nil))
	if methodType.In(0) != contextType {
		return nil
	}

	// Return a wrapper function
	return func(c *context.Context) {
		method.Call([]reflect.Value{reflect.ValueOf(c)})
	}
}

// ResourceController wraps a controller for resource registration
type ResourceController struct {
	controller interface{}
	handlers   map[string]context.HandlerFunc
}

// NewResourceController creates a new resource controller wrapper
func NewResourceController(controller interface{}) *ResourceController {
	rc := &ResourceController{
		controller: controller,
		handlers:   make(map[string]context.HandlerFunc),
	}

	// Extract methods using reflection
	actions := []string{"index", "create", "show", "update", "destroy"}
	for _, action := range actions {
		if handler := extractControllerMethod(controller, action); handler != nil {
			rc.handlers[action] = handler
		}
	}

	return rc
}

// GetHandler returns the handler for a specific action
func (rc *ResourceController) GetHandler(action string) context.HandlerFunc {
	return rc.handlers[action]
}

// HasHandler checks if a handler exists for the action
func (rc *ResourceController) HasHandler(action string) bool {
	_, exists := rc.handlers[action]
	return exists
}

// getHandlerForAction improved implementation
func (rb *ResourceBuilder) getHandlerForActionImproved(action string) context.HandlerFunc {
	// If controller is already a map of handlers
	if handlers, ok := rb.controller.(map[string]context.HandlerFunc); ok {
		return handlers[action]
	}

	// If controller is a ResourceController
	if rc, ok := rb.controller.(*ResourceController); ok {
		return rc.GetHandler(action)
	}

	// Try to extract method using reflection
	return extractControllerMethod(rb.controller, action)
}