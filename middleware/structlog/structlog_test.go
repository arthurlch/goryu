package structlog_test

import (
	"bytes"
	"encoding/json"
	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
	"github.com/arthurlch/goryu/middleware/structlog"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestContext(req *http.Request) (*context.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}
func TestStructlogMiddleware(t *testing.T) {
	t.Run("BasicLogging", func(t *testing.T) {
		var buf bytes.Buffer
		config := structlog.Config{
			Output: &buf,
			Level:  slog.LevelInfo,
		}
		middleware := structlog.New(config)
		handler := func(c *context.Context) {
			c.Text(http.StatusOK, "Hello, World!")
		}
		req := httptest.NewRequest("GET", "/test", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		logOutput := buf.String()
		if logOutput == "" {
			t.Error("Expected log output, got empty string")
		}
		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(logOutput), &logEntry); err != nil {
			t.Fatalf("Failed to parse log output as JSON: %v", err)
		}
		expectedFields := []string{"time", "level", "msg", "request_id", "method", "path", "status_code"}
		for _, field := range expectedFields {
			if _, exists := logEntry[field]; !exists {
				t.Errorf("Expected field '%s' in log entry", field)
			}
		}
		if logEntry["method"] != "GET" {
			t.Errorf("Expected method 'GET', got %v", logEntry["method"])
		}
		if logEntry["path"] != "/test" {
			t.Errorf("Expected path '/test', got %v", logEntry["path"])
		}
		if logEntry["status_code"] != float64(200) {
			t.Errorf("Expected status_code 200, got %v", logEntry["status_code"])
		}
	})
	t.Run("RequestIDFromHeader", func(t *testing.T) {
		var buf bytes.Buffer
		config := structlog.Config{
			Output: &buf,
		}
		middleware := structlog.New(config)
		handler := func(c *context.Context) {
			requestID := structlog.GetRequestID(c)
			c.Text(http.StatusOK, "Request ID: "+requestID)
		}
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Request-ID", "test-request-123")
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if !strings.Contains(rr.Body.String(), "test-request-123") {
			t.Errorf("Expected response to contain request ID 'test-request-123'")
		}
		if rr.Header().Get("X-Request-ID") != "test-request-123" {
			t.Errorf("Expected X-Request-ID header to be 'test-request-123', got %s", rr.Header().Get("X-Request-ID"))
		}
	})
	t.Run("ErrorLevelForServerErrors", func(t *testing.T) {
		var buf bytes.Buffer
		config := structlog.Config{
			Output: &buf,
		}
		middleware := structlog.New(config)
		handler := func(c *context.Context) {
			c.Text(http.StatusInternalServerError, "Server Error")
		}
		req := httptest.NewRequest("GET", "/error", nil)
		ctx, _ := newTestContext(req)
		middleware(handler)(ctx)
		logOutput := buf.String()
		var logEntry map[string]interface{}
		_ = json.Unmarshal([]byte(logOutput), &logEntry)
		if logEntry["level"] != "ERROR" {
			t.Errorf("Expected log level 'ERROR', got %v", logEntry["level"])
		}
	})
	t.Run("WarnLevelForClientErrors", func(t *testing.T) {
		var buf bytes.Buffer
		config := structlog.Config{
			Output: &buf,
		}
		middleware := structlog.New(config)
		handler := func(c *context.Context) {
			c.Text(http.StatusBadRequest, "Bad Request")
		}
		req := httptest.NewRequest("GET", "/bad", nil)
		ctx, _ := newTestContext(req)
		middleware(handler)(ctx)
		logOutput := buf.String()
		var logEntry map[string]interface{}
		_ = json.Unmarshal([]byte(logOutput), &logEntry)
		if logEntry["level"] != "WARN" {
			t.Errorf("Expected log level 'WARN', got %v", logEntry["level"])
		}
	})
	t.Run("SkipLogging", func(t *testing.T) {
		var buf bytes.Buffer
		config := structlog.Config{
			Output: &buf,
			BaseConfig: base.BaseConfig{
				Skip: func(c *context.Context) bool {
					return c.Request.URL.Path == "/health"
				},
			},
		}
		middleware := structlog.New(config)
		handler := func(c *context.Context) {
			c.Text(http.StatusOK, "OK")
		}
		req := httptest.NewRequest("GET", "/health", nil)
		ctx, _ := newTestContext(req)
		middleware(handler)(ctx)
		logOutput := buf.String()
		if logOutput != "" {
			t.Errorf("Expected no log output for skipped request, got: %s", logOutput)
		}
	})
	t.Run("CustomFields", func(t *testing.T) {
		var buf bytes.Buffer
		config := structlog.Config{
			Output: &buf,
			CustomFields: func(c *context.Context) map[string]any {
				return map[string]any{
					"service": "test-service",
					"version": "1.0.0",
				}
			},
		}
		middleware := structlog.New(config)
		handler := func(c *context.Context) {
			c.Text(http.StatusOK, "OK")
		}
		req := httptest.NewRequest("GET", "/", nil)
		ctx, _ := newTestContext(req)
		middleware(handler)(ctx)
		logOutput := buf.String()
		var logEntry map[string]interface{}
		_ = json.Unmarshal([]byte(logOutput), &logEntry)
		customFields, ok := logEntry["custom"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected custom fields in log entry")
		}
		if customFields["service"] != "test-service" {
			t.Errorf("Expected custom field 'service' to be 'test-service', got %v", customFields["service"])
		}
		if customFields["version"] != "1.0.0" {
			t.Errorf("Expected custom field 'version' to be '1.0.0', got %v", customFields["version"])
		}
	})
}
func TestLogHelpers(t *testing.T) {
	t.Run("LogInfo", func(t *testing.T) {
		var buf bytes.Buffer
		config := structlog.Config{
			Output: &buf,
		}
		middleware := structlog.New(config)
		handler := func(c *context.Context) {
			structlog.LogInfo(c, "Processing user request",
				slog.String("user_id", "123"),
				slog.String("action", "create"),
			)
			c.Text(http.StatusOK, "OK")
		}
		req := httptest.NewRequest("GET", "/", nil)
		ctx, _ := newTestContext(req)
		middleware(handler)(ctx)
		logOutput := buf.String()
		lines := strings.Split(strings.TrimSpace(logOutput), "\n")
		if len(lines) < 2 {
			t.Fatal("Expected at least 2 log entries")
		}
		var infoEntry map[string]interface{}
		if err := json.Unmarshal([]byte(lines[0]), &infoEntry); err != nil {
			t.Fatalf("Failed to parse info log: %v", err)
		}
		if infoEntry["msg"] != "Processing user request" {
			t.Errorf("Expected message 'Processing user request', got %v", infoEntry["msg"])
		}
		if infoEntry["user_id"] != "123" {
			t.Errorf("Expected user_id '123', got %v", infoEntry["user_id"])
		}
	})
	t.Run("CorrelatedLogger", func(t *testing.T) {
		var buf bytes.Buffer
		config := structlog.Config{
			Output: &buf,
		}
		middleware := structlog.New(config)
		handler := func(c *context.Context) {
			logger := structlog.CorrelatedLogger(c)
			logger.Info("Using correlated logger")
			c.Text(http.StatusOK, "OK")
		}
		req := httptest.NewRequest("GET", "/", nil)
		ctx, _ := newTestContext(req)
		middleware(handler)(ctx)
		logOutput := buf.String()
		lines := strings.Split(strings.TrimSpace(logOutput), "\n")
		if len(lines) < 2 {
			t.Fatal("Expected at least 2 log entries")
		}
		var correlatedEntry map[string]interface{}
		if err := json.Unmarshal([]byte(lines[0]), &correlatedEntry); err != nil {
			t.Fatalf("Failed to parse correlated log: %v", err)
		}
		if correlatedEntry["msg"] != "Using correlated logger" {
			t.Errorf("Expected message 'Using correlated logger', got %v", correlatedEntry["msg"])
		}
		if correlatedEntry["request_id"] == nil {
			t.Error("Expected request_id in correlated log entry")
		}
	})
}
