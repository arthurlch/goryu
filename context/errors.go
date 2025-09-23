package context

import (
	"fmt"
	"log"
	"strings"
)

type ErrorHandlingMode int

const (
	ErrorModeReturn ErrorHandlingMode = iota
	ErrorModeLog
	ErrorModePanic
	ErrorModeSilent
)

type ErrorType int

const (
	WriteError ErrorType = iota
	HeaderError
	ValidationError
	SerializationError
	FileError
	InternalError
)

type ResponseError struct {
	Type      ErrorType 
	Operation string    
	Err       error     
	Code      int       
	Critical  bool      
	Recovered bool      
}

func (e *ResponseError) Error() string {
	prefix := ""
	if e.Recovered {
		prefix = "recovered "
	}
	
	typeStr := ""
	switch e.Type {
	case WriteError:
		typeStr = "write error"
	case HeaderError:
		typeStr = "header error"
	case ValidationError:
		typeStr = "validation error"
	case SerializationError:
		typeStr = "serialization error"
	case FileError:
		typeStr = "file error"
	case InternalError:
		typeStr = "internal error"
	}
	
	if typeStr != "" {
		return fmt.Sprintf("%s%s in %s: %v", prefix, typeStr, e.Operation, e.Err)
	}
	return fmt.Sprintf("%serror in %s: %v", prefix, e.Operation, e.Err)
}

func (e *ResponseError) Unwrap() error {
	return e.Err
}

type ResponseConfig struct {
	ChainOnError bool
	
	AutoRespond bool
	
	ErrorHandler func(c *Context, err error)
}

func (c *Context) WithResponseConfig(config ResponseConfig) *Context {
	if c.Keys == nil {
		c.Keys = make(map[string]interface{})
	}
	c.Keys["response_config"] = config
	return c
}

func (c *Context) GetResponseConfig() ResponseConfig {
	if c.Keys == nil {
		return ResponseConfig{} // default config
	}
	if config, exists := c.Keys["response_config"]; exists {
		if cfg, ok := config.(ResponseConfig); ok {
			return cfg
		}
	}
	return ResponseConfig{} // default config
}

func (c *Context) SetErrorHandlingMode(mode ErrorHandlingMode) {
	if c.Keys == nil {
		c.Keys = make(map[string]interface{})
	}
	c.Keys["error_handling_mode"] = mode
}

func (c *Context) GetErrorHandlingMode() ErrorHandlingMode {
	if c.Keys == nil {
		return ErrorModeReturn
	}
	if mode, exists := c.Keys["error_handling_mode"]; exists {
		if m, ok := mode.(ErrorHandlingMode); ok {
			return m
		}
	}
	return ErrorModeReturn
}

func (c *Context) handleResponseError(operation string, err error, errorType ErrorType) error {
	if err == nil {
		return nil
	}
	
	responseErr, isResponseErr := err.(*ResponseError)
	if !isResponseErr {
		responseErr = &ResponseError{
			Type:      errorType,
			Operation: operation,
			Err:       err,
			Critical:  errorType == WriteError || errorType == InternalError,
		}
	}
	
	config := c.GetResponseConfig()
	
	if config.ErrorHandler != nil {
		config.ErrorHandler(c, responseErr)
	}
	
	c.callResponseErrorHandler(responseErr)
	
	if config.AutoRespond && responseErr.Critical {
		c.sendErrorResponse(responseErr)
	}
	
	mode := c.GetErrorHandlingMode()
	
	switch mode {
	case ErrorModeReturn:
		log.Printf("%s: %v", responseErr.Error(), err)
		return responseErr
		
	case ErrorModeLog:
		log.Printf("%s: %v", responseErr.Error(), err)
		return nil
		
	case ErrorModePanic:
		panic(responseErr)
		
	case ErrorModeSilent:
		return nil
		
	default:
		log.Printf("%s: %v", responseErr.Error(), err)
		return responseErr
	}
}

func (c *Context) safeExecute(operation string, errorType ErrorType, fn func() error) error {
	var err error
	
	if c.GetErrorHandlingMode() != ErrorModePanic {
		defer func() {
			if r := recover(); r != nil {
				var panicErr error
				if e, ok := r.(error); ok {
					panicErr = e
				} else {
					panicErr = fmt.Errorf("%v", r)
				}
				
				responseErr := &ResponseError{
					Type:      errorType,
					Operation: operation,
					Err:       panicErr,
					Critical:  true, // panics are always critical
					Recovered: true,
				}
				
				log.Printf("Recovered panic in %s: %v", operation, panicErr)
				err = responseErr
			}
		}()
	}
	
	err = fn()
	return c.handleResponseError(operation, err, errorType)
}

// SECUCHECK: Thread-safe to prevent race conditions
func (c *Context) sendErrorResponse(err *ResponseError) {
	// SECUCHECK: Check if response already sent to prevent race conditions
	if !c.markResponseSent() {
		return 
	}
	
	status := 500
	message := "Internal Server Error"
	
	if err.Code > 0 {
		status = err.Code
	} else {
		switch err.Type {
		case ValidationError:
			status = 400
			message = "Bad Request"
		case FileError:
			if err.Err != nil && strings.Contains(err.Err.Error(), "not found") {
				status = 404
				message = "Not Found"
			} else if strings.Contains(err.Err.Error(), "permission") {
				status = 403
				message = "Forbidden"
			}
		case SerializationError:
			status = 400
			message = "Invalid Request Data"
		}
	}
	
	// SECUCHECK: Safe to write response as we have exclusive access
	c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
	c.Writer.WriteHeader(status)
	c.Writer.Write([]byte(message + "\n"))
}

func (c *Context) OnResponseError(handler func(error)) {
	if c.Keys == nil {
		c.Keys = make(map[string]interface{})
	}
	c.Keys["response_error_handler"] = handler
}

func (c *Context) callResponseErrorHandler(err error) {
	if c.Keys == nil {
		return
	}
	
	if handler, exists := c.Keys["response_error_handler"]; exists {
		if h, ok := handler.(func(error)); ok {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic in custom response error handler: %v", r)
				}
			}()
			h(err)
		}
	}
}