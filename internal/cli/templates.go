package cli

import (
	"fmt"
)

func generateGoMod(projectName string) string {
	return fmt.Sprintf(`module %s

go 1.21

require (
	github.com/arthurlch/goryu v0.1.0
)
`, projectName)
}

func generateGoModWithDB(projectName, dbTool string) string {
	var dependencies string

	switch dbTool {
	case "sqlc":
		dependencies = `	github.com/arthurlch/goryu v0.1.0
	github.com/jackc/pgx/v5 v5.4.3
	github.com/golang-migrate/migrate/v4 v4.16.2`
	case "ent":
		dependencies = `	github.com/arthurlch/goryu v0.1.0
	entgo.io/ent v0.12.4
	github.com/jackc/pgx/v5 v5.4.3`
	case "gorm":
		dependencies = `	github.com/arthurlch/goryu v0.1.0
	gorm.io/gorm v1.25.5
	gorm.io/driver/postgres v1.5.3`
	default:
		dependencies = `	github.com/arthurlch/goryu v0.1.0
	github.com/jackc/pgx/v5 v5.4.3`
	}

	return fmt.Sprintf(`module %s

go 1.21

require (
%s
)
`, projectName, dependencies)
}

func generateReadme(projectName string) string {
	return fmt.Sprintf(`# %s

A Goryu web application.

## Getting Started

1. Install dependencies:
   `+"```"+`bash
   go mod tidy
   `+"```"+`

2. Run the application:
   `+"```"+`bash
   go run cmd/server/main.go
   `+"```"+`

3. Visit http://localhost:8080

## Configuration

Edit config/config.json to customize your application.

## Features

- Built with Goryu web framework
- Configuration management
- Health checks
- Docker support
`, projectName)
}

func generateMainFile(projectName string) string {
	return fmt.Sprintf(`package main

import (
	"log"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/config"
	"%s/internal/handlers"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %%v", err)
	}

	// Create Goryu app with configuration
	goryuCfg := cfg.ToGoryuConfig()
	app := goryu.New(goryu.Config{
		AppName:               goryuCfg.AppName,
		ServerHeader:          goryuCfg.ServerHeader,
		StrictRouting:         goryuCfg.StrictRouting,
		CaseSensitive:         goryuCfg.CaseSensitive,
		DisableStartupMessage: goryuCfg.DisableStartupMessage,
	})

	// Register routes
	app.GET("/", handlers.Home)
	app.GET("/health", handlers.Health)

	log.Printf("Starting server on %%s", cfg.GetServerAddress())
	if err := app.Run(cfg.GetServerAddress()); err != nil {
		log.Fatalf("Server failed to start: %%v", err)
	}
}
`, projectName)
}

func generateHealthHandler(projectName string) string {
	return fmt.Sprintf(`package handlers

import (
	"net/http"
	"time"

	"github.com/arthurlch/goryu"
)

// Health returns the health status of the application
func Health(c *goryu.Context) {
	c.JSON(http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"service":   "%s",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "1.0.0",
	})
}
`, projectName)
}

func generateHomeHandler(projectName string) string {
	return fmt.Sprintf(`package handlers

import (
	"net/http"

	"github.com/arthurlch/goryu"
)

// Home handles the home page
func Home(c *goryu.Context) {
	c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Welcome to %s!",
		"status":  "success",
	})
}
`, projectName)
}

func generateBasicConfig() string {
	return `{
  "app": {
    "name": "goryu-app",
    "version": "1.0.0"
  },
  "server": {
    "host": "localhost",
    "port": 8080,
    "read_timeout": "30s",
    "write_timeout": "30s"
  },
  "environment": "development"
}`
}

func generateAPIConfig() string {
	return `{
  "app": {
    "name": "api-service",
    "version": "1.0.0"
  },
  "server": {
    "host": "0.0.0.0",
    "port": 8080,
    "read_timeout": "30s",
    "write_timeout": "30s"
  },
  "environment": "development",
  "custom": {
    "database": {
      "driver": "postgres",
      "host": "localhost",
      "port": 5432,
      "database": "api_db",
      "username": "api_user"
    },
    "metrics": {
      "enabled": true,
      "port": 9090
    }
  }
}`
}

func generateWebConfig() string {
	return `{
  "app": {
    "name": "web-app",
    "version": "1.0.0"
  },
  "server": {
    "host": "localhost",
    "port": 8080,
    "read_timeout": "30s",
    "write_timeout": "30s"
  },
  "environment": "development",
  "custom": {
    "static_dir": "./public",
    "template_dir": "./templates"
  }
}`
}

func generateDBConfig(dbTool string) string {
	switch dbTool {
	case "sqlc":
		return generateSQLCConfig()
	case "ent":
		return generateEntConfig()
	case "gorm":
		return generateGormConfig()
	default:
		return generateSQLCConfig()
	}
}

func generateSQLCConfig() string {
	return `version: "2"
sql:
  - engine: "postgresql"
    queries: "./sql/queries/"
    schema: "./sql/migrations/"
    gen:
      go:
        package: "db"
        out: "./internal/db"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: true
        emit_exact_table_names: false
        emit_empty_slices: true
        emit_exported_queries: true
        emit_result_struct_pointers: true
        emit_params_struct_pointers: false
        emit_methods_with_db_argument: false
        emit_pointers_for_null_types: false
        emit_enum_valid_method: false
        emit_all_enum_values: false
        json_tags_case_style: "snake"
        omit_unused_structs: true
`
}

func generateEntConfig() string {
	return `// ent generate config
//go:generate go run -mod=mod entgo.io/ent/cmd/ent generate ./ent/schema

// To initialize ent, run:
// go run entgo.io/ent/cmd/ent init User

// Generated files will be in ent/
`
}

func generateGormConfig() string {
	return `# GORM Configuration

# No special config file needed for GORM
# GORM is configured programmatically in your Go code

# Example database configuration in config.json:
# {
#   "database": {
#     "host": "localhost",
#     "port": 5432,
#     "database": "myapp",
#     "username": "user",
#     "password": "password",
#     "sslmode": "disable"
#   }
# }

# Auto-migration is handled in your main.go:
# db.AutoMigrate(&models.User{})
`
}

func generateMakefile() string {
	return `# Goryu Project Makefile

.PHONY: help build run test clean migrate-up migrate-down sqlc-generate dev

help: ## Show this help message
	@echo 'Usage: make <target>'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	go build -o bin/server cmd/server/main.go

run: ## Run the application
	go run cmd/server/main.go

test: ## Run tests
	go test -v ./...

test-coverage: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

clean: ## Clean build artifacts
	rm -f bin/*
	rm -f coverage.out

# Database commands
migrate-up: ## Run database migrations up
	migrate -path sql/migrations -database "$(DATABASE_URL)" up

migrate-down: ## Run database migrations down
	migrate -path sql/migrations -database "$(DATABASE_URL)" down

migrate-create: ## Create a new migration file (usage: make migrate-create name=migration_name)
	migrate create -ext sql -dir sql/migrations -seq $(name)

# sqlc commands
sqlc-generate: ## Generate Go code from SQL
	sqlc generate

sqlc-verify: ## Verify SQL queries
	sqlc verify

# Development
dev: ## Start development server with hot reload (requires air)
	air

install-tools: ## Install development tools
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/cosmtrek/air@latest

# Docker
docker-build: ## Build Docker image
	docker build -t goryu-app .

docker-run: ## Run Docker container
	docker run -p 8080:8080 goryu-app

docker-compose-up: ## Start with docker-compose
	docker-compose up --build

docker-compose-down: ## Stop docker-compose
	docker-compose down
`
}

func generateDockerfile() string {
	return `FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install git and ca-certificates
RUN apk update && apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/server/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

# Copy config files if they exist
COPY --from=builder /app/config ./config/

EXPOSE 8080

CMD ["./main"]
`
}

func generateDBReadme(projectName string) string {
	return fmt.Sprintf(`# %s

A Goryu database-driven application with sqlc integration.

## Getting Started

1. Install dependencies and tools:
   `+"`"+`bash
   go mod tidy
   make install-tools
   `+"`"+`

2. Set up your database:
   `+"`"+`bash
   export DATABASE_URL="postgres://user:pass@localhost/dbname?sslmode=disable"
   `+"`"+`

3. Create your models and SQL:
   `+"`"+`bash
   goryu generate model user --type=db
   goryu generate sql user
   `+"`"+`

4. Run migrations and generate code:
   `+"`"+`bash
   make migrate-up
   make sqlc-generate
   `+"`"+`

5. Run the application:
   `+"`"+`bash
   go run cmd/server/main.go
   `+"`"+`

## Database Development

### Creating Models
- `+"`"+`goryu generate model <name> --type=db`+"`"+` - Generate model with database integration
- `+"`"+`goryu generate sql <name>`+"`"+` - Generate SQL schema and queries

### Managing Database
- `+"`"+`make migrate-up`+"`"+` - Run pending migrations
- `+"`"+`make migrate-down`+"`"+` - Rollback last migration
- `+"`"+`make migrate-create name=<migration_name>`+"`"+` - Create new migration

### Code Generation
- `+"`"+`make sqlc-generate`+"`"+` - Generate Go code from SQL queries
- `+"`"+`make sqlc-verify`+"`"+` - Verify SQL queries

## Features

- Built with Goryu web framework
- sqlc for type-safe database operations  
- Database migrations with golang-migrate
- Repository pattern implementation
- PostgreSQL optimized (configurable for other databases)
- Development tools integration (Makefile, hot reload)

## Project Structure

- `+"`"+`cmd/server/`+"`"+` - Application entrypoint with database connection
- `+"`"+`internal/handlers/`+"`"+` - HTTP handlers using repositories
- `+"`"+`internal/models/`+"`"+` - Domain models and validation
- `+"`"+`internal/db/`+"`"+` - sqlc generated code and connection management
- `+"`"+`internal/repository/`+"`"+` - Repository implementations
- `+"`"+`sql/migrations/`+"`"+` - Database schema migrations
- `+"`"+`sql/queries/`+"`"+` - SQL queries for sqlc
- `+"`"+`config/`+"`"+` - Configuration files

## Environment Variables

Set these environment variables or update config.json:

`+"`"+`bash
export DATABASE_URL="postgres://user:pass@localhost/dbname?sslmode=disable"
export GORYU_DATABASE_HOST=localhost
export GORYU_DATABASE_PORT=5432
export GORYU_DATABASE_DATABASE=dbname
export GORYU_DATABASE_USERNAME=user
export GORYU_DATABASE_PASSWORD=pass
`+"`"+`
`, projectName)
}

func generateDBMainFile(projectName string) string { // hmmm 
	return fmt.Sprintf(`package main

import (
	"context"
	"log"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/config"
	"%s/internal/db"
	"%s/internal/handlers"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %%v", err)
	}

	// Initialize database connection
	database, err := db.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %%v", err)
	}
	defer func() { _ = database.Close() }()

	// Test database connection
	if err := database.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %%v", err)
	}
	log.Println("✅ Database connected successfully")

	// Initialize sqlc queries
	queries := db.New(database)

	// Create Goryu app with configuration
	goryuCfg := cfg.ToGoryuConfig()
	app := goryu.New(goryu.Config{
		AppName:               goryuCfg.AppName,
		ServerHeader:          goryuCfg.ServerHeader,
		StrictRouting:         goryuCfg.StrictRouting,
		CaseSensitive:         goryuCfg.CaseSensitive,
		DisableStartupMessage: goryuCfg.DisableStartupMessage,
	})

	// Add database to context for handlers
	app.Use(func(next goryu.HandlerFunc) goryu.HandlerFunc {
		return func(c *goryu.Context) {
			c.Set("db", database)
			c.Set("queries", queries)
			next(c)
		}
	})

	// Register routes
	app.GET("/", handlers.Home)
	app.GET("/health", handlers.Health)

	log.Printf("🚀 Starting server on %%s", cfg.GetServerAddress())
	if err := app.Run(cfg.GetServerAddress()); err != nil {
		log.Fatalf("Server failed to start: %%v", err)
	}
}
`, projectName, projectName)
}

func generateDBConnection(projectName string) string {
	return `package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/arthurlch/goryu/db"
	"github.com/arthurlch/goryu/config"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var conn *db.Connection

// Connect creates a database connection using the provided config
func Connect(cfg *config.Config) (*sql.DB, error) {
	// Use goryu's db package to connect
	connection, err := db.Connect(cfg)
	if err != nil {
		return nil, err
	}
	
	conn = connection
	return connection.DB, nil
}

// Connection returns the current database connection
func Connection() *db.Connection {
	return conn
}

// Close closes the database connection
func Close() error {
	if conn != nil {
		return conn.Close()
	}
	return nil
}

// Ping tests the database connection
func Ping() error {
	ctx := context.Background()
	if conn == nil || conn.DB == nil {
		return fmt.Errorf("database connection not established")
	}
	return conn.DB.PingContext(ctx)
}
`
}

func generateBaseRepository(projectName string) string {
	return fmt.Sprintf(`package repository

import (
	"database/sql"
	"%s/internal/db"
)

// BaseRepository provides common database operations
type BaseRepository struct {
	DB      *sql.DB
	Queries *db.Queries
}

// NewBaseRepository creates a new base repository
func NewBaseRepository(database *sql.DB) *BaseRepository {
	return &BaseRepository{
		DB:      database,
		Queries: db.New(database),
	}
}

// WithTx executes a function within a database transaction
func (r *BaseRepository) WithTx(fn func(*db.Queries) error) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := r.Queries.WithTx(tx)
	if err := fn(qtx); err != nil {
		return err
	}

	return tx.Commit()
}
`, projectName)
}

func generateGitignore() string {
	return `# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with ` + "`go test -c`" + `
*.test

# Output of the go coverage tool
*.out

# Dependency directories
vendor/

# Go workspace file
go.work

# Environment variables
.env

# IDE files
.vscode/
.idea/
*.swp
*.swo

# OS generated files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db

# Application specific
logs/
tmp/
dist/`
}
