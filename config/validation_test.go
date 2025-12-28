package config

import (
	"testing"
	"time"
)

func TestDatabaseConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    DatabaseConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "Valid SQLite config",
			config: DatabaseConfig{
				Driver: "sqlite3",
				Path:   "./test.db",
			},
			wantError: false,
		},
		{
			name: "Valid PostgreSQL config",
			config: DatabaseConfig{
				Driver:   "postgres",
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				Username: "testuser",
				Password: "testpass",
				SSLMode:  "prefer",
			},
			wantError: false,
		},
		{
			name: "Valid MySQL config",
			config: DatabaseConfig{
				Driver:   "mysql",
				Host:     "localhost",
				Port:     3306,
				Database: "testdb",
				Username: "testuser",
				Password: "testpass",
				Charset:  "utf8mb4",
			},
			wantError: false,
		},
		{
			name: "Invalid driver",
			config: DatabaseConfig{
				Driver: "unsupported",
			},
			wantError: true,
			errorMsg:  "invalid database driver",
		},
		{
			name: "SQLite missing path",
			config: DatabaseConfig{
				Driver: "sqlite3",
				Path:   "",
			},
			wantError: true,
			errorMsg:  "path is required for SQLite database",
		},
		{
			name: "PostgreSQL missing database",
			config: DatabaseConfig{
				Driver:   "postgres",
				Username: "testuser",
			},
			wantError: true,
			errorMsg:  "database name is required for postgres",
		},
		{
			name: "PostgreSQL missing username",
			config: DatabaseConfig{
				Driver:   "postgres",
				Database: "testdb",
			},
			wantError: true,
			errorMsg:  "username is required for postgres",
		},
		{
			name: "MySQL missing database",
			config: DatabaseConfig{
				Driver:   "mysql",
				Username: "testuser",
			},
			wantError: true,
			errorMsg:  "database name is required for mysql",
		},
		{
			name: "Invalid PostgreSQL SSL mode",
			config: DatabaseConfig{
				Driver:   "postgres",
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				Username: "testuser",
				SSLMode:  "invalid",
			},
			wantError: true,
			errorMsg:  "invalid SSL mode",
		},
		{
			name: "Invalid port",
			config: DatabaseConfig{
				Driver:   "postgres",
				Port:     -1,
				Database: "testdb",
				Username: "testuser",
			},
			wantError: true,
			errorMsg:  "invalid database port",
		},
		{
			name: "Invalid connection pool settings",
			config: DatabaseConfig{
				Driver:       "sqlite3",
				Path:         "./test.db",
				MaxOpenConns: 10,
				MaxIdleConns: 20, // More than MaxOpenConns
			},
			wantError: true,
			errorMsg:  "max_idle_conns (20) cannot be greater than max_open_conns (10)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestAppConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    AppConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "Valid app config",
			config: AppConfig{
				Name:        "test-app",
				Environment: "development",
				LogLevel:    "info",
			},
			wantError: false,
		},
		{
			name: "Missing app name",
			config: AppConfig{
				Name:        "",
				Environment: "development",
				LogLevel:    "info",
			},
			wantError: true,
			errorMsg:  "app name is required",
		},
		{
			name: "Invalid environment",
			config: AppConfig{
				Name:        "test-app",
				Environment: "invalid",
				LogLevel:    "info",
			},
			wantError: true,
			errorMsg:  "invalid environment",
		},
		{
			name: "Invalid log level",
			config: AppConfig{
				Name:        "test-app",
				Environment: "development",
				LogLevel:    "invalid",
			},
			wantError: true,
			errorMsg:  "invalid log level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestServerConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    ServerConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "Valid server config",
			config: ServerConfig{
				Host:            "localhost",
				Port:            8080,
				ReadTimeout:     30 * time.Second,
				WriteTimeout:    30 * time.Second,
				ShutdownTimeout: 30 * time.Second,
			},
			wantError: false,
		},
		{
			name: "Invalid port - too low",
			config: ServerConfig{
				Port: 0,
			},
			wantError: true,
			errorMsg:  "invalid server port",
		},
		{
			name: "Invalid port - too high",
			config: ServerConfig{
				Port: 70000,
			},
			wantError: true,
			errorMsg:  "invalid server port",
		},
		{
			name: "Negative timeout",
			config: ServerConfig{
				Port:        8080,
				ReadTimeout: -1 * time.Second,
			},
			wantError: true,
			errorMsg:  "read timeout cannot be negative",
		},
		{
			name: "TLS enabled but missing cert file",
			config: ServerConfig{
				Port: 8080,
				TLS: TLSConfig{
					Enabled: true,
					AutoTLS: false,
				},
			},
			wantError: true,
			errorMsg:  "cert_file is required",
		},
		{
			name: "Valid TLS config with AutoTLS",
			config: ServerConfig{
				Port: 8080,
				TLS: TLSConfig{
					Enabled: true,
					AutoTLS: true,
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestNewConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError bool
		errorMsg  string
	}{
		{
			name: "Valid complete config",
			config: Config{
				App: AppConfig{
					Name:        "test-app",
					Environment: "development",
					LogLevel:    "info",
				},
				Server: ServerConfig{
					Port: 8080,
				},
				Database: DatabaseConfig{
					Driver: "sqlite3",
					Path:   "./test.db",
				},
				Framework: FrameworkConfig{},
			},
			wantError: false,
		},
		{
			name: "Invalid app config",
			config: Config{
				App: AppConfig{
					Name:        "", // Invalid
					Environment: "development",
					LogLevel:    "info",
				},
				Server: ServerConfig{
					Port: 8080,
				},
				Database: DatabaseConfig{
					Driver: "sqlite3",
					Path:   "./test.db",
				},
			},
			wantError: true,
			errorMsg:  "app config validation failed",
		},
		{
			name: "Invalid database config",
			config: Config{
				App: AppConfig{
					Name:        "test-app",
					Environment: "development",
					LogLevel:    "info",
				},
				Server: ServerConfig{
					Port: 8080,
				},
				Database: DatabaseConfig{
					Driver: "postgres",
					// Missing required fields
				},
			},
			wantError: true,
			errorMsg:  "database config validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
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
