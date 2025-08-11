package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSRFMiddleware(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	handler := CSRFMiddleware(dummyHandler)

	t.Run("get request receives csrf token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		tokenHeader := rr.Header().Get(csrfTokenHeader)
		if tokenHeader == "" {
			t.Error("X-CSRF-Token header not set on GET request")
		}

		cookie := rr.Result().Header.Get("Set-Cookie")
		if !strings.Contains(cookie, csrfTokenCookie) {
			t.Error("csrf-token cookie not set on GET request")
		}
	})

	t.Run("post request with valid token succeeds", func(t *testing.T) {
		getReq := httptest.NewRequest("GET", "/", nil)
		getRR := httptest.NewRecorder()
		handler.ServeHTTP(getRR, getReq)

		token := getRR.Header().Get(csrfTokenHeader)
		cookie := getRR.Result().Cookies()[0]

		postReq := httptest.NewRequest("POST", "/", nil)
		postReq.Header.Set(csrfTokenHeader, token)
		postReq.AddCookie(cookie)
		postRR := httptest.NewRecorder()
		handler.ServeHTTP(postRR, postReq)

		if status := postRR.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})

	t.Run("post request with invalid token fails", func(t *testing.T) {
		// Step 1: Get a valid cookie
		getReq := httptest.NewRequest("GET", "/", nil)
		getRR := httptest.NewRecorder()
		handler.ServeHTTP(getRR, getReq)
		cookie := getRR.Result().Cookies()[0]

		postReq := httptest.NewRequest("POST", "/", nil)
		postReq.Header.Set(csrfTokenHeader, "this-is-an-invalid-token")
		postReq.AddCookie(cookie)
		postRR := httptest.NewRecorder()
		handler.ServeHTTP(postRR, postReq)

		if status := postRR.Code; status != http.StatusForbidden {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusForbidden)
		}
	})

	t.Run("post request with no token header fails", func(t *testing.T) {
		postReq := httptest.NewRequest("POST", "/", nil)
		postRR := httptest.NewRecorder()
		handler.ServeHTTP(postRR, postReq)

		if status := postRR.Code; status != http.StatusForbidden {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusForbidden)
		}
	})
}
