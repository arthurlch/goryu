package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnvvarMiddleware(t *testing.T) {
	t.Setenv("APP_VERSION", "v1.2.3")
	t.Setenv("DATABASE_URL", "secret-db-url")
	t.Setenv("LOG_LEVEL", "debug")

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	t.Run("exposes only specified variables", func(t *testing.T) {
		config := EnvvarConfig{
			Path:   "/config",
			Expose: []string{"APP_VERSION", "LOG_LEVEL"},
		}
		handler := EnvvarMiddleware(config)(nextHandler)

		req := httptest.NewRequest("GET", "/config", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var body map[string]string
		if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode json response: %v", err)
		}

		if len(body) != 2 {
			t.Errorf("expected 2 env vars, got %d", len(body))
		}
		if body["APP_VERSION"] != "v1.2.3" {
			t.Errorf("unexpected APP_VERSION: got %s", body["APP_VERSION"])
		}
		if _, exists := body["DATABASE_URL"]; exists {
			t.Error("DATABASE_URL should not be exposed")
		}
	})

	t.Run("excludes specified variables", func(t *testing.T) {
		config := EnvvarConfig{
			Path:    "/config",
			Exclude: []string{"DATABASE_URL"},
		}
		handler := EnvvarMiddleware(config)(nextHandler)

		req := httptest.NewRequest("GET", "/config", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		var body map[string]string
		if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode json response: %v", err)
		}

		if _, exists := body["DATABASE_URL"]; exists {
			t.Error("DATABASE_URL should have been excluded")
		}
		if _, exists := body["LOG_LEVEL"]; !exists {
			t.Error("LOG_LEVEL should have been included")
		}
	})

	t.Run("passes through on non-matching path", func(t *testing.T) {
		config := EnvvarConfig{Path: "/config"}
		handler := EnvvarMiddleware(config)(nextHandler)

		req := httptest.NewRequest("GET", "/not-config", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusTeapot {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusTeapot)
		}
	})
}
