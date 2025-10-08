package db

import (
	"strings"
	"testing"

	"github.com/arthurlch/goryu/config"
)

func TestDatabaseConnection(t *testing.T) {
	t.Run("SQLite connection with new config", func(t *testing.T) {
		cfg := &config.Config{
			Database: config.DatabaseConfig{
				Driver: "sqlite3",
				Path:   ":memory:", // Use in-memory database for testing
			},
		}

		conn, err := Connect(cfg)
		if err != nil {
			t.Fatalf("Failed to connect to SQLite: %v", err)
		}
		defer conn.Close()

		if conn.Driver != "sqlite3" {
			t.Errorf("Expected driver 'sqlite3', got '%s'", conn.Driver)
		}
	})

	t.Run("PostgreSQL connection validation", func(t *testing.T) {
		cfg := &config.Config{
			Database: config.DatabaseConfig{
				Driver:   "postgres",
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				Username: "testuser",
				Password: "testpass",
				SSLMode:  "disable",
			},
		}

		// This will fail to connect but should validate configuration
		_, err := Connect(cfg)
		// We expect a connection error, not a validation error
		if err != nil && !contains(err.Error(), "failed to ping database") && !contains(err.Error(), "failed to open database") {
			t.Errorf("Expected connection error, got validation error: %v", err)
		}
	})

	t.Run("MySQL connection validation", func(t *testing.T) {
		cfg := &config.Config{
			Database: config.DatabaseConfig{
				Driver:   "mysql",
				Host:     "localhost",
				Port:     3306,
				Database: "testdb",
				Username: "testuser",
				Password: "testpass",
				Charset:  "utf8mb4",
			},
		}

		// This will fail to connect but should validate configuration
		_, err := Connect(cfg)
		// We expect a connection error, not a validation error
		if err != nil && !contains(err.Error(), "failed to ping database") && !contains(err.Error(), "failed to open database") {
			t.Errorf("Expected connection error, got validation error: %v", err)
		}
	})

	t.Run("Invalid configuration", func(t *testing.T) {
		cfg := &config.Config{
			Database: config.DatabaseConfig{
				Driver: "postgres",
				// Missing required fields
			},
		}

		_, err := Connect(cfg)
		if err == nil {
			t.Error("Expected validation error for invalid config")
		}
		if !contains(err.Error(), "database configuration validation failed") {
			t.Errorf("Expected validation error, got: %v", err)
		}
	})

	t.Run("Connection pool settings", func(t *testing.T) {
		cfg := &config.Config{
			Database: config.DatabaseConfig{
				Driver:          "sqlite3",
				Path:            ":memory:",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: 0,
				ConnMaxIdleTime: 0,
			},
		}

		conn, err := Connect(cfg)
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer conn.Close()

		// The connection pool settings should be applied
		// We can't easily test the exact values, but we can verify the connection works
		if err := conn.DB.Ping(); err != nil {
			t.Errorf("Failed to ping database after setting pool settings: %v", err)
		}
	})
}

func TestLegacyDatabaseConnection(t *testing.T) {
	t.Run("Legacy SQLite connection", func(t *testing.T) {
		cfg := &config.LegacyConfig{
			Custom: map[string]interface{}{
				"database": map[string]interface{}{
					"driver": "sqlite3",
					"path":   ":memory:",
				},
			},
		}

		conn, err := ConnectLegacy(cfg)
		if err != nil {
			t.Fatalf("Failed to connect to SQLite with legacy config: %v", err)
		}
		defer conn.Close()

		if conn.Driver != "sqlite3" {
			t.Errorf("Expected driver 'sqlite3', got '%s'", conn.Driver)
		}
	})

	t.Run("Legacy config missing database section", func(t *testing.T) {
		cfg := &config.LegacyConfig{
			Custom: map[string]interface{}{
				// No database section
			},
		}

		_, err := ConnectLegacy(cfg)
		if err == nil {
			t.Error("Expected error for missing database config")
		}
		if !contains(err.Error(), "database configuration not found") {
			t.Errorf("Expected missing config error, got: %v", err)
		}
	})
}

func TestDSNBuilding(t *testing.T) {
	t.Run("SQLite DSN", func(t *testing.T) {
		cfg := &config.DatabaseConfig{
			Driver: "sqlite3",
			Path:   "./test.db",
		}

		dsn, err := buildSQLiteDSNFromConfig(cfg)
		if err != nil {
			t.Fatalf("Failed to build SQLite DSN: %v", err)
		}

		if dsn != "./test.db" {
			t.Errorf("Expected DSN './test.db', got '%s'", dsn)
		}
	})

	t.Run("PostgreSQL DSN", func(t *testing.T) {
		cfg := &config.DatabaseConfig{
			Driver:   "postgres",
			Host:     "localhost",
			Port:     5432,
			Database: "testdb",
			Username: "testuser",
			Password: "testpass",
			SSLMode:  "require",
		}

		dsn, err := buildPostgresDSNFromConfig(cfg)
		if err != nil {
			t.Fatalf("Failed to build PostgreSQL DSN: %v", err)
		}

		expected := "postgres://testuser:testpass@localhost:5432/testdb?sslmode=require"
		if dsn != expected {
			t.Errorf("Expected DSN '%s', got '%s'", expected, dsn)
		}
	})

	t.Run("MySQL DSN", func(t *testing.T) {
		cfg := &config.DatabaseConfig{
			Driver:   "mysql",
			Host:     "localhost",
			Port:     3306,
			Database: "testdb",
			Username: "testuser",
			Password: "testpass",
			Charset:  "utf8mb4",
		}

		dsn, err := buildMySQLDSNFromConfig(cfg)
		if err != nil {
			t.Fatalf("Failed to build MySQL DSN: %v", err)
		}

		expected := "testuser:testpass@tcp(localhost:3306)/testdb?parseTime=true&charset=utf8mb4"
		if dsn != expected {
			t.Errorf("Expected DSN '%s', got '%s'", expected, dsn)
		}
	})

	t.Run("MySQL DSN without charset", func(t *testing.T) {
		cfg := &config.DatabaseConfig{
			Driver:   "mysql",
			Host:     "localhost",
			Port:     3306,
			Database: "testdb",
			Username: "testuser",
			Password: "testpass",
			// No charset specified
		}

		dsn, err := buildMySQLDSNFromConfig(cfg)
		if err != nil {
			t.Fatalf("Failed to build MySQL DSN: %v", err)
		}

		expected := "testuser:testpass@tcp(localhost:3306)/testdb?parseTime=true"
		if dsn != expected {
			t.Errorf("Expected DSN '%s', got '%s'", expected, dsn)
		}
	})
}

func TestSecurityValidations(t *testing.T) {
	t.Run("SQL Injection Prevention", func(t *testing.T) {
		// Test PostgreSQL with malicious input
		cfg := &config.DatabaseConfig{
			Driver:   "postgres",
			Host:     "localhost",
			Port:     5432,
			Database: "test'; DROP TABLE users; --",
			Username: "user",
			Password: "pass",
			SSLMode:  "prefer",
		}

		_, err := buildPostgresDSNFromConfig(cfg)
		if err == nil {
			t.Error("Expected error for SQL injection attempt in database name")
		}
		if !strings.Contains(err.Error(), "dangerous character") {
			t.Errorf("Expected SQL injection warning, got: %v", err)
		}
	})

	t.Run("MySQL Injection Prevention", func(t *testing.T) {
		cfg := &config.DatabaseConfig{
			Driver:   "mysql",
			Host:     "localhost",
			Port:     3306,
			Database: "test",
			Username: "admin'/*",
			Password: "password",
		}

		_, err := buildMySQLDSNFromConfig(cfg)
		if err == nil {
			t.Error("Expected error for SQL injection attempt in username")
		}
	})

	t.Run("SQLite Path Traversal Prevention", func(t *testing.T) {
		cfg := &config.DatabaseConfig{
			Driver: "sqlite3",
			Path:   "../../../etc/passwd",
		}

		_, err := buildSQLiteDSNFromConfig(cfg)
		if err == nil {
			t.Error("Expected error for directory traversal attempt")
		}
		if !strings.Contains(err.Error(), "traversal") {
			t.Errorf("Expected directory traversal warning, got: %v", err)
		}
	})

	t.Run("Control Character Prevention", func(t *testing.T) {
		cfg := &config.DatabaseConfig{
			Driver:   "postgres",
			Host:     "localhost",
			Port:     5432,
			Database: "test\x00db",
			Username: "user",
			Password: "pass",
			SSLMode:  "prefer",
		}

		_, err := buildPostgresDSNFromConfig(cfg)
		if err == nil {
			t.Error("Expected error for null byte injection")
		}
	})

	t.Run("Charset Validation", func(t *testing.T) {
		// Test invalid charset
		err := validateCharset("malicious'; DROP TABLE")
		if err == nil {
			t.Error("Expected error for malicious charset")
		}

		// Test valid charset
		err = validateCharset("utf8mb4")
		if err != nil {
			t.Errorf("Expected no error for valid charset, got: %v", err)
		}
	})

	t.Run("System Path Prevention", func(t *testing.T) {
		cfg := &config.DatabaseConfig{
			Driver: "sqlite3",
			Path:   "/etc/shadow.db",
		}

		_, err := buildSQLiteDSNFromConfig(cfg)
		if err == nil {
			t.Error("Expected error for system directory access")
		}
		if !strings.Contains(err.Error(), "system directory") {
			t.Errorf("Expected system directory warning, got: %v", err)
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && 
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
				func() bool {
					for i := 0; i <= len(s)-len(substr); i++ {
						if s[i:i+len(substr)] == substr {
							return true
						}
					}
					return false
				}())))
}