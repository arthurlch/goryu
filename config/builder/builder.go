package builder

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Builder struct {
	config *Config
	errors ValidationErrors
}

func New() *Builder {
	return &Builder{
		config: DefaultConfig(),
		errors: make(ValidationErrors, 0),
	}
}

func FromFile(path string) (*Builder, error) {
	b := New()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	ext := filepath.Ext(path)
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, b.config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, b.config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", ext)
	}

	return b, nil
}

func FromEnvironment() *Builder {
	b := New()

	// load from environment variables with GORYU_ prefix
	// Example: GORYU_SERVER_PORT=8080

	if port := os.Getenv("GORYU_SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			b.config.Server.Port = p
		}
	}

	if host := os.Getenv("GORYU_SERVER_HOST"); host != "" {
		b.config.Server.Host = host
	}

	if name := os.Getenv("GORYU_APP_NAME"); name != "" {
		b.config.App.Name = name
	}

	if env := os.Getenv("GORYU_APP_ENVIRONMENT"); env != "" {
		b.config.App.Environment = env
	}

	if header := os.Getenv("GORYU_APP_SERVER_HEADER"); header != "" {
		b.config.App.ServerHeader = header
	}

	if disable := os.Getenv("GORYU_APP_DISABLE_STARTUP_MESSAGE"); disable == "true" {
		b.config.App.DisableStartupMessage = true
	}

	if certFile := os.Getenv("GORYU_TLS_CERT_FILE"); certFile != "" {
		b.config.Server.TLS.CertFile = certFile
		if keyFile := os.Getenv("GORYU_TLS_KEY_FILE"); keyFile != "" {
			b.config.Server.TLS.KeyFile = keyFile
			b.config.Server.TLS.Enabled = true
		}
	}

	if minVersion := os.Getenv("GORYU_TLS_MIN_VERSION"); minVersion != "" {
		b.config.Server.TLS.MinVersion = minVersion
	}

	if strict := os.Getenv("GORYU_ROUTER_STRICT"); strict == "true" {
		b.config.Router.StrictRouting = true
	}

	if caseSensitive := os.Getenv("GORYU_ROUTER_CASE_SENSITIVE"); caseSensitive == "true" {
		b.config.Router.CaseSensitive = true
	}

	if csrf := os.Getenv("GORYU_SECURITY_CSRF"); csrf == "false" {
		b.config.Security.CSRFProtection = false
	}

	if hsts := os.Getenv("GORYU_SECURITY_HSTS"); hsts == "true" {
		b.config.Security.HSTS.Enabled = true
	}

	if staticRoot := os.Getenv("GORYU_STATIC_ROOT"); staticRoot != "" {
		b.config.Static.Root = staticRoot
	}

	if staticIndex := os.Getenv("GORYU_STATIC_INDEX"); staticIndex != "" {
		b.config.Static.Index = staticIndex
	}

	return b
}

func (b *Builder) Configure(fn func(*Config)) *Builder {
	fn(b.config)
	return b
}

func (b *Builder) App(fn func(*AppConfig)) *Builder {
	fn(&b.config.App)
	return b
}

func (b *Builder) Server(fn func(*ServerConfig)) *Builder {
	fn(&b.config.Server)
	return b
}

func (b *Builder) Router(fn func(*RouterConfig)) *Builder {
	fn(&b.config.Router)
	return b
}

func (b *Builder) Static(fn func(*StaticConfig)) *Builder {
	fn(&b.config.Static)
	return b
}

func (b *Builder) Security(fn func(*SecurityConfig)) *Builder {
	fn(&b.config.Security)
	return b
}

func (b *Builder) Limits(fn func(*LimitsConfig)) *Builder {
	fn(&b.config.Limits)
	return b
}

func (b *Builder) SetAppName(name string) *Builder {
	b.config.App.Name = name
	return b
}

func (b *Builder) SetPort(port int) *Builder {
	b.config.Server.Port = port
	return b
}

func (b *Builder) SetEnvironment(env string) *Builder {
	b.config.App.Environment = env
	return b
}

func (b *Builder) EnableTLS(certFile, keyFile string) *Builder {
	b.config.Server.TLS.Enabled = true
	b.config.Server.TLS.CertFile = certFile
	b.config.Server.TLS.KeyFile = keyFile
	return b
}

func (b *Builder) SetStaticRoot(root string) *Builder {
	b.config.Static.Root = root
	return b
}

func (b *Builder) EnableProduction() *Builder {
	b.config.App.Environment = "production"
	b.config.App.DisableStartupMessage = true
	b.config.Security.ContentTypeNosniff = true
	b.config.Security.XSSProtection = "1; mode=block"
	b.config.Security.XFrameOptions = "DENY"
	b.config.Security.HSTS.Enabled = true
	return b
}

func (b *Builder) Merge(other *Config) *Builder {
	b.config.Merge(other)
	return b
}

func (b *Builder) Validate() *Builder {
	validator := NewValidator()
	b.errors = validator.Validate(b.config)
	return b
}

func (b *Builder) Errors() ValidationErrors {
	return b.errors
}

func (b *Builder) HasErrors() bool {
	return len(b.errors) > 0
}

func (b *Builder) Build() (*Config, error) {
	b.Validate()

	if b.HasErrors() {
		return nil, b.errors
	}

	return b.config.Clone(), nil
}

func (b *Builder) MustBuild() *Config {
	cfg, err := b.Build()
	if err != nil {
		panic(fmt.Sprintf("invalid configuration: %v", err))
	}
	return cfg
}

func (b *Builder) WriteTo(w io.Writer, format string) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(b.config)
	case "yaml":
		encoder := yaml.NewEncoder(w)
		return encoder.Encode(b.config)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func (b *Builder) SaveToFile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	ext := filepath.Ext(path)
	format := "json"
	if ext == ".yaml" || ext == ".yml" {
		format = "yaml"
	}

	return b.WriteTo(file, format)
}

func (b *Builder) Print() *Builder {
	fmt.Println(b.config.String())
	return b
}
