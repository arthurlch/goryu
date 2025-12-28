package metrics

import (
	"strconv"
	"time"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
)

type Metrics interface {
	IncrementCounter(name string, tags map[string]string)
	AddToCounter(name string, value float64, tags map[string]string)
	RecordHistogram(name string, value float64, tags map[string]string)
	SetGauge(name string, value float64, tags map[string]string)
	AddToGauge(name string, value float64, tags map[string]string)
}
type Config struct {
	base.BaseConfig
	Metrics         Metrics
	CustomTags      func(c *context.Context) map[string]string
	RecordBody      bool
	GroupStatusCode bool
	Prefix          string
}

func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if c.Metrics == nil {
		c.Metrics = &noopMetrics{}
	}
	if c.Prefix == "" {
		c.Prefix = "http"
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
				base.DefaultErrorHandler(c, err, "Metrics")
			}
		}
	}
	preHandler := func(c *context.Context) error {
		rw := base.NewStandardResponseWriter(c.Writer)
		c.Writer = rw
		c.Set("metrics.start_time", time.Now())
		c.Set("metrics.response_writer", rw)
		cfg.Metrics.AddToGauge(cfg.Prefix+"_requests_active", 1, map[string]string{})
		return nil
	}
	postHandler := func(c *context.Context) error {
		startVal, exists := c.Get("metrics.start_time")
		if !exists {
			return nil
		}
		start := startVal.(time.Time)
		rwVal, exists := c.Get("metrics.response_writer")
		if !exists {
			return nil
		}
		rw := rwVal.(*base.StandardResponseWriter)
		duration := time.Since(start)
		tags := map[string]string{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
		}
		if cfg.GroupStatusCode {
			tags["status_class"] = getStatusClass(rw.Status())
		} else {
			tags["status"] = strconv.Itoa(rw.Status())
		}
		if cfg.CustomTags != nil {
			for k, v := range cfg.CustomTags(c) {
				tags[k] = v
			}
		}
		cfg.Metrics.IncrementCounter(cfg.Prefix+"_requests_total", tags)
		cfg.Metrics.RecordHistogram(cfg.Prefix+"_request_duration_seconds", duration.Seconds(), tags)
		if cfg.RecordBody {
			cfg.Metrics.RecordHistogram(cfg.Prefix+"_request_size_bytes", float64(c.Request.ContentLength), tags)
			cfg.Metrics.RecordHistogram(cfg.Prefix+"_response_size_bytes", float64(rw.Size()), tags)
		}
		cfg.Metrics.AddToGauge(cfg.Prefix+"_requests_active", -1, map[string]string{})
		return nil
	}
	return base.PostProcessMiddleware("Metrics", cfg.BaseConfig, preHandler, postHandler)
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New()
}
func getStatusClass(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return "2xx"
	case statusCode >= 300 && statusCode < 400:
		return "3xx"
	case statusCode >= 400 && statusCode < 500:
		return "4xx"
	case statusCode >= 500:
		return "5xx"
	default:
		return "1xx"
	}
}

type noopMetrics struct{}

func (n *noopMetrics) IncrementCounter(name string, tags map[string]string)               {}
func (n *noopMetrics) AddToCounter(name string, value float64, tags map[string]string)    {}
func (n *noopMetrics) RecordHistogram(name string, value float64, tags map[string]string) {}
func (n *noopMetrics) SetGauge(name string, value float64, tags map[string]string)        {}
func (n *noopMetrics) AddToGauge(name string, value float64, tags map[string]string)      {}
