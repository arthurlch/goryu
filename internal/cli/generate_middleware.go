package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func runGenerateMiddleware(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("middleware name is required")
	}

	name := args[0]
	path := "internal/middleware"

	// Parse arguments
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "--path=") {
			path = strings.TrimPrefix(arg, "--path=")
		}
	}

	fmt.Printf("🚀 Generating middleware: %s\n", name)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate middleware file
	filename := filepath.Join(path, strings.ToLower(name)+".go")
	content := generateMiddlewareContent(name)

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write middleware file: %w", err)
	}

	fmt.Printf("✅ Middleware created: %s\n", filename)
	return nil
}
