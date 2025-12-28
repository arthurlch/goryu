package monitoring

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// safeWriter wraps a bytes.Buffer with a mutex for thread-safe writes
type safeWriter struct {
	buf *bytes.Buffer
	mu  *sync.Mutex
}

func (sw *safeWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.buf.Write(p)
}

func TestEventHandlerPanicRecovery(t *testing.T) {
	t.Run("PanicHandler_SafeExecuteEnabled", func(t *testing.T) {
		monitor := New(Config{Enabled: true, SafeExecute: true})

		panicHandlerCalled := make(chan bool, 1)
		normalHandlerCalled := make(chan bool, 1)

		monitor.AddEventHandler(func(event Event) {
			if event.Type != EventCustom {
				return
			}
			panicHandlerCalled <- true
			panic("test panic in event handler")
		})

		monitor.AddEventHandler(func(event Event) {
			if event.Type != EventCustom {
				return
			}
			normalHandlerCalled <- true
		})

		monitor.EmitEvent(EventCustom, "test event", nil)

		select {
		case <-panicHandlerCalled:
			// Good
		case <-time.After(1 * time.Second):
			t.Error("Panic handler was not called within timeout")
		}

		select {
		case <-normalHandlerCalled:
			// Good
		case <-time.After(1 * time.Second):
			t.Error("Normal handler was not called within timeout")
		}

		time.Sleep(100 * time.Millisecond)

		events := monitor.GetEvents(10)
		var errorEventFound bool
		for _, event := range events {
			if event.Type == EventError && event.Message == "Event handler panicked" {
				errorEventFound = true
				if event.Data["handler_error"] != true {
					t.Error("Expected handler_error flag in error event data")
				}
				if !strings.Contains(event.Data["panic_value"].(string), "test panic in event handler") {
					t.Errorf("Expected panic value in error event data, got: %v", event.Data["panic_value"])
				}
				break
			}
		}
		if !errorEventFound {
			t.Error("Expected error event to be created for panicked handler")
		}
	})

	t.Run("PanicHandler_SafeExecuteDisabled", func(t *testing.T) {
		monitor := New(Config{Enabled: true, SafeExecute: false})

		var handlerCallCount int32
		monitor.AddEventHandler(func(event Event) {
			if event.Type == EventStartup {
				return
			}
			atomic.AddInt32(&handlerCallCount, 1)
		})

		monitor.EmitEvent(EventCustom, "test event", nil)

		time.Sleep(100 * time.Millisecond)

		calls := atomic.LoadInt32(&handlerCallCount)
		if calls != 1 {
			t.Errorf("Expected handler to be called once, got %d calls", calls)
		}

		if monitor.safeExecute {
			t.Error("Expected SafeExecute to be false")
		}
	})
}

func TestHealthCheckPanicRecovery(t *testing.T) {
	t.Run("HealthCheckPanic_SafeExecuteEnabled", func(t *testing.T) {
		var buf bytes.Buffer
		var bufMu sync.Mutex

		// Use a safe writer that protects concurrent access to the buffer
		log.SetOutput(&safeWriter{buf: &buf, mu: &bufMu})
		defer log.SetOutput(os.Stderr)

		monitor := New(Config{Enabled: true, SafeExecute: true, HealthInterval: 0}) // Disable automatic checks

		monitor.AddHealthCheck("panic_check", &HealthCheck{
			Check: func() (HealthStatus, error) {
				panic("test panic in health check")
			},
			Critical: true,
		})

		monitor.AddHealthCheck("normal_check", &HealthCheck{
			Check: func() (HealthStatus, error) {
				return StatusHealthy, nil
			},
			Critical: false,
		})

		monitor.executeHealthChecks()

		time.Sleep(500 * time.Millisecond)

		bufMu.Lock()
		logOutput := buf.String()
		bufMu.Unlock()

		if !strings.Contains(logOutput, "Health check 'panic_check' panicked") {
			t.Errorf("Expected health check panic to be logged, got: %s", logOutput)
		}

		results := monitor.GetHealthResults()

		panicResult, exists := results["panic_check"]
		if !exists {
			t.Error("Expected result for panic_check")
		} else {
			if panicResult.Status != StatusUnhealthy {
				t.Errorf("Expected panic check to be unhealthy, got: %s", panicResult.Status)
			}
			if !strings.Contains(panicResult.Message, "Health check panicked") {
				t.Errorf("Expected panic message in result, got: %s", panicResult.Message)
			}
		}

		normalResult, exists := results["normal_check"]
		if !exists {
			t.Error("Expected result for normal_check")
		} else {
			if normalResult.Status != StatusHealthy {
				t.Errorf("Expected normal check to be healthy, got: %s", normalResult.Status)
			}
		}

		events := monitor.GetEvents(10)
		var errorEventFound bool
		for _, event := range events {
			if event.Type == EventUnhealthy && strings.Contains(event.Message, "panic_check' panicked") {
				errorEventFound = true
				if event.Data["health_error"] != true {
					t.Error("Expected health_error flag in error event data")
				}
				break
			}
		}
		if !errorEventFound {
			t.Error("Expected error event to be created for panicked health check")
		}
	})

	t.Run("HealthCheckPanic_SafeExecuteDisabled", func(t *testing.T) {
		monitor := New(Config{Enabled: true, SafeExecute: false, HealthInterval: 0})

		if monitor.safeExecute {
			t.Error("Expected SafeExecute to be false")
		}

		monitor.AddHealthCheck("normal_check", &HealthCheck{
			Check: func() (HealthStatus, error) {
				return StatusHealthy, nil
			},
			Critical: false,
		})

		monitor.executeHealthChecks()

		time.Sleep(200 * time.Millisecond)

		results := monitor.GetHealthResults()
		if result, exists := results["normal_check"]; !exists {
			t.Error("Expected result for normal_check in unsafe mode")
		} else if result.Status != StatusHealthy {
			t.Errorf("Expected normal check to be healthy, got: %s", result.Status)
		}
	})
}

func TestHealthCheckErrorHandling(t *testing.T) {
	monitor := New(Config{Enabled: true, SafeExecute: true, HealthInterval: 0})

	monitor.AddHealthCheck("error_check", &HealthCheck{
		Check: func() (HealthStatus, error) {
			return StatusHealthy, errors.New("test error")
		},
		Critical: false,
	})

	monitor.executeHealthChecks()

	time.Sleep(100 * time.Millisecond)

	results := monitor.GetHealthResults()
	result, exists := results["error_check"]
	if !exists {
		t.Error("Expected result for error_check")
	} else {
		if result.Status != StatusUnhealthy {
			t.Errorf("Expected error check to be unhealthy, got: %s", result.Status)
		}
		if result.Message != "test error" {
			t.Errorf("Expected error message in result, got: %s", result.Message)
		}
	}
}

func TestConcurrentEventHandlers(t *testing.T) {
	monitor := New(Config{Enabled: true, SafeExecute: true})

	const numHandlers = 5
	const numEvents = 50

	handlerCallCount := make([]int32, numHandlers)

	for i := 0; i < numHandlers; i++ {
		handlerIndex := i
		monitor.AddEventHandler(func(event Event) {
			// Ignore startup event which might race
			if event.Type != EventCustom {
				return
			}

			atomic.AddInt32(&handlerCallCount[handlerIndex], 1)

			if handlerIndex%3 == 0 {
				panic(fmt.Sprintf("handler %d panic", handlerIndex))
			}
		})
	}

	var wg sync.WaitGroup
	for i := 0; i < numEvents; i++ {
		wg.Add(1)
		go func(eventNum int) {
			defer wg.Done()
			monitor.EmitEvent(EventCustom, fmt.Sprintf("event %d", eventNum), nil)
		}(i)
	}

	wg.Wait()

	// Give some time for handlers to complete before closing
	time.Sleep(200 * time.Millisecond)

	monitor.Close()
	monitor.Wait()

	// Additional buffer for handlers to finish processing
	time.Sleep(100 * time.Millisecond)

	for i, count := range handlerCallCount {
		actualCount := atomic.LoadInt32(&count)
		if actualCount != numEvents {
			t.Errorf("Handler %d was called %d times, expected %d", i, actualCount, numEvents)
		}
	}
}

func TestConcurrentHealthChecks(t *testing.T) {
	monitor := New(Config{Enabled: true, SafeExecute: true, HealthInterval: 0})

	const numHealthChecks = 20
	const numConcurrentExecutions = 5

	for i := 0; i < numHealthChecks; i++ {
		checkIndex := i
		monitor.AddHealthCheck(fmt.Sprintf("check_%d", checkIndex), &HealthCheck{
			Check: func() (HealthStatus, error) {
				switch checkIndex % 4 {
				case 0:
					return StatusHealthy, nil
				case 1:
					time.Sleep(10 * time.Millisecond)
					return StatusHealthy, nil
				case 2:
					// Error
					return StatusUnhealthy, fmt.Errorf("check %d failed", checkIndex)
				case 3:
					// Panic
					panic(fmt.Sprintf("check %d panicked", checkIndex))
				}
				return StatusHealthy, nil
			},
			Critical: checkIndex%2 == 0,
		})
	}

	var wg sync.WaitGroup
	for i := 0; i < numConcurrentExecutions; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			monitor.executeHealthChecks()
		}()
	}

	wg.Wait()

	time.Sleep(200 * time.Millisecond)

	results := monitor.GetHealthResults()
	if len(results) != numHealthChecks {
		t.Errorf("Expected %d health check results, got %d", numHealthChecks, len(results))
	}

	var healthyCount, unhealthyCount, panicCount int
	for name, result := range results {
		switch result.Status {
		case StatusHealthy:
			healthyCount++
		case StatusUnhealthy:
			unhealthyCount++
			if strings.Contains(name, "_") {
				checkNum := name[6:]
				if checkNum != "" && strings.Contains(result.Message, "panicked") {
					panicCount++
				}
			}
		}
	}

	if healthyCount == 0 {
		t.Error("Expected at least some healthy checks")
	}
	if unhealthyCount == 0 {
		t.Error("Expected at least some unhealthy checks")
	}
}

func TestConcurrentEventSubscription(t *testing.T) {
	monitor := New(Config{Enabled: true, SafeExecute: true, EventBufferSize: 10000})

	const numHandlers = 15
	const numEmitters = 5
	const eventsPerEmitter = 50

	handlerCalls := make([]int, numHandlers)
	var callMutex sync.Mutex

	var setupWg sync.WaitGroup
	for i := 0; i < numHandlers; i++ {
		setupWg.Add(1)
		go func(handlerIndex int) {
			defer setupWg.Done()

			monitor.AddEventHandler(func(event Event) {
				if event.Type != EventCustom {
					return
				}

				callMutex.Lock()
				handlerCalls[handlerIndex]++
				callMutex.Unlock()

				switch handlerIndex % 5 {
				case 0:
					return
				case 1:
					time.Sleep(1 * time.Millisecond)
				case 2:
					if event.Data != nil {
						_ = event.Data["test"]
					}
				case 3:
					if handlerIndex == 3 && event.Message == "panic_trigger" {
						panic("intentional test panic")
					}
				case 4:
					_ = len(event.Message)
				}
			})
		}(i)
	}
	setupWg.Wait()

	var emitWg sync.WaitGroup
	for i := 0; i < numEmitters; i++ {
		emitWg.Add(1)
		go func(emitterIndex int) {
			defer emitWg.Done()

			for j := 0; j < eventsPerEmitter; j++ {
				eventMsg := fmt.Sprintf("emitter_%d_event_%d", emitterIndex, j)
				eventData := map[string]interface{}{
					"emitter": emitterIndex,
					"event":   j,
					"test":    "value",
				}

				// Emit special panic trigger occasionally
				if j == 25 {
					monitor.EmitEvent(EventCustom, "panic_trigger", eventData)
				} else {
					monitor.EmitEvent(EventCustom, eventMsg, eventData)
				}
			}
		}(i)
	}
	emitWg.Wait()

	monitor.Close()
	monitor.Wait()
	time.Sleep(100 * time.Millisecond) // buffer for handlers

	expectedCalls := numEmitters * eventsPerEmitter
	callMutex.Lock()
	for i, calls := range handlerCalls {
		if calls != expectedCalls {
			t.Errorf("Handler %d received %d calls, expected %d", i, calls, expectedCalls)
		}
	}
	callMutex.Unlock()
}

func TestMonitoringStressTest(t *testing.T) {
	monitor := New(Config{Enabled: true, SafeExecute: true, HealthInterval: 0, EventBufferSize: 10000})

	const duration = 100 * time.Millisecond
	const numWorkers = 10

	var (
		eventCount   int
		healthCount  int
		handlerCount int
		mu           sync.Mutex
	)

	for i := 0; i < 5; i++ {
		monitor.AddEventHandler(func(event Event) {
			mu.Lock()
			handlerCount++
			mu.Unlock()
		})
	}

	for i := 0; i < 3; i++ {
		checkName := fmt.Sprintf("stress_check_%d", i)
		monitor.AddHealthCheck(checkName, &HealthCheck{
			Check: func() (HealthStatus, error) {
				mu.Lock()
				healthCount++
				mu.Unlock()
				return StatusHealthy, nil
			},
			Critical: false,
		})
	}

	var wg sync.WaitGroup
	stopChan := make(chan bool)

	for i := 0; i < numWorkers/2; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-stopChan:
					return
				default:
					monitor.EmitEvent(EventCustom, fmt.Sprintf("stress_event_%d", workerID), nil)
					mu.Lock()
					eventCount++
					mu.Unlock()
					time.Sleep(1 * time.Millisecond)
				}
			}
		}(i)
	}

	for i := numWorkers / 2; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopChan:
					return
				default:
					monitor.executeHealthChecks()
					time.Sleep(5 * time.Millisecond)
				}
			}
		}()
	}

	time.Sleep(duration)
	close(stopChan)
	wg.Wait()

	monitor.Close()
	monitor.Wait()

	// Give handlers launched by worker a moment to finish (since they are goroutines)
	// Or we should wait for them. But we don't track them.
	// For now, a small sleep is still needed for *handlers* to finish, but queue is drained.
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if eventCount == 0 {
		t.Error("Expected some events to be emitted")
	}
	if healthCount == 0 {
		t.Error("Expected some health checks to be executed")
	}
	if handlerCount == 0 {
		t.Error("Expected some handlers to be called")
	}

	expectedHandlerCalls := eventCount * 5
	if handlerCount < expectedHandlerCalls/2 || handlerCount > expectedHandlerCalls*2 {
		t.Errorf("Handler calls (%d) outside expected range for %d events with 5 handlers", handlerCount, eventCount)
	}
}

func TestSafeExecuteConfig(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		monitor := New()
		if !monitor.safeExecute {
			t.Error("Expected SafeExecute to be true by default")
		}
	})

	t.Run("ExplicitConfig", func(t *testing.T) {
		monitor := New(Config{Enabled: true, SafeExecute: false})
		if monitor.safeExecute {
			t.Error("Expected SafeExecute to be false when explicitly configured")
		}
	})
}
