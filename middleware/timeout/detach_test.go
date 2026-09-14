package timeout_test

import (
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/timeout"
)

func TestTimeoutSlowHandlerDoesNotCorruptPool(t *testing.T) {
	mw := timeout.New(timeout.Config{Timeout: 20 * time.Millisecond})

	handler := func(c *context.Context) {
		time.Sleep(200 * time.Millisecond)
		c.Set("late", "value")
		_ = c.Request
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/", nil)
			rr := httptest.NewRecorder()
			c := context.NewContext(rr, req)
			mw(handler)(c)
			c.Release()
			if rr.Code != 0 && rr.Code != 408 {
				t.Errorf("unexpected status %d", rr.Code)
			}
		}()
	}
	wg.Wait()
	time.Sleep(300 * time.Millisecond)
}
