package config

import (
	"fmt"
	"testing"
)

// Example demonstrating the new configuration system
func Example() {
	// Load configuration using the new system
	config, err := NewBuilder().
		WithDefaults().
		WithFile("examples/example_config.json").
		WithEnvironment("GORYU").
		Build()
	
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}
	
	// Access different configuration sections
	fmt.Printf("App Name: %s\n", config.App.Name)
	fmt.Printf("Environment: %s\n", config.App.Environment)
	fmt.Printf("Server Address: %s\n", config.GetServerAddress())
	fmt.Printf("Database Driver: %s\n", config.Database.Driver)
	
	// Use framework adapter for goryu-specific config
	adapter := NewFrameworkAdapter(config)
	goryuConfig := adapter.ToGoryuConfig()
	fmt.Printf("Framework config available: %T\n", goryuConfig)
	
	// Validate configuration
	if err := config.Validate(); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
		return
	}
	
	fmt.Println("Configuration loaded and validated successfully!")
	
	// Output:
	// App Name: goryu-api
	// Environment: production
	// Server Address: 0.0.0.0:8080
	// Database Driver: postgres
	// Framework config available: config.GoryuConfigCompatibility
	// Configuration loaded and validated successfully!
}

// Example demonstrating validation errors
func ExampleConfig_Validate() {
	config := &Config{
		App: AppConfig{
			Name:        "", // Invalid: empty name
			Environment: "invalid", // Invalid: not a valid environment
			LogLevel:    "debug", // Valid
		},
		Server: ServerConfig{
			Port: 99999, // Invalid: port out of range
		},
		Database: DatabaseConfig{
			Driver: "postgres",
			// Missing required fields
		},
	}
	
	if err := config.Validate(); err != nil {
		fmt.Printf("Validation error: %v\n", err)
	}
	
	// Output:
	// Validation error: app config validation failed: app name is required
}

// Example demonstrating database configuration with validation
func ExampleDatabaseConfig_Validate() {
	// Valid PostgreSQL configuration
	dbConfig := DatabaseConfig{
		Driver:          "postgres",
		Host:            "localhost",
		Port:            5432,
		Database:        "myapp",
		Username:        "dbuser",
		Password:        "dbpass",
		SSLMode:         "require",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
	}
	
	if err := dbConfig.Validate(); err != nil {
		fmt.Printf("Database config error: %v\n", err)
		return
	}
	
	fmt.Println("Database configuration is valid")
	
	// Invalid configuration - missing required fields
	invalidConfig := DatabaseConfig{
		Driver: "postgres",
		// Missing database name and username
	}
	
	if err := invalidConfig.Validate(); err != nil {
		fmt.Printf("Invalid config error: %v\n", err)
	}
	
	// Output:
	// Database configuration is valid
	// Invalid config error: database name is required for postgres
}

// Test showing migration from legacy to new config format
func TestLegacyToNewMigration(t *testing.T) {
	// Old format config
	legacyConfig := &LegacyConfig{
		App: LegacyAppConfig{
			Name:              "old-app",
			Version:           "1.0.0",
			ServerHeader:      "OldServer/1.0",
			StrictRouting:     true,
			CaseSensitive:     false,
			DisableStartupMsg: false,
		},
		Server: LegacyServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Custom: map[string]interface{}{
			"database": map[string]interface{}{
				"driver":   "postgres",
				"host":     "db.example.com",
				"port":     float64(5432), // JSON numbers are float64
				"database": "olddb",
				"username": "olduser",
				"password": "oldpass",
			},
		},
	}
	
	// Convert to new format
	newConfig := legacyConfig.ToNewConfig()
	
	// Verify migration worked
	if newConfig.App.Name != "old-app" {
		t.Errorf("Expected app name 'old-app', got '%s'", newConfig.App.Name)
	}
	
	if newConfig.Framework.ServerHeader != "OldServer/1.0" {
		t.Errorf("Expected server header 'OldServer/1.0', got '%s'", newConfig.Framework.ServerHeader)
	}
	
	if newConfig.Database.Driver != "postgres" {
		t.Errorf("Expected database driver 'postgres', got '%s'", newConfig.Database.Driver)
	}
	
	if newConfig.Database.Host != "db.example.com" {
		t.Errorf("Expected database host 'db.example.com', got '%s'", newConfig.Database.Host)
	}
	
	// Validate the migrated config
	if err := newConfig.Validate(); err != nil {
		t.Errorf("Migrated config should be valid: %v", err)
	}
}

// Test showing conversion back to legacy format for backward compatibility
func TestNewToLegacyConversion(t *testing.T) {
	newConfig := &Config{
		App: AppConfig{
			Name:        "new-app",
			Version:     "2.0.0",
			Environment: "production",
			LogLevel:    "info",
			Custom: map[string]interface{}{
				"feature_flags": map[string]interface{}{
					"enable_cache": true,
				},
			},
		},
		Server: ServerConfig{
			Host: "api.example.com",
			Port: 443,
		},
		Database: DatabaseConfig{
			Driver:   "mysql",
			Host:     "mysql.example.com",
			Port:     3306,
			Database: "appdb",
			Username: "appuser",
			Password: "apppass",
		},
		Framework: FrameworkConfig{
			ServerHeader:          "NewServer/2.0",
			StrictRouting:         false,
			CaseSensitive:         true,
			DisableStartupMessage: false,
		},
	}
	
	// Convert to legacy format
	legacyConfig := ConvertNewToLegacy(newConfig)
	
	// Verify conversion
	if legacyConfig.App.Name != "new-app" {
		t.Errorf("Expected app name 'new-app', got '%s'", legacyConfig.App.Name)
	}
	
	if legacyConfig.App.ServerHeader != "NewServer/2.0" {
		t.Errorf("Expected server header 'NewServer/2.0', got '%s'", legacyConfig.App.ServerHeader)
	}
	
	// Check that database config was moved to custom section
	if dbCustom, ok := legacyConfig.Custom["database"].(map[string]interface{}); ok {
		if dbCustom["driver"] != "mysql" {
			t.Errorf("Expected database driver 'mysql' in custom, got '%v'", dbCustom["driver"])
		}
	} else {
		t.Error("Expected database config in custom section")
	}
	
	// Check that custom app settings were preserved
	if featureFlags, ok := legacyConfig.Custom["feature_flags"].(map[string]interface{}); ok {
		if !featureFlags["enable_cache"].(bool) {
			t.Error("Expected feature flag 'enable_cache' to be true")
		}
	} else {
		t.Error("Expected feature flags in custom section")
	}
}