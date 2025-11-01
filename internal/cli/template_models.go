package cli

import (
	"fmt"
	"strings"

	"github.com/arthurlch/goryu/internal/utils"
)

func generateBasicModelContent(name, fields string) string {
	modelName := utils.ToGoIdentifier(name)
	var structFields string
	
	if fields != "" {
		fieldList := parseFields(fields)
		var fieldStrings []string
		fieldStrings = append(fieldStrings, "	ID        int       `json:\"id\"`")
		for _, field := range fieldList {
			fieldStrings = append(fieldStrings, fmt.Sprintf("	%s %s `%s`", field.Name, field.Type, field.Tag))
		}
		fieldStrings = append(fieldStrings, "	CreatedAt time.Time `json:\"created_at\"`")
		fieldStrings = append(fieldStrings, "	UpdatedAt time.Time `json:\"updated_at\"`")
		structFields = strings.Join(fieldStrings, "\n")
	} else {
		structFields = `	ID        int       ` + "`json:\"id\"`" + `
	Name      string    ` + "`json:\"name\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
	UpdatedAt time.Time ` + "`json:\"updated_at\"`"
	}

	return fmt.Sprintf(`package models

import (
	"time"
)

// %s represents a %s in the system
type %s struct {
%s
}

// Validate validates the %s model
func (m *%s) Validate() error {
	return nil
}
`, modelName, strings.ToLower(name), modelName, structFields, strings.ToLower(name), modelName)
}

func generateDBModelContent(name, dbTool, fields string) string {
	switch dbTool {
	case "gorm":
		return generateGormModelContentSimple(name, fields)
	default:
		return generateBasicModelContent(name, fields)
	}
}

func generateGormModelContentSimple(name, fields string) string {
	modelName := utils.ToGoIdentifier(name)
	var structFields string
	
	if fields != "" {
		fieldList := parseFields(fields)
		var fieldStrings []string
		for _, field := range fieldList {
			gormTag := generateGormTag(field)
			fieldStrings = append(fieldStrings, fmt.Sprintf("	%s %s `%s`", field.Name, field.Type, gormTag))
		}
		structFields = strings.Join(fieldStrings, "\n")
	} else {
		structFields = `	Name      string ` + "`gorm:\"size:100;not null\" json:\"name\"`" + `
	Email     string ` + "`gorm:\"size:255;not null;uniqueIndex\" json:\"email\"`"
	}

	return fmt.Sprintf(`package models

import (
	"gorm.io/gorm"
)

// %s represents a %s in the database using GORM
type %s struct {
	gorm.Model
%s
}

// %sRepository provides database operations
type %sRepository struct {
	db *gorm.DB
}

// New%sRepository creates a new repository
func New%sRepository(db *gorm.DB) *%sRepository {
	return &%sRepository{db: db}
}
`, modelName, strings.ToLower(name), modelName, structFields, modelName, modelName, modelName, modelName, modelName, modelName)
}

func generateGormTag(field Field) string {
	gormTags := []string{}
	
	switch field.Type {
	case "string":
		gormTags = append(gormTags, "size:255")
		if strings.ToLower(field.Name) == "email" {
			gormTags = append(gormTags, "uniqueIndex")
		}
	case "int", "int64":
		// No special gorm tags needed for integers
	case "float64":
		gormTags = append(gormTags, "type:decimal(10,2)")
	case "bool":
		gormTags = append(gormTags, "default:false")
	case "time.Time":
		// GORM handles time automatically so well 
	}
	
	if len(gormTags) > 0 {
		return fmt.Sprintf("gorm:\"%s\" %s", strings.Join(gormTags, ";"), field.Tag)
	}
	
	return field.Tag
}
