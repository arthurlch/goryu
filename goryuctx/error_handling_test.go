package goryuctx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestErrorHandlingModes(t *testing.T) {
	t.Run("ErrorModeReturn", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		ctx := NewContext(rr, req)
		
		// Default mode should be return
		if ctx.GetErrorHandlingMode() != ErrorModeReturn {
			t.Errorf("Expected default mode to be ErrorModeReturn")
		}
		
		// Test with mock writer that returns error
		ctx.Writer = &errorWriter{rr, true}
		
		err := ctx.JSON(200, map[string]string{"test": "data"})
		if err == nil {
			t.Error("Expected error from JSON with errorWriter")
		}
		
		if respErr, ok := err.(*ResponseError); ok {
			if respErr.Operation != "JSON" {
				t.Errorf("Expected operation 'JSON', got '%s'", respErr.Operation)
			}
		} else {
			t.Error("Expected ResponseError type")
		}
	})
	
	t.Run("ErrorModeLog", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		ctx := NewContext(rr, req)
		
		ctx.SetErrorHandlingMode(ErrorModeLog)
		
		// Test with mock writer that returns error
		ctx.Writer = &errorWriter{rr, true}
		
		err := ctx.JSON(200, map[string]string{"test": "data"})
		if err != nil {
			t.Error("Expected no error in ErrorModeLog, errors should be logged only")
		}
	})
	
	t.Run("ErrorModeSilent", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		ctx := NewContext(rr, req)
		
		ctx.SetErrorHandlingMode(ErrorModeSilent)
		
		// Test with mock writer that returns error
		ctx.Writer = &errorWriter{rr, true}
		
		err := ctx.JSON(200, map[string]string{"test": "data"})
		if err != nil {
			t.Error("Expected no error in ErrorModeSilent")
		}
	})
	
	t.Run("ErrorModePanic", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		ctx := NewContext(rr, req)
		
		ctx.SetErrorHandlingMode(ErrorModePanic)
		
		// Test with mock writer that returns error
		ctx.Writer = &errorWriter{rr, true}
		
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic in ErrorModePanic")
			}
		}()
		
		_ = ctx.JSON(200, map[string]string{"test": "data"})
	})
}

func TestCustomErrorHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	ctx := NewContext(rr, req)
	
	var capturedError error
	ctx.OnResponseError(func(err error) {
		capturedError = err
	})
	
	// Test with mock writer that returns error
	ctx.Writer = &errorWriter{rr, true}
	
	err := ctx.JSON(200, map[string]string{"test": "data"})
	if err == nil {
		t.Error("Expected error from JSON with errorWriter")
	}
	
	if capturedError == nil {
		t.Error("Expected custom error handler to be called")
	}
	
	if respErr, ok := capturedError.(*ResponseError); ok {
		if respErr.Operation != "JSON" {
			t.Errorf("Expected operation 'JSON' in custom handler, got '%s'", respErr.Operation)
		}
	} else {
		t.Error("Expected ResponseError type in custom handler")
	}
}

func TestStatusChaining(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	ctx := NewContext(rr, req)
	
	// Test that Status returns *Context for chaining
	result := ctx.Status(404)
	if result != ctx {
		t.Error("Status should return *Context for chaining")
	}
	
	// Create new context for clean test
	req2 := httptest.NewRequest("GET", "/", nil)
	rr2 := httptest.NewRecorder()
	ctx2 := NewContext(rr2, req2)
	
	// Test chaining with JSON (Status will be called again in JSON)
	err := ctx2.Status(200).JSON(200, map[string]string{"message": "success"})
	if err != nil {
		t.Errorf("Chaining Status().JSON() should work: %v", err)
	}
	
	if rr2.Code != 200 {
		t.Errorf("Expected status 200, got %d", rr2.Code)
	}
	
	if !strings.Contains(rr2.Body.String(), "success") {
		t.Error("Expected JSON response to contain 'success'")
	}
}

func TestResponseError(t *testing.T) {
	err := &ResponseError{
		Type:      WriteError,
		Operation: "JSON",
		Err:       errors.New("mock error"),
		Recovered: false,
	}
	
	expected := "write error in JSON: mock error"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
	
	if err.Unwrap() != err.Err {
		t.Error("Unwrap should return underlying error")
	}
	
	// Test recovered error
	err.Recovered = true
	expected = "recovered write error in JSON: mock error"
	if err.Error() != expected {
		t.Errorf("Expected recovered error message '%s', got '%s'", expected, err.Error())
	}
}

func TestWriteMethodsReturnErrors(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	ctx := NewContext(rr, req)
	
	// Test with error writer
	ctx.Writer = &errorWriter{rr, true}
	
	tests := []struct {
		name string
		fn   func() error
	}{
		{"JSON", func() error { return ctx.JSON(200, map[string]string{"test": "data"}) }},
		{"Text", func() error { return ctx.Text(200, "test") }},
		{"Data", func() error { return ctx.Data(200, "text/plain", []byte("test")) }},
		{"Abort", func() error { return ctx.Abort(500, "test error") }},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.fn()
			if err == nil {
				t.Errorf("Expected error from %s with errorWriter", test.name)
			}
			
			if respErr, ok := err.(*ResponseError); ok {
				if respErr.Operation != test.name {
					t.Errorf("Expected operation '%s', got '%s'", test.name, respErr.Operation)
				}
			} else {
				t.Errorf("Expected ResponseError type for %s", test.name)
			}
		})
	}
}

func TestHeaderMethodsWork(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	ctx := NewContext(rr, req)
	
	// These methods primarily set headers and should succeed with normal writer
	tests := []struct {
		name string
		fn   func() error
	}{
		{"SetHeader", func() error { return ctx.SetHeader("X-Test", "value") }},
		{"Location", func() error { return ctx.Location("/test") }},
		{"Type", func() error { return ctx.Type("json") }},
		{"Vary", func() error { return ctx.Vary("Accept-Encoding") }},
		{"Append", func() error { return ctx.Append("X-Test", "value1", "value2") }},
		{"Attachment", func() error { return ctx.Attachment("test.txt") }},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.fn()
			if err != nil {
				t.Errorf("Expected no error from %s with normal writer, got: %v", test.name, err)
			}
		})
	}
}

func TestSendMethods(t *testing.T) {
	t.Run("Send with no data", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		ctx := NewContext(rr, req)
		
		err := ctx.Send(404)
		if err != nil {
			t.Errorf("Send with status only should not error: %v", err)
		}
		
		if rr.Code != 404 {
			t.Errorf("Expected status 404, got %d", rr.Code)
		}
	})
	
	t.Run("Send with string", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		ctx := NewContext(rr, req)
		
		err := ctx.Send(200, "Hello World")
		if err != nil {
			t.Errorf("Send with string should not error: %v", err)
		}
		
		if rr.Body.String() != "Hello World" {
			t.Errorf("Expected 'Hello World', got '%s'", rr.Body.String())
		}
	})
	
	t.Run("Send with JSON", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		ctx := NewContext(rr, req)
		
		err := ctx.Send(200, map[string]string{"message": "success"})
		if err != nil {
			t.Errorf("Send with JSON should not error: %v", err)
		}
		
		if !strings.Contains(rr.Body.String(), "success") {
			t.Error("Expected JSON response to contain 'success'")
		}
	})
	
	t.Run("Send with bytes", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		ctx := NewContext(rr, req)
		
		data := []byte("binary data")
		err := ctx.Send(200, data)
		if err != nil {
			t.Errorf("Send with bytes should not error: %v", err)
		}
		
		if rr.Body.String() != "binary data" {
			t.Errorf("Expected 'binary data', got '%s'", rr.Body.String())
		}
	})
}

// Mock writer that returns errors
type errorWriter struct {
	*httptest.ResponseRecorder
	shouldError bool
}

func (e *errorWriter) Write(b []byte) (int, error) {
	if e.shouldError {
		return 0, errors.New("mock write error")
	}
	return e.ResponseRecorder.Write(b)
}

func (e *errorWriter) WriteHeader(statusCode int) {
	// WriteHeader typically doesn't error in real scenarios,
	// so we'll let it succeed for most tests
	e.ResponseRecorder.WriteHeader(statusCode)
}

func (e *errorWriter) Header() http.Header {
	// For header operations that might not cause actual errors,
	// we'll simulate by checking if we should panic
	if e.shouldError {
		// Simulate an error condition in Header operations
		header := make(http.Header)
		return header // Return a basic header that won't cause write errors immediately
	}
	return e.ResponseRecorder.Header()
}

