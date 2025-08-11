package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

type Probe func(ctx context.Context) error

type HealthChecker struct {
	liveProbes  map[string]Probe
	readyProbes map[string]Probe
	livePath    string
	readyPath   string
	timeout     time.Duration
}

func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		liveProbes:  make(map[string]Probe),
		readyProbes: make(map[string]Probe),
		livePath:    "/live",
		readyPath:   "/ready",
		timeout:     5 * time.Second,
	}
}

func (h *HealthChecker) AddLivenessCheck(name string, probe Probe) {
	h.liveProbes[name] = probe
}

func (h *HealthChecker) AddReadinessCheck(name string, probe Probe) {
	h.readyProbes[name] = probe
}

func (h *HealthChecker) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case h.livePath:
			h.runProbes(w, r, h.liveProbes)
		case h.readyPath:
			h.runProbes(w, r, h.readyProbes)
		default:
			next.ServeHTTP(w, r)
		}
	})
}

func (h *HealthChecker) runProbes(w http.ResponseWriter, r *http.Request, probes map[string]Probe) {
	if len(probes) == 0 {
		h.writeResponse(w, http.StatusOK, map[string]string{"status": "UP"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	var wg sync.WaitGroup
	errsChan := make(chan map[string]string, len(probes))

	for name, probe := range probes {
		wg.Add(1)
		go func(name string, probe Probe) {
			defer wg.Done()
			done := make(chan error, 1)

			go func() {
				done <- probe(ctx)
			}()

			select {
			case err := <-done:
				if err != nil {
					errsChan <- map[string]string{name: err.Error()}
				}
			case <-ctx.Done():
				errsChan <- map[string]string{name: ctx.Err().Error()}
			}
		}(name, probe)
	}

	wg.Wait()
	close(errsChan)

	failedChecks := make(map[string]string)
	for errMap := range errsChan {
		for k, v := range errMap {
			failedChecks[k] = v
		}
	}

	if len(failedChecks) > 0 {
		response := map[string]interface{}{
			"status": "DOWN",
			"errors": failedChecks,
		}
		h.writeResponse(w, http.StatusServiceUnavailable, response)
		return
	}

	h.writeResponse(w, http.StatusOK, map[string]string{"status": "UP"})
}

func (h *HealthChecker) writeResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error writing health check response: %v", err)
	}
}
