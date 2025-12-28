package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/arthurlch/goryu/config/builder"
)

// TODO: Consider exposing show, set, and get commands in the future

func runConfigInit(args []string) error {
	configType := "basic"
	filename := "config.json"

	for _, arg := range args {
		if strings.HasPrefix(arg, "--type=") {
			configType = strings.TrimPrefix(arg, "--type=")
		} else if strings.HasPrefix(arg, "--file=") {
			filename = strings.TrimPrefix(arg, "--file=")
		}
	}

	fmt.Printf("🔧 Creating %s configuration file: %s\n", configType, filename)

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
	filename := "config/config.json"

	for _, arg := range args {
		if strings.HasPrefix(arg, "--file=") {
			filename = strings.TrimPrefix(arg, "--file=")
		}
	}

	fmt.Printf("🔍 Validating configuration file: %s\n", filename)

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s", filename)
	}

	cfg, err := builder.FromFile(filename)
	if err != nil {
		fmt.Printf("❌ Configuration loading failed:\n")
		fmt.Printf("   %v\n", err)
		return err
	}

	cfg.Validate()
	if cfg.HasErrors() {
		fmt.Printf("❌ Configuration validation failed:\n")
		for _, validationErr := range cfg.Errors() {
			fmt.Printf("   %v\n", validationErr)
		}
		return fmt.Errorf("configuration validation failed")
	}

	config, err := cfg.Build()
	if err != nil {
		return err
	}

	fmt.Printf("✅ Configuration is valid!\n")
	fmt.Printf("\n📋 Configuration summary:\n")
	fmt.Printf("  • App: %s v%s\n", config.App.Name, config.App.Version)
	fmt.Printf("  • Server: %s:%d\n", config.Server.Host, config.Server.Port)
	fmt.Printf("  • Environment: %s\n", config.App.Environment)

	return nil
}

// MEMQ: The following functions are not currently exposed through the CLI but could be useful in the future:
// - runConfigShow: Display current configuration in different formats
// - runConfigSet: Modify configuration values using dot notation
// - runConfigGet: Read specific configuration values
