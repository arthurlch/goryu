package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func runGenerateHandler(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("handler name is required")
	}

	name := args[0]
	path := "internal/handlers"
	handlerType := "basic" // basic, crud, api (websocket later), crud default ;/

	// Parse arguments
	dbTool := ""
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "--path=") {
			path = strings.TrimPrefix(arg, "--path=")
		} else if strings.HasPrefix(arg, "--type=") {
			handlerType = strings.TrimPrefix(arg, "--type=")
		} else if strings.HasPrefix(arg, "--db-tool=") {
			dbTool = strings.TrimPrefix(arg, "--db-tool=")
		}
	}

	fmt.Printf("🚀 Generating %s handler: %s\n", handlerType, name)
	if dbTool != "" {
		fmt.Printf("   Database Tool: %s\n", dbTool)
		handlerType = "db" // Override type to db specific
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	filename := filepath.Join(path, strings.ToLower(name)+".go")
	var content string

	switch handlerType {
	case "basic":
		content = generateBasicHandlerContent(name)
	case "crud":
		content = generateCRUDHandlerContent(name)
	case "api":
		content = generateAPIHandlerContent(name)
	case "db":
		switch dbTool {
		case "sqlc":
			content = generateSQLCHandlerContent(name)
		case "ent":
			content = generateEntHandlerContent(name)
		case "gorm":
			content = generateGormHandlerContent(name)
		default:
			return fmt.Errorf("unknown db-tool: %s", dbTool)
		}
	default:
		return fmt.Errorf("unknown handler type: %s (available: basic, crud, api)", handlerType)
	}

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write handler file: %w", err)
	}

	fmt.Printf("✅ Handler created: %s\n", filename)
	printHandlerTips(handlerType)
	return nil
}

func printHandlerTips(handlerType string) {
	fmt.Printf("\n💡 Handler Tips:\n")
	switch handlerType {
	case "basic":
		fmt.Printf("  • Register route: app.GET(\"/path\", handlers.%s)\n", "HandlerName")
		fmt.Printf("  • Implement your business logic\n")
		fmt.Printf("  • Add input validation as needed\n")
	case "crud":
		fmt.Printf("  • Register CRUD routes:\n")
		fmt.Printf("    - app.GET(\"/items\", handlers.ListItems)\n")
		fmt.Printf("    - app.GET(\"/items/:id\", handlers.GetItem)\n")
		fmt.Printf("    - app.POST(\"/items\", handlers.CreateItem)\n")
		fmt.Printf("    - app.PUT(\"/items/:id\", handlers.UpdateItem)\n")
		fmt.Printf("    - app.DELETE(\"/items/:id\", handlers.DeleteItem)\n")
		fmt.Printf("  • Replace map[string]interface{} with proper structs\n")
		fmt.Printf("  • Add validation and business logic\n")
	case "api":
		fmt.Printf("  • Register route: app.Any(\"/api/resource\", api.HandleResource)\n")
		fmt.Printf("  • Customize Request/Response structs\n")
		fmt.Printf("  • Add validation tags and business logic\n")
		fmt.Printf("  • Consider adding authentication middleware\n")
		// MEMRO: recheck for websocket later ...
	}
}
