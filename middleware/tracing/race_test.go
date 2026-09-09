package tracing

import (
	stdContext "context"
	"sync"
	"testing"
)

func TestSimpleTracerConcurrent(t *testing.T) {
	tracer := NewSimpleTracer()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			span, _ := tracer.StartSpan(stdContext.Background(), "op")
			span.SetTag("k", "v")
			span.AddEvent("e", map[string]interface{}{"a": 1})
			span.SetStatus(StatusCodeOk, "ok")
			span.End()
			_ = tracer.GetSpans()
		}()
	}
	wg.Wait()
	if got := len(tracer.GetSpans()); got != 50 {
		t.Fatalf("expected 50 spans, got %d", got)
	}
}
