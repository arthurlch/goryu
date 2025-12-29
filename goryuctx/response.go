package goryuctx

import (
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/arthurlch/goryu/internal/bytesconv"
	goryujson "github.com/arthurlch/goryu/internal/json"
)

// isComplexObject determines if an object is complex enough to benefit from sonic
func isComplexObject(obj interface{}) bool {
	// Quick type assertions for common cases to avoid reflection overhead
	switch v := obj.(type) {
	case map[string]string:
		return len(v) > 3
	case map[string]interface{}:
		return len(v) > 3
	case []string:
		return len(v) > 5
	case []interface{}:
		return len(v) > 5
	case []int:
		return len(v) > 5
	case string, int, int32, int64, float32, float64, bool:
		return false
	default:
		// Fall back to reflection for unknown types
		if obj == nil {
			return false
		}

		rv := reflect.ValueOf(obj)
		switch rv.Kind() {
		case reflect.Struct:
			return rv.Type().NumField() > 3
		case reflect.Map:
			return rv.Len() > 3
		case reflect.Slice, reflect.Array:
			return rv.Len() > 5
		case reflect.Ptr, reflect.Interface:
			if rv.IsNil() {
				return false
			}
			return isComplexObject(rv.Elem().Interface())
		default:
			return false
		}
	}
}

func (c *Context) SetHeader(key string, value string) error {
	return c.safeExecute("SetHeader", HeaderError, func() error {
		c.Writer.Header().Set(key, value)
		return nil
	})
}

func (c *Context) JSON(code int, obj interface{}) error {
	// Fast path for non-panic mode
	if c.GetErrorHandlingMode() == ErrorModeReturn {
		c.Writer.Header().Set("Content-Type", "application/json")
		c.Writer.WriteHeader(code)

		// Smart JSON selection: use sonic for complex objects, standard for simple ones
		var data []byte
		var err error
		if isComplexObject(obj) {
			// Use sonic for complex/large objects
			data, err = goryujson.Default.Marshal(obj)
		} else {
			// Use standard library for simple objects
			data, err = json.Marshal(obj)
		}
		if err != nil {
			responseErr := &ResponseError{
				Type:      SerializationError,
				Operation: "JSON",
				Err:       err,
				Code:      code,
				Critical:  true,
			}
			return c.handleResponseError("JSON", responseErr, SerializationError)
		}

		_, err = c.Writer.Write(data)
		if err != nil {
			responseErr := &ResponseError{
				Type:      WriteError,
				Operation: "JSON",
				Err:       err,
				Code:      code,
				Critical:  true,
			}
			return c.handleResponseError("JSON", responseErr, WriteError)
		}
		return nil
	}

	// Slow path with panic recovery
	return c.safeExecute("JSON", SerializationError, func() error {
		c.Writer.Header().Set("Content-Type", "application/json")
		c.Writer.WriteHeader(code)

		// Smart JSON selection: use sonic for complex objects, standard for simple ones
		var data []byte
		var err error
		if isComplexObject(obj) {
			// Use sonic for complex/large objects
			data, err = goryujson.Default.Marshal(obj)
		} else {
			// Use standard library for simple objects
			data, err = json.Marshal(obj)
		}
		if err != nil {
			return &ResponseError{
				Type:      SerializationError,
				Operation: "JSON",
				Err:       err,
				Code:      code,
				Critical:  true,
			}
		}

		_, err = c.Writer.Write(data)
		if err != nil {
			return &ResponseError{
				Type:      WriteError,
				Operation: "JSON",
				Err:       err,
				Code:      code,
				Critical:  true,
			}
		}
		return nil
	})
}

func (c *Context) Text(code int, text string) error {
	// Fast path for non-panic mode
	if c.GetErrorHandlingMode() == ErrorModeReturn {
		c.Writer.Header().Set("Content-Type", "text/plain")
		c.Writer.WriteHeader(code)
		_, err := c.Writer.Write(bytesconv.StringToBytes(text))
		if err != nil {
			responseErr := &ResponseError{
				Type:      WriteError,
				Operation: "Text",
				Err:       err,
				Code:      code,
				Critical:  true,
			}
			return c.handleResponseError("Text", responseErr, WriteError)
		}
		return nil
	}

	// Slow path with panic recovery
	return c.safeExecute("Text", WriteError, func() error {
		c.Writer.Header().Set("Content-Type", "text/plain")
		c.Writer.WriteHeader(code)
		_, err := c.Writer.Write(bytesconv.StringToBytes(text))
		if err != nil {
			return &ResponseError{
				Type:      WriteError,
				Operation: "Text",
				Err:       err,
				Code:      code,
				Critical:  true,
			}
		}
		return nil
	})
}

// JSONHeavy is optimized for complex/large JSON objects using the configured JSON engine
func (c *Context) JSONHeavy(code int, obj interface{}) error {
	// Fast path for non-panic mode
	if c.GetErrorHandlingMode() == ErrorModeReturn {
		c.Writer.Header().Set("Content-Type", "application/json")
		c.Writer.WriteHeader(code)

		// Use configured JSON engine for potentially better performance on large objects
		data, err := goryujson.Default.Marshal(obj)
		if err != nil {
			responseErr := &ResponseError{
				Type:      SerializationError,
				Operation: "JSONHeavy",
				Err:       err,
				Code:      code,
				Critical:  true,
			}
			return c.handleResponseError("JSONHeavy", responseErr, SerializationError)
		}

		_, err = c.Writer.Write(data)
		if err != nil {
			responseErr := &ResponseError{
				Type:      WriteError,
				Operation: "JSONHeavy",
				Err:       err,
				Code:      code,
				Critical:  true,
			}
			return c.handleResponseError("JSONHeavy", responseErr, WriteError)
		}
		return nil
	}

	// Slow path with panic recovery
	return c.safeExecute("JSONHeavy", SerializationError, func() error {
		c.Writer.Header().Set("Content-Type", "application/json")
		c.Writer.WriteHeader(code)

		// Use configured JSON engine
		data, err := goryujson.Default.Marshal(obj)
		if err != nil {
			return &ResponseError{
				Type:      SerializationError,
				Operation: "JSONHeavy",
				Err:       err,
				Code:      code,
				Critical:  true,
			}
		}

		_, err = c.Writer.Write(data)
		if err != nil {
			return &ResponseError{
				Type:      WriteError,
				Operation: "JSONHeavy",
				Err:       err,
				Code:      code,
				Critical:  true,
			}
		}
		return nil
	})
}

// WriteString is a fast, minimal method for writing strings without safety checks
// this is for maximum performance when we know the response won't fail
func (c *Context) WriteString(code int, s string) {
	c.Writer.Header().Set("Content-Type", "text/plain")
	c.Writer.WriteHeader(code)
	c.Writer.Write(bytesconv.StringToBytes(s))
}

func (c *Context) Data(code int, contentType string, data []byte) error {
	return c.safeExecute("Data", WriteError, func() error {
		c.Writer.Header().Set("Content-Type", contentType)
		c.Writer.WriteHeader(code)
		_, err := c.Writer.Write(data)
		if err != nil {
			return &ResponseError{
				Type:      WriteError,
				Operation: "Data",
				Err:       err,
				Code:      code,
				Critical:  true,
			}
		}
		return nil
	})
}

func (c *Context) Status(code int) *Context {
	err := c.safeExecute("Status", HeaderError, func() error {
		c.Writer.WriteHeader(code)
		return nil
	})
	if err != nil {
		c.Set("last_error", err)
		c.callResponseErrorHandler(err)
	}
	return c
}

func (c *Context) Redirect(code int, location string) error {
	return c.safeExecute("Redirect", HeaderError, func() error {
		http.Redirect(c.Writer, c.Request, location, code)
		return nil
	})
}

func (c *Context) SetCookie(cookie *http.Cookie) error {
	return c.safeExecute("SetCookie", HeaderError, func() error {
		http.SetCookie(c.Writer, cookie)
		return nil
	})
}

func (c *Context) Error(err error, statusCode ...int) error {
	code := http.StatusInternalServerError
	if len(statusCode) > 0 {
		code = statusCode[0]
	}

	log.Printf("HTTP %d Error occurred", code)

	message := http.StatusText(code)
	if code == http.StatusInternalServerError {
		message = "Internal Server Error"
	}

	http.Error(c.Writer, message, code)
	return err
}

func (c *Context) ErrorWithMessage(err error, statusCode int, message string) error {
	// SECUCHECK: generic
	log.Printf("HTTP %d Error occurred", statusCode)
	http.Error(c.Writer, message, statusCode)
	return err
}

func (c *Context) Abort(statusCode int, message ...string) error {
	return c.safeExecute("Abort", WriteError, func() error {
		msg := http.StatusText(statusCode)
		if len(message) > 0 && message[0] != "" {
			msg = message[0]
		}
		c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.WriteHeader(statusCode)
		_, err := c.Writer.Write([]byte(msg + "\n"))
		if err != nil {
			return &ResponseError{
				Type:      WriteError,
				Operation: "Abort",
				Err:       err,
				Code:      statusCode,
				Critical:  true,
			}
		}
		return nil
	})
}

func (c *Context) ClearCookie(name string) error {
	return c.safeExecute("ClearCookie", HeaderError, func() error {
		cookie := &http.Cookie{
			Name:    name,
			Value:   "",
			Path:    "/",
			Expires: time.Unix(0, 0),
			MaxAge:  -1,
		}
		http.SetCookie(c.Writer, cookie)
		return nil
	})
}

func (c *Context) Attachment(filename ...string) error {
	return c.safeExecute("Attachment", HeaderError, func() error {
		disposition := "attachment"
		if len(filename) > 0 {
			// again using filepath.Base to prevent directory traversal attacks ......
			fname := filepath.Base(filename[0])
			disposition = fmt.Sprintf("attachment; filename=\"%s\"", fname)
		}
		c.Writer.Header().Set("Content-Disposition", disposition)
		return nil
	})
}

func (c *Context) SendFile(path string) error {
	return c.safeExecute("SendFile", FileError, func() error {
		// SECUCHECK: I should clean the path to prevent directory traversal
		cleanPath := filepath.Clean(path)

		file, err := os.Open(cleanPath)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(c.Writer, "File Not Found", http.StatusNotFound)
			} else if os.IsPermission(err) {
				http.Error(c.Writer, "Forbidden", http.StatusForbidden)
			} else {
				// SECURITY: genreric message
				log.Printf("File operation error: %v", err)
				http.Error(c.Writer, "Internal Server Error", http.StatusInternalServerError)
			}
			return &ResponseError{
				Type:      FileError,
				Operation: "SendFile",
				Err:       err,
				Code:      0,
				Critical:  true,
			}
		}
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				log.Printf("Error closing file %s: %v", cleanPath, closeErr)
			}
		}()

		fileInfo, err := file.Stat()
		if err != nil {
			// SECUCHECK:  generic error message
			log.Printf("File stat operation error: %v", err)
			http.Error(c.Writer, "Internal Server Error", http.StatusInternalServerError)
			return &ResponseError{
				Type:      FileError,
				Operation: "SendFile",
				Err:       fmt.Errorf("error getting file info for %s: %w", cleanPath, err),
				Code:      0,
				Critical:  true,
			}
		}

		if fileInfo.IsDir() {
			http.Error(c.Writer, "Forbidden", http.StatusForbidden)
			return &ResponseError{
				Type:      FileError,
				Operation: "SendFile",
				Err:       fmt.Errorf("path is a directory: %s", cleanPath),
				Code:      0,
				Critical:  true,
			}
		}

		contentType := mime.TypeByExtension(filepath.Ext(fileInfo.Name()))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		c.Writer.Header().Set("Content-Type", contentType)

		http.ServeContent(c.Writer, c.Request, fileInfo.Name(), fileInfo.ModTime(), file)
		return nil
	})
}

func (c *Context) Location(path string) error {
	return c.safeExecute("Location", HeaderError, func() error {
		c.Writer.Header().Set("Location", path)
		return nil
	})
}

func (c *Context) Type(ext string) error {
	return c.safeExecute("Type", HeaderError, func() error {
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		contentType := mime.TypeByExtension(ext)
		if contentType != "" {
			c.Writer.Header().Set("Content-Type", contentType)
		}
		return nil
	})
}

func (c *Context) Vary(fields ...string) error {
	return c.safeExecute("Vary", HeaderError, func() error {
		for _, field := range fields {
			c.Writer.Header().Add("Vary", field)
		}
		return nil
	})
}

func (c *Context) Append(field string, values ...string) error {
	return c.safeExecute("Append", HeaderError, func() error {
		for _, value := range values {
			c.Writer.Header().Add(field, value)
		}
		return nil
	})
}
