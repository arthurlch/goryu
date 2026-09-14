package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func isSafeName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

func runGenerateHandler(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("handler name is required")
	}

	name := args[0]
	if !isSafeName(name) {
		return fmt.Errorf("invalid handler name %q: use letters, digits, '-' or '_'", name)
	}
	path := "internal/handlers"
	handlerType := "basic" // basic, crud, api (websocket later), crud default ;/

	// Parse arguments
	dbTool := ""
	kind := "chat"
	for _, arg := range args[1:] {
		switch {
		case strings.HasPrefix(arg, "--path="):
			path = strings.TrimPrefix(arg, "--path=")
		case strings.HasPrefix(arg, "--type="):
			handlerType = strings.TrimPrefix(arg, "--type=")
		case strings.HasPrefix(arg, "--db-tool="):
			dbTool = strings.TrimPrefix(arg, "--db-tool=")
		case strings.HasPrefix(arg, "--kind="):
			kind = strings.TrimPrefix(arg, "--kind=")
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
	case "ai":
		if kind != "chat" && kind != "rag" && kind != "agent" {
			return fmt.Errorf("unknown ai kind: %s (available: chat, rag, agent)", kind)
		}
		content = generateAIHandlerContent(name, kind)
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
		return fmt.Errorf("unknown handler type: %s (available: basic, crud, api, ai)", handlerType)
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
	case "ai":
		fmt.Printf("  • Register routes:\n")
		fmt.Printf("    - app.POST(\"/chat/stream\", handlers.ChatStream)  // SSE token streaming\n")
		fmt.Printf("    - app.GET(\"/chat/schema\", handlers.ChatSchema)   // JSON Schema\n")
		fmt.Printf("  • Meter tokens/cost: app.Use(aimeter.New(aimeter.Config{...}))\n")
		fmt.Printf("  • Cache prompts:     app.Use(promptcache.New())\n")
		fmt.Printf("  • Record for evals:  app.Use(recorder.New(recorder.Config{Sink: sink}))\n")
		fmt.Printf("  • Replace the stubbed reply with a real provider call (see aiproxy)\n")
	}
}
