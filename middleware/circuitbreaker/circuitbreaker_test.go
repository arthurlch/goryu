package circuitbreaker_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/circuitbreaker"
)

func newTestContext(req *http.Request) (*goryu.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}

func TestCircuitBreakerMiddleware(t *testing.T) {
	t.Run("SuccessfulRequests", func(t *testing.T) {
		config := circuitbreaker.Config{
			Name: "test",
		}
		middleware := circuitbreaker.New(config)

		handler := func(c *goryu.Context) {
			c.Text(http.StatusOK, "Success")
		}

		// Send several successful requests
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			ctx, rr := newTestContext(req)

			middleware(handler)(ctx)

			if rr.Code != http.StatusOK {
				t.Errorf("Request %d: Expected status 200, got %d", i, rr.Code)
			}
		}
	})

	t.Run("FailingRequestsOpenCircuit", func(t *testing.T) {
		config := circuitbreaker.Config{
			Name:        "test",
			Timeout:     100 * time.Millisecond,
			MaxRequests: 1,
			ReadyToTrip: func(counts circuitbreaker.Counts) bool {
				return counts.ConsecutiveFailures >= 3
			},
		}
		middleware := circuitbreaker.New(config)

		handler := func(c *goryu.Context) {
			c.Text(http.StatusInternalServerError, "Server Error")
		}

		// Send failing requests to trip the circuit
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			ctx, rr := newTestContext(req)

			middleware(handler)(ctx)

			if rr.Code != http.StatusInternalServerError {
				t.Errorf("Request %d: Expected status 500, got %d", i, rr.Code)
			}
		}

		// Next request should be blocked by open circuit
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 503 (circuit open), got %d", rr.Code)
		}
	})

	t.Run("HalfOpenRecovery", func(t *testing.T) {
		config := circuitbreaker.Config{
			Name:        "test",
			Timeout:     50 * time.Millisecond,
			MaxRequests: 2,
			ReadyToTrip: func(counts circuitbreaker.Counts) bool {
				return counts.ConsecutiveFailures >= 2
			},
		}
		middleware := circuitbreaker.New(config)

		failingHandler := func(c *goryu.Context) {
			c.Text(http.StatusInternalServerError, "Server Error")
		}

		successHandler := func(c *goryu.Context) {
			c.Text(http.StatusOK, "Success")
		}

		// Trip the circuit with failing requests
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			ctx, _ := newTestContext(req)
			middleware(failingHandler)(ctx)
		}

		// Wait for circuit to transition to half-open
		time.Sleep(60 * time.Millisecond)

		// Send successful request to close the circuit
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)

		middleware(successHandler)(ctx)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 (half-open success), got %d", rr.Code)
		}

		// Circuit should now be closed, allowing more requests
		req2 := httptest.NewRequest("GET", "/", nil)
		ctx2, rr2 := newTestContext(req2)

		middleware(successHandler)(ctx2)

		if rr2.Code != http.StatusOK {
			t.Errorf("Expected status 200 (circuit closed), got %d", rr2.Code)
		}
	})

	t.Run("CustomFallbackHandler", func(t *testing.T) {
		config := circuitbreaker.Config{
			Name:        "test",
			Timeout:     100 * time.Millisecond,
			MaxRequests: 1,
			ReadyToTrip: func(counts circuitbreaker.Counts) bool {
				return counts.ConsecutiveFailures >= 1
			},
			FallbackHandler: func(c *goryu.Context) {
				c.Text(http.StatusTooManyRequests, "Custom fallback")
			},
		}
		middleware := circuitbreaker.New(config)

		handler := func(c *goryu.Context) {
			c.Text(http.StatusInternalServerError, "Server Error")
		}

		// Trip the circuit
		req1 := httptest.NewRequest("GET", "/", nil)
		ctx1, _ := newTestContext(req1)
		middleware(handler)(ctx1)

		// Next request should use custom fallback
		req2 := httptest.NewRequest("GET", "/", nil)
		ctx2, rr2 := newTestContext(req2)

		middleware(handler)(ctx2)

		if rr2.Code != http.StatusTooManyRequests {
			t.Errorf("Expected status 429, got %d", rr2.Code)
		}

		if rr2.Body.String() != "Custom fallback" {
			t.Errorf("Expected 'Custom fallback', got %s", rr2.Body.String())
		}
	})

	t.Run("SkipMiddleware", func(t *testing.T) {
		config := circuitbreaker.Config{
			Name: "test",
			Skip: func(c *goryu.Context) bool {
				return c.Request.URL.Path == "/skip"
			},
		}
		middleware := circuitbreaker.New(config)

		handler := func(c *goryu.Context) {
			c.Text(http.StatusInternalServerError, "Server Error")
		}

		req := httptest.NewRequest("GET", "/skip", nil)
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		// Should execute handler despite being a failing request
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", rr.Code)
		}
	})

	t.Run("StateCallbacks", func(t *testing.T) {
		var stateChanges []string

		config := circuitbreaker.Config{
			Name:        "test",
			Timeout:     50 * time.Millisecond,
			MaxRequests: 1,
			ReadyToTrip: func(counts circuitbreaker.Counts) bool {
				return counts.ConsecutiveFailures >= 2
			},
			OnStateChange: func(name string, from circuitbreaker.State, to circuitbreaker.State) {
				stateChanges = append(stateChanges, name+":"+from.String()+"->"+to.String())
			},
		}
		middleware := circuitbreaker.New(config)

		failingHandler := func(c *goryu.Context) {
			c.Text(http.StatusInternalServerError, "Server Error")
		}

		// Trip the circuit
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			ctx, _ := newTestContext(req)
			middleware(failingHandler)(ctx)
		}

		// Wait for half-open transition
		time.Sleep(60 * time.Millisecond)

		// Trigger half-open state
		req := httptest.NewRequest("GET", "/", nil)
		ctx, _ := newTestContext(req)
		middleware(failingHandler)(ctx)

		if len(stateChanges) == 0 {
			t.Error("Expected state change callbacks to be called")
		}
	})
}
