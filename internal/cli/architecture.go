package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ProjectArchitecture struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Folders     []string          `json:"folders"`
	Files       map[string]string `json:"files"`
	Templates   map[string]string `json:"templates"`
	Variables   map[string]string `json:"variables"`
}

// built in architectures
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
	"db": {
		Name:        "Database Application",
		Description: "Web application with database integration (PostgreSQL by default)",
		Folders: []string{
			"cmd/server",
			"internal/handlers",
			"internal/models",
			"internal/repository",
			"internal/db",
			"sql/migrations",
			"sql/queries",
			"config",
		},
		Files: map[string]string{
			"go.mod":                            "gomod_db",
			"README.md":                         "readme_db",
			"config/config.json":                "api_config",
			"cmd/server/main.go":                "main_db",
			"internal/handlers/health.go":       "health_handler",
			"internal/handlers/home.go":         "home_handler",
			"internal/db/connection.go":         "db_connection",
			"internal/repository/base.go":       "base_repository",
			".gitignore":                        "gitignore",
			"Makefile":                          "makefile",
			"Dockerfile":                        "dockerfile",
		},
		Variables: map[string]string{
			"app_name":  "{{.ProjectName}}",
			"framework": "github.com/arthurlch/goryu",
		},
	},
}

type FlexibleArchitecture struct {
	architectures map[string]ProjectArchitecture
	customPath    string
}

func NewFlexibleArchitecture() *FlexibleArchitecture {
	return &FlexibleArchitecture{
		architectures: make(map[string]ProjectArchitecture),
		customPath:    ".goryu/architectures",
	}
}

func (fa *FlexibleArchitecture) LoadArchitectures() error {
	for name, arch := range defaultArchitectures {
		fa.architectures[name] = arch
	}

	// TODO: Load custom architectures from .goryu/architectures/
	// So we would allow users to define their own project templates and structures
	// I dont want to be too restrictive as Golang conv are about freedom
	// its not rails after all. ..

	return nil
}

func (fa *FlexibleArchitecture) GetArchitecture(name string) (ProjectArchitecture, bool) {
	arch, exists := fa.architectures[name]
	return arch, exists
}

func (fa *FlexibleArchitecture) ListArchitectures() map[string]ProjectArchitecture {
	return fa.architectures
}

func (fa *FlexibleArchitecture) GenerateProject(projectName, archName string, customOptions map[string]string) error {
	arch, exists := fa.GetArchitecture(archName)
	if !exists {
		return fmt.Errorf("architecture '%s' not found", archName)
	}

	fmt.Printf("🏗️  Generating project with '%s' architecture\n", arch.Name)
	fmt.Printf("📋 %s\n", arch.Description)

	// pro direct
	if err := os.MkdirAll(projectName, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	for _, folder := range arch.Folders {
		folderPath := filepath.Join(projectName, folder)
		if err := os.MkdirAll(folderPath, 0755); err != nil {
			return fmt.Errorf("failed to create folder %s: %w", folder, err)
		}
	}

	variables := make(map[string]string)
	for k, v := range arch.Variables {
		variables[k] = v
	}
	for k, v := range customOptions {
		variables[k] = v
	}
	variables["ProjectName"] = projectName

	for filePath, templateName := range arch.Files {
		fullPath := filepath.Join(projectName, filePath)

		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		content, err := fa.generateFileContent(templateName, variables)
		if err != nil {
			return fmt.Errorf("failed to generate content for %s: %w", filePath, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", filePath, err)
		}
	}

	if archName == "db" && customOptions["db_tool"] != "" {
		dbTool := customOptions["db_tool"]
		switch dbTool {
		case "sqlc":
			sqlcFile := filepath.Join(projectName, "sqlc.yaml")
			sqlcContent := generateSQLCConfig()
			if err := os.WriteFile(sqlcFile, []byte(sqlcContent), 0644); err != nil {
				return fmt.Errorf("failed to write sqlc.yaml: %w", err)
			}
		case "ent":
			entDir := filepath.Join(projectName, "ent", "schema")
			if err := os.MkdirAll(entDir, 0755); err != nil {
				return fmt.Errorf("failed to create ent directory: %w", err)
			}
			entConfigFile := filepath.Join(projectName, "ent.go")
			entConfigContent := generateEntConfig()
			if err := os.WriteFile(entConfigFile, []byte(entConfigContent), 0644); err != nil {
				return fmt.Errorf("failed to write ent.go: %w", err)
			}
		case "gorm":
			// GORM doesn't need a specific config file
			gormReadmeFile := filepath.Join(projectName, "GORM.md")
			gormContent := generateGormConfig()
			if err := os.WriteFile(gormReadmeFile, []byte(gormContent), 0644); err != nil {
				return fmt.Errorf("failed to write GORM.md: %w", err)
			}
		}
		
		fmt.Printf("   ✓ Configured for %s\n", dbTool)
	}

	fmt.Printf("✅ Project '%s' created successfully!\n", projectName)
	return nil
}

func (fa *FlexibleArchitecture) generateFileContent(templateName string, variables map[string]string) (string, error) {
	switch templateName {
	case "gomod":
		moduleName := variables["module"]
		if moduleName == "" {
			moduleName = variables["ProjectName"]
		}
		return fa.replaceVariables(generateGoMod(moduleName), variables), nil
	case "gomod_db":
		moduleName := variables["module"]
		if moduleName == "" {
			moduleName = variables["ProjectName"]
		}
		dbTool := variables["db_tool"]
		if dbTool == "" {
			dbTool = "sqlc"
		}
		return fa.replaceVariables(generateGoModWithDB(moduleName, dbTool), variables), nil
	case "readme":
		return fa.replaceVariables(generateReadme(variables["ProjectName"]), variables), nil
	case "readme_db":
		return fa.replaceVariables(generateDBReadme(variables["ProjectName"]), variables), nil
	case "basic_config":
		return generateBasicConfig(), nil
	case "api_config":
		return generateAPIConfig(), nil
	case "main":
		moduleName := variables["module"]
		if moduleName == "" {
			moduleName = variables["ProjectName"]
		}
		return fa.replaceVariables(generateMainFile(moduleName), variables), nil
	case "main_db":
		moduleName := variables["module"]
		if moduleName == "" {
			moduleName = variables["ProjectName"]
		}
		return fa.replaceVariables(generateDBMainFile(moduleName), variables), nil
	case "health_handler":
		return fa.replaceVariables(generateHealthHandler(variables["ProjectName"]), variables), nil
	case "home_handler":
		return fa.replaceVariables(generateHomeHandler(variables["ProjectName"]), variables), nil
	case "db_connection":
		return fa.replaceVariables(generateDBConnection(variables["ProjectName"]), variables), nil
	case "base_repository":
		return fa.replaceVariables(generateBaseRepository(variables["ProjectName"]), variables), nil
	case "makefile":
		return generateMakefile(), nil
	case "dockerfile":
		return generateDockerfile(), nil
	case "sqlc_config":
		return generateSQLCConfig(), nil
	case "keep_file":
		return "", nil
	case "gitignore":
		return generateGitignore(), nil
	default:
		return "", fmt.Errorf("unknown template: %s", templateName)
	}
}

func (fa *FlexibleArchitecture) replaceVariables(content string, variables map[string]string) string {
	result := content
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}
