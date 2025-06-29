package logger_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/logger"
)

func newTestContext(req *http.Request) (*goryu.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}

func TestLoggerMiddleware(t *testing.T) {
	dummyHandler := func(c *goryu.Context) {
		_ = c.Text(http.StatusOK, "OK")
	}

	t.Run("Default Logger", func(t *testing.T) {
		var buf bytes.Buffer
		middleware := logger.New(logger.Config{
			Output: &buf,
		})

		req := httptest.NewRequest("GET", "/test", nil)
		ctx, _ := newTestContext(req)

		handlerToTest := middleware(dummyHandler)
		handlerToTest(ctx)

		logOutput := buf.String()

		if !strings.Contains(logOutput, "\033[32m200\033[0m") {
			t.Errorf("log output does not contain colored status code 200. Got: %s", logOutput)
		}
		if !strings.Contains(logOutput, "\033[34mGET\033[0m /test") {
			t.Errorf("log output does not contain colored method and path. Got: %s", logOutput)
		}
	})

	t.Run("Custom Format with Colors Disabled", func(t *testing.T) {
		var buf bytes.Buffer
		middleware := logger.New(logger.Config{
			Output:        &buf,
			DisableColors: true,
			Format:        "METHOD=${method} PATH=${path} STATUS=${status}",
		})

		req := httptest.NewRequest("POST", "/custom", nil)
		ctx, _ := newTestContext(req)

		handlerToTest := middleware(dummyHandler)
		handlerToTest(ctx)

		logOutput := strings.TrimSpace(buf.String())
		expected := "METHOD=POST PATH=/custom STATUS=200"

		if logOutput != expected {
			t.Errorf("expected log output '%s', got '%s'", expected, logOutput)
		}
		if strings.Contains(logOutput, "\033") {
			t.Errorf("log output should not contain color codes. Got: %s", logOutput)
		}
	})

	t.Run("Error Logging", func(t *testing.T) {
		var buf bytes.Buffer
		middleware := logger.New(logger.Config{
			Output:        &buf,
			DisableColors: true,
			Format:        "STATUS=${status} ERROR='${error}'",
		})

		errorHandler := func(c *goryu.Context) {
			err := errors.New("database connection failed")
			c.Set("error", err)
			c.Status(http.StatusInternalServerError)
		}

		req := httptest.NewRequest("GET", "/db-error", nil)
		ctx, _ := newTestContext(req)

		handlerToTest := middleware(errorHandler)
		handlerToTest(ctx)

		logOutput := strings.TrimSpace(buf.String())
		expected := "STATUS=500 ERROR='database connection failed'"

		if logOutput != expected {
			t.Errorf("expected log output '%s', got '%s'", expected, logOutput)
		}
	})

	t.Run("Skip Logging with Next", func(t *testing.T) {
		var buf bytes.Buffer
		middleware := logger.New(logger.Config{
			Output: &buf,
			Next: func(c *goryu.Context) bool {
				return strings.HasPrefix(c.Request.URL.Path, "/health")
			},
		})

		req1 := httptest.NewRequest("GET", "/api/v1/users", nil)
		ctx1, _ := newTestContext(req1)
		middleware(dummyHandler)(ctx1)

		if buf.Len() == 0 {
			t.Error("expected /api/v1/users to be logged, but output is empty")
		}

		buf.Reset()
		req2 := httptest.NewRequest("GET", "/healthz", nil)
		ctx2, _ := newTestContext(req2)
		middleware(dummyHandler)(ctx2)

		if buf.Len() > 0 {
			t.Errorf("expected /healthz to be skipped, but got log output: %s", buf.String())
		}
	})

	t.Run("Request ID", func(t *testing.T) {
		var buf bytes.Buffer
		middleware := logger.New(logger.Config{
			Output:        &buf,
			DisableColors: true,
			Format:        "ID=${request_id}",
		})

		req1 := httptest.NewRequest("GET", "/", nil)
		req1.Header.Set("X-Request-ID", "my-custom-id-123")
		ctx1, _ := newTestContext(req1)
		middleware(dummyHandler)(ctx1)

		if !strings.Contains(buf.String(), "ID=my-custom-id-123") {
			t.Errorf("log did not use existing X-Request-ID. Got: %s", buf.String())
		}

		buf.Reset()
		req2 := httptest.NewRequest("GET", "/", nil)
		ctx2, _ := newTestContext(req2)
		middleware(dummyHandler)(ctx2)

		logOutput := buf.String()
		re := regexp.MustCompile(`ID=([a-f0-9]{32})`)
		if !re.MatchString(logOutput) {
			t.Errorf("log did not contain a generated 32-char hex request ID. Got: %s", logOutput)
		}
	})
}
