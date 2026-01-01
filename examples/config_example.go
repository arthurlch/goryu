package main

import (
	"log"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/config/builder"
	"github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/compress"
	// "github.com/arthurlch/goryu/middleware/ratelimit" // Assuming this doesn't exist yet or is named limiter
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
        // app.Use(goryu.CSRF()) // Assuming Helper exists, or use middleware directly
        // For example purposes using what's available
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
