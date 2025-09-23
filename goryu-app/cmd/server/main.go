package main

import (
	"log"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/config"
	"goryu-app/internal/handlers"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create Goryu app with configuration
	goryuCfg := cfg.ToGoryuConfig()
	app := goryu.New(goryu.Config{
		AppName:               goryuCfg.AppName,
		ServerHeader:          goryuCfg.ServerHeader,
		StrictRouting:         goryuCfg.StrictRouting,
		CaseSensitive:         goryuCfg.CaseSensitive,
		DisableStartupMessage: goryuCfg.DisableStartupMessage,
	})

	// Routes
	app.GET("/", func(c *goryu.Context) {
		c.JSON(200, map[string]string{
			"message": "Hello from goryu-app!",
		})
	})
	app.GET("/health", handlers.Health)

	log.Printf("Starting server on %s", cfg.GetServerAddress())
	if err := app.Listen(cfg.GetServerAddress()); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
%!(EXTRA string=goryu-app)