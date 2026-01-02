# Goryu Configuration Management

A comprehensive, flexible configuration management system for Goryu applications supporting multiple configuration sources with priority-based merging, validation, and advanced features.

## Features

- **Multiple Sources**: Environment variables, JSON/YAML files, and defaults
- **Priority System**: Environment variables override files, files override defaults
- **Type Safety**: Strongly typed configuration with comprehensive validation
- **Auto-Discovery**: Automatic config file detection in common locations
- **Fluent Builder**: Chain configuration sources with a fluent API
- **Framework Integration**: Direct integration with goryu.Config
- **Database Support**: Built-in database configuration with connection pooling
- **Security Features**: TLS, CSRF, HSTS, and security headers configuration
- **Performance Limits**: Request body size, concurrent requests, and route limits
- **Static File Serving**: Configurable static file serving with caching
- **Router Options**: Advanced routing configuration options

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

The configuration system supports two main configuration formats:

### Core Configuration (New Structure)
```json
{
  "app": {
    "name": "my-goryu-app",
    "version": "1.0.0",
    "environment": "production",
    "log_level": "info",
    "custom": {
      // Add your app-specific configuration here
    }
  },
  "server": {
    "host": "0.0.0.0",
    "port": 8080,
    "read_timeout": "30s",
    "write_timeout": "30s",
    "shutdown_timeout": "30s",
    "tls": {
      "enabled": false,
      "cert_file": "",
      "key_file": "",
      "auto_tls": false
    }
  },
  "database": {
    "driver": "sqlite3",
    "path": "./app.db",
    "max_open_conns": 25,
    "max_idle_conns": 5,
    "conn_max_lifetime": "1h",
    "conn_max_idle_time": "30m"
  },
  "framework": {
    "server_header": "MyApp/1.0",
    "strict_routing": false,
    "case_sensitive": false,
    "redirect_trailing_slash": true,
    "enable_head_fallback": true,
    "disable_startup_msg": false
  }
}
```

### Advanced Configuration (Builder Pattern)
```json
{
  "app": {
    "name": "my-goryu-app",
    "version": "1.0.0",
    "environment": "production",
    "disable_startup_message": false,
    "server_header": "MyApp/1.0"
  },
  "server": {
    "port": 8080,
    "host": "0.0.0.0",
    "read_timeout": "30s",
    "write_timeout": "30s",
    "idle_timeout": "120s",
    "shutdown_timeout": "30s",
    "max_header_size": 1048576,
    "disable_keepalive": false,
    "tls": {
      "enabled": false,
      "cert_file": "",
      "key_file": "",
      "min_version": "TLS1.2"
    }
  },
  "router": {
    "strict_routing": false,
    "case_sensitive": false,
    "redirect_trailing_slash": true,
    "redirect_fixed_path": true,
    "handle_method_not_allowed": true,
    "handle_options": true,
    "enable_head_fallback": true
  },
  "static": {
    "root": "./public",
    "index": "index.html",
    "browse": false,
    "max_age": "3600s",
    "compress": true,
    "byte_range": true,
    "download": false,
    "cache_duration": "3600s"
  },
  "security": {
    "csrf_protection": true,
    "csrf_token_length": 32,
    "xss_protection": "1; mode=block",
    "content_type_nosniff": true,
    "x_frame_options": "DENY",
    "hsts": {
      "enabled": true,
      "max_age": "31536000s",
      "include_subdomains": true,
      "preload": false
    },
    "allowed_hosts": [],
    "trusted_proxies": []
  },
  "limits": {
    "max_route_depth": 20,
    "max_total_routes": 10000,
    "max_parameters_per_route": 20,
    "max_request_body_size": 10485760,
    "max_multipart_memory": 10485760,
    "max_concurrent_requests": 1000
  }
}
```

### Database Connection Examples

**SQLite (Simple - Recommended)**
```json
{
  "database": {
    "driver": "sqlite3",
    "path": "./data/app.db",
    "max_open_conns": 25,
    "max_idle_conns": 5
  }
}
```

**PostgreSQL**
```json
{
  "database": {
    "driver": "postgres",
    "host": "localhost",
    "port": 5432,
    "database": "myapp",
    "username": "user",
    "password": "password",
    "ssl_mode": "prefer",
    "max_open_conns": 50,
    "max_idle_conns": 10,
    "conn_max_lifetime": "1h"
  }
}
```

**MySQL**
```json
{
  "database": {
    "driver": "mysql",
    "host": "localhost", 
    "port": 3306,
    "database": "myapp",
    "username": "user",
    "password": "password",
    "charset": "utf8mb4",
    "parse_time": true,
    "max_open_conns": 50,
    "max_idle_conns": 10
  }
}
```

**Custom App Configuration**
```json
{
  "app": {
    "custom": {
      "redis": {
        "url": "redis://localhost:6379"
      },
      "api_keys": {
        "stripe": "sk_test_...",
        "sendgrid": "SG..."
      },
      "features": {
        "enable_analytics": true,
        "enable_notifications": false
      }
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
GORYU_APP_ENVIRONMENT=production
GORYU_APP_LOG_LEVEL=info

# Server configuration
GORYU_SERVER_HOST=0.0.0.0
GORYU_SERVER_PORT=8080
GORYU_SERVER_READ_TIMEOUT=30s
GORYU_SERVER_WRITE_TIMEOUT=30s

# TLS configuration
GORYU_SERVER_TLS_ENABLED=true
GORYU_SERVER_TLS_CERT_FILE=/path/to/cert.pem
GORYU_SERVER_TLS_KEY_FILE=/path/to/key.pem

# Database configuration
GORYU_DATABASE_DRIVER=postgres
GORYU_DATABASE_HOST=localhost
GORYU_DATABASE_PORT=5432
GORYU_DATABASE_DATABASE=myapp
GORYU_DATABASE_USERNAME=user
GORYU_DATABASE_PASSWORD=secret

# Framework configuration
GORYU_FRAMEWORK_SERVER_HEADER=MyApp/1.0
GORYU_FRAMEWORK_STRICT_ROUTING=false
GORYU_FRAMEWORK_CASE_SENSITIVE=false

# Custom configuration (as JSON)
GORYU_APP_CUSTOM={"api_timeout":"30s","features":{"analytics":true}}
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
- **Server**: Port must be 1-65535, timeouts cannot be negative
- **Database**: Driver must be sqlite3/postgres/mysql, required fields vary by driver
- **App**: Environment must be development/staging/production, log level must be debug/info/warn/error
- **TLS**: If enabled, cert/key files required unless auto_tls is true
- **Security**: CSRF token length must be positive, HSTS max_age must be valid duration
- **Limits**: All limits must be positive integers, max_idle_conns cannot exceed max_open_conns

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
customData := cfg.App.Custom["features"].(map[string]interface{})

// Database helpers
if cfg.Database.Driver != "" {
    connStr := cfg.Database.ConnectionString()
}

// Validation
if err := cfg.Validate(); err != nil {
    log.Fatal("Configuration invalid:", err)
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
   GORYU_SERVER_TLS_CERT_FILE=/secure/cert.pem
   GORYU_APP_CUSTOM='{"api_keys":{"stripe":"sk_live_..."}}'
   ```

2. **Set Appropriate Defaults**: Provide sensible defaults for development
   ```go
   // Defaults are applied automatically
   cfg, _ := config.LoadConfig()
   // Development defaults: localhost, SQLite, debug logging
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

4. **Use Feature Flags**: Control features via custom configuration
   ```json
   {
     "app": {
       "custom": {
         "features": {
           "enable_analytics": true,
           "enable_notifications": false,
           "enable_beta_features": false
         }
       }
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

6. **Security Best Practices**:
   - Enable TLS in production
   - Configure HSTS for HTTPS sites
   - Set appropriate CORS and security headers
   - Use strong CSRF tokens
   - Limit request sizes and concurrent connections

7. **Database Connection Pooling**: Configure based on load
   ```json
   {
     "database": {
       "max_open_conns": 100,    // High-traffic apps
       "max_idle_conns": 10,     // Keep some connections ready
       "conn_max_lifetime": "1h" // Refresh connections periodically
     }
   }
   ```

## Examples

See `examples/config_example.go` for a complete working example and `examples/config.json.example` for a sample configuration file.

## Integration Examples

### Basic Application Setup

```go
package main

import (
    "log"
    
    "github.com/arthurlch/goryu"\n    "github.com/arthurlch/goryu/goryuctx"
    "github.com/arthurlch/goryu/config"
)

func main() {
    // Load configuration
    cfg, err := config.LoadConfig()
    if err != nil {
        log.Fatal("Configuration error:", err)
    }
    
    // Create app with configuration
    app := goryu.New(goryu.Config{
        AppName:           cfg.App.Name,
        ServerHeader:      cfg.Framework.ServerHeader,
        StrictRouting:     cfg.Framework.StrictRouting,
        CaseSensitive:     cfg.Framework.CaseSensitive,
        DisableStartupMsg: cfg.Framework.DisableStartupMessage,
    })
    
    // Configure based on environment
    if cfg.App.Environment == "production" {
        app.Use(goryu.Logger())
        app.Use(goryu.Recover())
    }
    
    // Routes
    app.GET("/", func(c *goryuctx.Context) {
        c.JSON(200, goryuctx.Map{
            "app":         cfg.App.Name,
            "version":     cfg.App.Version,
            "environment": cfg.App.Environment,
        })
    })
    
    // Start server
    log.Printf("Starting %s on %s", cfg.App.Name, cfg.GetServerAddress())
    app.Listen(cfg.GetServerAddress())
}
```

### Advanced Setup with Builder Pattern

```go
package main

import (
    "log"
    
    "github.com/arthurlch/goryu"\n    "github.com/arthurlch/goryu/goryuctx"
    "github.com/arthurlch/goryu/config/builder"
    "github.com/arthurlch/goryu/middleware/compress"
    "github.com/arthurlch/goryu/middleware/cors"
    "github.com/arthurlch/goryu/middleware/ratelimit"
)

func main() {
    // Load advanced configuration
    cfg, err := builder.NewBuilder().
        WithDefaults().
        WithFile("config.json").
        WithEnvironment("GORYU").
        Build()
    
    if err != nil {
        log.Fatal("Config error:", err)
    }
    
    // Create app
    app := goryu.New()
    
    // Configure middleware based on config
    if cfg.Security.CSRFProtection {
        app.Use(goryu.CSRF())
    }
    
    if cfg.Static.Compress {
        app.Use(compress.New())
    }
    
    // Configure static files
    if cfg.Static.Root != "" {
        app.Static("/", cfg.Static.Root, goryu.Static{
            Compress: cfg.Static.Compress,
            ByteRange: cfg.Static.ByteRange,
            Browse: cfg.Static.Browse,
            Index: cfg.Static.Index,
            MaxAge: int(cfg.Static.MaxAge.Seconds()),
        })
    }
    
    // Configure security headers
    app.Use(func(c *goryuctx.Context) {
        if cfg.Security.ContentTypeNosniff {
            c.SetHeader("X-Content-Type-Options", "nosniff")
        }
        if cfg.Security.XFrameOptions != "" {
            c.SetHeader("X-Frame-Options", cfg.Security.XFrameOptions)
        }
        if cfg.Security.XSSProtection != "" {
            c.SetHeader("X-XSS-Protection", cfg.Security.XSSProtection)
        }
        c.Next()
    })
    
    // Start with TLS if configured
    if cfg.Server.TLS.Enabled {
        log.Fatal(app.ListenTLS(
            cfg.GetServerAddress(),
            cfg.Server.TLS.CertFile,
            cfg.Server.TLS.KeyFile,
        ))
    } else {
        log.Fatal(app.Listen(cfg.GetServerAddress()))
    }
}
```

