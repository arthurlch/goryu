package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func probeThatSucceeds(ctx context.Context) error {
	return nil
}

func probeThatFails(ctx context.Context) error {
	return errors.New("dependency failed")
}

func probeThatTimesOut(ctx context.Context) error {
	time.Sleep(100 * time.Millisecond)
	return nil
}

func TestHealthChecker(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	t.Run("liveness success", func(t *testing.T) {
		t.Parallel()
		health := NewHealthChecker()
		health.AddLivenessCheck("passing-check", probeThatSucceeds)
		handler := health.Middleware(nextHandler)

		req := httptest.NewRequest("GET", "/live", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("wrong status code: got %v want %v", status, http.StatusOK)
		}
	})

	t.Run("readiness failure", func(t *testing.T) {
		t.Parallel()
		health := NewHealthChecker()
		health.AddReadinessCheck("failing-check", probeThatFails)
		handler := health.Middleware(nextHandler)

		req := httptest.NewRequest("GET", "/ready", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusServiceUnavailable {
			t.Errorf("wrong status code: got %v want %v", status, http.StatusServiceUnavailable)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode json: %v", err)
		}

		errorsMap, ok := body["errors"].(map[string]interface{})
		if !ok {
			t.Fatalf("Response 'errors' field is not a map")
		}
		if errorsMap["failing-check"] != "dependency failed" {
			t.Errorf("unexpected error message in json: got %v", errorsMap["failing-check"])
		}
	})

	t.Run("probe timeout", func(t *testing.T) {
		t.Parallel()
		health := NewHealthChecker()
		health.timeout = 50 * time.Millisecond // Set a short timeout
		health.AddLivenessCheck("slow-probe", probeThatTimesOut)
		handler := health.Middleware(nextHandler)

		req := httptest.NewRequest("GET", "/live", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusServiceUnavailable {
			t.Errorf("wrong status code: got %v want %v", status, http.StatusServiceUnavailable)
		}
	})

	t.Run("passthrough non-health path", func(t *testing.T) {
		t.Parallel()
		health := NewHealthChecker()
		handler := health.Middleware(nextHandler)

		req := httptest.NewRequest("GET", "/api/v1/data", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusAccepted {
			t.Errorf("handler did not pass through: got %v want %v", status, http.StatusAccepted)
		}
	})
}
