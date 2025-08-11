package middleware

import (
	"expvar"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExpvarMiddleware(t *testing.T) {
	requestCounter := expvar.NewInt("test_requests")
	requestCounter.Set(42)

	debugPath := "/debug/metrics"

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte("Not a debug path")); err != nil {
			t.Fatalf("Failed to write response in dummy handler: %v", err)
		}
	})

	expvarWrapper := ExpvarMiddleware(debugPath)
	handler := expvarWrapper(nextHandler)

	t.Run("serves expvar on matching path", func(t *testing.T) {
		req := httptest.NewRequest("GET", debugPath, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		body, _ := io.ReadAll(rr.Body)
		if !strings.Contains(string(body), `"test_requests": 42`) {
			t.Errorf("expvar body does not contain expected metric. Body: %s", string(body))
		}
	})

	t.Run("passes through on non-matching path", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/some/other/path", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}

		body, _ := io.ReadAll(rr.Body)
		if string(body) != "Not a debug path" {
			t.Errorf("handler did not call the next handler correctly. Body: %s", string(body))
		}
	})
}
