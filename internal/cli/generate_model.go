package cli

import (
	"fmt"
	"github.com/arthurlch/goryu/internal/utils"
	"os"
	"path/filepath"
	"strings"
)

func runGenerateModel(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("model name is required")
	}

	name := args[0]
	path := "internal/models"
	modelType := "basic" // basic, db
	dbTool := "sqlc"     // sqlc, ent, gorm
	fields := ""

	// Parse arguments
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "--path=") {
			path = strings.TrimPrefix(arg, "--path=")
		} else if strings.HasPrefix(arg, "--type=") {
			modelType = strings.TrimPrefix(arg, "--type=")
		} else if strings.HasPrefix(arg, "--db-tool=") {
			dbTool = strings.TrimPrefix(arg, "--db-tool=")
		} else if strings.HasPrefix(arg, "--fields=") {
			fields = strings.TrimPrefix(arg, "--fields=")
		}
	}

	if modelType == "db" {
		validDBTools := []string{"sqlc", "ent", "gorm"}
		isValid := false
		for _, valid := range validDBTools {
			if dbTool == valid {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("unknown db-tool: %s (available: sqlc, ent, gorm)", dbTool)
		}
		fmt.Printf("🚀 Generating %s model with %s: %s\n", modelType, dbTool, name)
	} else {
		fmt.Printf("🚀 Generating %s model: %s\n", modelType, name)
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	filename := filepath.Join(path, strings.ToLower(name)+".go")
	var content string

	switch modelType {
	case "basic":
		content = generateBasicModelContent(name, fields)
	case "db":
		content = generateDBModelContent(name, dbTool, fields)
	default:
		return fmt.Errorf("unknown model type: %s (available: basic, db)", modelType)
	}

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write model file: %w", err)
	}

	fmt.Printf("✅ Model created: %s\n", filename)
	printModelTips(modelType, name, dbTool)
	return nil
}

func printModelTips(modelType, name, dbTool string) {
	fmt.Printf("\n💡 Model Tips:\n")
	switch modelType {
	case "basic":
		fmt.Printf("  • Import in handlers: \"your-app/internal/models\"\n")
		fmt.Printf("  • Add validation logic in Validate() method\n")
		fmt.Printf("  • Customize fields as needed\n")
	case "db":
		switch dbTool {
		case "sqlc":
			fmt.Printf("  • Create SQL schema in sql/migrations/\n")
			fmt.Printf("  • Write SQL queries in sql/queries/\n")
			fmt.Printf("  • Run: sqlc generate (generates Go code from SQL)\n")
			fmt.Printf("  • Use the generated queries in your repository\n")
			fmt.Printf("  • Update your model struct to match the generated types\n")
			fmt.Printf("  • Example workflow:\n")
			fmt.Printf("    1. Create migration: make migrate-create name=create_%s_table\n", strings.ToLower(name))
			fmt.Printf("    2. Write SQL queries in sql/queries/%s.sql\n", strings.ToLower(name))
			fmt.Printf("    3. Generate Go code: make sqlc-generate\n")
		case "ent":
			fmt.Printf("  • Initialize Ent: go run entgo.io/ent/cmd/ent init %s\n", utils.ToGoIdentifier(name))
			fmt.Printf("  • Update the schema file in ent/schema/%s.go\n", strings.ToLower(name))
			fmt.Printf("  • Generate code: go generate ./ent\n")
			fmt.Printf("  • Use the generated client in your handlers\n")
			fmt.Printf("  • Example workflow:\n")
			fmt.Printf("    1. Update Fields() method with your database fields\n")
			fmt.Printf("    2. Add relationships in Edges() method\n")
			fmt.Printf("    3. Generate: go generate ./ent\n")
			fmt.Printf("    4. Use client.%s methods in your repository\n", utils.ToGoIdentifier(name))
		case "gorm":
			fmt.Printf("  • Model is ready to use with GORM\n")
			fmt.Printf("  • Auto-migrate: db.AutoMigrate(&%s{})\n", utils.ToGoIdentifier(name))
			fmt.Printf("  • Use the repository methods or GORM directly\n")
			fmt.Printf("  • Example workflow:\n")
			fmt.Printf("    1. Initialize GORM connection\n")
			fmt.Printf("    2. Auto-migrate your models\n")
			fmt.Printf("    3. Use repository pattern or GORM methods directly\n")
			fmt.Printf("  • GORM Documentation: https://gorm.io/\n")
		default:
			fmt.Printf("  • Database tool: %s\n", dbTool)
			fmt.Printf("  • Check the generated model for specific integration steps\n")
		}
	}
}
