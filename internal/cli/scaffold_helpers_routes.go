package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arthurlch/goryu/internal/utils"
)

func generateAPIRoutes(resource string, moduleName string, middleware string) error {
	routesDir := "internal/routes"
	if err := os.MkdirAll(routesDir, 0755); err != nil {
		return err
	}

	resourceName := utils.ToGoIdentifier(resource)
	lowerName := strings.ToLower(resource)
	filename := filepath.Join(routesDir, lowerName+".go")

	middlewareStr := ""
	imports := fmt.Sprintf(`"%s/internal/handlers"`, moduleName)
	
	if middleware != "" {
		imports += fmt.Sprintf(`
	"%s/internal/middleware"`, moduleName)
		
		mws := strings.Split(middleware, ",")
		for _, mw := range mws {
			mwName := strings.Title(strings.TrimSpace(mw))
			if mwName == "Validator" {
				// Special handling for validator if needed, or stick to convention
				// For scaffold, we likely generated a validator called Check? Or generic?
				// The scaffold command adds "validator" to middleware list.
				// The generic middleware generator might not have generated "validator" unless we asked.
				// But scaffold usually expects standard ones.
				// Let's assume standard middleware are available as methods or exported builders.
				// If generated middleware is a Builder, we need .Build() or similar.
				// If we look at template_builders.go logic:
				// content += fmt.Sprintf("\n\tgroup.Use(middleware.%s())", mwName)
			}
			middlewareStr += fmt.Sprintf("\n\tgroup.Use(middleware.%s())", mwName)
		}
	}

	content := fmt.Sprintf(`package routes

import (
	"github.com/arthurlch/goryu"
	%s
)

// Register%sRoutes registers all %s routes
func Register%sRoutes(app *goryu.App) {
	// Create a group for %s routes
	group := app.Group("/api/%s")%s

	// Register CRUD routes
	group.GET("/", handlers.List%s)
	group.GET("/:id", handlers.Get%s)
	group.POST("/", handlers.Create%s)
	group.PUT("/:id", handlers.Update%s)
	group.DELETE("/:id", handlers.Delete%s)
}
`, imports, resourceName, lowerName, resourceName, lowerName, lowerName, middlewareStr,
		resourceName, resourceName, resourceName, resourceName, resourceName)

	return os.WriteFile(filename, []byte(content), 0644)
}
