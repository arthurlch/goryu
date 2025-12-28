package goryu

// honestly it is too embeded inside main pack and api related test and design will be stay inside main folder
import (
	"testing"
	"time"

	"github.com/arthurlch/goryu/config/builder"
	routebuilder "github.com/arthurlch/goryu/router/builder"
)

func TestFluentAPI(t *testing.T) {
	// Test configuration API
	app := NewWithConfig(func(cfg *builder.Builder) *builder.Builder {
		return cfg.
			SetAppName("TestApp").
			SetPort(8080).
			Static(func(s *builder.StaticConfig) {
				s.Root = "."
			})
	})

	if app.Config.AppName != "TestApp" {
		t.Errorf("Expected AppName to be TestApp, got %s", app.Config.AppName)
	}

	if app.Config.ServerPort != 8080 {
		t.Errorf("Expected ServerPort to be 8080, got %d", app.Config.ServerPort)
	}

	app.Route().
		Group("/api", func(g *routebuilder.SimpleGroupBuilder) {
		})

	logger := Logger().Development().Build()
	if logger == nil {
		t.Error("Expected logger middleware, got nil")
	}

	recovery := Recovery().Production().Build()
	if recovery == nil {
		t.Error("Expected recovery middleware, got nil")
	}

	cors := CORS().AllowOrigins("https://example.com").Build()
	if cors == nil {
		t.Error("Expected CORS middleware, got nil")
	}

	rateLimit := RateLimit(100, time.Minute).Build()
	if rateLimit == nil {
		t.Error("Expected rate limit middleware, got nil")
	}
}

func TestPluginRegistry(t *testing.T) {
	plugins := ListPlugins()
	expectedPlugins := []string{"logger", "recovery", "cors", "ratelimit"}

	for _, expected := range expectedPlugins {
		found := false
		for _, plugin := range plugins {
			if plugin == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected plugin %s to be registered", expected)
		}
	}
}
