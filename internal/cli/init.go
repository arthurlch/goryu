package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

func newInitCommand() *Command {
	return &Command{
		Name:        "init",
		Description: "Initialize a new Goryu project",
		Usage:       "goryu init [project-name] [flags]",
		Flags: []Flag{
			{Name: "template", Shorthand: "t", Description: "Project template (basic, api, db)", Default: "basic"},
			{Name: "path", Shorthand: "p", Description: "Project path", Default: "."},
			{Name: "module", Shorthand: "m", Description: "Go module name"},
			{Name: "git", Description: "Initialize git repository", Default: true},
			{Name: "docker", Description: "Include Docker files", Default: false},
			{Name: "ci", Description: "Include CI/CD configs", Default: false},
			{Name: "db-tool", Description: "Database tool (sqlc, ent, gorm) - only for db template", Default: "sqlc"},
		},
		Action: cmdInit,
	}
}

func runInit(projectName string, template string, projectPath string, module string, dbTool string) error {
	if projectName == "" {
		projectName = "goryu-app"
	}

	targetPath := projectName
	if projectPath != "." && projectPath != "" {
		targetPath = filepath.Join(projectPath, projectName)
	}

	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("directory %s already exists", targetPath)
	}

	fa := NewFlexibleArchitecture()
	if err := fa.LoadArchitectures(); err != nil {
		return fmt.Errorf("failed to load architectures: %w", err)
	}

	customOptions := map[string]string{
		"module":  module,
		"path":    targetPath,
		"db_tool": dbTool,
	}

	return fa.GenerateProject(projectName, template, customOptions)
}
