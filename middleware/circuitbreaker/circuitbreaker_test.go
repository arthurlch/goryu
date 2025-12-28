package circuitbreaker

import (
	"errors"
	context "github.com/arthurlch/goryu/goryuctx"
	"net/http"
	"testing"
	"time"
)

func TestNewCircuitBreaker(t *testing.T) {
	config := Config{
		MaxRequests:  5,
		Interval:     60 * time.Second,
		Timeout:      30 * time.Second,
		FailureRatio: 0.5,
		MinRequests:  3,
	}
	cb := NewCircuitBreaker(config)
	if cb == nil {
		t.Fatal("Expected circuit breaker to be created")
	}
	if cb.config.MaxRequests != 5 {
		t.Errorf("Expected MaxRequests=5, got %d", cb.config.MaxRequests)
	}
	if cb.state != StateClosed {
		t.Errorf("Expected initial state to be Closed, got %s", cb.state.String())
	}
}
func TestNewCircuitBreakerDefaults(t *testing.T) {
	config := Config{}
	cb := NewCircuitBreaker(config)
	if cb.config.MaxRequests != 1 {
		t.Errorf("Expected default MaxRequests=1, got %d", cb.config.MaxRequests)
	}
	if cb.config.Interval != 60*time.Second {
		t.Errorf("Expected default Interval=60s, got %v", cb.config.Interval)
	}
	if cb.config.Timeout != 60*time.Second {
		t.Errorf("Expected default Timeout=60s, got %v", cb.config.Timeout)
	}
	if cb.config.FailureRatio != 0.6 {
		t.Errorf("Expected default FailureRatio=0.6, got %f", cb.config.FailureRatio)
	}
	if cb.config.MinRequests != 3 {
		t.Errorf("Expected default MinRequests=3, got %d", cb.config.MinRequests)
	}
}
func TestCircuitBreakerClosedState(t *testing.T) {
	config := Config{
		MaxRequests:  1,
		Timeout:      1 * time.Second,
		FailureRatio: 0.5,
		MinRequests:  2,
	}
	cb := NewCircuitBreaker(config)
	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected success in closed state, got error: %v", err)
	}
	if cb.State() != StateClosed {
		t.Errorf("Expected state to remain Closed, got %s", cb.State().String())
	}
}
func TestCircuitBreakerOpenState(t *testing.T) {
	config := Config{
		MaxRequests:  1,
		Timeout:      1 * time.Second,
		FailureRatio: 0.5,
		MinRequests:  2,
	}
	cb := NewCircuitBreaker(config)
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error {
			return errors.New("failure")
		})
	}
	if cb.State() != StateOpen {
		t.Errorf("Expected state to be Open, got %s", cb.State().String())
	}
	err := cb.Execute(func() error {
		return nil
	})
	if err == nil {
		t.Error("Expected request to be rejected in open state")
	}
}
func TestCircuitBreakerHalfOpenState(t *testing.T) {
	config := Config{
		MaxRequests:  2,
		Timeout:      50 * time.Millisecond,
		FailureRatio: 0.5,
		MinRequests:  2,
	}
	cb := NewCircuitBreaker(config)
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error {
			return errors.New("failure")
		})
	}
	time.Sleep(100 * time.Millisecond)
	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected request to be allowed in half-open state, got error: %v", err)
	}
	if cb.State() != StateClosed {
		t.Errorf("Expected state to transition to Closed after success, got %s", cb.State().String())
	}
}
func TestCircuitBreakerStateTransitions(t *testing.T) {
	var stateChanges []State
	config := Config{
		MaxRequests:  1,
		Timeout:      50 * time.Millisecond,
		FailureRatio: 0.5,
		MinRequests:  2,
		OnStateChange: func(state State) {
			stateChanges = append(stateChanges, state)
		},
	}
	cb := NewCircuitBreaker(config)
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error {
			return errors.New("failure")
		})
	}
	if len(stateChanges) == 0 || stateChanges[len(stateChanges)-1] != StateOpen {
		t.Error("Expected state change to Open")
	}
	time.Sleep(100 * time.Millisecond)
	_ = cb.Execute(func() error {
		return nil
	})
	foundHalfOpen := false
	foundClosed := false
	for _, state := range stateChanges {
		if state == StateHalfOpen {
			foundHalfOpen = true
		}
		if state == StateClosed {
			foundClosed = true
		}
	}
	if !foundHalfOpen {
		t.Error("Expected transition to HalfOpen state")
	}
	if !foundClosed {
		t.Error("Expected transition back to Closed state")
	}
}
func TestCircuitBreakerMetrics(t *testing.T) {
	config := Config{
		MaxRequests:  1,
		Timeout:      1 * time.Second,
		FailureRatio: 0.5,
		MinRequests:  2,
	}
	cb := NewCircuitBreaker(config)
	_ = cb.Execute(func() error {
		return nil
	})
	_ = cb.Execute(func() error {
		return errors.New("failure")
	})
	metrics := cb.Metrics()
	state, ok := metrics["state"].(string)
	if !ok || state != "closed" {
		t.Errorf("Expected state='closed', got %v", metrics["state"])
	}
	requests, ok := metrics["requests"].(uint32)
	if !ok || requests != 2 {
		t.Errorf("Expected requests=2, got %v", metrics["requests"])
	}
	failures, ok := metrics["failures"].(uint32)
	if !ok || failures != 1 {
		t.Errorf("Expected failures=1, got %v", metrics["failures"])
	}
	failureRate, ok := metrics["failure_rate"].(float64)
	if !ok || failureRate != 0.5 {
		t.Errorf("Expected failure_rate=0.5, got %v", metrics["failure_rate"])
	}
}
func TestCircuitBreakerMiddleware(t *testing.T) {
	config := Config{
		MaxRequests:  1,
		Timeout:      1 * time.Second,
		FailureRatio: 0.5,
		MinRequests:  1,
	}
	_, middleware := WithCircuitBreaker(config)
	called := false
	handler := func(c *context.Context) {
		called = true
	}
	wrappedHandler := middleware(handler)
	ctx := &context.Context{
		Writer: &mockResponseWriter{},
	}
	wrappedHandler(ctx)
	if !called {
		t.Error("Expected handler to be called")
	}
}
func TestStateString(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(999), "unknown"},
	}
	for _, test := range tests {
		if test.state.String() != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, test.state.String())
		}
	}
}
func BenchmarkCircuitBreakerExecute(b *testing.B) {
	config := Config{
		MaxRequests:  10,
		Timeout:      1 * time.Second,
		FailureRatio: 0.9,
		MinRequests:  10,
	}
	cb := NewCircuitBreaker(config)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.Execute(func() error {
			return nil
		})
	}
}

type mockResponseWriter struct {
	statusCode int
	data       []byte
	headers    map[string][]string
}

func (m *mockResponseWriter) Header() http.Header {
	if m.headers == nil {
		m.headers = make(map[string][]string)
	}
	return http.Header(m.headers)
}
func (m *mockResponseWriter) Write(data []byte) (int, error) {
	if m.statusCode == 0 {
		m.statusCode = 200
	}
	m.data = append(m.data, data...)
	return len(data), nil
}
func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
}
