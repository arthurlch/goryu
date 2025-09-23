package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type EnvironmentSource struct {
	prefix string
}

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

		if e.prefix != "" && !strings.HasPrefix(key, e.prefix+"_") {
			continue
		}

		if e.prefix != "" {
			key = strings.TrimPrefix(key, e.prefix+"_")
		}

		key = strings.ToLower(key)

		// Handle nested keys (e.g., SERVER_HOST -> server.host)
		if strings.Contains(key, "_") {
			parts := strings.Split(key, "_")
			if len(parts) >= 2 {
				current := data
				for i, part := range parts[:len(parts)-1] {
					if _, exists := current[part]; !exists {
						current[part] = make(map[string]interface{})
					}
					if i == len(parts)-2 {
						if nested, ok := current[part].(map[string]interface{}); ok {
							nested[parts[len(parts)-1]] = value
						}
					} else {
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

	if _, err := os.Stat(f.filePath); os.IsNotExist(err) {
		return data, nil
	}

	content, err := os.ReadFile(f.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", f.filePath, err)
	}

	ext := strings.ToLower(filepath.Ext(f.filePath))
	switch ext {
	case ".json":
		if err := json.Unmarshal(content, &data); err != nil {
			return nil, fmt.Errorf("failed to parse JSON file %s: %w", f.filePath, err)
		}

	case ".yaml", ".yml":
		// For YAML support, I would use gopkg.in/yaml.v2 or similar
		// For now, I'll just return an error indicating YAML is not supported, better and easier.
		// Actually I don't want to add a new dependency just for YAML parsing ????? 
		// The less dependency the better I feel, goryu should rely on my unreliable code
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

type DefaultSource struct {
	defaults map[string]interface{}
}

func NewDefaultSource() *DefaultSource {
	defaults := map[string]interface{}{
		"app": map[string]interface{}{
			"name":        "goryu-app",
			"version":     "1.0.0",
			"environment": "development",
			"log_level":   "info",
		},
		"server": map[string]interface{}{
			"host": "localhost",
			"port": 8080,
			"read_timeout": "30s",
			"write_timeout": "30s",
			"shutdown_timeout": "30s",
		},
		"database": map[string]interface{}{
			"driver": "sqlite3",
			"path":   "./app.db",
			"max_open_conns": 25,
			"max_idle_conns": 5,
			"conn_max_lifetime": "1h",
			"conn_max_idle_time": "30m",
		},
		"framework": map[string]interface{}{
			"strict_routing": false,
			"case_sensitive": false,
			"redirect_trailing_slash": true,
			"enable_head_fallback": true,
			"disable_startup_msg": false,
		},
	}

	return &DefaultSource{
		defaults: defaults,
	}
}

func (d *DefaultSource) Load() (map[string]interface{}, error) {
	data := make(map[string]interface{})
	copyMap(d.defaults, data)
	return data, nil
}

func (d *DefaultSource) Name() string {
	return "defaults"
}

func (d *DefaultSource) Priority() int {
	return 1 // Low priority 
}

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

// NOTE: use builder pattern to make it easier to add sources and build the final config
type ConfigBuilder struct {
	manager *Manager
}

func NewBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		manager: NewManager(),
	}
}

func (b *ConfigBuilder) WithDefaults() *ConfigBuilder {
	b.manager.AddSource(NewDefaultSource())
	return b
}

func (b *ConfigBuilder) WithFile(filePath string) *ConfigBuilder {
	b.manager.AddSource(NewFileSource(filePath))
	return b
}

func (b *ConfigBuilder) WithEnvironment(prefix string) *ConfigBuilder {
	b.manager.AddSource(NewEnvironmentSource(prefix))
	return b
}

func (b *ConfigBuilder) WithConfigDir(dir string) *ConfigBuilder {
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

func (b *ConfigBuilder) Build() (*Config, error) {
	return b.manager.Load()
}

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

func LoadConfigWithFile(filePath string) (*Config, error) {
	builder := NewBuilder().
		WithDefaults().
		WithFile(filePath).
		WithEnvironment("GORYU")

	return builder.Build()
}

func LoadConfigFromEnv(prefix string) (*Config, error) {
	builder := NewBuilder().
		WithDefaults().
		WithEnvironment(prefix)

	return builder.Build()
}
