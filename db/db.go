package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/arthurlch/goryu/config"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
)

// this was more complicated than I thought it would be...
// but I think it's better now, more secure and robust now,
// I feel i will need to go back here and refactor again in the future
// maybe add support for non sql DB ?? ? like mongodb or redis ??

type Connection struct {
	DB     *sql.DB
	Driver string
	DSN    string
}

func Connect(cfg *config.Config) (*Connection, error) {
	if err := cfg.Database.Validate(); err != nil {
		return nil, fmt.Errorf("database configuration validation failed: %w", err)
	}

	var dsn string
	var err error

	switch cfg.Database.Driver {
	case "sqlite3":
		dsn, err = buildSQLiteDSNFromConfig(&cfg.Database)
	case "postgres", "pgx":
		dsn, err = buildPostgresDSNFromConfig(&cfg.Database)
	case "mysql":
		dsn, err = buildMySQLDSNFromConfig(&cfg.Database)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Database.Driver)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to build DSN: %w", err)
	}

	db, err := sql.Open(cfg.Database.Driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.Database.ConnMaxIdleTime)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Connection{
		DB:     db,
		Driver: cfg.Database.Driver,
		DSN:    dsn,
	}, nil
}

func ConnectLegacy(cfg *config.LegacyConfig) (*Connection, error) {
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
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Connection{
		DB:     db,
		Driver: driver,
		DSN:    dsn,
	}, nil
}

func (c *Connection) Close() error {
	if c.DB != nil {
		return c.DB.Close()
	}
	return nil
}

func buildSQLiteDSNFromConfig(config *config.DatabaseConfig) (string, error) {
	// SECUCHECK: Validate and sanitize SQLite path to prevent directory traversal
	if err := validateSQLitePath(config.Path); err != nil {
		return "", err
	}
	return config.Path, nil
}

func buildPostgresDSNFromConfig(config *config.DatabaseConfig) (string, error) {
	// SECUCHECK: Validate all components before building DSN
	if err := validateDatabaseComponents(config.Username, config.Password, config.Host, config.Database, config.SSLMode); err != nil {
		return "", err
	}

	// SECUCHECK: Properly URL encode credentials and database name to prevent injection !
	username := url.QueryEscape(config.Username)
	password := url.QueryEscape(config.Password)
	database := url.QueryEscape(config.Database)
	host := url.QueryEscape(config.Host)
	sslmode := url.QueryEscape(config.SSLMode)

	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		username, password, host, config.Port, database, sslmode), nil
}

func buildMySQLDSNFromConfig(config *config.DatabaseConfig) (string, error) {
	// SECUCHECK: Validate all components before building DSN
	if err := validateDatabaseComponents(config.Username, config.Password, config.Host, config.Database, config.Charset); err != nil {
		return "", err
	}

	// SECUCHECK: Properly URL encode credentials and database name to prevent injection
	username := url.QueryEscape(config.Username)
	password := url.QueryEscape(config.Password)
	database := url.QueryEscape(config.Database)
	host := url.QueryEscape(config.Host)

	params := "parseTime=true"
	if config.Charset != "" {
		// SECUCHECK: Validate and escape charset parameter
		if err := validateCharset(config.Charset); err != nil {
			return "", err
		}
		params += "&charset=" + url.QueryEscape(config.Charset)
	}

	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
		username, password, host, config.Port, database, params), nil
}

func buildSQLiteDSN(config map[string]interface{}) (string, error) {
	path := getStringFromConfig(config, "path", "./app.db")
	return path, nil
}

// (legacy)
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

	// SECUCHECK: Validate all components before building DSN
	if err := validateDatabaseComponents(username, password, host, database, sslmode); err != nil {
		return "", err
	}

	// SECUCHECK: Properly URL encode credentials and database name to prevent injection
	usernameEncoded := url.QueryEscape(username)
	passwordEncoded := url.QueryEscape(password)
	databaseEncoded := url.QueryEscape(database)
	hostEncoded := url.QueryEscape(host)
	sslmodeEncoded := url.QueryEscape(sslmode)

	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		usernameEncoded, passwordEncoded, hostEncoded, port, databaseEncoded, sslmodeEncoded), nil
}

// (legacy)
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

	// SECUCHECK: Validate all components before building DSN
	if err := validateDatabaseComponents(username, password, host, database, ""); err != nil {
		return "", err
	}

	// SECUCHECK: Properly URL encode credentials and database name to prevent injection
	usernameEncoded := url.QueryEscape(username)
	passwordEncoded := url.QueryEscape(password)
	databaseEncoded := url.QueryEscape(database)
	hostEncoded := url.QueryEscape(host)

	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		usernameEncoded, passwordEncoded, hostEncoded, port, databaseEncoded), nil
}

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

// SECUCHECK validation functions

func validateDatabaseComponents(username, password, host, database, extra string) error {
	// SECUCHECK: Check for SQL injection patterns and dangerous characters
	dangerousChars := []string{
		";",    // SQL statement separator
		"--",   // SQL comment
		"/*",   // SQL comment start
		"*/",   // SQL comment end
		"'",    // SQL string delimiter
		"\"",   // SQL string delimiter
		"\x00", // Null byte
		"\n",   // Newline
		"\r",   // Carriage return
		"\t",   // Tab
		"\\",   // Backslash escape
	}

	components := map[string]string{
		"username": username,
		"password": password,
		"host":     host,
		"database": database,
		"extra":    extra,
	}

	for name, value := range components {
		if value == "" && (name == "username" || name == "database" || name == "host") {
			continue
		}

		for _, dangerous := range dangerousChars {
			if strings.Contains(value, dangerous) {
				return fmt.Errorf("invalid %s: contains dangerous character '%s' (potential SQL injection)", name, dangerous)
			}
		}

		// SECUCHECK: Check length limits to prevent buffer overflow attacks
		if len(value) > 255 {
			return fmt.Errorf("invalid %s: exceeds maximum length of 255 characters", name)
		}

		// SECUCHECK: Check for control characters
		for _, char := range value {
			if char < 32 && char != 9 && char != 10 && char != 13 { // Allow tab, LF, CR
				return fmt.Errorf("invalid %s: contains control character (potential injection)", name)
			}
		}
	}

	return nil
}

func validateSQLitePath(path string) error {
	if path == "" {
		return fmt.Errorf("SQLite path cannot be empty")
	}

	// SECUCHECK: traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("invalid SQLite path: contains directory traversal (..) - potential security risk")
	}

	// SECUCHECK: Clean the path and allow reasonable normalization
	cleaned := filepath.Clean(path)
	if strings.Contains(path, "..") && strings.Contains(cleaned, "..") {
		return fmt.Errorf("invalid SQLite path: contains directory traversal after normalization")
	}

	// SECUCHECK: Check for dangerous path components
	dangerousPaths := []string{
		"/etc/",
		"/var/",
		"/usr/",
		"/bin/",
		"/sbin/",
		"C:\\Windows\\",
		"C:\\Program Files\\",
	}

	for _, dangerous := range dangerousPaths {
		if strings.HasPrefix(strings.ToLower(cleaned), strings.ToLower(dangerous)) {
			return fmt.Errorf("invalid SQLite path: access to system directory '%s' not allowed", dangerous)
		}
	}

	return nil
}

func validateCharset(charset string) error {
	if charset == "" {
		return nil
	}

	// SECUCHECK: Whitelist allowed charset values to prevent injection
	allowedCharsets := map[string]bool{
		"utf8":    true,
		"utf8mb3": true,
		"utf8mb4": true,
		"latin1":  true,
		"ascii":   true,
		"binary":  true,
		"ucs2":    true,
		"utf16":   true,
		"utf32":   true,
	}

	if !allowedCharsets[strings.ToLower(charset)] {
		return fmt.Errorf("invalid charset: '%s' is not a recognized safe charset", charset)
	}

	return nil
}
