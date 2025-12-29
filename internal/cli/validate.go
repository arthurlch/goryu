package cli

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/arthurlch/goryu/config/builder"
)

func newValidateCommand() *Command {
	return &Command{
		Name:        "validate",
		Description: "Validate project structure",
		Usage:       "goryu validate [flags]",
		Flags: []Flag{
			{Name: "fix", Description: "Auto-fix issues", Default: false},
		},
		Action: cmdValidate,
	}
}

func runValidate(args []string) error {
	configFile := "config/config.json"
	validateProject := true

	for _, arg := range args {
		if strings.HasPrefix(arg, "--config=") {
			configFile = strings.TrimPrefix(arg, "--config=")
		} else if arg == "--project" {
			validateProject = true
		}
	}

	fmt.Println("🔍 Validating Goryu project...")

	var issues []string
	var warnings []string

	fmt.Printf("\n📋 Validating configuration (%s)...\n", configFile)
	if configIssues := validateConfiguration(configFile); len(configIssues) > 0 {
		issues = append(issues, configIssues...)
	} else {
		fmt.Println("✅ Configuration is valid")
	}

	if validateProject {
		fmt.Println("\n📁 Validating project structure...")
		if projectIssues, projectWarnings := validateProjectStructure(); len(projectIssues) > 0 {
			issues = append(issues, projectIssues...)
			warnings = append(warnings, projectWarnings...)
		} else {
			fmt.Println("✅ Project structure looks good")
		}

		fmt.Println("\n🔧 Validating Go files...")
		if goIssues := validateGoFiles(); len(goIssues) > 0 {
			issues = append(issues, goIssues...)
		} else {
			fmt.Println("✅ Go files are syntactically valid")
		}

		fmt.Println("\n📦 Validating dependencies...")
		if depIssues := validateDependencies(); len(depIssues) > 0 {
			warnings = append(warnings, depIssues...)
		} else {
			fmt.Println("✅ Dependencies look good")
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("VALIDATION SUMMARY")
	fmt.Println(strings.Repeat("=", 50))

	if len(issues) == 0 && len(warnings) == 0 {
		fmt.Println("🎉 All validations passed! Your project looks great.")
		return nil
	}

	if len(issues) > 0 {
		fmt.Printf("❌ Found %d issue(s):\n", len(issues))
		for i, issue := range issues {
			fmt.Printf("  %d. %s\n", i+1, issue)
		}
	}

	if len(warnings) > 0 {
		fmt.Printf("⚠️  Found %d warning(s):\n", len(warnings))
		for i, warning := range warnings {
			fmt.Printf("  %d. %s\n", i+1, warning)
		}
	}

	if len(issues) > 0 {
		fmt.Println("\n💡 Fix the issues above and run validation again.")
		return fmt.Errorf("validation failed with %d issue(s)", len(issues))
	}

	fmt.Println("\n💡 Consider addressing the warnings to improve your project.")
	return nil
}

func validateConfiguration(configFile string) []string {
	var issues []string

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		issues = append(issues, fmt.Sprintf("Configuration file not found: %s", configFile))
		return issues
	}

	cfgBuilder, err := builder.FromFile(configFile)
	if err != nil {
		issues = append(issues, fmt.Sprintf("Configuration loading failed: %v", err))
		return issues
	}

	cfgBuilder.Validate()
	if cfgBuilder.HasErrors() {
		for _, validationErr := range cfgBuilder.Errors() {
			issues = append(issues, fmt.Sprintf("Configuration validation error: %v", validationErr))
		}
		return issues
	}

	cfg, err := cfgBuilder.Build()
	if err != nil {
		issues = append(issues, fmt.Sprintf("Configuration build failed: %v", err))
		return issues
	}

	if cfg.Server.Port < 1024 && cfg.Server.Host != "localhost" && cfg.Server.Host != "127.0.0.1" {
		issues = append(issues, "Using privileged port (<1024) with non-localhost host requires root privileges")
	}

	if cfg.App.Environment == "production" && (cfg.App.Name == "goryu-app" || cfg.App.Name == "") {
		issues = append(issues, "Using default or empty app name in production environment - consider setting a unique name")
	}

	if cfg.App.Environment == "production" && cfg.Server.TLS.Enabled && cfg.Server.TLS.MinVersion != "TLS1.2" && cfg.Server.TLS.MinVersion != "TLS1.3" {
		issues = append(issues, "TLS minimum version should be TLS1.2 or higher in production")
	}

	return issues
}

func validateProjectStructure() ([]string, []string) {
	var issues []string
	var warnings []string

	essentialFiles := []string{
		"go.mod",
		"cmd/server/main.go",
	}

	for _, file := range essentialFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("Missing essential file: %s", file))
		}
	}

	recommendedDirs := []string{
		"internal/handlers",
		"internal/models",
		"config",
	}

	for _, dir := range recommendedDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			warnings = append(warnings, fmt.Sprintf("Recommended directory missing: %s", dir))
		}
	}

	dockerFiles := []string{"Dockerfile", "docker-compose.yml"}
	hasDocker := false
	for _, file := range dockerFiles {
		if _, err := os.Stat(file); err == nil {
			hasDocker = true
			break
		}
	}
	if !hasDocker {
		warnings = append(warnings, "No Docker configuration found (Dockerfile or docker-compose.yml)")
	}

	if _, err := os.Stat(".gitignore"); os.IsNotExist(err) {
		warnings = append(warnings, "No .gitignore file found")
	}

	readmeFiles := []string{"README.md", "README.txt", "README"}
	hasReadme := false
	for _, file := range readmeFiles {
		if _, err := os.Stat(file); err == nil {
			hasReadme = true
			break
		}
	}
	if !hasReadme {
		warnings = append(warnings, "No README file found")
	}

	return issues, warnings
}

func validateGoFiles() []string {
	var issues []string

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && (info.Name() == "vendor" || info.Name() == ".git") {
			return filepath.SkipDir
		}

		if strings.HasSuffix(path, ".go") {
			if goIssues := validateGoFile(path); len(goIssues) > 0 {
				issues = append(issues, goIssues...)
			}
		}

		return nil
	})

	if err != nil {
		issues = append(issues, fmt.Sprintf("Error walking directory: %v", err))
	}

	return issues
}

func validateGoFile(filename string) []string {
	var issues []string

	fset := token.NewFileSet()
	src, err := os.ReadFile(filename)
	if err != nil {
		issues = append(issues, fmt.Sprintf("Cannot read file %s: %v", filename, err))
		return issues
	}

	_, err = parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		issues = append(issues, fmt.Sprintf("Syntax error in %s: %v", filename, err))
	}

	return issues
}

func validateDependencies() []string {
	var warnings []string

	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		warnings = append(warnings, "go.mod file not found - project may not be a Go module")
		return warnings
	}

	content, err := os.ReadFile("go.mod")
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Cannot read go.mod: %v", err))
		return warnings
	}

	goModContent := string(content)

	if !strings.Contains(goModContent, "github.com/arthurlch/goryu") {
		warnings = append(warnings, "Goryu dependency not found in go.mod")
	}

	if !strings.Contains(goModContent, "go 1.") {
		warnings = append(warnings, "Go version not specified in go.mod")
	} else {
		if strings.Contains(goModContent, "go 1.20") || strings.Contains(goModContent, "go 1.19") {
			warnings = append(warnings, "Consider updating to Go 1.21+ for better performance")
		}
	}

	return warnings
}

func validateGoSyntax(filename string) error {
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("syntax error: %v", err)
	}

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if x.Name.IsExported() && x.Doc == nil {
				// Maybe shall be warning ??
			}
		case *ast.GenDecl:
			if x.Tok == token.TYPE {
				for _, spec := range x.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						if ts.Name.IsExported() && x.Doc == nil {
							// same than above ..
						}
					}
				}
			}
		}
		return true
	})

	return nil
}
