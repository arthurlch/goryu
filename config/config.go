package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Manager struct {
	sources []Source
	cache   map[string]interface{}
}

type Source interface {
	Load() (map[string]interface{}, error)
	Name() string
	Priority() int 
}

// legacy types for backward compatibility - DEPRECATED
// Need to use the new types in types.go instead
// Cause it's fine to shoot yourself in the foot once in a while
// happens to the best of us.... right?

// DEPRECATEDO
type LegacyConfig struct {
	App    LegacyAppConfig    `json:"app" env:"APP" yaml:"app"`
	Server LegacyServerConfig `json:"server" env:"SERVER" yaml:"server"`
	Custom map[string]interface{} `json:"custom" env:"CUSTOM" yaml:"custom"`
}

type LegacyAppConfig struct {
	Name              string `json:"name" env:"NAME" default:"goryu-app"`
	Version           string `json:"version" env:"VERSION" default:"1.0.0"`
	ServerHeader      string `json:"server_header" env:"SERVER_HEADER"`
	StrictRouting     bool   `json:"strict_routing" env:"STRICT_ROUTING" default:"false"`
	CaseSensitive     bool   `json:"case_sensitive" env:"CASE_SENSITIVE" default:"false"`
	DisableStartupMsg bool   `json:"disable_startup_msg" env:"DISABLE_STARTUP_MSG" default:"false"`
}

type LegacyServerConfig struct {
	Host            string        `json:"host" env:"HOST" default:"localhost"`
	Port            int           `json:"port" env:"PORT" default:"8080"`
	ReadTimeout     time.Duration `json:"read_timeout" env:"READ_TIMEOUT" default:"30s"`
	WriteTimeout    time.Duration `json:"write_timeout" env:"WRITE_TIMEOUT" default:"30s"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout" env:"SHUTDOWN_TIMEOUT" default:"30s"`
}

// ToNewConfig converts legacy config to new config structure
func (c *LegacyConfig) ToNewConfig() *Config {
	config := &Config{
		App: AppConfig{
			Name:        c.App.Name,
			Version:     c.App.Version,
			Environment: "development", // default
			LogLevel:    "info",        // default
			Custom:      c.Custom,
		},
		Server: ServerConfig{
			Host:            c.Server.Host,
			Port:            c.Server.Port,
			ReadTimeout:     c.Server.ReadTimeout,
			WriteTimeout:    c.Server.WriteTimeout,
			ShutdownTimeout: c.Server.ShutdownTimeout,
		},
		Framework: FrameworkConfig{
			ServerHeader:          c.App.ServerHeader,
			StrictRouting:         c.App.StrictRouting,
			CaseSensitive:         c.App.CaseSensitive,
			DisableStartupMessage: c.App.DisableStartupMsg,
		},
		Database: DatabaseConfig{
			Driver: "sqlite3", // default (sqlite pretty good)
			Path:   "./app.db",
		},
	}
	
	if dbConfig, ok := c.Custom["database"].(map[string]interface{}); ok {
		migrateDBConfig(&config.Database, dbConfig)
	}
	
	return config
}

// migrateDBConfig migrates a database config from custom map to the new structure
func migrateDBConfig(target *DatabaseConfig, source map[string]interface{}) {
	if driver, ok := source["driver"].(string); ok {
		target.Driver = driver
	}
	if host, ok := source["host"].(string); ok {
		target.Host = host
	}
	if port, ok := source["port"].(float64); ok {
		target.Port = int(port)
	}
	if database, ok := source["database"].(string); ok {
		target.Database = database
	}
	if username, ok := source["username"].(string); ok {
		target.Username = username
	}
	if password, ok := source["password"].(string); ok {
		target.Password = password
	}
	if path, ok := source["path"].(string); ok {
		target.Path = path
	}
	if sslMode, ok := source["sslmode"].(string); ok {
		target.SSLMode = sslMode
	}
}

func NewManager() *Manager {
	return &Manager{
		sources: make([]Source, 0),
		cache:   make(map[string]interface{}),
	}
}

func (m *Manager) AddSource(source Source) *Manager {
	m.sources = append(m.sources, source)
	return m
}

func (m *Manager) Load() (*Config, error) {
	config := &Config{
		App: AppConfig{
			Custom: make(map[string]interface{}),
		},
	}

	if err := applyDefaults(config); err != nil {
		return nil, fmt.Errorf("failed to apply defaults: %w", err)
	}

	sources := m.getSortedSources()
	for _, source := range sources {
		data, err := source.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load from source %s: %w", source.Name(), err)
		}

		if err := mergeData(config, data); err != nil {
			return nil, fmt.Errorf("failed to merge data from source %s: %w", source.Name(), err)
		}
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

func (m *Manager) LoadLegacy() (*LegacyConfig, error) {
	config := &LegacyConfig{
		Custom: make(map[string]interface{}),
	}

	if err := applyDefaults(config); err != nil {
		return nil, fmt.Errorf("failed to apply defaults: %w", err)
	}

	sources := m.getSortedSources()
	for _, source := range sources {
		data, err := source.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load from source %s: %w", source.Name(), err)
		}

		if err := mergeData(config, data); err != nil {
			return nil, fmt.Errorf("failed to merge data from source %s: %w", source.Name(), err)
		}
	}

	if err := validateLegacyConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

func (m *Manager) getSortedSources() []Source {
	sources := make([]Source, len(m.sources))
	copy(sources, m.sources)

	// bubble sort by priority 
	// if I grind leetcode I will make it quickSort
	for i := 0; i < len(sources)-1; i++ {
		for j := 0; j < len(sources)-i-1; j++ {
			if sources[j].Priority() > sources[j+1].Priority() {
				sources[j], sources[j+1] = sources[j+1], sources[j]
			}
		}
	}

	return sources
}

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

		if field.Kind() == reflect.Struct && fieldType.Type.Name() != "Duration" {
			if err := applyDefaultsToValue(field); err != nil {
				return err
			}
			continue
		}

		defaultValue := fieldType.Tag.Get("default")
		if defaultValue == "" {
			continue
		}

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

		jsonTag := fieldType.Tag.Get("json")
		fieldName := strings.Split(jsonTag, ",")[0]
		if fieldName == "" {
			fieldName = strings.ToLower(fieldType.Name)
		}

		fullName := fieldName
		if prefix != "" {
			fullName = prefix + "." + fieldName
		}

		
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

func validateLegacyConfig(config *LegacyConfig) error {
	if config.Server.Port < 1 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	if config.App.Name == "" {
		config.App.Name = "goryu-app"
	}

	return nil
}

func (c *Config) ToJSON() (string, error) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

