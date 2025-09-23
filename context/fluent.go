package context

import (
	"encoding/json"
	"net/http"
)

// Overall fluent API design inspired by Echo framework but adapted for Goryu context structure
// Each method returns *Context to allow chaining
// Error handling is done internally and can be checked after the chaining with c.Errors()

func (c *Context) FluentStatus(code int) *Context {
	err := c.safeExecute("Status", HeaderError, func() error {
		c.Writer.WriteHeader(code)
		return nil
	})
	return c.collectError(err)
}

func (c *Context) FluentHeader(key, value string) *Context {
	err := c.safeExecute("Header", HeaderError, func() error {
		c.Writer.Header().Set(key, value)
		return nil
	})
	return c.collectError(err)
}

func (c *Context) FluentHeaders(headers map[string]string) *Context {
	for key, value := range headers {
		c.FluentHeader(key, value)
	}
	return c
}

func (c *Context) FluentContentType(contentType string) *Context {
	return c.FluentHeader("Content-Type", contentType)
}

// Note: Using this for fluent chaining. For error-returning behavior, use JSON()
func (c *Context) FluentJSON(code int, obj interface{}) *Context {
	err := c.safeExecute("JSON", SerializationError, func() error {
		c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		c.Writer.WriteHeader(code)
		enc := json.NewEncoder(c.Writer)
		if err := enc.Encode(obj); err != nil {
			return &ResponseError{
				Type:      SerializationError,
				Operation: "JSON",
				Err:       err,
				Code:      code,
				Critical:  true,
			}
		}
		return nil
	})
	return c.collectError(err)
}

func (c *Context) FluentText(code int, text string) *Context {
	err := c.safeExecute("Text", WriteError, func() error {
		c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		c.Writer.WriteHeader(code)
		_, writeErr := c.Writer.Write([]byte(text))
		return writeErr
	})
	return c.collectError(err)
}

func (c *Context) FluentHTML(code int, html string) *Context {
	err := c.safeExecute("HTML", WriteError, func() error {
		c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		c.Writer.WriteHeader(code)
		_, writeErr := c.Writer.Write([]byte(html))
		return writeErr
	})
	return c.collectError(err)
}

func (c *Context) FluentXML(code int, xml string) *Context {
	err := c.safeExecute("XML", WriteError, func() error {
		c.Writer.Header().Set("Content-Type", "application/xml; charset=utf-8")
		c.Writer.WriteHeader(code)
		_, writeErr := c.Writer.Write([]byte(xml))
		return writeErr
	})
	return c.collectError(err)
}

func (c *Context) FluentData(code int, contentType string, data []byte) *Context {
	err := c.safeExecute("Data", WriteError, func() error {
		if contentType != "" {
			c.Writer.Header().Set("Content-Type", contentType)
		}
		c.Writer.WriteHeader(code)
		_, writeErr := c.Writer.Write(data)
		return writeErr
	})
	return c.collectError(err)
}

func (c *Context) FluentRedirect(code int, url string) *Context {
	err := c.safeExecute("Redirect", WriteError, func() error {
		if code < 300 || code >= 400 {
			code = http.StatusFound // Default to 302
		}
		http.Redirect(c.Writer, c.Request, url, code)
		return nil
	})
	return c.collectError(err)
}

func (c *Context) FluentCookie(cookie *http.Cookie) *Context {
	err := c.safeExecute("SetCookie", HeaderError, func() error {
		http.SetCookie(c.Writer, cookie)
		return nil
	})
	return c.collectError(err)
}

func (c *Context) FluentVary(fields ...string) *Context {
	err := c.safeExecute("Vary", HeaderError, func() error {
		for _, field := range fields {
			current := c.Writer.Header().Get("Vary")
			if current == "" {
				c.Writer.Header().Set("Vary", field)
			} else if !contains(current, field) {
				c.Writer.Header().Set("Vary", current+", "+field)
			}
		}
		return nil
	})
	return c.collectError(err)
}

func (c *Context) FluentLocation(path string) *Context {
	return c.FluentHeader("Location", path)
}

func (c *Context) FluentAttachment(filename ...string) *Context {
	err := c.safeExecute("Attachment", HeaderError, func() error {
		if len(filename) > 0 && filename[0] != "" {
			c.Writer.Header().Set("Content-Disposition", `attachment; filename="`+filename[0]+`"`)
		} else {
			c.Writer.Header().Set("Content-Disposition", "attachment")
		}
		return nil
	})
	return c.collectError(err)
}

func (c *Context) NoCache() *Context {
	return c.FluentHeaders(map[string]string{
		"Cache-Control": "no-cache, no-store, must-revalidate",
		"Pragma":        "no-cache",
		"Expires":       "0",
	})
}

// CORS sets CORS headers using fluent API
func (c *Context) CORS(origin string) *Context {
	return c.FluentHeaders(map[string]string{
		"Access-Control-Allow-Origin":  origin,
		"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, PATCH, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization, X-Requested-With",
	})
}

// Security sets common security headers using fluent API
func (c *Context) Security() *Context {
	return c.FluentHeaders(map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	})
}


func (c *Context) OK(data interface{}) *Context {
	return c.FluentJSON(200, data)
}

func (c *Context) Created(data interface{}) *Context {
	return c.FluentJSON(201, data)
}

func (c *Context) BadRequest(message string) *Context {
	return c.FluentJSON(400, map[string]string{"error": message})
}

func (c *Context) Unauthorized(message string) *Context {
	return c.FluentJSON(401, map[string]string{"error": message})
}

func (c *Context) Forbidden(message string) *Context {
	return c.FluentJSON(403, map[string]string{"error": message})
}

func (c *Context) NotFound(message string) *Context {
	return c.FluentJSON(404, map[string]string{"error": message})
}

func (c *Context) InternalError(message string) *Context {
	return c.FluentJSON(500, map[string]string{"error": message})
}

func (c *Context) Success(message string) *Context {
	return c.OK(map[string]string{"message": message})
}

func (c *Context) FluentError(code int, message string) *Context {
	return c.FluentJSON(code, map[string]string{"error": message})
}


func (c *Context) WithStatus(code int) *Context {
	return c.FluentStatus(code)
}

func (c *Context) WithHeader(key, value string) *Context {
	return c.FluentHeader(key, value)
}

func (c *Context) WithHeaders(headers map[string]string) *Context {
	return c.FluentHeaders(headers)
}

func (c *Context) FluentSend(code int, data interface{}) *Context {
	switch v := data.(type) {
	case string:
		return c.FluentText(code, v)
	case []byte:
		return c.FluentData(code, "", v)
	default:
		return c.FluentJSON(code, v)
	}
}


func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr)))
}