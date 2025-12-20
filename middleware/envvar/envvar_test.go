package envvar_test
import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/envvar"
)
func newTestContext(req *http.Request) (*context.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}
func TestEnvvarMiddleware(t *testing.T) {
	os.Setenv("TEST_APP_VERSION", "v1.2.3")
	os.Setenv("TEST_DATABASE_URL", "secret-db-url")
	os.Setenv("TEST_LOG_LEVEL", "debug")
	defer func() {
		os.Unsetenv("TEST_APP_VERSION")
		os.Unsetenv("TEST_DATABASE_URL")
		os.Unsetenv("TEST_LOG_LEVEL")
	}()
	t.Run("DefaultEndpoint", func(t *testing.T) {
		middleware := envvar.New(envvar.Config{})
		handler := func(c *context.Context) {
			c.Text(http.StatusOK, "Should not reach here")
		}
		req := httptest.NewRequest("GET", "/envvar", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		var body map[string]string
		if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode json response: %v", err)
		}
		if _, exists := body["TEST_APP_VERSION"]; !exists {
			t.Error("TEST_APP_VERSION should be present in response")
		}
	})
	t.Run("ExposeOnlySpecified", func(t *testing.T) {
		config := envvar.Config{
			Path:   "/config",
			Expose: []string{"TEST_APP_VERSION", "TEST_LOG_LEVEL"},
		}
		middleware := envvar.New(config)
		handler := func(c *context.Context) {
			c.Text(http.StatusOK, "Should not reach here")
		}
		req := httptest.NewRequest("GET", "/config", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		var body map[string]string
		if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode json response: %v", err)
		}
		if body["TEST_APP_VERSION"] != "v1.2.3" {
			t.Errorf("Expected TEST_APP_VERSION=v1.2.3, got %s", body["TEST_APP_VERSION"])
		}
		if body["TEST_LOG_LEVEL"] != "debug" {
			t.Errorf("Expected TEST_LOG_LEVEL=debug, got %s", body["TEST_LOG_LEVEL"])
		}
		if _, exists := body["TEST_DATABASE_URL"]; exists {
			t.Error("TEST_DATABASE_URL should not be exposed")
		}
	})
	t.Run("ExcludeSpecified", func(t *testing.T) {
		config := envvar.Config{
			Path:    "/config",
			Exclude: []string{"TEST_DATABASE_URL"},
		}
		middleware := envvar.New(config)
		handler := func(c *context.Context) {
			c.Text(http.StatusOK, "Should not reach here")
		}
		req := httptest.NewRequest("GET", "/config", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		var body map[string]string
		if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode json response: %v", err)
		}
		if _, exists := body["TEST_DATABASE_URL"]; exists {
			t.Error("TEST_DATABASE_URL should have been excluded")
		}
		if _, exists := body["TEST_LOG_LEVEL"]; !exists {
			t.Error("TEST_LOG_LEVEL should have been included")
		}
	})
	t.Run("NonMatchingPath", func(t *testing.T) {
		config := envvar.Config{Path: "/config"}
		middleware := envvar.New(config)
		handler := func(c *context.Context) {
			c.Text(http.StatusTeapot, "Regular handler")
		}
		req := httptest.NewRequest("GET", "/not-config", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusTeapot {
			t.Errorf("Expected status 418, got %d", rr.Code)
		}
		if rr.Body.String() != "Regular handler" {
			t.Errorf("Expected 'Regular handler', got %s", rr.Body.String())
		}
	})
	t.Run("WithPathHelper", func(t *testing.T) {
		middleware := envvar.WithPath("/custom-env")
		handler := func(c *context.Context) {
			c.Text(http.StatusOK, "Should not reach here")
		}
		req := httptest.NewRequest("GET", "/custom-env", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		contentType := rr.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected content-type 'application/json', got %s", contentType)
		}
	})
	t.Run("WithExposeHelper", func(t *testing.T) {
		middleware := envvar.WithExpose([]string{"TEST_APP_VERSION"})
		handler := func(c *context.Context) {
			c.Text(http.StatusOK, "Should not reach here")
		}
		req := httptest.NewRequest("GET", "/envvar", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		var body map[string]string
		if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode json response: %v", err)
		}
		if len(body) != 1 {
			t.Errorf("Expected 1 env var, got %d", len(body))
		}
		if body["TEST_APP_VERSION"] != "v1.2.3" {
			t.Errorf("Expected TEST_APP_VERSION=v1.2.3, got %s", body["TEST_APP_VERSION"])
		}
	})
}