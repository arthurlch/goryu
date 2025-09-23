# Goryu CLI

A powerful command-line tool for generating and managing Goryu web applications.

## Installation

Build the CLI from source:

```bash
go build ./cmd/goryu
```

Or add to your PATH:

```bash
go install ./cmd/goryu
```

## Quick Start

Create a new Goryu project:

```bash
goryu init my-app
cd my-app
go mod tidy
go run cmd/server/main.go
```

Your app will be running at http://localhost:8080

## Commands

### `goryu init`

Initialize a new Goryu project with customizable templates.

```bash
goryu init [project-name] [--template=basic|api|web]
```

**Examples:**
```bash
goryu init my-app                    # Basic project
goryu init my-api --template=api     # API-focused project  
goryu init my-web --template=web     # Web application project
```

**Templates:**
- `basic` - Simple web application (default)
- `api` - REST API with enhanced configuration
- `web` - Web application with static file serving
- `db` - Database-driven application with sqlc integration

### `goryu generate`

Generate boilerplate code for common components.

#### Generate Handlers

Create HTTP handlers with different complexity levels:

```bash
goryu generate handler <name> [--type=basic|crud|api] [--path=internal/handlers]
```

**Handler Types:**

**Basic Handler** - Simple endpoint handler
```bash
goryu generate handler welcome --type=basic
```
Generates a single function handler perfect for simple endpoints.

**CRUD Handler** - Full CRUD operations
```bash
goryu generate handler users --type=crud
```
Generates complete Create, Read, Update, Delete operations with:
- `ListUsers()` - GET /users
- `GetUser()` - GET /users/:id  
- `CreateUser()` - POST /users
- `UpdateUser()` - PUT /users/:id
- `DeleteUser()` - DELETE /users/:id

**API Handler** - Structured API handler
```bash
goryu generate handler auth --type=api
```
Generates professional API handler with:
- Request/Response structs
- Method-based routing
- Dependency injection pattern
- Validation structure

#### Generate Middleware

Create custom middleware:

```bash
goryu generate middleware <name> [--path=internal/middleware]
```

**Examples:**
```bash
goryu generate middleware cors
goryu generate middleware auth
goryu generate middleware ratelimit
```

#### Generate Models

Create data models with different complexity levels:

```bash
goryu generate model <name> [--type=basic|db] [--path=internal/models]
```

**Model Types:**

**Basic Model** - Simple struct for data modeling
```bash
goryu generate model user --type=basic
```
Generates a basic struct with validation methods, perfect for simple data models.

**Database Model** - Advanced model with database integration
```bash
goryu generate model user --type=db
```
Generates database-ready model with:
- Database tags and validation
- Repository pattern structure
- sqlc integration ready
- Parameter structs for CRUD operations

#### Generate SQL

Create SQL schema and queries for your models:

```bash
goryu generate sql <model-name>
```

**Examples:**
```bash
goryu generate sql user
```
Generates:
- `sql/migrations/001_create_user_table.sql` - Database schema with triggers
- `sql/queries/user.sql` - Complete CRUD queries for sqlc

### `goryu config`

Manage application configuration files.

#### Create Configuration
```bash
goryu config init [--type=basic|api|web] [--file=config.json]
```

#### Validate Configuration
```bash
goryu config validate [--file=config.json]
```

#### Show Configuration
```bash
goryu config show [--file=config.json]
```

#### Set Configuration Values
```bash
goryu config set <key> <value> [--file=config.json]
```

**Examples:**
```bash
goryu config set server.port 3000
goryu config set database.host localhost
goryu config set features.enable_metrics true
```

#### Get Configuration Values
```bash
goryu config get <key> [--file=config.json]
```

### `goryu validate`

Validate project structure and configuration:

```bash
goryu validate [--config=config.json] [--project]
```

Checks for:
- Valid configuration files
- Proper project structure
- Go syntax validation
- Dependency verification

## Project Structure

When you create a new project, Goryu CLI generates this structure:

```
my-app/
├── cmd/
│   └── server/
│       └── main.go          # Application entry point
├── internal/
│   ├── handlers/            # HTTP handlers
│   │   ├── health.go
│   │   └── home.go
│   ├── middleware/          # Custom middleware
│   └── models/              # Data models
├── config/
│   └── config.json          # Application configuration
├── go.mod                   # Go module file
├── README.md               # Project documentation
└── .gitignore              # Git ignore rules
```

## Configuration

Goryu applications use JSON configuration files with environment variable support. Example configuration:

```json
{
  "server": {
    "host": "localhost",
    "port": 8080,
    "read_timeout": "30s",
    "write_timeout": "30s"
  },
  "logging": {
    "level": "info",
    "format": "json"
  },
  "features": {
    "enable_metrics": false,
    "enable_tracing": false,
    "enable_structured_logging": true,
    "enable_compression": true
  }
}
```

Override with environment variables:
```bash
export GORYU_SERVER_PORT=3000
export GORYU_LOGGING_LEVEL=debug
```

## Examples

### Create a REST API

```bash
# Create API project
goryu init blog-api --template=api

# Generate user management handlers
goryu generate handler users --type=crud

# Generate authentication handler  
goryu generate handler auth --type=api

# Generate middleware
goryu generate middleware cors
goryu generate middleware jwt

# Validate project
goryu validate
```

### Create a Database-Driven Application

```bash
# Create database project with sqlc integration
goryu init ecommerce-api --template=db

cd ecommerce-api

# Install development tools
make install-tools

# Generate models and SQL
goryu generate model user --type=db
goryu generate model product --type=db
goryu generate sql user
goryu generate sql product

# Generate handlers that work with models
goryu generate handler users --type=crud
goryu generate handler products --type=api

# Set up database and run migrations
export DATABASE_URL="postgres://user:pass@localhost/ecommerce?sslmode=disable"
make migrate-up

# Generate Go database code
make sqlc-generate

# Run the application
go run cmd/server/main.go
```

### Create a Web Application

```bash
# Create web project
goryu init blog-web --template=web

# Generate page handlers
goryu generate handler blog --type=basic
goryu generate handler admin --type=crud

# Generate middleware
goryu generate middleware auth
goryu generate middleware session
```

### Working with Configuration

```bash
# Create production config
goryu config init --type=api --file=config.prod.json

# Set production values
goryu config set server.host 0.0.0.0 --file=config.prod.json
goryu config set database.host db.prod.example.com --file=config.prod.json

# Validate configuration
goryu config validate --file=config.prod.json
```

## Generated Code Examples

### Basic Handler
```go
// Welcome handles welcome requests
func Welcome(c *goryu.Context) {
    c.JSON(http.StatusOK, map[string]interface{}{
        "message": "welcome handler",
        "status":  "success",
    })
}
```

### CRUD Handler
```go
// ListUsers handles GET /users - list all users
func ListUsers(c *goryu.Context) {
    // TODO: Implement list logic
    c.JSON(http.StatusOK, map[string]interface{}{
        "data":  []interface{}{},
        "count": 0,
    })
}

// GetUser handles GET /users/:id - get user by ID
func GetUser(c *goryu.Context) {
    idStr := c.Param("id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, map[string]string{
            "error": "Invalid ID",
        })
        return
    }
    // TODO: Implement get by ID logic ...
}
```

### API Handler
```go
// AuthRequest represents the request body for auth operations
type AuthRequest struct {
    // TODO: Add your request fields
    // Email    string `json:"email" validate:"required,email"`
    // Password string `json:"password" validate:"required,min=8"`
}

// AuthResponse represents the response body for auth operations  
type AuthResponse struct {
    ID      int    `json:"id"`
    Message string `json:"message"`
    // TODO: Add your response fields
}

// AuthAPI provides API endpoints for auth management
type AuthAPI struct {
    // TODO: Add dependencies (services, repositories, etc.)
}
```

## Tips

1. **Start with templates**: Use `--template` flag to get the right project structure for your use case

2. **Use CRUD handlers**: For resource management, `--type=crud` gives you a complete set of operations

3. **Structure your API**: Use `--type=api` for professional API handlers with proper request/response types

4. **Validate early**: Run `goryu validate` to catch configuration and structure issues

5. **Environment-specific configs**: Create multiple config files for different environments

6. **Custom paths**: Use `--path` flag to organize code in custom directories

## Contributing

The CLI is designed to be easily extensible. To add new generator types:

1. Add the generator function in `internal/cli/generate.go`
2. Add the template function in `internal/cli/templates.go`
3. Update the help text and examples

## Version

```bash
goryu version
```

Shows the current CLI version and framework information.

---

**🐉 Happy coding with Goryu! 🐉** 