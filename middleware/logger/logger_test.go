package logger_test

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/logger"
)

var logMutex sync.Mutex

func TestLoggerMiddleware(t *testing.T) {
	loggerMiddleware := logger.New()

	dummyHandler := func(c *context.Context) {
		c.Text(http.StatusAccepted, "Logged!")
	}

	req, _ := http.NewRequest("GET", "/logtest", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("User-Agent", "TestLoggerAgent")

	rr := httptest.NewRecorder()
	ctx := context.NewContext(rr, req)

	handlerToTest := loggerMiddleware(dummyHandler)

	logMutex.Lock()
	defer logMutex.Unlock()

	var buf bytes.Buffer
	originalFlags := log.Flags()
	originalOutput := log.Writer()

	log.SetOutput(&buf)
	log.SetFlags(0)

	handlerToTest(ctx)

	log.SetOutput(originalOutput)
	log.SetFlags(originalFlags)
	logOutput := buf.String()

	if rr.Code != http.StatusAccepted {
		t.Errorf("Expected status %d from dummy handler, got %d", http.StatusAccepted, rr.Code)
	}
	if !strings.Contains(logOutput, "method=GET") {
		t.Errorf("Log output missing 'method=GET'. Got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "path=\"/logtest\"") {
		t.Errorf("Log output missing 'path=\"/logtest\"'. Got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "status=202") {
		t.Errorf("Log output missing 'status=202'. Got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "user_agent=\"TestLoggerAgent\"") {
		t.Errorf("Log output missing 'user_agent=\"TestLoggerAgent\"'. Got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "size=7") {
		t.Errorf("Log output missing 'size=7'. Got: %s", logOutput)
	}
}
