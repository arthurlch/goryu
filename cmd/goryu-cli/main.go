package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		showHelp()
		return
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "init":
		handleInit(args)
	case "config":
		handleConfig(args)
	case "version":
		handleVersion()
	case "help":
		showHelp()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		showHelp()
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Printf("Goryu CLI v%s - A GOated web framework \n\n", version)
	fmt.Println("Usage: goryu <command> [arguments]")
	fmt.Println("\nAvailable commands:")
	fmt.Println("  init      Initialize a new Goryu project")
	fmt.Println("  config    Manage application configuration")
	fmt.Println("  version   Show version information")
	fmt.Println("  help      Show this help message")
	fmt.Println("\nExamples:")
	fmt.Println("  goryu init my-app --template=api")
	fmt.Println("  goryu config init --type=web")
	fmt.Println("\nFor more information about a command:")
	fmt.Println("  goryu <command> --help")
}

func handleInit(args []string) {
	projectName := "goryu-app"
	template := "basic"

	// Parse arguments
	for i, arg := range args {
		if strings.HasPrefix(arg, "--template=") {
			template = strings.TrimPrefix(arg, "--template=")
		} else if !strings.HasPrefix(arg, "--") && i == 0 {
			projectName = arg
		}
	}

	fmt.Printf("🚀 Initializing new Goryu project: %s\n", projectName)
	fmt.Printf("📋 Template: %s\n", template)

	// Check if directory already exists
	if _, err := os.Stat(projectName); err == nil {
		fmt.Printf("❌ Directory %s already exists\n", projectName)
		return
	}

	// Create project directory
	if err := os.MkdirAll(projectName, 0755); err != nil {
		fmt.Printf("❌ Failed to create directory: %v\n", err)
		return
	}

	// Create basic project structure
	createProjectFiles(projectName, template)

	fmt.Printf("✅ Project created successfully!\n")
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  cd %s\n", projectName)
	fmt.Printf("  go mod tidy\n")
	fmt.Printf("  go run cmd/server/main.go\n")
}

func handleConfig(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: goryu config <subcommand>")
		fmt.Println("Subcommands:")
		fmt.Println("  init     Create a new configuration file")
		fmt.Println("  show     Show current configuration")
		return
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "init":
		handleConfigInit(subargs)
	case "show":
		handleConfigShow(subargs)
	default:
		fmt.Printf("Unknown config subcommand: %s\n", subcommand)
	}
}

func handleConfigInit(args []string) {
	configType := "basic"
	filename := "config.json"

	// Parse arguments
	for _, arg := range args {
		if strings.HasPrefix(arg, "--type=") {
			configType = strings.TrimPrefix(arg, "--type=")
		} else if strings.HasPrefix(arg, "--file=") {
			filename = strings.TrimPrefix(arg, "--file=")
		}
	}

	fmt.Printf("🔧 Creating %s configuration: %s\n", configType, filename)

	var config map[string]interface{}
	switch configType {
	case "basic":
		config = getBasicConfig()
	case "api":
		config = getAPIConfig()
	case "web":
		config = getWebConfig()
	default:
		fmt.Printf("❌ Unknown config type: %s\n", configType)
		return
	}

	// Write config file
	data, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(filename, data, 0644); err != nil {
		fmt.Printf("❌ Failed to write config: %v\n", err)
		return
	}

	fmt.Printf("✅ Configuration created: %s\n", filename)
}

func handleConfigShow(args []string) {
	filename := "config.json"
	if len(args) > 0 && strings.HasPrefix(args[0], "--file=") {
		filename = strings.TrimPrefix(args[0], "--file=")
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("❌ Cannot read %s: %v\n", filename, err)
		return
	}

	var config map[string]interface{}
	if err := _ = json.Unmarshal(data, &config); err != nil {
		fmt.Printf("❌ Invalid JSON in %s: %v\n", filename, err)
		return
	}

	prettyData, _ := json.MarshalIndent(config, "", "  ")
	fmt.Println(string(prettyData))
}

func handleVersion() {
	fmt.Printf("Goryu CLI v%s\n", version)
	fmt.Println("A powerful web framework for Go")
}

func createProjectFiles(projectName, template string) {
	// Create directories
	dirs := []string{
		"cmd/server",
		"internal/handlers",
		"config",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(projectName, dir), 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create files
	files := map[string]string{
		"go.mod":                      createGoMod(projectName),
		"README.md":                   createReadme(projectName),
		"config/config.json":          createConfigJSON(template),
		"cmd/server/main.go":          createMainGo(projectName, template),
		"internal/handlers/health.go": createHealthHandler(projectName),
		".gitignore":                  createGitignore(),
	}

	for filename, content := range files {
		path := filepath.Join(projectName, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			log.Fatalf("Failed to write file %s: %v", filename, err)
		}
	}
}

func createGoMod(projectName string) string {
	return fmt.Sprintf(`module %s

go 1.21

require (
	github.com/arthurlch/goryu v1.0.0
)
`, projectName)
}

func createReadme(projectName string) string {
	return fmt.Sprintf(`# %s

A Goryu web application.

## Getting Started

1. Install dependencies:
   `+"```"+`bash
   go mod tidy
   `+"```"+`

2. Run the application:
   `+"```"+`bash
   go run cmd/server/main.go
   `+"```"+`

3. Visit http://localhost:8080

## Configuration

Edit config/config.json to customize your application.

## Features

- Built with Goryu web framework
- Configuration management
- Health checks
- Docker support
`, projectName)
}

func createConfigJSON(template string) string {
	var config map[string]interface{}
	switch template {
	case "api":
		config = getAPIConfig()
	case "web":
		config = getWebConfig()
	default:
		config = getBasicConfig()
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	return string(data)
}

func createMainGo(projectName, template string) string {
	return fmt.Sprintf(`package main

import (
	"log"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/config"
	"%s/internal/handlers"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %%v", err)
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
			"message": "Hello from %s!",
		})
	})
	app.GET("/health", handlers.Health)

	log.Printf("Starting server on %%s", cfg.GetServerAddress())
	if err := app.Listen(cfg.GetServerAddress()); err != nil {
		log.Fatalf("Server failed: %%v", err)
	}
}
`, projectName, projectName)
}

func createHealthHandler(projectName string) string {
	return fmt.Sprintf(`package handlers

import (
	"net/http"
	"time"

	"github.com/arthurlch/goryu"
)

// Health returns the health status
func Health(c *goryu.Context) {
	c.JSON(http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"service":   "%s",
		"timestamp": time.Now().UTC(),
	})
}
`, projectName)
}

func createGitignore() string {
	return `# Binaries
*.exe
*.dll
*.so
*.dylib
*.test
*.out

# Dependencies
vendor/

# Environment
.env

# IDE
.vscode/
.idea/
*.swp

# OS
.DS_Store
Thumbs.db

# Logs
logs/
*.log
`
}

func getBasicConfig() map[string]interface{} {
	return map[string]interface{}{
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
}

func getAPIConfig() map[string]interface{} {
	return map[string]interface{}{
		"app": map[string]interface{}{
			"name":    "api-service",
			"version": "1.0.0",
		},
		"server": map[string]interface{}{
			"host": "0.0.0.0",
			"port": 8080,
		},
		"environment": "development",
		"custom": map[string]interface{}{
			"database": map[string]interface{}{
				"driver": "postgres",
				"host":   "localhost",
				"port":   5432,
			},
			"metrics": map[string]interface{}{
				"enabled": true,
				"port":    9090,
			},
		},
	}
}

func getWebConfig() map[string]interface{} {
	return map[string]interface{}{
		"app": map[string]interface{}{
			"name":    "web-app",
			"version": "1.0.0",
		},
		"server": map[string]interface{}{
			"host": "localhost",
			"port": 8080,
		},
		"environment": "development",
		"custom": map[string]interface{}{
			"static_dir":   "./public",
			"template_dir": "./templates",
		},
	}
}
