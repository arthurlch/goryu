package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectArchitecture defines the structure of a project
type ProjectArchitecture struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Folders     []string          `json:"folders"`
	Files       map[string]string `json:"files"`
	Templates   map[string]string `json:"templates"`
	Variables   map[string]string `json:"variables"`
}

// Built-in architectures - simplified to avoid boilerplate
var defaultArchitectures = map[string]ProjectArchitecture{
	"basic": {
		Name:        "Basic Web Application",
		Description: "Simple web application with minimal structure",
		Folders: []string{
			"cmd/server",
			"internal/handlers",
			"internal/models",
			"config",
		},
		Files: map[string]string{
			"go.mod":                      "gomod",
			"README.md":                   "readme",
			"config/config.json":          "basic_config",
			"cmd/server/main.go":          "main",
			"internal/handlers/health.go": "health_handler",
			"internal/handlers/home.go":   "home_handler",
			".gitignore":                  "gitignore",
		},
		Variables: map[string]string{
			"app_name":  "{{.ProjectName}}",
			"framework": "github.com/arthurlch/goryu",
		},
	},
	"api": {
		Name:        "REST API Application",
		Description: "REST API with enhanced structure",
		Folders: []string{
			"cmd/server",
			"internal/handlers",
			"internal/models",
			"internal/repository",
			"config",
		},
		Files: map[string]string{
			"go.mod":                      "gomod",
			"README.md":                   "readme",
			"config/config.json":          "api_config",
			"cmd/server/main.go":          "main",
			"internal/handlers/health.go": "health_handler",
			"internal/handlers/home.go":   "home_handler",
			".gitignore":                  "gitignore",
		},
		Variables: map[string]string{
			"app_name":  "{{.ProjectName}}",
			"framework": "github.com/arthurlch/goryu",
		},
	},
}

// FlexibleArchitecture allows for custom architectures
type FlexibleArchitecture struct {
	architectures map[string]ProjectArchitecture
	customPath    string
}

// NewFlexibleArchitecture creates a new flexible architecture manager
func NewFlexibleArchitecture() *FlexibleArchitecture {
	return &FlexibleArchitecture{
		architectures: make(map[string]ProjectArchitecture),
		customPath:    ".goryu/architectures",
	}
}

// LoadArchitectures loads both default and custom architectures
func (fa *FlexibleArchitecture) LoadArchitectures() error {
	// Load default architectures
	for name, arch := range defaultArchitectures {
		fa.architectures[name] = arch
	}

	// TODO: Load custom architectures from .goryu/architectures/
	// This would allow users to define their own project templates

	return nil
}

// GetArchitecture returns an architecture by name
func (fa *FlexibleArchitecture) GetArchitecture(name string) (ProjectArchitecture, bool) {
	arch, exists := fa.architectures[name]
	return arch, exists
}

// ListArchitectures returns all available architectures
func (fa *FlexibleArchitecture) ListArchitectures() map[string]ProjectArchitecture {
	return fa.architectures
}

// GenerateProject creates a project using the specified architecture
func (fa *FlexibleArchitecture) GenerateProject(projectName, archName string, customOptions map[string]string) error {
	arch, exists := fa.GetArchitecture(archName)
	if !exists {
		return fmt.Errorf("architecture '%s' not found", archName)
	}

	fmt.Printf("🏗️  Generating project with '%s' architecture\n", arch.Name)
	fmt.Printf("📋 %s\n", arch.Description)

	// Create project directory
	if err := os.MkdirAll(projectName, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Create folder structure
	for _, folder := range arch.Folders {
		folderPath := filepath.Join(projectName, folder)
		if err := os.MkdirAll(folderPath, 0755); err != nil {
			return fmt.Errorf("failed to create folder %s: %w", folder, err)
		}
	}

	// Prepare template variables
	variables := make(map[string]string)
	for k, v := range arch.Variables {
		variables[k] = v
	}
	// Override with custom options
	for k, v := range customOptions {
		variables[k] = v
	}
	variables["ProjectName"] = projectName

	// Generate files
	for filePath, templateName := range arch.Files {
		fullPath := filepath.Join(projectName, filePath)

		// Ensure directory exists
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Generate file content
		content, err := fa.generateFileContent(templateName, variables)
		if err != nil {
			return fmt.Errorf("failed to generate content for %s: %w", filePath, err)
		}

		// Write file
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", filePath, err)
		}
	}

	fmt.Printf("✅ Project '%s' created successfully!\n", projectName)
	return nil
}

// generateFileContent generates content for a file based on template name and variables
func (fa *FlexibleArchitecture) generateFileContent(templateName string, variables map[string]string) (string, error) {
	switch templateName {
	case "gomod":
		return fa.replaceVariables(generateGoMod(variables["ProjectName"]), variables), nil
	case "readme":
		return fa.replaceVariables(generateReadme(variables["ProjectName"]), variables), nil
	case "basic_config":
		return generateBasicConfig(), nil
	case "api_config":
		return generateAPIConfig(), nil
	case "main":
		return fa.replaceVariables(generateMainFile(variables["ProjectName"]), variables), nil
	case "health_handler":
		return fa.replaceVariables(generateHealthHandler(variables["ProjectName"]), variables), nil
	case "home_handler":
		return fa.replaceVariables(generateHomeHandler(variables["ProjectName"]), variables), nil
	case "gitignore":
		return generateGitignore(), nil
	default:
		return "", fmt.Errorf("unknown template: %s", templateName)
	}
}

// replaceVariables replaces template variables in content
func (fa *FlexibleArchitecture) replaceVariables(content string, variables map[string]string) string {
	result := content
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}
