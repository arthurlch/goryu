package middleware_test

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/arthurlch/goryu/pkg/context"
	"github.com/arthurlch/goryu/pkg/middleware"
)

// Helper to capture log output
func captureLogOutput(fn func()) string {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	fn()
	log.SetOutput(os.Stderr) // Reset to default
	return buf.String()
}

// To safely swap log output in concurrent tests if needed
var logMutex sync.Mutex

func TestLoggerMiddleware(t *testing.T) {
	loggerMiddleware := middleware.Logger()
	dummyHandler := func(c *context.Context) {
		c.Text(http.StatusAccepted, "Logged!") // StatusAccepted and some body to check size
	}

	req, _ := http.NewRequest("GET", "/logtest", nil)
	req.RemoteAddr = "127.0.0.1:12345" // Mock RemoteAddr
	req.Header.Set("User-Agent", "TestLoggerAgent")

	rr := httptest.NewRecorder()
	ctx := context.New(rr, req)

	handlerToTest := loggerMiddleware(dummyHandler)

	// Capture log output
	logMutex.Lock()
	defer logMutex.Unlock() // Ensure unlock even on panic

	var logOutput string
	// Store original logger flags and output
	originalFlags := log.Flags()
	originalOutput := log.Writer()

	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0) // Remove timestamp for predictable output in tests

	handlerToTest(ctx)

	log.SetOutput(originalOutput) // Restore original logger output
	log.SetFlags(originalFlags)   // Restore original logger flags

	logOutput = buf.String()
	// End capture

	if rr.Code != http.StatusAccepted {
		t.Errorf("Expected status %d from dummy handler, got %d", http.StatusAccepted, rr.Code)
	}

	// Check if the log output contains expected elements
	// This is a basic check; you might want to parse the log line for more specific assertions
	if !strings.Contains(logOutput, "method=GET") {
		t.Errorf("Log output missing 'method=GET'. Got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "path=\"/logtest\"") {
		t.Errorf("Log output missing 'path=\"/logtest\"'. Got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "status=202") { // http.StatusAccepted
		t.Errorf("Log output missing 'status=202'. Got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "remote_addr=\"127.0.0.1:12345\"") {
		t.Errorf("Log output missing 'remote_addr=\"127.0.0.1:12345\"'. Got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "user_agent=\"TestLoggerAgent\"") {
		t.Errorf("Log output missing 'user_agent=\"TestLoggerAgent\"'. Got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "size=7") { // "Logged!" is 7 bytes
		t.Errorf("Log output missing 'size=7'. Got: %s", logOutput)
	}

}