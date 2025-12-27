package cli

import (
	"fmt"
	"strings"
)

// Any good framework needs a big yaml or json file with many config to tweak for fun

func generateRouteBuilderContent(name, group, middleware, methods, moduleName string) string {
	routeName := strings.Title(strings.ToLower(name))
	varName := strings.ToLower(name)

	// middlewareSetup := "" // No longer needed in this form
	// if middleware != "" {
	// 	mws := strings.Split(middleware, ",")
	// 	for _, mw := range mws {
	// 		middlewareSetup += fmt.Sprintf("\t\t\t.Middleware(middleware.%s())\n", strings.Title(strings.TrimSpace(mw)))
	// 	}
	// }

	// methodSetup := "" // No longer needed in this form
	// for _, method := range strings.Split(methods, ",") {
	// 	m := strings.TrimSpace(method)
	// 	handlerName := fmt.Sprintf("handle%s%s", routeName, strings.Title(strings.ToLower(m)))
	// 	methodSetup += fmt.Sprintf("\t\t\t.%s(handlers.%s)\n", strings.Title(strings.ToLower(m)), handlerName)
	// }

	content := fmt.Sprintf(`package routes

import (
	"github.com/arthurlch/goryu"
	"%s/internal/handlers"
	"%s/internal/middleware"
)

// Register%sRoutes registers all %s routes
func Register%sRoutes(app *goryu.App) {
	// Create a group for %s routes
	group := app.Group("/%s")`,
		moduleName, moduleName, routeName, varName, routeName, varName, group)

	if middleware != "" {
		mws := strings.Split(middleware, ",")
		for _, mw := range mws {
			content += fmt.Sprintf("\n\tgroup.Use(middleware.%s())", strings.Title(strings.TrimSpace(mw)))
		}
	} else {
		// If no middleware, we should remove the unused import to avoid compile error
		content = strings.Replace(content, fmt.Sprintf("\t\"%s/internal/middleware\"\n", moduleName), "", 1)
	}

	content += "\n\n\t// Register routes"
	for _, method := range strings.Split(methods, ",") {
		m := strings.TrimSpace(method)
		handlerName := fmt.Sprintf("Handle%s%s", routeName, strings.Title(strings.ToLower(m)))
		content += fmt.Sprintf("\n\tgroup.%s(\"/\", handlers.%s)", strings.ToUpper(m), handlerName)
	}

	content += "\n}\n"

	// Add ConfigureAPI function
	content += fmt.Sprintf(`
// Configure%sAPI sets up the complete API configuration
func Configure%sAPI(app *goryu.App) {
	// Configure global middleware
	app.Use(goryu.Logger().Build())
	app.Use(goryu.Recovery().Build())
	
	// Register routes
	Register%sRoutes(app)
}
`, routeName, routeName, routeName)

	return content
}

func generateStandardRouteContent(name, group, middleware, methods, moduleName string) string {
	routeName := strings.Title(strings.ToLower(name))
	varName := strings.ToLower(name)

	content := fmt.Sprintf(`package routes

import (
	"github.com/arthurlch/goryu"
	"%s/internal/handlers"
)

// Register%sRoutes registers all %s routes
func Register%sRoutes(app *goryu.App) {
	// Basic route registration
`, moduleName, routeName, varName, routeName)

	prefix := ""
	if group != "" {
		prefix = group + "/"
	}

	for _, method := range strings.Split(methods, ",") {
		m := strings.ToUpper(strings.TrimSpace(method))
		handlerName := fmt.Sprintf("Handle%s%s", routeName, strings.Title(strings.ToLower(m)))
		content += fmt.Sprintf(`	app.%s("%s%s", handlers.%s)
`, strings.Title(strings.ToLower(m)), prefix, varName, handlerName)
	}

	content += "}\n"
	return content
}

func generateConfigBuilderContent(name, configType string) string {
	configName := strings.Title(strings.ToLower(name))
	
	var configStruct string
	switch configType {
	case "server":
		configStruct = `	config := &%sConfig{
		Config: &builder.Config{
			Server: builder.ServerConfig{
				Host: "0.0.0.0",
				Port: 8080,
				ReadTimeout: 30 * time.Second,
				WriteTimeout: 30 * time.Second,
			},
			App: builder.AppConfig{
				Name: "My Goryu App",
				Version: "1.0.0",
			},
		},
	}`
		
	case "database":
		configStruct = `	config := &%sConfig{
		Config: &builder.Config{
			// Initialize database config here
		},
	}`
		
	case "cache":
		configStruct = `	config := &%sConfig{
		Config: &builder.Config{
			// Initialize cache config here
		},
	}`
	}

	return fmt.Sprintf(`package config

import (
	"os"
	"time"
	
	"github.com/arthurlch/goryu/config/builder"
)

// %sConfig holds the %s configuration
type %sConfig struct {
	*builder.Config
}

// New%sConfig creates a new %s configuration
func New%sConfig() (*%sConfig, error) {
%s

	return config, nil
}

// LoadFrom%s loads configuration from %s sources
func (c *%sConfig) LoadFrom%s() error {
	// Load from environment variables
	// c.Config.LoadFromEnv("APP_")
	
	// Load from config file if exists
	if _, err := os.Stat("config.json"); err == nil {
		if err := c.Config.LoadFromFile("config.json"); err != nil {
			return err
		}
	}
	
	return nil
}

// Validate validates the configuration
func (c *%sConfig) Validate() error {
	return c.Config.Validate()
}
`, configName, configType, configName, configName, configType, configName, configName, fmt.Sprintf(configStruct, configName), configName, configName, configName, configName, configName)
}

func generateStandardConfigContent(name, configType string) string {
	configName := strings.Title(strings.ToLower(name)) // oh need upt here
	fields := ""
	switch configType {
	case "server":
		fields = `	Host              string        ` + "`json:\"host\" env:\"HOST\"`" + `
	Port              int           ` + "`json:\"port\" env:\"PORT\"`" + `
	ReadTimeout       time.Duration ` + "`json:\"read_timeout\" env:\"READ_TIMEOUT\"`" + `
	WriteTimeout      time.Duration ` + "`json:\"write_timeout\" env:\"WRITE_TIMEOUT\"`" + `
	ShutdownTimeout   time.Duration ` + "`json:\"shutdown_timeout\" env:\"SHUTDOWN_TIMEOUT\"`"
		
	case "database":
		fields = `	Driver            string ` + "`json:\"driver\" env:\"DB_DRIVER\"`" + `
	Host              string ` + "`json:\"host\" env:\"DB_HOST\"`" + `
	Port              int    ` + "`json:\"port\" env:\"DB_PORT\"`" + `
	Database          string ` + "`json:\"database\" env:\"DB_DATABASE\"`" + `
	Username          string ` + "`json:\"username\" env:\"DB_USERNAME\"`" + `
	Password          string ` + "`json:\"password\" env:\"DB_PASSWORD\"`" + `
	MaxConnections    int    ` + "`json:\"max_connections\" env:\"DB_MAX_CONNECTIONS\"`"
		
	case "cache":
		fields = `	Driver   string ` + "`json:\"driver\" env:\"CACHE_DRIVER\"`" + `
	Host     string ` + "`json:\"host\" env:\"CACHE_HOST\"`" + `
	Port     int    ` + "`json:\"port\" env:\"CACHE_PORT\"`" + `
	Password string ` + "`json:\"password\" env:\"CACHE_PASSWORD\"`" + `
	Database int    ` + "`json:\"database\" env:\"CACHE_DATABASE\"`" + `
	TTL      int    ` + "`json:\"ttl\" env:\"CACHE_TTL\"`"
	}

	return fmt.Sprintf(`package config

import (
	"encoding/json"
	"os"
	"time"
)

// %sConfig represents %s configuration
type %sConfig struct {
%s
}

// Load%sConfig loads the %s configuration
func Load%sConfig() (*%sConfig, error) {
	config := &%sConfig{}
	
	// Load from environment variables
	// Implementation would use a library like envconfig
	
	// Load from config file if exists
	if data, err := os.ReadFile("config.json"); err == nil {
		if err := json.Unmarshal(data, config); err != nil {
			return nil, err
		}
	}
	
	// Set defaults
	config.setDefaults()
	
	return config, nil
}

// setDefaults sets default values
func (c *%sConfig) setDefaults() {
	// Set default values based on config type
}

// Validate validates the configuration
func (c *%sConfig) Validate() error {
	// Add validation logic
	return nil
}
`, configName, configType, configName, fields, configName, configType, configName, configName, configName, configName, configName)
}

func generateExampleConfig(configType, format string) string {
	switch format {
	case "json":
		return generateJSONExampleConfig(configType)
	case "yaml":
		return generateYAMLExampleConfig(configType)
	case "toml":
		return generateTOMLExampleConfig(configType)
	case "env":
		return generateEnvExampleConfig(configType)
	}
	return ""
}

func generateJSONExampleConfig(configType string) string {
	switch configType {
	case "server":
		return `{
  "host": "0.0.0.0",
  "port": 8080,
  "read_timeout": "30s",
  "write_timeout": "30s",
  "shutdown_timeout": "10s",
  "app": {
    "name": "My Goryu App",
    "version": "1.0.0",
    "environment": "development"
  },
  "router": {
    "strict_routing": false,
    "case_sensitive": false,
    "max_param_length": 1024
  },
  "middleware": {
    "logger": {
      "enabled": true,
      "format": "combined",
      "skip_paths": ["/health", "/metrics"]
    },
    "cors": {
      "enabled": true,
      "allowed_origins": ["*"],
      "allowed_methods": ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
      "allowed_headers": ["Content-Type", "Authorization"]
    },
    "rate_limit": {
      "enabled": true,
      "max_requests": 100,
      "window": "1m"
    }
  }
}`
	case "database":
		return `{
  "driver": "postgres",
  "host": "localhost",
  "port": 5432,
  "database": "myapp",
  "username": "postgres",
  "password": "",
  "max_connections": 25,
  "max_idle_connections": 5,
  "connection_max_lifetime": "1h",
  "ssl_mode": "disable"
}`
	case "cache":
		return `{
  "driver": "redis",
  "host": "localhost", 
  "port": 6379,
  "password": "",
  "database": 0,
  "pool_size": 10,
  "min_idle_conns": 5,
  "max_retries": 3,
  "default_ttl": 3600
}`
	}
	return "{}"
}

func generateYAMLExampleConfig(configType string) string {
	switch configType {
	case "server":
		return `host: 0.0.0.0
port: 8080
read_timeout: 30s
write_timeout: 30s
shutdown_timeout: 10s

app:
  name: My Goryu App
  version: 1.0.0
  environment: development

router:
  strict_routing: false
  case_sensitive: false
  max_param_length: 1024

middleware:
  logger:
    enabled: true
    format: combined
    skip_paths:
      - /health
      - /metrics
  cors:
    enabled: true
    allowed_origins:
      - "*"
    allowed_methods:
      - GET
      - POST
      - PUT
      - DELETE
      - OPTIONS
    allowed_headers:
      - Content-Type
      - Authorization
  rate_limit:
    enabled: true
    max_requests: 100
    window: 1m`
	}
	return ""
}

func generateTOMLExampleConfig(configType string) string {
	switch configType {
	case "server":
		return `host = "0.0.0.0"
port = 8080
read_timeout = "30s"
write_timeout = "30s"
shutdown_timeout = "10s"

[app]
name = "My Goryu App"
version = "1.0.0"
environment = "development"

[router]
strict_routing = false
case_sensitive = false
max_param_length = 1024

[middleware.logger]
enabled = true
format = "combined"
skip_paths = ["/health", "/metrics"]

[middleware.cors]
enabled = true
allowed_origins = ["*"]
allowed_methods = ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
allowed_headers = ["Content-Type", "Authorization"]

[middleware.rate_limit]
enabled = true
max_requests = 100
window = "1m"`
	}
	return ""
}

func generateEnvExampleConfig(configType string) string {
	switch configType {
	case "server":
		return `# Server Configuration
HOST=0.0.0.0
PORT=8080
READ_TIMEOUT=30s
WRITE_TIMEOUT=30s
SHUTDOWN_TIMEOUT=10s

# App Configuration
APP_NAME=My Goryu App
APP_VERSION=1.0.0
APP_ENVIRONMENT=development

# Router Configuration
ROUTER_STRICT_ROUTING=false
ROUTER_CASE_SENSITIVE=false
ROUTER_MAX_PARAM_LENGTH=1024

# Middleware Configuration
MIDDLEWARE_LOGGER_ENABLED=true
MIDDLEWARE_LOGGER_FORMAT=combined
MIDDLEWARE_CORS_ENABLED=true
MIDDLEWARE_RATE_LIMIT_ENABLED=true
MIDDLEWARE_RATE_LIMIT_MAX=100
MIDDLEWARE_RATE_LIMIT_WINDOW=1m`
	case "database":
		return `# Database Configuration
DB_DRIVER=postgres
DB_HOST=localhost
DB_PORT=5432
DB_DATABASE=myapp
DB_USERNAME=postgres
DB_PASSWORD=
DB_MAX_CONNECTIONS=25
DB_MAX_IDLE_CONNECTIONS=5
DB_CONNECTION_MAX_LIFETIME=1h
DB_SSL_MODE=disable`
	case "cache":
		return `# Cache Configuration
CACHE_DRIVER=redis
CACHE_HOST=localhost
CACHE_PORT=6379
CACHE_PASSWORD=
CACHE_DATABASE=0
CACHE_POOL_SIZE=10
CACHE_MIN_IDLE_CONNS=5
CACHE_MAX_RETRIES=3
CACHE_DEFAULT_TTL=3600`
	}
	return ""
}