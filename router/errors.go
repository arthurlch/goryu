package router

import (
	"fmt"
	"log"
)

// RouterError represents errors that occur during route registration
type RouterError struct {
	Operation string // The operation that failed (Add, SetName, etc.)
	Message   string // Error message
}

func (e *RouterError) Error() string {
	return fmt.Sprintf("router %s error: %s", e.Operation, e.Message)
}

// ErrorHandlingMode defines how the router should handle registration errors
type RouterErrorMode int

const (
	// RouterErrorModePanic panics on errors (current behavior, backward compatible)
	RouterErrorModePanic RouterErrorMode = iota
	// RouterErrorModeLog logs errors but continues execution
	RouterErrorModeLog
	// RouterErrorModeReturn returns errors to caller (requires API changes)
	RouterErrorModeReturn
	// RouterErrorModeSilent ignores errors completely
	RouterErrorModeSilent
)

// SetErrorHandlingMode sets how the router should handle registration errors
func (r *Router) SetErrorHandlingMode(mode RouterErrorMode) {
	r.Config.ErrorMode = mode
}

// GetErrorHandlingMode returns the current error handling mode
func (r *Router) GetErrorHandlingMode() RouterErrorMode {
	return r.Config.ErrorMode
}

// handleRouterError handles router errors according to the configured mode
func (r *Router) handleRouterError(operation, message string) {
	err := &RouterError{
		Operation: operation,
		Message:   message,
	}

	mode := r.GetErrorHandlingMode()

	switch mode {
	case RouterErrorModePanic:
		// Maintain backward compatibility - panic as before
		panic(err.Error())

	case RouterErrorModeLog:
		// Log the error but continue execution
		log.Printf("Router error: %v", err)

	case RouterErrorModeReturn:
		// This would require API changes to return errors
		// For now, treat as log mode since we can't change method signatures
		log.Printf("Router error (would return): %v", err)

	case RouterErrorModeSilent:
		// Completely ignore the error
		return

	default:
		// Fallback to panic mode for unknown modes
		panic(err.Error())
	}
}
