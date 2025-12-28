package healthcheck

import (
	stdContext "context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
)

type Probe func(ctx stdContext.Context) error
type Config struct {
	base.BaseConfig
	LivenessPath    string
	ReadinessPath   string
	HealthPath      string
	Timeout         time.Duration
	LivenessProbes  map[string]Probe
	ReadinessProbes map[string]Probe
	HealthProbes    map[string]Probe
}
type HealthStatus struct {
	Status string                 `json:"status"`
	Errors map[string]string      `json:"errors,omitempty"`
	Checks map[string]interface{} `json:"checks,omitempty"`
}

func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if c.LivenessPath == "" {
		c.LivenessPath = "/health/live"
	}
	if c.ReadinessPath == "" {
		c.ReadinessPath = "/health/ready"
	}
	if c.HealthPath == "" {
		c.HealthPath = "/health"
	}
	if c.Timeout <= 0 {
		c.Timeout = 5 * time.Second
	}
	if c.LivenessProbes == nil {
		c.LivenessProbes = make(map[string]Probe)
	}
	if c.ReadinessProbes == nil {
		c.ReadinessProbes = make(map[string]Probe)
	}
	if c.HealthProbes == nil {
		c.HealthProbes = make(map[string]Probe)
	}
	return nil
}
func New(config ...Config) func(next context.HandlerFunc) context.HandlerFunc {
	cfg := Config{}
	if len(config) > 0 {
		cfg = config[0]
	}

	if err := cfg.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "HealthCheck")
			}
		}
	}
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if cfg.Skip != nil && cfg.Skip(c) {
				next(c)
				return
			}
			path := c.Request.URL.Path
			var probes map[string]Probe
			switch path {
			case cfg.LivenessPath:
				probes = cfg.LivenessProbes
			case cfg.ReadinessPath:
				probes = cfg.ReadinessProbes
			case cfg.HealthPath:
				probes = cfg.HealthProbes
			default:
				next(c)
				return
			}
			status := runHealthChecks(c.Request.Context(), probes, cfg.Timeout)
			c.Writer.Header().Set("Content-Type", "application/json")
			c.Writer.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			statusCode := http.StatusOK
			if status.Status == "DOWN" {
				statusCode = http.StatusServiceUnavailable
			}
			c.Writer.WriteHeader(statusCode)
			if err := json.NewEncoder(c.Writer).Encode(status); err != nil {
				logger := cfg.Logger
				if logger == nil {
					logger = base.DefaultLogger("HealthCheck")
				}
				logger.Printf("could not encode health check response: %v", err)
			}
		}
	}
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New()
}
func WithProbes(livenessProbes, readinessProbes map[string]Probe) func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{
		LivenessProbes:  livenessProbes,
		ReadinessProbes: readinessProbes,
	})
}
func runHealthChecks(ctx stdContext.Context, probes map[string]Probe, timeout time.Duration) *HealthStatus {
	if len(probes) == 0 {
		return &HealthStatus{
			Status: "UP",
		}
	}
	ctx, cancel := stdContext.WithTimeout(ctx, timeout)
	defer cancel()
	var wg sync.WaitGroup
	results := make(chan probeResult, len(probes))
	for name, probe := range probes {
		wg.Add(1)
		go func(name string, probe Probe) {
			defer wg.Done()
			done := make(chan error, 1)
			go func() {
				done <- probe(ctx)
			}()
			var err error
			select {
			case err = <-done:
			case <-ctx.Done():
				err = ctx.Err()
			}
			results <- probeResult{
				name: name,
				err:  err,
			}
		}(name, probe)
	}
	wg.Wait()
	close(results)
	status := &HealthStatus{
		Status: "UP",
		Errors: make(map[string]string),
		Checks: make(map[string]interface{}),
	}
	for result := range results {
		if result.err != nil {
			status.Status = "DOWN"
			status.Errors[result.name] = result.err.Error()
			status.Checks[result.name] = map[string]interface{}{
				"status": "DOWN",
				"error":  result.err.Error(),
			}
		} else {
			status.Checks[result.name] = map[string]interface{}{
				"status": "UP",
			}
		}
	}
	return status
}

type probeResult struct {
	name string
	err  error
}

func DatabaseProbe(pingFunc func(stdContext.Context) error) Probe {
	return func(ctx stdContext.Context) error {
		return pingFunc(ctx)
	}
}
func HTTPProbe(url string, client *http.Client) Probe {
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	return func(ctx stdContext.Context) error {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return base.NewConfigError("HTTP Probe", "received status "+resp.Status)
		}
		return nil
	}
}
func AlwaysUpProbe() Probe {
	return func(ctx stdContext.Context) error {
		return nil
	}
}
func AlwaysDownProbe(message string) Probe {
	return func(ctx stdContext.Context) error {
		return base.NewConfigError("Test Probe", message)
	}
}
