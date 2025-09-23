package cli

import (
	"fmt"
	"os"
	"strings"
)

func newInitCommand() *Command {
	return &Command{
		Name:        "init",
		Description: "Initialize a new Goryu project",
		Usage:       "goryu init [project-name] [--template=basic|api|web]",
		Action:      runInit,
	}
}

func runInit(args []string) error {
	var projectName string
	template := "basic"
	// Parse arguments
	for i, arg := range args {
		if strings.HasPrefix(arg, "--template=") {
			template = strings.TrimPrefix(arg, "--template=")
		} else if i == 0 {
			projectName = arg
		}
	}

	if projectName == "" {
		projectName = "goryu-app"
	}

	// Check if directory already exists
	if _, err := os.Stat(projectName); err == nil {
		return fmt.Errorf("directory %s already exists", projectName)
	}

	// Use FlexibleArchitecture for all templates
	fa := NewFlexibleArchitecture()
	if err := fa.LoadArchitectures(); err != nil {
		return fmt.Errorf("failed to load architectures: %w", err)
	}

	return fa.GenerateProject(projectName, template, nil)
}
