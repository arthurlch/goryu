package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireTLSMiddleware(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireTLSMiddleware(dummyHandler)

	t.Run("redirects http request with x-forwarded-proto", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		req.Header.Set("X-Forwarded-Proto", "http")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusMovedPermanently {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMovedPermanently)
		}

		expectedURL := "https://example.com/test"
		if location := rr.Header().Get("Location"); location != expectedURL {
			t.Errorf("handler redirected to wrong location: got %s want %s", location, expectedURL)
		}
	})

	t.Run("allows https request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "https://example.com/test", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})
}
