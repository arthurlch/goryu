package requestid_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/requestid"
)

func newTestContext(req *http.Request) (*goryu.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}

func TestRequestIDMiddleware(t *testing.T) {
	handler := func(c *goryu.Context) {
	}

	t.Run("Generates ID if none exists", func(t *testing.T) {
		middleware := requestid.New()
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		generatedID := rr.Header().Get(requestid.DefaultRequestIDHeader)
		if generatedID == "" {
			t.Error("Expected a request ID to be generated, but header is empty")
		}

		contextID, exists := ctx.Get("requestid")
		if !exists || contextID.(string) != generatedID {
			t.Errorf("Expected request ID to be in context. Got exists=%v, value=%v", exists, contextID)
		}
	})

	t.Run("Uses existing ID from header", func(t *testing.T) {
		middleware := requestid.New()
		req := httptest.NewRequest("GET", "/", nil)
		existingID := "my-existing-id-123"
		req.Header.Set(requestid.DefaultRequestIDHeader, existingID)
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		responseID := rr.Header().Get(requestid.DefaultRequestIDHeader)
		if responseID != existingID {
			t.Errorf("Expected request ID '%s', got '%s'", existingID, responseID)
		}

		contextID, _ := ctx.Get("requestid")
		if contextID.(string) != existingID {
			t.Errorf("Expected context ID to be '%s', got '%s'", existingID, contextID)
		}
	})

	t.Run("Uses custom configuration", func(t *testing.T) {
		customHeader := "X-Correlation-ID"
		customContextKey := "correlationID"
		customGenerator := func() string {
			return "fixed-generated-id"
		}

		config := requestid.Config{
			Header:     customHeader,
			ContextKey: customContextKey,
			Generator:  customGenerator,
		}
		middleware := requestid.New(config)

		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)

		if rr.Header().Get(customHeader) != "fixed-generated-id" {
			t.Errorf("Expected custom header to have generated ID, got '%s'", rr.Header().Get(customHeader))
		}
		contextID, _ := ctx.Get(customContextKey)
		if contextID.(string) != "fixed-generated-id" {
			t.Errorf("Expected custom context key to have generated ID, got '%s'", contextID)
		}
	})
}
