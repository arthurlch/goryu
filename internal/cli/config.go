package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/arthurlch/goryu/config"
)

func newConfigCommand() *Command {
	return &Command{
		Name:        "config",
		Description: "Manage application configuration",
		Usage:       "goryu config <subcommand> [options]",
		Subcommands: []*Command{
			{
				Name:        "init",
				Description: "Create a new configuration file",
				Usage:       "goryu config init [--type=basic|api|web] [--file=config.json]",
				Action:      runConfigInit,
			},
			{
				Name:        "validate",
				Description: "Validate configuration file",
				Usage:       "goryu config validate [--file=config.json]",
				Action:      runConfigValidate,
			},
			{
				Name:        "show",
				Description: "Show current configuration",
				Usage:       "goryu config show [--file=config.json] [--format=json|yaml]",
				Action:      runConfigShow,
			},
			{
				Name:        "set",
				Description: "Set a configuration value",
				Usage:       "goryu config set <key> <value> [--file=config.json]",
				Action:      runConfigSet,
			},
			{
				Name:        "get",
				Description: "Get a configuration value",
				Usage:       "goryu config get <key> [--file=config.json]",
				Action:      runConfigGet,
			},
		},
	}
}

func runConfigInit(args []string) error {
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

	fmt.Printf("🔧 Creating %s configuration file: %s\n", configType, filename)

	// Check if file already exists
	if _, err := os.Stat(filename); err == nil {
		fmt.Printf("❓ Configuration file %s already exists. Overwrite? (y/N): ", filename)
		var response string
		_, _ = fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("❌ Configuration creation cancelled.")
			return nil
		}
	}

	var content string
	switch configType {
	case "basic":
		content = generateBasicConfig()
	case "api":
		content = generateAPIConfig()
	case "web":
		content = generateWebConfig()
	default:
		return fmt.Errorf("unknown configuration type: %s", configType)
	}

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	fmt.Printf("✅ Configuration file created: %s\n", filename)
	fmt.Println("\n💡 Next steps:")
	fmt.Printf("  • Edit %s to customize your settings\n", filename)
	fmt.Printf("  • Use 'goryu config validate --file=%s' to validate\n", filename)
	fmt.Printf("  • Set environment variables to override config values\n")

	return nil
}

func runConfigValidate(args []string) error {
	filename := "config.json"

	// Parse arguments
	for _, arg := range args {
		if strings.HasPrefix(arg, "--file=") {
			filename = strings.TrimPrefix(arg, "--file=")
		}
	}

	fmt.Printf("🔍 Validating configuration file: %s\n", filename)

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s", filename)
	}

	// Load and validate configuration
	cfg, err := config.LoadConfigWithFile(filename)
	if err != nil {
		fmt.Printf("❌ Configuration validation failed:\n")
		fmt.Printf("   %v\n", err)
		return err
	}

	fmt.Printf("✅ Configuration is valid!\n")
	fmt.Printf("\n📋 Configuration summary:\n")
	fmt.Printf("  • App: %s v%s\n", cfg.App.Name, cfg.App.Version)
	fmt.Printf("  • Server: %s\n", cfg.GetServerAddress())
	fmt.Printf("  • Environment: %s\n", cfg.Environment)

	return nil
}

func runConfigShow(args []string) error {
	filename := "config.json"
	format := "json"

	// Parse arguments
	for _, arg := range args {
		if strings.HasPrefix(arg, "--file=") {
			filename = strings.TrimPrefix(arg, "--file=")
		} else if strings.HasPrefix(arg, "--format=") {
			format = strings.TrimPrefix(arg, "--format=")
		}
	}

	// Load configuration
	cfg, err := config.LoadConfigWithFile(filename)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	switch format {
	case "json":
		jsonStr, err := cfg.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to convert to JSON: %w", err)
		}
		fmt.Println(jsonStr)
	case "yaml":
		fmt.Println("YAML format not yet implemented")
		return fmt.Errorf("YAML format not supported yet")
	default:
		return fmt.Errorf("unknown format: %s", format)
	}

	return nil
}

func runConfigSet(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: goryu config set <key> <value> [--file=config.json]")
	}

	key := args[0]
	value := args[1]
	filename := "config.json"

	// Parse additional arguments
	for i := 2; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--file=") {
			filename = strings.TrimPrefix(arg, "--file=")
		}
	}

	fmt.Printf("🔧 Setting configuration: %s = %s in %s\n", key, value, filename)

	// Load existing configuration
	var configData map[string]interface{}
	if _, err := os.Stat(filename); err == nil {
		content, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		if err := _ = json.Unmarshal(content, &configData); err != nil {
			return fmt.Errorf("failed to parse config file: %w", err)
		}
	} else {
		configData = make(map[string]interface{})
	}

	// Set the value using dot notation
	if err := setNestedValue(configData, key, value); err != nil {
		return fmt.Errorf("failed to set value: %w", err)
	}

	// Write back to file
	updatedContent, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filename, updatedContent, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("✅ Configuration updated successfully!\n")
	return nil
}

func runConfigGet(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: goryu config get <key> [--file=config.json]")
	}

	key := args[0]
	filename := "config.json"

	// Parse additional arguments
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--file=") {
			filename = strings.TrimPrefix(arg, "--file=")
		}
	}

	// Load configuration
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var configData map[string]interface{}
	if err := _ = json.Unmarshal(content, &configData); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Get the value using dot notation
	value, err := getNestedValue(configData, key)
	if err != nil {
		return fmt.Errorf("failed to get value: %w", err)
	}

	// Format output
	if valueStr, ok := value.(string); ok {
		fmt.Println(valueStr)
	} else {
		valueJSON, _ := json.Marshal(value)
		fmt.Println(string(valueJSON))
	}

	return nil
}

// Helper function to set nested values using dot notation (e.g., "server.port")
func setNestedValue(data map[string]interface{}, key string, value string) error {
	keys := strings.Split(key, ".")
	current := data

	for i, k := range keys[:len(keys)-1] {
		if _, exists := current[k]; !exists {
			current[k] = make(map[string]interface{})
		}
		if nested, ok := current[k].(map[string]interface{}); ok {
			current = nested
		} else {
			return fmt.Errorf("key path '%s' is not an object at level %d", strings.Join(keys[:i+1], "."), i)
		}
	}

	finalKey := keys[len(keys)-1]

	// Try to convert value to appropriate type
	if value == "true" {
		current[finalKey] = true
	} else if value == "false" {
		current[finalKey] = false
	} else if val, err := json.Number(value).Int64(); err == nil {
		current[finalKey] = val
	} else if val, err := json.Number(value).Float64(); err == nil {
		current[finalKey] = val
	} else {
		current[finalKey] = value
	}

	return nil
}

// Helper function to get nested values using dot notation
func getNestedValue(data map[string]interface{}, key string) (interface{}, error) {
	keys := strings.Split(key, ".")
	current := data

	for i, k := range keys[:len(keys)-1] {
		if nested, ok := current[k].(map[string]interface{}); ok {
			current = nested
		} else {
			return nil, fmt.Errorf("key path '%s' not found at level %d", strings.Join(keys[:i+1], "."), i)
		}
	}

	finalKey := keys[len(keys)-1]
	if value, exists := current[finalKey]; exists {
		return value, nil
	}

	return nil, fmt.Errorf("key '%s' not found", key)
}
