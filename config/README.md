# Goryu Configuration Management

A simple, flexible configuration management system for Goryu applications supporting multiple configuration sources with priority-based merging.

## Features

- **Multiple Sources**: Environment variables, JSON files, and defaults
- **Priority System**: Environment variables override files, files override defaults
- **Type Safety**: Strongly typed configuration with validation
- **Auto-Discovery**: Automatic config file detection in common locations
- **Fluent Builder**: Chain configuration sources with a fluent API
- **Framework Integration**: Direct integration with goryu.Config
- **Custom Config**: Extensible custom configuration section for app-specific needs

## Quick Start

### Basic Usage

```go
package main

import (
    "log"
    "github.com/arthurlch/goryu/config"
)

func main() {
    // Load configuration from defaults, files, and environment
    cfg, err := config.LoadConfig()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    
    fmt.Printf("Server will run on: %s\n", cfg.GetServerAddress())
}
```

### Custom Configuration

```go
// Build configuration from specific sources
cfg, err := config.NewBuilder().
    WithDefaults().                    // Apply defaults first
    WithFile("config.json").           // Override with file
    WithFile("/etc/myapp/config.json"). // Try system config
    WithEnvironment("MYAPP").          // Override with env vars
    Build()
```

## Configuration Structure

### Basic Configuration
```json
{
  "app": {
    "name": "my-goryu-app",
    "version": "1.0.0",
    "server_header": "MyApp/1.0",
    "strict_routing": false,
    "case_sensitive": false,
    "disable_startup_msg": false
  },
  "server": {
    "host": "0.0.0.0",
    "port": 8080,
    "read_timeout": "30s",
    "write_timeout": "30s",
    "shutdown_timeout": "30s"
  },
  "environment": "production",
  "custom": {
    // Add your app-specific configuration here
  }
}
```

### Database Connection Examples

**SQLite (Simple - Recommended)**
```json
{
  "custom": {
    "database": {
      "driver": "sqlite3",
      "path": "./data/app.db"
    }
  }
}
```

**PostgreSQL**
```json
{
  "custom": {
    "database": {
      "driver": "postgres",
      "host": "localhost",
      "port": 5432,
      "database": "myapp",
      "username": "user",
      "password": "password",
      "sslmode": "prefer"
    }
  }
}
```

**MySQL**
```json
{
  "custom": {
    "database": {
      "driver": "mysql",
      "host": "localhost", 
      "port": 3306,
      "database": "myapp",
      "username": "user",
      "password": "password"
    }
  }
}
```

**Other Custom Config**
```json
{
  "custom": {
    "redis": {
      "url": "redis://localhost:6379"
    },
    "api_keys": {
      "stripe": "sk_test_...",
      "sendgrid": "SG..."
    }
  }
}
```

## Environment Variables

Environment variables use the pattern `PREFIX_SECTION_FIELD`:

```bash
# App configuration
GORYU_APP_NAME=my-service
GORYU_APP_VERSION=2.0.0

# Server configuration
GORYU_SERVER_HOST=0.0.0.0
GORYU_SERVER_PORT=8080

# Environment
GORYU_ENVIRONMENT=production

# Custom configuration (as JSON)
GORYU_CUSTOM={"api_timeout":"30s"}
```

## Configuration Sources Priority

1. **Environment Variables** (Highest Priority)
2. **Configuration Files** (Medium Priority)
3. **Defaults** (Lowest Priority)

Later sources override earlier ones. Environment variables always win.

## File Auto-Discovery

The system automatically looks for configuration files in:

- `./config.json`, `./config.yaml`, `./config.yml`
- `./app.json`, `./app.yaml`, `./app.yml`
- `./config/config.json`, `./config/config.yaml`
- `/etc/goryu/config.json`, `/etc/goryu/config.yaml`

## Validation

Configuration is automatically validated on load:

```go
cfg, err := config.LoadConfig()
if err != nil {
    // Validation failed - err contains details
    log.Fatalf("Invalid configuration: %v", err)
}
```

Common validation rules:
- Server port must be 1-65535
- JWT secret is required (gets default if missing)
- Metrics port must be valid if metrics enabled
- Tracing sample rate must be 0.0-1.0

## Helper Methods

```go
cfg, _ := config.LoadConfig()

// Get formatted server address
serverAddr := cfg.GetServerAddress()      // "localhost:8080"

// Convert to Goryu framework config
goryuConfig := cfg.ToGoryuConfig()        // Returns goryu.Config{}

// Export as JSON
jsonStr, _ := cfg.ToJSON()
fmt.Println(jsonStr)

// Access custom configuration
if dbConfig, ok := cfg.Custom["database"].(map[string]interface{}); ok {
    dbHost := dbConfig["host"].(string)
}
```

## Builder Pattern

```go
// Custom builder with specific sources
cfg, err := config.NewBuilder().
    WithDefaults().                       // Start with defaults
    WithFile("config.json").              // Add file source
    WithConfigDir("/etc/myapp").          // Auto-discover in directory
    WithEnvironment("MYAPP").             // Add environment source
    Build()                               // Build final config
```

## Advanced Usage

### Custom Configuration Sources

Implement the `Source` interface:

```go
type Source interface {
    Load() (map[string]interface{}, error)
    Name() string
    Priority() int  // Higher priority overrides lower
}
```

### Environment-Only Configuration

```go
cfg, err := config.LoadConfigFromEnv("MYAPP")
```

### File-Specific Configuration

```go
cfg, err := config.LoadConfigWithFile("production.json")
```

## Best Practices

1. **Use Environment Variables for Secrets**: Never put secrets in config files
   ```bash
   GORYU_DATABASE_PASSWORD=secret
   GORYU_SECURITY_JWT_SECRET=jwt-secret
   ```

2. **Set Appropriate Defaults**: Provide sensible defaults for development
   ```go
   // Defaults are applied automatically
   cfg, _ := config.LoadConfig()
   ```

3. **Validate Early**: Load and validate config at application startup
   ```go
   func main() {
       cfg, err := config.LoadConfig()
       if err != nil {
           log.Fatalf("Config error: %v", err)
       }
       // Continue with validated config
   }
   ```

4. **Use Feature Flags**: Control middleware and features via configuration
   ```json
   {
     "features": {
       "enable_metrics": true,
       "enable_tracing": false,
       "enable_compression": true
     }
   }
   ```

5. **Environment-Specific Configs**: Use different config files per environment
   ```bash
   # Development
   GORYU_CONFIG_FILE=config.dev.json
   
   # Production  
   GORYU_CONFIG_FILE=config.prod.json
   ```

## Examples

See `examples/config_example.go` for a complete working example and `examples/config.json.example` for a sample configuration file.

## Database Integration

Simple database connection with Goryu:

```go
package main

import (
    "log"
    
    "github.com/arthurlch/goryu"
    "github.com/arthurlch/goryu/config"
    "github.com/arthurlch/goryu/db"
)

func main() {
    // Load configuration
    cfg, err := config.LoadConfig()
    if err != nil {
        log.Fatal(err)
    }
    
    // Connect to database (optional)
    var database *db.Connection
    if _, hasDB := cfg.Custom["database"]; hasDB {
        database, err = db.Connect(cfg)
        if err != nil {
            log.Fatal("Database connection failed:", err)
        }
        defer database.Close()
        log.Println("Database connected:", database.Driver)
    }
    
    // Create app with configuration
    goryuCfg := cfg.ToGoryuConfig()
    app := goryu.New(goryu.Config{
        AppName: goryuCfg.AppName,
        // ... other config fields
    })
    
    // Use database in handlers
    app.GET("/users", func(c *goryu.Context) {
        if database != nil {
            // Query your database
            rows, err := database.DB.Query("SELECT id, name FROM users")
            // ... handle rows
        }
        c.JSON(200, map[string]string{"status": "ok"})
    })
    
    app.Listen(cfg.GetServerAddress())
}
```

