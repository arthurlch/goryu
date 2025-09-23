package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Manager handles configuration loading from multiple sources
type Manager struct {
	sources []Source
	cache   map[string]interface{}
}

// Source represents a configuration source (env, file, etc.)
type Source interface {
	Load() (map[string]interface{}, error)
	Name() string
	Priority() int // Higher priority sources override lower ones
}

// Config represents the application configuration
type Config struct {
	// App settings
	App AppConfig `json:"app" env:"APP" yaml:"app"`

	// Server settings
	Server ServerConfig `json:"server" env:"SERVER" yaml:"server"`

	// Environment
	Environment string `json:"environment" env:"ENVIRONMENT" default:"development"`

	// Custom app-specific configuration can be added here by users
	Custom map[string]interface{} `json:"custom" env:"CUSTOM" yaml:"custom"`
}

type AppConfig struct {
	Name              string `json:"name" env:"NAME" default:"goryu-app"`
	Version           string `json:"version" env:"VERSION" default:"1.0.0"`
	ServerHeader      string `json:"server_header" env:"SERVER_HEADER"`
	StrictRouting     bool   `json:"strict_routing" env:"STRICT_ROUTING" default:"false"`
	CaseSensitive     bool   `json:"case_sensitive" env:"CASE_SENSITIVE" default:"false"`
	DisableStartupMsg bool   `json:"disable_startup_msg" env:"DISABLE_STARTUP_MSG" default:"false"`
}

type ServerConfig struct {
	Host            string        `json:"host" env:"HOST" default:"localhost"`
	Port            int           `json:"port" env:"PORT" default:"8080"`
	ReadTimeout     time.Duration `json:"read_timeout" env:"READ_TIMEOUT" default:"30s"`
	WriteTimeout    time.Duration `json:"write_timeout" env:"WRITE_TIMEOUT" default:"30s"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout" env:"SHUTDOWN_TIMEOUT" default:"30s"`
}

// GoryuConfig represents the configuration for goryu.New()
type GoryuConfig struct {
	AppName               string
	ServerHeader          string
	StrictRouting         bool
	CaseSensitive         bool
	DisableStartupMessage bool
}

// ToGoryuConfig converts this config to GoryuConfig for framework initialization
func (c *Config) ToGoryuConfig() GoryuConfig {
	return GoryuConfig{
		AppName:               c.App.Name,
		ServerHeader:          c.App.ServerHeader,
		StrictRouting:         c.App.StrictRouting,
		CaseSensitive:         c.App.CaseSensitive,
		DisableStartupMessage: c.App.DisableStartupMsg,
	}
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	return &Manager{
		sources: make([]Source, 0),
		cache:   make(map[string]interface{}),
	}
}

// AddSource adds a configuration source to the manager
func (m *Manager) AddSource(source Source) *Manager {
	m.sources = append(m.sources, source)
	return m
}

// Load loads configuration from all sources and returns a merged config
func (m *Manager) Load() (*Config, error) {
	// Create default config
	config := &Config{
		Custom: make(map[string]interface{}), // Initialize Custom field
	}

	// Apply defaults first
	if err := applyDefaults(config); err != nil {
		return nil, fmt.Errorf("failed to apply defaults: %w", err)
	}

	// Load from sources in priority order (lower priority first)
	sources := m.getSortedSources()
	for _, source := range sources {
		data, err := source.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load from source %s: %w", source.Name(), err)
		}

		// Merge data into config
		if err := mergeData(config, data); err != nil {
			return nil, fmt.Errorf("failed to merge data from source %s: %w", source.Name(), err)
		}
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// getSortedSources returns sources sorted by priority (lowest first)
func (m *Manager) getSortedSources() []Source {
	sources := make([]Source, len(m.sources))
	copy(sources, m.sources)

	// Simple bubble sort by priority
	for i := 0; i < len(sources)-1; i++ {
		for j := 0; j < len(sources)-i-1; j++ {
			if sources[j].Priority() > sources[j+1].Priority() {
				sources[j], sources[j+1] = sources[j+1], sources[j]
			}
		}
	}

	return sources
}

// applyDefaults applies default values to the config struct
func applyDefaults(config interface{}) error {
	return applyDefaultsToValue(reflect.ValueOf(config).Elem())
}

func applyDefaultsToValue(v reflect.Value) error {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if !field.CanSet() {
			continue
		}

		// Handle nested structs
		if field.Kind() == reflect.Struct && fieldType.Type.Name() != "Duration" {
			if err := applyDefaultsToValue(field); err != nil {
				return err
			}
			continue
		}

		// Get default value from tag
		defaultValue := fieldType.Tag.Get("default")
		if defaultValue == "" {
			continue
		}

		// Set default value based on field type
		if err := setFieldValue(field, defaultValue); err != nil {
			return fmt.Errorf("failed to set default for field %s: %w", fieldType.Name, err)
		}
	}

	return nil
}

func setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			duration, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			field.SetInt(int64(duration))
		} else {
			intValue, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return err
			}
			field.SetInt(intValue)
		}

	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolValue)

	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatValue)

	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			if value != "" {
				values := strings.Split(value, ",")
				for i, v := range values {
					values[i] = strings.TrimSpace(v)
				}
				field.Set(reflect.ValueOf(values))
			}
		}

	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}

// mergeData merges configuration data into the target struct
func mergeData(target interface{}, data map[string]interface{}) error {
	return mergeDataToValue(reflect.ValueOf(target).Elem(), data, "")
}

func mergeDataToValue(v reflect.Value, data map[string]interface{}, prefix string) error {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if !field.CanSet() {
			continue
		}

		// Get field name from json tag or use field name
		jsonTag := fieldType.Tag.Get("json")
		fieldName := strings.Split(jsonTag, ",")[0]
		if fieldName == "" {
			fieldName = strings.ToLower(fieldType.Name)
		}

		fullName := fieldName
		if prefix != "" {
			fullName = prefix + "." + fieldName
		}

		// Handle nested structs
		if field.Kind() == reflect.Struct && fieldType.Type.Name() != "Duration" {
			if nestedData, exists := data[fieldName]; exists {
				if nestedMap, ok := nestedData.(map[string]interface{}); ok {
					if err := mergeDataToValue(field, nestedMap, fullName); err != nil {
						return err
					}
				}
			}
			continue
		}

		// Set field value from data
		if value, exists := data[fieldName]; exists {
			if err := setFieldFromInterface(field, value); err != nil {
				return fmt.Errorf("failed to set field %s: %w", fullName, err)
			}
		}
	}

	return nil
}

func setFieldFromInterface(field reflect.Value, value interface{}) error {
	switch field.Kind() {
	case reflect.String:
		if str, ok := value.(string); ok {
			field.SetString(str)
		} else {
			field.SetString(fmt.Sprintf("%v", value))
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			if str, ok := value.(string); ok {
				duration, err := time.ParseDuration(str)
				if err != nil {
					return err
				}
				field.SetInt(int64(duration))
			}
		} else {
			switch v := value.(type) {
			case int:
				field.SetInt(int64(v))
			case int64:
				field.SetInt(v)
			case float64:
				field.SetInt(int64(v))
			case string:
				intValue, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return err
				}
				field.SetInt(intValue)
			}
		}

	case reflect.Bool:
		if boolValue, ok := value.(bool); ok {
			field.SetBool(boolValue)
		} else if str, ok := value.(string); ok {
			boolValue, err := strconv.ParseBool(str)
			if err != nil {
				return err
			}
			field.SetBool(boolValue)
		}

	case reflect.Float32, reflect.Float64:
		switch v := value.(type) {
		case float64:
			field.SetFloat(v)
		case float32:
			field.SetFloat(float64(v))
		case string:
			floatValue, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}
			field.SetFloat(floatValue)
		}

	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			switch v := value.(type) {
			case []string:
				field.Set(reflect.ValueOf(v))
			case []interface{}:
				strings := make([]string, len(v))
				for i, item := range v {
					strings[i] = fmt.Sprintf("%v", item)
				}
				field.Set(reflect.ValueOf(strings))
			case string:
				if v != "" {
					values := strings.Split(v, ",")
					for i, val := range values {
						values[i] = strings.TrimSpace(val)
					}
					field.Set(reflect.ValueOf(values))
				}
			}
		}
	}

	return nil
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Basic validation
	if config.Server.Port < 1 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	if config.App.Name == "" {
		config.App.Name = "goryu-app"
	}

	return nil
}

// ToJSON returns the configuration as a JSON string
func (c *Config) ToJSON() (string, error) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetServerAddress returns the full server address
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
