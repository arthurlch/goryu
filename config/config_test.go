package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfigDefaults(t *testing.T) {
	config := &Config{}
	err := applyDefaults(config)
	if err != nil {
		t.Fatalf("Failed to apply defaults: %v", err)
	}

	// Test some default values
	if config.Server.Host != "localhost" {
		t.Errorf("Expected default host 'localhost', got '%s'", config.Server.Host)
	}
	if config.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", config.Server.Port)
	}
	if config.App.Name != "goryu-app" {
		t.Errorf("Expected default app name 'goryu-app', got '%s'", config.App.Name)
	}
	if config.App.Environment != "development" {
		t.Errorf("Expected default environment 'development', got '%s'", config.App.Environment)
	}
}

func TestConfigBuilder(t *testing.T) {
	t.Run("BuildWithDefaults", func(t *testing.T) {
		config, err := NewBuilder().
			WithDefaults().
			Build()

		if err != nil {
			t.Fatalf("Failed to build config: %v", err)
		}

		if config == nil {
			t.Fatal("Config is nil")
		}

		// Should have defaults applied
		if config.Server.Port != 8080 {
			t.Errorf("Expected default port 8080, got %d", config.Server.Port)
		}
	})

	t.Run("BuildWithEnvironment", func(t *testing.T) {
		// Set test environment variables
		_ = os.Setenv("TEST_SERVER_PORT", "9090")
		_ = os.Setenv("TEST_SERVER_HOST", "0.0.0.0")
		_ = os.Setenv("TEST_APP_NAME", "test-app")
		_ = os.Setenv("TEST_APP_ENVIRONMENT", "production")
		defer func() {
			_ = os.Unsetenv("TEST_SERVER_PORT")
			_ = os.Unsetenv("TEST_SERVER_HOST")
			_ = os.Unsetenv("TEST_APP_NAME")
			_ = os.Unsetenv("TEST_APP_ENVIRONMENT")
		}()

		config, err := NewBuilder().
			WithDefaults().
			WithEnvironment("TEST").
			Build()

		if err != nil {
			t.Fatalf("Failed to build config: %v", err)
		}

		// Environment should override defaults
		if config.Server.Port != 9090 {
			t.Errorf("Expected port 9090 from env, got %d", config.Server.Port)
		}
		if config.Server.Host != "0.0.0.0" {
			t.Errorf("Expected host '0.0.0.0' from env, got '%s'", config.Server.Host)
		}
		if config.App.Name != "test-app" {
			t.Errorf("Expected app name 'test-app' from env, got '%s'", config.App.Name)
		}
		if config.App.Environment != "production" {
			t.Errorf("Expected environment 'production' from env, got '%s'", config.App.Environment)
		}
	})

	t.Run("BuildWithFile", func(t *testing.T) {
		// Create temporary config file
		configData := map[string]interface{}{
			"server": map[string]interface{}{
				"host": "api.example.com",
				"port": 3000,
			},
			"app": map[string]interface{}{
				"name":        "file-app",
				"version":     "2.0.0",
				"environment": "staging",
			},
		}

		tempFile := createTempConfigFile(t, configData)
		defer func() { _ = os.Remove(tempFile) }()

		config, err := NewBuilder().
			WithDefaults().
			WithFile(tempFile).
			Build()

		if err != nil {
			t.Fatalf("Failed to build config: %v", err)
		}

		// File should override defaults
		if config.Server.Host != "api.example.com" {
			t.Errorf("Expected host 'api.example.com' from file, got '%s'", config.Server.Host)
		}
		if config.Server.Port != 3000 {
			t.Errorf("Expected port 3000 from file, got %d", config.Server.Port)
		}
		if config.App.Name != "file-app" {
			t.Errorf("Expected app name 'file-app' from file, got '%s'", config.App.Name)
		}
		if config.App.Environment != "staging" {
			t.Errorf("Expected environment 'staging' from file, got '%s'", config.App.Environment)
		}
	})

	t.Run("PriorityOrder", func(t *testing.T) {
		// Environment should override file
		configData := map[string]interface{}{
			"server": map[string]interface{}{
				"port": 3000,
			},
		}

		tempFile := createTempConfigFile(t, configData)
		defer func() { _ = os.Remove(tempFile) }()

		// Set environment variable
		_ = os.Setenv("PRIORITY_SERVER_PORT", "4000")
		defer func() { _ = os.Unsetenv("PRIORITY_SERVER_PORT") }()

		config, err := NewBuilder().
			WithDefaults().
			WithFile(tempFile).
			WithEnvironment("PRIORITY").
			Build()

		if err != nil {
			t.Fatalf("Failed to build config: %v", err)
		}

		// Environment should win over file
		if config.Server.Port != 4000 {
			t.Errorf("Expected port 4000 from env (highest priority), got %d", config.Server.Port)
		}
	})
}

func TestEnvironmentSource(t *testing.T) {
	t.Run("LoadBasic", func(t *testing.T) {
		// Set test environment variables
		_ = os.Setenv("TEST_SIMPLE", "value")
		_ = os.Setenv("TEST_SERVER_HOST", "localhost")
		_ = os.Setenv("TEST_SERVER_PORT", "8080")
		defer func() {
			_ = os.Unsetenv("TEST_SIMPLE")
			_ = os.Unsetenv("TEST_SERVER_HOST")
			_ = os.Unsetenv("TEST_SERVER_PORT")
		}()

		source := NewEnvironmentSource("TEST")
		data, err := source.Load()
		if err != nil {
			t.Fatalf("Failed to load environment: %v", err)
		}

		// Check simple value
		if data["simple"] != "value" {
			t.Errorf("Expected simple='value', got '%v'", data["simple"])
		}

		// Check nested structure
		if serverData, ok := data["server"].(map[string]interface{}); ok {
			if serverData["host"] != "localhost" {
				t.Errorf("Expected server.host='localhost', got '%v'", serverData["host"])
			}
			if serverData["port"] != "8080" {
				t.Errorf("Expected server.port='8080', got '%v'", serverData["port"])
			}
		} else {
			t.Error("Expected server to be a map")
		}
	})
}

func TestFileSource(t *testing.T) {
	t.Run("LoadJSON", func(t *testing.T) {
		configData := map[string]interface{}{
			"server": map[string]interface{}{
				"host": "example.com",
				"port": 9000,
			},
			"custom": map[string]interface{}{
				"api_timeout": "30s",
			},
		}

		tempFile := createTempConfigFile(t, configData)
		defer func() { _ = os.Remove(tempFile) }()

		source := NewFileSource(tempFile)
		data, err := source.Load()
		if err != nil {
			t.Fatalf("Failed to load file: %v", err)
		}

		// Verify data structure
		if serverData, ok := data["server"].(map[string]interface{}); ok {
			if serverData["host"] != "example.com" {
				t.Errorf("Expected host='example.com', got '%v'", serverData["host"])
			}
			if serverData["port"] != float64(9000) { // JSON numbers are float64
				t.Errorf("Expected port=9000, got '%v'", serverData["port"])
			}
		} else {
			t.Error("Expected server to be a map")
		}
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		source := NewFileSource("nonexistent.json")
		data, err := source.Load()
		if err != nil {
			t.Fatalf("Should not error on non-existent file, got: %v", err)
		}

		if len(data) != 0 {
			t.Error("Expected empty data for non-existent file")
		}
	})
}

func TestConfigValidation(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		config := &Config{
			App: AppConfig{
				Name: "test-app",
				Environment: "development",
				LogLevel: "info",
			},
			Server: ServerConfig{
				Port: 8080,
			},
			Database: DatabaseConfig{
				Driver: "sqlite3",
				Path: "./test.db",
			},
		}

		err := config.Validate()
		if err != nil {
			t.Errorf("Valid config should not error: %v", err)
		}
	})

	t.Run("InvalidPort", func(t *testing.T) {
		config := &Config{
			App: AppConfig{
				Name: "test",
				Environment: "development",
				LogLevel: "info",
			},
			Server: ServerConfig{
				Port: 99999, // Invalid port
			},
			Database: DatabaseConfig{
				Driver: "sqlite3",
				Path: "./test.db",
			},
		}

		err := config.Validate()
		if err == nil {
			t.Error("Invalid port should cause validation error")
		}
	})

	t.Run("MissingAppName", func(t *testing.T) {
		config := &Config{
			App: AppConfig{
				// Missing name - should cause error
				Environment: "development",
				LogLevel: "info",
			},
			Server: ServerConfig{
				Port: 8080,
			},
			Database: DatabaseConfig{
				Driver: "sqlite3",
				Path: "./test.db",
			},
		}

		err := config.Validate()
		if err == nil {
			t.Error("Missing app name should cause validation error")
		}
	})
}

func TestConfigHelpers(t *testing.T) {
	config := &Config{
		App: AppConfig{
			Name:        "test-app",
			Version:     "1.0.0",
			Environment: "development",
			LogLevel:    "info",
		},
		Server: ServerConfig{
			Host: "api.example.com",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Driver: "sqlite3",
			Path:   "./test.db",
		},
	}

	t.Run("GetServerAddress", func(t *testing.T) {
		addr := config.GetServerAddress()
		expected := "api.example.com:8080"
		if addr != expected {
			t.Errorf("Expected server address '%s', got '%s'", expected, addr)
		}
	})

	t.Run("ToGoryuConfig", func(t *testing.T) {
		// Use the new adapter
		adapter := NewFrameworkAdapter(config)
		goryuCfg := adapter.ToGoryuConfig()
		
		// No type assertion needed, directly access fields
		if goryuCfg.AppName != "test-app" {
			t.Errorf("Expected AppName 'test-app', got '%s'", goryuCfg.AppName)
		}
	})
}

func TestConfigJSON(t *testing.T) {
	config := &Config{
		App: AppConfig{
			Name:        "test",
			Environment: "development",
			LogLevel:    "info",
		},
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Driver: "sqlite3",
			Path:   "./test.db",
		},
	}

	jsonStr, err := config.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert config to JSON: %v", err)
	}

	// Verify it's valid JSON by parsing it back
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Check some values
	if server, ok := parsed["server"].(map[string]interface{}); ok {
		if server["host"] != "localhost" {
			t.Errorf("Expected host='localhost' in JSON, got '%v'", server["host"])
		}
	} else {
		t.Error("Expected server object in JSON")
	}
	if app, ok := parsed["app"].(map[string]interface{}); ok {
		if app["name"] != "test" {
			t.Errorf("Expected app name='test' in JSON, got '%v'", app["name"])
		}
	} else {
		t.Error("Expected app object in JSON")
	}
}

func TestLoadConfig(t *testing.T) {
	t.Run("LoadWithDefaults", func(t *testing.T) {
		config, err := LoadConfigFromEnv("NONEXISTENT")
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Should have defaults
		if config.Server.Port != 8080 {
			t.Errorf("Expected default port 8080, got %d", config.Server.Port)
		}
	})
}

// Helper function to create temporary config files for testing
func createTempConfigFile(t *testing.T, data map[string]interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	tempFile := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(tempFile, jsonData, 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	return tempFile
}

func TestDurationParsing(t *testing.T) {
	config := &Config{}
	err := applyDefaults(config)
	if err != nil {
		t.Fatalf("Failed to apply defaults: %v", err)
	}

	// Test default duration values
	if config.Server.ReadTimeout != 30*time.Second {
		t.Errorf("Expected read timeout 30s, got %v", config.Server.ReadTimeout)
	}
	if config.Server.WriteTimeout != 30*time.Second {
		t.Errorf("Expected write timeout 30s, got %v", config.Server.WriteTimeout)
	}
	if config.Server.ShutdownTimeout != 30*time.Second {
		t.Errorf("Expected shutdown timeout 30s, got %v", config.Server.ShutdownTimeout)
	}
}

func BenchmarkConfigLoad(b *testing.B) {
	// Create a temporary config file
	configData := map[string]interface{}{
		"server": map[string]interface{}{
			"host": "localhost",
			"port": 8080,
		},
		"app": map[string]interface{}{
			"name": "bench-app",
		},
	}

	tempFile := filepath.Join(b.TempDir(), "config.json")
	jsonData, _ := json.Marshal(configData)
	_ = os.WriteFile(tempFile, jsonData, 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewBuilder().
			WithDefaults().
			WithFile(tempFile).
			WithEnvironment("BENCH").
			Build()
		if err != nil {
			b.Fatalf("Failed to build config: %v", err)
		}
	}
}
