package healthcheck_test

import (
	"encoding/json"
	goryuContext "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/healthcheck"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestContext(req *http.Request) (*goryuContext.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return goryuContext.NewContext(rr, req), rr
}
func TestHealthCheckMiddleware(t *testing.T) {
	t.Run("DefaultHealthEndpoint", func(t *testing.T) {
		middleware := healthcheck.New(healthcheck.Config{})
		handler := func(c *goryuContext.Context) {
			c.Text(http.StatusOK, "Should not reach here")
		}
		req := httptest.NewRequest("GET", "/health", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		var response healthcheck.HealthStatus
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		if response.Status != "UP" {
			t.Errorf("Expected status UP, got %s", response.Status)
		}
	})
	t.Run("LivenessCheck", func(t *testing.T) {
		config := healthcheck.Config{
			LivenessProbes: map[string]healthcheck.Probe{
				"always-up": healthcheck.AlwaysUpProbe(),
			},
		}
		middleware := healthcheck.New(config)
		handler := func(c *goryuContext.Context) {
			c.Text(http.StatusOK, "Should not reach here")
		}
		req := httptest.NewRequest("GET", "/health/live", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		var response healthcheck.HealthStatus
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		if response.Status != "UP" {
			t.Errorf("Expected status UP, got %s", response.Status)
		}
		if len(response.Checks) != 1 {
			t.Errorf("Expected 1 check, got %d", len(response.Checks))
		}
	})
	t.Run("ReadinessCheckFails", func(t *testing.T) {
		config := healthcheck.Config{
			ReadinessProbes: map[string]healthcheck.Probe{
				"always-down": healthcheck.AlwaysDownProbe("service not ready"),
			},
		}
		middleware := healthcheck.New(config)
		handler := func(c *goryuContext.Context) {
			c.Text(http.StatusOK, "Should not reach here")
		}
		req := httptest.NewRequest("GET", "/health/ready", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 503, got %d", rr.Code)
		}
		var response healthcheck.HealthStatus
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		if response.Status != "DOWN" {
			t.Errorf("Expected status DOWN, got %s", response.Status)
		}
		if len(response.Errors) != 1 {
			t.Errorf("Expected 1 error, got %d", len(response.Errors))
		}
	})
	t.Run("NonHealthRequest", func(t *testing.T) {
		middleware := healthcheck.New(healthcheck.Config{})
		handler := func(c *goryuContext.Context) {
			c.Text(http.StatusOK, "Homepage")
		}
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		if rr.Body.String() != "Homepage" {
			t.Errorf("Expected 'Homepage', got %s", rr.Body.String())
		}
	})
	t.Run("CustomPaths", func(t *testing.T) {
		config := healthcheck.Config{
			HealthPath:   "/status",
			LivenessPath: "/alive",
			HealthProbes: map[string]healthcheck.Probe{
				"test": healthcheck.AlwaysUpProbe(),
			},
		}
		middleware := healthcheck.New(config)
		handler := func(c *goryuContext.Context) {
			c.Text(http.StatusOK, "Should not reach here")
		}
		req := httptest.NewRequest("GET", "/status", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		req2 := httptest.NewRequest("GET", "/health", nil)
		ctx2, rr2 := newTestContext(req2)
		handler2 := func(c *goryuContext.Context) {
			c.Text(http.StatusOK, "Default handler")
		}
		middleware(handler2)(ctx2)
		if rr2.Body.String() != "Default handler" {
			t.Error("Expected default handler to be called for /health")
		}
	})
	t.Run("WithProbesHelper", func(t *testing.T) {
		livenessProbes := map[string]healthcheck.Probe{
			"liveness": healthcheck.AlwaysUpProbe(),
		}
		readinessProbes := map[string]healthcheck.Probe{
			"readiness": healthcheck.AlwaysUpProbe(),
		}
		middleware := healthcheck.WithProbes(livenessProbes, readinessProbes)
		handler := func(c *goryuContext.Context) {
			c.Text(http.StatusOK, "Should not reach here")
		}
		req := httptest.NewRequest("GET", "/health/live", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})
}
