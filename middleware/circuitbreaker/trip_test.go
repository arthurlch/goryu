package circuitbreaker_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/circuitbreaker"
)

func TestBreakerOpensOnDownstreamFailures(t *testing.T) {
	mw := circuitbreaker.New(circuitbreaker.Config{
		MinRequests:  3,
		FailureRatio: 0.5,
	})

	calls := 0
	failing := func(c *context.Context) {
		calls++
		c.Writer.WriteHeader(http.StatusInternalServerError)
	}

	// Drive enough 5xx responses to trip the breaker.
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		mw(failing)(context.NewContext(httptest.NewRecorder(), req))
	}
	callsBeforeOpen := calls

	// Next request must be short-circuited: handler not called, 503 returned.
	rr := httptest.NewRecorder()
	mw(failing)(context.NewContext(rr, httptest.NewRequest("GET", "/", nil)))

	if calls != callsBeforeOpen {
		t.Fatalf("handler was called after breaker should have opened (calls %d -> %d)", callsBeforeOpen, calls)
	}
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when breaker is open, got %d", rr.Code)
	}
}
