package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnvironmentSource loads configuration from environment variables
type EnvironmentSource struct {
	prefix string
}

// NewEnvironmentSource creates a new environment variable source
func NewEnvironmentSource(prefix string) *EnvironmentSource {
	return &EnvironmentSource{
		prefix: prefix,
	}
}

func (e *EnvironmentSource) Load() (map[string]interface{}, error) {
	data := make(map[string]interface{})

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		// Check if key starts with prefix
		if e.prefix != "" && !strings.HasPrefix(key, e.prefix+"_") {
			continue
		}

		// Remove prefix
		if e.prefix != "" {
			key = strings.TrimPrefix(key, e.prefix+"_")
		}

		// Convert key to lowercase and replace underscores
		key = strings.ToLower(key)

		// Handle nested keys (e.g., SERVER_HOST -> server.host)
		if strings.Contains(key, "_") {
			parts := strings.Split(key, "_")
			if len(parts) >= 2 {
				// Create nested structure
				current := data
				for i, part := range parts[:len(parts)-1] {
					if _, exists := current[part]; !exists {
						current[part] = make(map[string]interface{})
					}
					if i == len(parts)-2 {
						// Last level - set the value
						if nested, ok := current[part].(map[string]interface{}); ok {
							nested[parts[len(parts)-1]] = value
						}
					} else {
						// Continue nesting
						if nested, ok := current[part].(map[string]interface{}); ok {
							current = nested
						}
					}
				}
			} else {
				data[key] = value
			}
		} else {
			data[key] = value
		}
	}

	return data, nil
}

func (e *EnvironmentSource) Name() string {
	return "environment"
}

func (e *EnvironmentSource) Priority() int {
	return 100 // High priority - environment variables override everything
}

// FileSource loads configuration from JSON/YAML files
type FileSource struct {
	filePath string
}

// NewFileSource creates a new file-based configuration source
func NewFileSource(filePath string) *FileSource {
	return &FileSource{
		filePath: filePath,
	}
}

func (f *FileSource) Load() (map[string]interface{}, error) {
	data := make(map[string]interface{})

	// Check if file exists
	if _, err := os.Stat(f.filePath); os.IsNotExist(err) {
		// File doesn't exist - return empty data
		return data, nil
	}

	// Read file content
	content, err := os.ReadFile(f.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", f.filePath, err)
	}

	// Parse based on file extension
	ext := strings.ToLower(filepath.Ext(f.filePath))
	switch ext {
	case ".json":
		if err := _ = json.Unmarshal(content, &data); err != nil {
			return nil, fmt.Errorf("failed to parse JSON file %s: %w", f.filePath, err)
		}

	case ".yaml", ".yml":
		// For YAML support, you would use gopkg.in/yaml.v2 or similar
		// For now, we'll just return an error
		return nil, fmt.Errorf("YAML support not implemented yet")

	default:
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}

	return data, nil
}

func (f *FileSource) Name() string {
	return fmt.Sprintf("file:%s", f.filePath)
}

func (f *FileSource) Priority() int {
	return 50 // Medium priority - files override defaults but not environment
}

// DefaultSource provides default configuration values
type DefaultSource struct {
	defaults map[string]interface{}
}

// NewDefaultSource creates a new default source with predefined values
func NewDefaultSource() *DefaultSource {
	defaults := map[string]interface{}{
		"app": map[string]interface{}{
			"name":    "goryu-app",
			"version": "1.0.0",
		},
		"server": map[string]interface{}{
			"host": "localhost",
			"port": 8080,
		},
		"environment": "development",
	}

	return &DefaultSource{
		defaults: defaults,
	}
}

func (d *DefaultSource) Load() (map[string]interface{}, error) {
	// Return a copy to avoid modification
	data := make(map[string]interface{})
	copyMap(d.defaults, data)
	return data, nil
}

func (d *DefaultSource) Name() string {
	return "defaults"
}

func (d *DefaultSource) Priority() int {
	return 1 // Lowest priority - defaults are overridden by everything
}

// copyMap performs a deep copy of a map
func copyMap(src, dst map[string]interface{}) {
	for key, value := range src {
		switch v := value.(type) {
		case map[string]interface{}:
			nested := make(map[string]interface{})
			copyMap(v, nested)
			dst[key] = nested
		default:
			dst[key] = value
		}
	}
}

// ConfigBuilder provides a fluent interface for building configuration
type ConfigBuilder struct {
	manager *Manager
}

// NewBuilder creates a new configuration builder
func NewBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		manager: NewManager(),
	}
}

// WithDefaults adds default configuration source
func (b *ConfigBuilder) WithDefaults() *ConfigBuilder {
	b.manager.AddSource(NewDefaultSource())
	return b
}

// WithFile adds a file-based configuration source
func (b *ConfigBuilder) WithFile(filePath string) *ConfigBuilder {
	b.manager.AddSource(NewFileSource(filePath))
	return b
}

// WithEnvironment adds environment variable source
func (b *ConfigBuilder) WithEnvironment(prefix string) *ConfigBuilder {
	b.manager.AddSource(NewEnvironmentSource(prefix))
	return b
}

// WithConfigDir searches for config files in a directory
func (b *ConfigBuilder) WithConfigDir(dir string) *ConfigBuilder {
	// Look for common config file names
	configFiles := []string{
		"config.json",
		"config.yaml",
		"config.yml",
		"app.json",
		"app.yaml",
		"app.yml",
	}

	for _, filename := range configFiles {
		path := filepath.Join(dir, filename)
		if _, err := os.Stat(path); err == nil {
			b.manager.AddSource(NewFileSource(path))
		}
	}

	return b
}

// Build builds the final configuration
func (b *ConfigBuilder) Build() (*Config, error) {
	return b.manager.Load()
}

// LoadConfig is a convenience function for common configuration loading patterns
func LoadConfig() (*Config, error) {
	builder := NewBuilder().
		WithDefaults().
		WithFile("config.json").
		WithFile("config.yaml").
		WithConfigDir("./config").
		WithConfigDir("/etc/goryu").
		WithEnvironment("GORYU")

	return builder.Build()
}

// LoadConfigWithFile loads configuration from a specific file
func LoadConfigWithFile(filePath string) (*Config, error) {
	builder := NewBuilder().
		WithDefaults().
		WithFile(filePath).
		WithEnvironment("GORYU")

	return builder.Build()
}

// LoadConfigFromEnv loads configuration only from environment variables
func LoadConfigFromEnv(prefix string) (*Config, error) {
	builder := NewBuilder().
		WithDefaults().
		WithEnvironment(prefix)

	return builder.Build()
}
