package db

import (
	"database/sql"
	"fmt"

	"github.com/arthurlch/goryu/config"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
)

// Connection holds database connection and metadata
type Connection struct {
	DB     *sql.DB
	Driver string
	DSN    string
}

// Connect creates a database connection from config
func Connect(cfg *config.Config) (*Connection, error) {
	// Check if database config exists in custom section
	dbConfig, ok := cfg.Custom["database"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("database configuration not found - add 'database' section to 'custom' in config")
	}

	driver := getStringFromConfig(dbConfig, "driver", "sqlite3")

	var dsn string
	var err error

	switch driver {
	case "sqlite3":
		dsn, err = buildSQLiteDSN(dbConfig)
	case "postgres", "pgx":
		dsn, err = buildPostgresDSN(dbConfig)
	case "mysql":
		dsn, err = buildMySQLDSN(dbConfig)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to build DSN: %w", err)
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		if err := db.Close(); err != nil {
			fmt.Printf("Error closing database: %v\n", err)
		}
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Connection{
		DB:     db,
		Driver: driver,
		DSN:    dsn,
	}, nil
}

// Close closes the database connection
func (c *Connection) Close() error {
	if c.DB != nil {
		return c.DB.Close()
	}
	return nil
}

// buildSQLiteDSN builds SQLite connection string
func buildSQLiteDSN(config map[string]interface{}) (string, error) {
	path := getStringFromConfig(config, "path", "./app.db")
	return path, nil
}

// buildPostgresDSN builds PostgreSQL connection string
func buildPostgresDSN(config map[string]interface{}) (string, error) {
	host := getStringFromConfig(config, "host", "localhost")
	port := getIntFromConfig(config, "port", 5432)
	database := getStringFromConfig(config, "database", "")
	username := getStringFromConfig(config, "username", "")
	password := getStringFromConfig(config, "password", "")
	sslmode := getStringFromConfig(config, "sslmode", "prefer")

	if database == "" {
		return "", fmt.Errorf("database name is required for PostgreSQL")
	}
	if username == "" {
		return "", fmt.Errorf("username is required for PostgreSQL")
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		username, password, host, port, database, sslmode), nil
}

// buildMySQLDSN builds MySQL connection string
func buildMySQLDSN(config map[string]interface{}) (string, error) {
	host := getStringFromConfig(config, "host", "localhost")
	port := getIntFromConfig(config, "port", 3306)
	database := getStringFromConfig(config, "database", "")
	username := getStringFromConfig(config, "username", "")
	password := getStringFromConfig(config, "password", "")

	if database == "" {
		return "", fmt.Errorf("database name is required for MySQL")
	}
	if username == "" {
		return "", fmt.Errorf("username is required for MySQL")
	}

	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		username, password, host, port, database), nil
}

// Helper functions for extracting values from config
func getStringFromConfig(config map[string]interface{}, key, defaultValue string) string {
	if value, ok := config[key].(string); ok {
		return value
	}
	return defaultValue
}

func getIntFromConfig(config map[string]interface{}, key string, defaultValue int) int {
	if value, ok := config[key].(float64); ok {
		return int(value)
	}
	if value, ok := config[key].(int); ok {
		return value
	}
	return defaultValue
}
