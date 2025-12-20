package errors

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arthurlch/goryu"
)
func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name         string
		errorFunc    func() *AppError
		expectedCode string
		expectedMsg  string
		expectedStatus int
	}{
		{
			name:         "BadRequest",
			errorFunc:    func() *AppError { return BadRequest("Invalid input") },
			expectedCode: "BAD_REQUEST",
			expectedMsg:  "Invalid input",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "Unauthorized",
			errorFunc:    func() *AppError { return Unauthorized("Please login") },
			expectedCode: "UNAUTHORIZED",
			expectedMsg:  "Please login",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:         "NotFound",
			errorFunc:    func() *AppError { return NotFound("user") },
			expectedCode: "NOT_FOUND",
			expectedMsg:  "user not found",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:         "InternalError",
			errorFunc:    func() *AppError { return InternalError(errors.New("db error")) },
			expectedCode: "INTERNAL_ERROR",
			expectedMsg:  "An internal error occurred",
			expectedStatus: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errorFunc()
			if err.Code != tt.expectedCode {
				t.Errorf("Expected code %s, got %s", tt.expectedCode, err.Code)
			}
			if err.Message != tt.expectedMsg {
				t.Errorf("Expected message %s, got %s", tt.expectedMsg, err.Message)
			}
			if err.Status != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, err.Status)
			}
		})
	}
}
func TestErrorBuilder(t *testing.T) {
	err := NewError("CUSTOM_ERROR", "Custom error message").
		Status(http.StatusTeapot).
		Detail("field", "value").
		Detail("another", 123).
		Internal(errors.New("internal error")).
		RequestID("req-123").
		Source("test.go:42").
		Build()
	if err.Code != "CUSTOM_ERROR" {
		t.Errorf("Expected code CUSTOM_ERROR, got %s", err.Code)
	}
	if err.Status != http.StatusTeapot {
		t.Errorf("Expected status %d, got %d", http.StatusTeapot, err.Status)
	}
	if err.Details["field"] != "value" {
		t.Errorf("Expected detail field=value, got %v", err.Details["field"])
	}
	if err.Details["another"] != 123 {
		t.Errorf("Expected detail another=123, got %v", err.Details["another"])
	}
	if err.RequestID != "req-123" {
		t.Errorf("Expected request ID req-123, got %s", err.RequestID)
	}
	if err.Source != "test.go:42" {
		t.Errorf("Expected source test.go:42, got %s", err.Source)
	}
}
func TestErrorMiddleware(t *testing.T) {
	app := goryu.New(goryu.Config{
		DisableStartupMessage: true,
	})
	app.Use(New(Config{
		ShowDetails:    true,
		ShowStackTrace: false,
		LogErrors:      false, 
		DevMode:        false,
	}))
	app.GET("/error", Handle(func(c *goryu.Ctx) error {
		return BadRequest("Test error")
	}))
	app.GET("/panic", func(c *goryu.Ctx) {
		panic("test panic")
	})
	app.GET("/validation", Handle(func(c *goryu.Ctx) error {
		return ValidationError("email", "Invalid email format")
	}))
	t.Run("ErrorHandler", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/error", nil)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		errorData := response["error"].(map[string]interface{})
		if errorData["code"] != "BAD_REQUEST" {
			t.Errorf("Expected error code BAD_REQUEST, got %v", errorData["code"])
		}
		if errorData["message"] != "Test error" {
			t.Errorf("Expected error message 'Test error', got %v", errorData["message"])
		}
	})
	t.Run("PanicHandler", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/panic", nil)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		errorData := response["error"].(map[string]interface{})
		if errorData["code"] != "PANIC" {
			t.Errorf("Expected error code PANIC, got %v", errorData["code"])
		}
	})
	t.Run("ValidationError", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/validation", nil)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		errorData := response["error"].(map[string]interface{})
		if errorData["code"] != "VALIDATION_ERROR" {
			t.Errorf("Expected error code VALIDATION_ERROR, got %v", errorData["code"])
		}
		details := errorData["details"].(map[string]interface{})
		if details["field"] != "email" {
			t.Errorf("Expected field=email, got %v", details["field"])
		}
		if details["error"] != "Invalid email format" {
			t.Errorf("Expected error='Invalid email format', got %v", details["error"])
		}
	})
}
func TestContextError(t *testing.T) {
	app := goryu.New(goryu.Config{
		DisableStartupMessage: true,
	})
	app.Use(New())
	app.GET("/bad-request", func(c *goryu.Ctx) {
		Error(c).BadRequest("Bad request test")
	})
	app.GET("/unauthorized", func(c *goryu.Ctx) {
		Error(c).Unauthorized("Unauthorized test")
	})
	app.GET("/not-found", func(c *goryu.Ctx) {
		Error(c).NotFound("resource")
	})
	app.GET("/custom", func(c *goryu.Ctx) {
		Error(c).Custom("CUSTOM_CODE", "Custom message", http.StatusTeapot)
	})
	tests := []struct {
		path           string
		expectedStatus int
		expectedCode   string
	}{
		{"/bad-request", http.StatusBadRequest, "BAD_REQUEST"},
		{"/unauthorized", http.StatusUnauthorized, "UNAUTHORIZED"},
		{"/not-found", http.StatusNotFound, "NOT_FOUND"},
		{"/custom", http.StatusTeapot, "CUSTOM_CODE"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}
			errorData := response["error"].(map[string]interface{})
			if errorData["code"] != tt.expectedCode {
				t.Errorf("Expected error code %s, got %v", tt.expectedCode, errorData["code"])
			}
		})
	}
}
func TestErrorChain(t *testing.T) {
	app := goryu.New(goryu.Config{
		DisableStartupMessage: true,
	})
	app.Use(New())
	app.GET("/chain-success", func(c *goryu.Ctx) {
		chain := NewChain(c).
			Do(func() error {
				return nil
			}).
			DoWithResult(func() (interface{}, error) {
				return "success", nil
			})
		chain.OnSuccess(func() {
			result, _ := chain.Result()
			c.JSON(200, map[string]interface{}{
				"result": result,
			})
		})
		chain.SendError("CHAIN_ERROR", "Chain operation failed")
	})
	app.GET("/chain-error", func(c *goryu.Ctx) {
		chain := NewChain(c).
			Do(func() error {
				return errors.New("operation failed")
			}).
			OnError(func(err error) {
			})
		if chain.SendError("CHAIN_ERROR", "Chain operation failed") {
			return
		}
		c.JSON(200, map[string]string{"status": "ok"})
	})
	t.Run("ChainSuccess", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/chain-success", nil)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		if response["result"] != "success" {
			t.Errorf("Expected result=success, got %v", response["result"])
		}
	})
	t.Run("ChainError", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/chain-error", nil)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		errorData := response["error"].(map[string]interface{})
		if errorData["code"] != "CHAIN_ERROR" {
			t.Errorf("Expected error code CHAIN_ERROR, got %v", errorData["code"])
		}
	})
}
func TestWrapFunctions(t *testing.T) {
	t.Run("Wrap", func(t *testing.T) {
		if err := Wrap(nil, "CODE", "message"); err != nil {
			t.Error("Expected nil when wrapping nil error")
		}
		originalErr := errors.New("original error")
		wrapped := Wrap(originalErr, "WRAPPED", "Wrapped error")
		if wrapped.Code != "WRAPPED" {
			t.Errorf("Expected code WRAPPED, got %s", wrapped.Code)
		}
		if wrapped.Internal != originalErr {
			t.Error("Expected internal error to be preserved")
		}
		appErr := BadRequest("test")
		if wrapped := Wrap(appErr, "OTHER", "other"); wrapped != appErr {
			t.Error("Expected original AppError to be returned")
		}
	})
	t.Run("WrapWithStatus", func(t *testing.T) {
		err := errors.New("test error")
		wrapped := WrapWithStatus(err, http.StatusConflict, "CONFLICT_ERROR", "Conflict occurred")
		if wrapped.Status != http.StatusConflict {
			t.Errorf("Expected status %d, got %d", http.StatusConflict, wrapped.Status)
		}
		if wrapped.Code != "CONFLICT_ERROR" {
			t.Errorf("Expected code CONFLICT_ERROR, got %s", wrapped.Code)
		}
	})
}
func TestHelperFunctions(t *testing.T) {
	t.Run("Must", func(t *testing.T) {
		Must(nil) 
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic from Must with error")
			}
		}()
		Must(errors.New("test error"))
	})
	t.Run("MustGet", func(t *testing.T) {
		value := MustGet("success", nil)
		if value != "success" {
			t.Errorf("Expected 'success', got %v", value)
		}
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic from MustGet with error")
			}
		}()
		MustGet("value", errors.New("test error"))
	})
	t.Run("Try", func(t *testing.T) {
		err := Try(func() error {
			return nil
		})
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
		err = Try(func() error {
			panic("test panic")
		})
		if err == nil {
			t.Error("Expected error from Try with panic")
		}
		if err.Error() != "panic: test panic" {
			t.Errorf("Expected 'panic: test panic', got %v", err.Error())
		}
	})
	t.Run("TryGet", func(t *testing.T) {
		value, err := TryGet(func() (string, error) {
			return "success", nil
		})
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
		if value != "success" {
			t.Errorf("Expected 'success', got %v", value)
		}
		value, err = TryGet(func() (string, error) {
			panic("test panic")
		})
		if err == nil {
			t.Error("Expected error from TryGet with panic")
		}
		if value != "" {
			t.Errorf("Expected empty value, got %v", value)
		}
	})
}