package goryuctx

import (
	"log"
	"net/http"
)

// Compatibility and convenience methods for fluent API support

func (c *Context) StatusWithError(code int) error {
	c.Status(code)
	if lastErr, exists := c.Get("last_error"); exists {
		if err, ok := lastErr.(error); ok {
			return err
		}
	}
	return nil
}

func (c *Context) SetHeaderSilent(key string, value string) {
	err := c.SetHeader(key, value)
	if err != nil {
		// SECUCHECK: Don't expose header details in logs - use generic message
		log.Printf("Warning: Error setting response header")
	}
}

func (c *Context) JSONSilent(code int, obj interface{}) {
	err := c.JSON(code, obj)
	if err != nil {
		// SECUCHECK: Shall not expose error details
		log.Printf("Warning: Error sending JSON response")
	}
}

func (c *Context) TextSilent(code int, text string) {
	err := c.Text(code, text)
	if err != nil {
		// SECUCHECK: Shall not expose error details
		log.Printf("Warning: Error sending text response")
	}
}

func (c *Context) RedirectSilent(code int, location string) {
	err := c.Redirect(code, location)
	if err != nil {
		// SECUCHECK: Shall not expose redirect details - use generic message
		log.Printf("Warning: Error performing redirect")
	}
}

func (c *Context) SetCookieSilent(cookie *http.Cookie) {
	err := c.SetCookie(cookie)
	if err != nil {
		// SECUCHECK: generic
		log.Printf("Warning: Error setting cookie")
	}
}

func (c *Context) AbortSilent(statusCode int, message ...string) {
	err := c.Abort(statusCode, message...)
	if err != nil {
		// SECURITY: generic
		log.Printf("Warning: Error aborting request")
	}
}

// Fluent API methods that always return *Context for chaining
// NOTE: WithHeader and WithStatus are now defined in fluent.go for

func (c *Context) WithType(ext string) *Context {
	originalMode := c.GetErrorHandlingMode()
	c.SetErrorHandlingMode(ErrorModeLog)
	_ = c.Type(ext)
	c.SetErrorHandlingMode(originalMode)
	return c
}

func (c *Context) WithVary(fields ...string) *Context {
	originalMode := c.GetErrorHandlingMode()
	c.SetErrorHandlingMode(ErrorModeLog)
	_ = c.Vary(fields...)
	c.SetErrorHandlingMode(originalMode)
	return c
}

func (c *Context) WithAppend(field string, values ...string) *Context {
	originalMode := c.GetErrorHandlingMode()
	c.SetErrorHandlingMode(ErrorModeLog)
	_ = c.Append(field, values...)
	c.SetErrorHandlingMode(originalMode)
	return c
}

func (c *Context) WithAttachment(filename ...string) *Context {
	originalMode := c.GetErrorHandlingMode()
	c.SetErrorHandlingMode(ErrorModeLog)
	_ = c.Attachment(filename...)
	c.SetErrorHandlingMode(originalMode)
	return c
}

func (c *Context) WithLocation(path string) *Context {
	originalMode := c.GetErrorHandlingMode()
	c.SetErrorHandlingMode(ErrorModeLog)
	_ = c.Location(path)
	c.SetErrorHandlingMode(originalMode)
	return c
}

func (c *Context) Send(code int, data ...interface{}) error {
	if len(data) == 0 {
		c.Status(code)
		if lastErr, exists := c.Get("last_error"); exists {
			if err, ok := lastErr.(error); ok {
				return err
			}
		}
		return nil
	}

	switch v := data[0].(type) {
	case string:
		return c.Text(code, v)
	case []byte:
		return c.Data(code, "application/octet-stream", v)
	default:
		return c.JSON(code, v)
	}
}

func (c *Context) SendString(code int, text string) error {
	return c.Text(code, text)
}

func (c *Context) SendJSON(code int, obj interface{}) error {
	return c.JSON(code, obj)
}

func (c *Context) SendBytes(code int, contentType string, data []byte) error {
	return c.Data(code, contentType, data)
}
