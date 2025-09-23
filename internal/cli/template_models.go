package cli

import (
	"fmt"
	"github.com/arthurlch/goryu/internal/utils"
	"strings"
)

// Model template generators
func generateBasicModelContent(name string) string {
	modelName := utils.ToGoIdentifier(name)

	return fmt.Sprintf(`package models

import (
	"time"
)

// %s represents a %s in the system
type %s struct {
	ID        int       `+"`json:\"id\"`"+`
	Name      string    `+"`json:\"name\"`"+`
	CreatedAt time.Time `+"`json:\"created_at\"`"+`
	UpdatedAt time.Time `+"`json:\"updated_at\"`"+`
}

// Validate validates the %s model
func (m *%s) Validate() error {
	return nil
}
`, modelName, strings.ToLower(name), modelName, strings.ToLower(name), modelName)
}

func generateDBModelContent(name, dbTool string) string {
	switch dbTool {
	case "gorm":
		return generateGormModelContentSimple(name)
	default:
		return generateBasicModelContent(name)
	}
}

func generateGormModelContentSimple(name string) string {
	modelName := utils.ToGoIdentifier(name)

	return fmt.Sprintf(`package models

import (
	"gorm.io/gorm"
)

// %s represents a %s in the database using GORM
type %s struct {
	gorm.Model
	Name      string `+"`gorm:\"size:100;not null\" json:\"name\"`"+`
	Email     string `+"`gorm:\"size:255;not null;uniqueIndex\" json:\"email\"`"+`
}

// %sRepository provides database operations
type %sRepository struct {
	db *gorm.DB
}

// New%sRepository creates a new repository
func New%sRepository(db *gorm.DB) *%sRepository {
	return &%sRepository{db: db}
}
`, modelName, strings.ToLower(name), modelName, modelName, modelName, modelName, modelName, modelName, modelName)
}
