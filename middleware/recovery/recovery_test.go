package recovery_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/recovery"
)

func newTestContext(req *http.Request) (*goryu.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}

func TestRecoveryMiddleware(t *testing.T) {
	panicHandler := func(c *goryu.Context) {
		panic("something went wrong")
	}

	normalHandler := func(c *goryu.Context) {
		_ = c.Text(http.StatusOK, "OK")
	}

	t.Run("Recovers from panic", func(t *testing.T) {
		middleware := recovery.New()

		req := httptest.NewRequest("GET", "/panic", nil)
		ctx, rr := newTestContext(req)

		handlerToTest := middleware(panicHandler)

		handlerToTest(ctx)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
		expectedBodyContent := `"error":"Internal Server Error"`
		if body := rr.Body.String(); !strings.Contains(body, expectedBodyContent) {
			t.Errorf("expected body to contain '%s', got '%s'", expectedBodyContent, body)
		}
	})

	t.Run("Does not interfere with normal requests", func(t *testing.T) {
		middleware := recovery.New()

		req := httptest.NewRequest("GET", "/normal", nil)
		ctx, rr := newTestContext(req)

		handlerToTest := middleware(normalHandler)
		handlerToTest(ctx)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
		if body := rr.Body.String(); body != "OK" {
			t.Errorf("expected body 'OK', got '%s'", body)
		}
	})

	t.Run("Skip middleware with Next", func(t *testing.T) {
		config := recovery.Config{
			Next: func(c *goryu.Context) bool {
				return c.Request.URL.Path == "/panic-and-skip"
			},
		}
		middleware := recovery.New(config)

		req := httptest.NewRequest("GET", "/panic-and-skip", nil)
		ctx, _ := newTestContext(req)

		handlerToTest := middleware(panicHandler)

		defer func() {
			if r := recover(); r == nil {
				t.Error("The code did not panic as expected because it should have been skipped by the middleware.")
			}
		}()

		handlerToTest(ctx)
	})
}
