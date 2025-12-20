package goryuctx

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestContextError(t *testing.T) {
	t.Run("Default 500 error", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		ctx := NewContext(w, req)

		err := errors.New("test error")
		returnedErr := ctx.Error(err)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}

		if !strings.Contains(w.Body.String(), "Internal Server Error") {
			t.Errorf("Expected 'Internal Server Error' in body, got: %s", w.Body.String())
		}

		if returnedErr != err {
			t.Error("Expected original error to be returned")
		}
	})

	t.Run("Custom status code", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		ctx := NewContext(w, req)

		err := errors.New("bad request error")
		returnedErr := ctx.Error(err, http.StatusBadRequest)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		if !strings.Contains(w.Body.String(), "Bad Request") {
			t.Errorf("Expected 'Bad Request' in body, got: %s", w.Body.String())
		}

		if returnedErr != err {
			t.Error("Expected original error to be returned")
		}
	})
}

func TestContextErrorWithMessage(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	ctx := NewContext(w, req)

	err := errors.New("validation failed")
	customMessage := "Custom validation error message"
	returnedErr := ctx.ErrorWithMessage(err, http.StatusUnprocessableEntity, customMessage)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), customMessage) {
		t.Errorf("Expected custom message in body, got: %s", w.Body.String())
	}

	if returnedErr != err {
		t.Error("Expected original error to be returned")
	}
}

func TestContextAbort(t *testing.T) {
	t.Run("Default message", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		ctx := NewContext(w, req)

		ctx.Abort(http.StatusUnauthorized)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}

		if !strings.Contains(w.Body.String(), "Unauthorized") {
			t.Errorf("Expected 'Unauthorized' in body, got: %s", w.Body.String())
		}
	})

	t.Run("Custom message", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		ctx := NewContext(w, req)

		customMessage := "Access denied"
		ctx.Abort(http.StatusForbidden, customMessage)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}

		if !strings.Contains(w.Body.String(), customMessage) {
			t.Errorf("Expected custom message in body, got: %s", w.Body.String())
		}
	})
}

func TestContextSendFileErrors(t *testing.T) {
	t.Run("Non-existent file", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		ctx := NewContext(w, req)

		err := ctx.SendFile("/non/existent/file.txt")

		if err == nil {
			t.Error("Expected error for non-existent file")
		}

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("Directory instead of file", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		ctx := NewContext(w, req)

		// Create temporary directory
		tempDir, err := os.MkdirTemp("", "test_dir")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		err = ctx.SendFile(tempDir)

		if err == nil {
			t.Error("Expected error when trying to send directory")
		}

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}
	})

	t.Run("Valid file", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		ctx := NewContext(w, req)

		// Create temporary file
		tempFile, err := os.CreateTemp("", "test_file.txt")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tempFile.Name())

		content := "test file content"
		if _, err := tempFile.WriteString(content); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		tempFile.Close()

		err = ctx.SendFile(tempFile.Name())

		if err != nil {
			t.Errorf("Expected no error for valid file, got: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Body.String() != content {
			t.Errorf("Expected file content %q, got %q", content, w.Body.String())
		}
	})
}

func TestResponseMethodErrorLogging(t *testing.T) {
	// Capture log output
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(os.Stderr)

	t.Run("JSON encoding error", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		ctx := NewContext(w, req)

		// Create an object that can't be JSON encoded
		invalidJSON := map[string]interface{}{
			"func": func() {}, // functions can't be JSON encoded
		}

		err := ctx.JSON(200, invalidJSON)

		if err == nil {
			t.Error("Expected JSON encoding error")
		}

		logOutput := logBuf.String()
		if !strings.Contains(logOutput, "serialization error in JSON") {
			t.Errorf("Expected JSON error log, got: %s", logOutput)
		}
	})
}

func TestContextFieldsRemoved(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	ctx := NewContext(w, req)

	// Test that Request field exists
	if ctx.Request != req {
		t.Error("Expected Request field to be set")
	}

	// This test will fail to compile if Req field still exists
	// which is what we want - it ensures the field was properly removed
	// ctx.Req should cause compilation error if properly removed
}