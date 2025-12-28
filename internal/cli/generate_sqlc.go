package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/arthurlch/goryu/internal/utils"
)

func generateSQLCSetup(name, fields string) error {
	// 1. Ensure directories exist
	if err := os.MkdirAll("sql/migrations", 0755); err != nil {
		return err
	}
	if err := os.MkdirAll("sql/queries", 0755); err != nil {
		return err
	}
	if err := os.MkdirAll("internal/database", 0755); err != nil {
		return err
	}

	// 2. Create sqlc.yaml if not exists
	if _, err := os.Stat("sqlc.yaml"); os.IsNotExist(err) {
		yamlContent := `version: "2"
sql:
  - engine: "postgresql"
    queries: "sql/queries"
    schema: "sql/migrations"
    gen:
      go:
        package: "database"
        out: "internal/database"
        sql_package: "pgx/v5"
`
		if err := os.WriteFile("sqlc.yaml", []byte(yamlContent), 0644); err != nil {
			return fmt.Errorf("failed to create sqlc.yaml: %w", err)
		}
		fmt.Println("✅ Created sqlc.yaml configuration")
	}

	// 3. Generate Schema Migration (basic)
	tableName := strings.ToLower(utils.ToSnakeCase(name)) + "s"
	migrationContent := generateSQLMigration(tableName, fields)
	// Check if already exists to avoid overwrite?
	// specific logic omitted for brevity, let's just write a generic one or skip if specific 001 exists?
	// For now let's just create a file with a likely unique name or just use the name provided
	migrationFile := fmt.Sprintf("sql/migrations/%s_create_%s.sql", utils.GetTimestamp(), tableName)

	if err := os.WriteFile(migrationFile, []byte(migrationContent), 0644); err != nil {
		return err
	}
	fmt.Printf("✅ Created migration: %s\n", migrationFile)

	// 4. Generate Queries
	queryContent := generateSQLQueries(name, tableName, fields)
	queryFile := fmt.Sprintf("sql/queries/%s.sql", strings.ToLower(name))
	if err := os.WriteFile(queryFile, []byte(queryContent), 0644); err != nil {
		return err
	}
	fmt.Printf("✅ Created queries: %s\n", queryFile)

	return nil
}

func generateSQLMigration(tableName, fields string) string {
	var fieldDefs []string
	fieldList := parseFields(fields)

	fieldDefs = append(fieldDefs, "id BIGSERIAL PRIMARY KEY")

	for _, field := range fieldList {
		sqlType := "TEXT"
		switch field.Type {
		case "int", "int64":
			sqlType = "BIGINT"
		case "float64":
			sqlType = "DECIMAL(10,2)"
		case "bool":
			sqlType = "BOOLEAN"
		case "time.Time":
			sqlType = "TIMESTAMPTZ"
		case "uuid.UUID":
			sqlType = "UUID"
		}

		nullable := "NOT NULL"
		if strings.HasPrefix(field.Type, "*") {
			nullable = ""
		}

		fieldDefs = append(fieldDefs, fmt.Sprintf("%s %s %s", utils.ToSnakeCase(field.Name), sqlType, nullable))
	}

	fieldDefs = append(fieldDefs, "created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()")
	fieldDefs = append(fieldDefs, "updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()")

	return fmt.Sprintf(`CREATE TABLE %s (
  %s
);
`, tableName, strings.Join(fieldDefs, ",\n  "))
}

func generateSQLQueries(name, tableName, fields string) string {
	fieldList := parseFields(fields)
	// Build column lists
	var columns []string
	var insertCols []string
	var insertProject []string
	var updateSet []string

	for i, field := range fieldList {
		col := utils.ToSnakeCase(field.Name)
		columns = append(columns, col)
		insertCols = append(insertCols, col)
		insertProject = append(insertProject, fmt.Sprintf("$%d", i+1))
		updateSet = append(updateSet, fmt.Sprintf("%s = $%d", col, i+2))
	}

	colsJoined := strings.Join(columns, ", ")
	insertColsJoined := strings.Join(insertCols, ", ")
	insertParams := strings.Join(insertProject, ", ")
	updateSetJoined := strings.Join(updateSet, ", ")

	return fmt.Sprintf(`-- name: Get%s :one
SELECT id, %s, created_at, updated_at FROM %s
WHERE id = $1 LIMIT 1;

-- name: List%ss :many
SELECT id, %s, created_at, updated_at FROM %s
ORDER BY name;

-- name: Create%s :one
INSERT INTO %s (
  %s
) VALUES (
  %s
)
RETURNING id, %s, created_at, updated_at;

-- name: Update%s :one
UPDATE %s
SET %s, updated_at = NOW()
WHERE id = $1
RETURNING id, %s, created_at, updated_at;

-- name: Delete%s :exec
DELETE FROM %s
WHERE id = $1;
`, name, colsJoined, tableName,
		name, colsJoined, tableName,
		name, tableName, insertColsJoined, insertParams, colsJoined,
		name, tableName, updateSetJoined, colsJoined,
		name, tableName)
}
