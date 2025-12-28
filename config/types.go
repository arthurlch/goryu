package config

import (
	"fmt"
	"time"
)

type AppConfig struct {
	Name    string `json:"name" env:"NAME" default:"goryu-app" validate:"required"`
	Version string `json:"version" env:"VERSION" default:"1.0.0"`

	Environment string `json:"environment" env:"ENVIRONMENT" default:"development" validate:"oneof=development staging production"`

	LogLevel string `json:"log_level" env:"LOG_LEVEL" default:"info" validate:"oneof=debug info warn error"`

	Custom map[string]interface{} `json:"custom,omitempty" env:"CUSTOM"`
}

type ServerConfig struct {
	Host            string        `json:"host" env:"HOST" default:"localhost"`
	Port            int           `json:"port" env:"PORT" default:"8080" validate:"min=1,max=65535"`
	ReadTimeout     time.Duration `json:"read_timeout" env:"READ_TIMEOUT" default:"30s"`
	WriteTimeout    time.Duration `json:"write_timeout" env:"WRITE_TIMEOUT" default:"30s"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout" env:"SHUTDOWN_TIMEOUT" default:"30s"`

	TLS TLSConfig `json:"tls,omitempty" env:"TLS"`
}

type TLSConfig struct {
	Enabled  bool   `json:"enabled" env:"ENABLED" default:"false"`
	CertFile string `json:"cert_file" env:"CERT_FILE"`
	KeyFile  string `json:"key_file" env:"KEY_FILE"`
	AutoTLS  bool   `json:"auto_tls" env:"AUTO_TLS" default:"false"`
}

type DatabaseConfig struct {
	Driver   string `json:"driver" env:"DRIVER" default:"sqlite3" validate:"oneof=sqlite3 postgres mysql"`
	Host     string `json:"host" env:"HOST" default:"localhost"`
	Port     int    `json:"port" env:"PORT" default:"5432"`
	Database string `json:"database" env:"DATABASE" validate:"required_unless=Driver sqlite3"`
	Username string `json:"username" env:"USERNAME" validate:"required_unless=Driver sqlite3"`
	Password string `json:"password" env:"PASSWORD"`
	Path     string `json:"path" env:"PATH" default:"./app.db" validate:"required_if=Driver sqlite3"`

	MaxOpenConns    int           `json:"max_open_conns" env:"MAX_OPEN_CONNS" default:"25"`
	MaxIdleConns    int           `json:"max_idle_conns" env:"MAX_IDLE_CONNS" default:"5"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" env:"CONN_MAX_LIFETIME" default:"1h"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time" env:"CONN_MAX_IDLE_TIME" default:"30m"`

	SSLMode string `json:"ssl_mode" env:"SSL_MODE" default:"prefer" validate:"omitempty,oneof=disable require verify-ca verify-full prefer"`

	ParseTime bool   `json:"parse_time" env:"PARSE_TIME" default:"true"`
	Charset   string `json:"charset" env:"CHARSET" default:"utf8mb4"`
}

type FrameworkConfig struct {
	ServerHeader string `json:"server_header" env:"SERVER_HEADER"`

	StrictRouting         bool `json:"strict_routing" env:"STRICT_ROUTING" default:"false"`
	CaseSensitive         bool `json:"case_sensitive" env:"CASE_SENSITIVE" default:"false"`
	RedirectTrailingSlash bool `json:"redirect_trailing_slash" env:"REDIRECT_TRAILING_SLASH" default:"true"`
	EnableHEADFallback    bool `json:"enable_head_fallback" env:"ENABLE_HEAD_FALLBACK" default:"true"`

	DisableStartupMessage bool `json:"disable_startup_msg" env:"DISABLE_STARTUP_MSG" default:"false"`
}

type Config struct {
	App AppConfig `json:"app" env:"APP"`

	Server ServerConfig `json:"server" env:"SERVER"`

	Database DatabaseConfig `json:"database" env:"DATABASE"`

	Framework FrameworkConfig `json:"framework" env:"FRAMEWORK"`
}

func (c *Config) Validate() error {
	if err := c.App.Validate(); err != nil {
		return fmt.Errorf("app config validation failed: %w", err)
	}

	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server config validation failed: %w", err)
	}

	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database config validation failed: %w", err)
	}

	if err := c.Framework.Validate(); err != nil {
		return fmt.Errorf("framework config validation failed: %w", err)
	}

	return nil
}

func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

func (a *AppConfig) Validate() error {
	if a.Name == "" {
		return fmt.Errorf("app name is required")
	}

	validEnvs := map[string]bool{
		"development": true,
		"staging":     true,
		"production":  true,
	}
	if !validEnvs[a.Environment] {
		return fmt.Errorf("invalid environment: %s (must be development, staging, or production)", a.Environment)
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[a.LogLevel] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", a.LogLevel)
	}

	return nil
}

func (s *ServerConfig) Validate() error {
	if s.Port < 1 || s.Port > 65535 {
		return fmt.Errorf("invalid server port: %d (must be between 1 and 65535)", s.Port)
	}

	if s.ReadTimeout < 0 {
		return fmt.Errorf("read timeout cannot be negative")
	}

	if s.WriteTimeout < 0 {
		return fmt.Errorf("write timeout cannot be negative")
	}

	if s.ShutdownTimeout < 0 {
		return fmt.Errorf("shutdown timeout cannot be negative")
	}

	if s.TLS.Enabled {
		if err := s.TLS.Validate(); err != nil {
			return fmt.Errorf("TLS config validation failed: %w", err)
		}
	}

	return nil
}

func (t *TLSConfig) Validate() error {
	if !t.Enabled {
		return nil
	}

	if !t.AutoTLS {
		if t.CertFile == "" {
			return fmt.Errorf("cert_file is required when TLS is enabled and auto_tls is false")
		}
		if t.KeyFile == "" {
			return fmt.Errorf("key_file is required when TLS is enabled and auto_tls is false")
		}
	}

	return nil
}

func (d *DatabaseConfig) Validate() error {
	validDrivers := map[string]bool{
		"sqlite3":  true,
		"postgres": true,
		"mysql":    true,
	}
	if !validDrivers[d.Driver] {
		return fmt.Errorf("invalid database driver: %s (must be sqlite3, postgres, or mysql)", d.Driver)
	}

	switch d.Driver {
	case "sqlite3":
		if d.Path == "" {
			return fmt.Errorf("path is required for SQLite database")
		}

	case "postgres", "mysql":
		if d.Database == "" {
			return fmt.Errorf("database name is required for %s", d.Driver)
		}
		if d.Username == "" {
			return fmt.Errorf("username is required for %s", d.Driver)
		}
		if d.Port < 1 || d.Port > 65535 {
			return fmt.Errorf("invalid database port: %d (must be between 1 and 65535)", d.Port)
		}
	}

	if d.Driver == "postgres" && d.SSLMode != "" {
		validSSLModes := map[string]bool{
			"disable":     true,
			"require":     true,
			"verify-ca":   true,
			"verify-full": true,
			"prefer":      true,
		}
		if !validSSLModes[d.SSLMode] {
			return fmt.Errorf("invalid SSL mode: %s", d.SSLMode)
		}
	}

	if d.MaxOpenConns < 0 {
		return fmt.Errorf("max_open_conns cannot be negative")
	}
	if d.MaxIdleConns < 0 {
		return fmt.Errorf("max_idle_conns cannot be negative")
	}
	if d.MaxIdleConns > d.MaxOpenConns && d.MaxOpenConns > 0 {
		return fmt.Errorf("max_idle_conns (%d) cannot be greater than max_open_conns (%d)", d.MaxIdleConns, d.MaxOpenConns)
	}

	return nil
}

func (f *FrameworkConfig) Validate() error {
	// Framework config validation is minimal as most settings are just boolean flags
	// Could add validation for server header format if needed ??
	return nil
}
