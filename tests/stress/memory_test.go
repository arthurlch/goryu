package stress

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/arthurlch/goryu"
)

// TestMemoryLeak checks for goroutine leaks and major memory growth
func TestMemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	falsePtr := false
	app := goryu.New(goryu.Config{
		EnableMonitoring: &falsePtr,
	})
	app.GET("/stress", func(c *goryu.Context) {
		c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
			"data":   "some reasonably sized payload to test allocations",
		})
	})

	// Warmup
	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/stress", nil)
		app.ServeHTTP(w, req)
	}

	// Force GC to get baseline
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)
	initialGoroutines := runtime.NumGoroutine()

	t.Logf("Initial: Goroutines=%d, HeapAlloc=%d bytes", initialGoroutines, m1.HeapAlloc)

	// Stress loop
	iterations := 10000
	for i := 0; i < iterations; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/stress", nil)
		
		// Simulate different request patterns?
		if i%2 == 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		
		app.ServeHTTP(w, req)
		
		if w.Code != 200 {
			t.Fatalf("Status not 200: %d", w.Code)
		}
	}

	// Allow for cleanup
	for k := 0; k < 5; k++ {
		runtime.GC()
		time.Sleep(50 * time.Millisecond)
	}
	
	runtime.ReadMemStats(&m2)
	finalGoroutines := runtime.NumGoroutine()
	
	t.Logf("Final: Goroutines=%d, HeapAlloc=%d bytes, HeapInUse=%d bytes", finalGoroutines, m2.HeapAlloc, m2.HeapInuse)
	t.Logf("Growth: %d bytes", int64(m2.HeapAlloc)-int64(m1.HeapAlloc))

	// Verify Goroutines
	if finalGoroutines > initialGoroutines+2 {
		t.Errorf("Potential Goroutine leak: grew from %d to %d", initialGoroutines, finalGoroutines)
	}

	// Verify Memory
	// 5MB growth was seen. Let's allow 1MB for variance, but 5MB is too much.
	// If it persists, we have a real leak.
	growth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	// 10000 reqs * 500 bytes = 5MB. If we leak context or parts of it, that's possible.
	if growth > 2*1024*1024 { // 2MB threshold
		t.Errorf("Potential Memory leak: Heap grew by %d bytes", growth)
	}
}

// BenchmarkRequestPerformance performs a standard benchmark to track regression
func BenchmarkRequestPerformance(b *testing.B) {
	app := goryu.New()
	app.GET("/bench", func(c *goryu.Context) {
		c.JSON(http.StatusOK, map[string]string{"result": "ok"})
	})

	req := httptest.NewRequest("GET", "/bench", nil)
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}
