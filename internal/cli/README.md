# Goryu CLI

The powerful goryu-cli that provides code generation, project scaffolding, and development tools etc

## Installation

```bash
go install github.com/arthurlch/goryu/cmd/goryu@latest
```

## Features

- **Interactive TUI Mode** - Launch without arguments for a visual interface
- **Project Generation** - Initialize new projects with templates
- **Code Generation** - Generate handlers, middleware, models, routes, and configs
- **Middleware Management** - List and inspect available middleware
- **Configuration Tools** - Initialize, validate, and manage configurations

## Quick Start

```bash
# Interactive mode
goryu

# Create a new project
goryu init my-app

# Generate components
goryu generate handler user
goryu generate middleware auth
goryu generate model product --type db
```

## Commands

### `goryu` (Interactive Mode)

Launch without arguments to enter the Terminal User Interface with visual menus and guided workflows.

### `init`

Initialize a new Goryu project.

```bash
goryu init [project-name] [flags]
```

**Flags:**
- `--template` - Project template (`basic`, `api`) (default: `basic`)
- `--path` - Project path (default: `.`)
- `--module` - Go module name
- `--git` - Initialize git repository (default: `true`)
- `--docker` - Include Docker files
- `--ci` - Include CI/CD configs

**Available Templates:**
- `basic` - Simple web application with minimal structure
- `api` - REST API with enhanced structure

### `generate` / `g`

Generate code using templates.

#### `generate handler`

```bash
goryu generate handler <name> [flags]
```

**Flags:**
- `--type` - Handler type (`basic`, `crud`, `api`, `websocket`) (default: `basic`)
- `--path` - Output path (default: `internal/handlers`)
- `--model` - Associated model name
- `--middleware` - Middleware to apply (comma-separated)
- `--route` - Route pattern (default: `/{name}`)

#### `generate middleware`

```bash
goryu generate middleware <name> [flags]
```

**Flags:**
- `--type` - Middleware type (`standard`, `builder`, `plugin`) (default: `builder`)
- `--path` - Output path (default: `internal/middleware`)
- `--global` - Make middleware global

#### `generate model`

```bash
goryu generate model <name> [flags]
```

**Flags:**
- `--type` - Model type (`basic`, `db`) (default: `basic`)
- `--db-tool` - Database tool (`gorm`, `sqlc`, `ent`) (default: `gorm`)
- `--fields` - Model fields (comma-separated)
- `--path` - Output path (default: `internal/models`)

#### `generate route`

```bash
goryu generate route <name> [flags]
```

**Flags:**
- `--builder` - Use route builder pattern (default: `true`)
- `--group` - Route group prefix
- `--middleware` - Route middleware (comma-separated)
- `--methods` - HTTP methods (default: `GET,POST,PUT,DELETE`)

#### `generate config`

```bash
goryu generate config <name> [flags]
```

**Flags:**
- `--builder` - Use config builder pattern (default: `true`)
- `--type` - Config type (`server`, `database`, `cache`, etc.) (default: `server`)
- `--format` - Config format (`json`, `yaml`, `toml`, `env`) (default: `json`)

### `scaffold` (Planned)

Scaffold complete features. Currently includes:
- `scaffold api` - Complete REST API (not yet implemented)
- `scaffold service` - Microservice scaffolding (not yet implemented)

### `dev` (Planned)

Start development server with hot-reload. (Not yet implemented)

### `build` (Planned)

Build the application for production. (Not yet implemented)

### `middleware`

Manage middleware components.

#### `middleware list`

List all available middleware:

```bash
goryu middleware list
```

Shows all 26 built-in middleware components including:
- Authentication (auth, basicauth)
- Security (cors, csrf, secure, secure_cookie)
- Performance (cache, compress, limiter)
- Monitoring (metrics, healthcheck, tracing)
- Request handling (timeout, requestid, recovery)
- And more...

#### `middleware info` (Planned)

Get detailed information about a specific middleware.

### `config`

Configuration management commands.

#### `config init`

Initialize configuration file:

```bash
goryu config init [flags]
```

**Flags:**
- `--type` - Config type (default: `server`)
- `--format` - Config format (default: `json`)

#### `config validate`

Validate configuration:

```bash
goryu config validate [flags]
```

**Flags:**
- `--file` - Config file (default: `config.json`)

#### `config migrate` (Planned)

Migrate configuration between formats.

### `routes` (Planned)

Route management commands:
- `routes list` - List all routes
- `routes test` - Test route matching

### `validate`

Validate project structure:

```bash
goryu validate [flags]
```

**Flags:**
- `--fix` - Auto-fix issues

### `version`

Display version information:

```bash
goryu version
```

## Examples

### Create a Basic Web App

```bash
# Initialize project
goryu init my-web-app

# Generate handlers
cd my-web-app
goryu g handler user --type crud
goryu g handler auth --middleware auth,cors

# Generate middleware
goryu g middleware logger --type builder
```

### Create an API Project

```bash
# Initialize with API template
goryu init my-api --template api

# Generate API resources
cd my-api
goryu g model user --type db --fields "name:string,email:string"
goryu g handler user --type api --model User
goryu g route user --group "/api/v1" --middleware "auth,limiter"
```

## Project Structure

Projects created with Goryu CLI follow this structure:

```
my-app/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── handlers/
│   ├── middleware/
│   ├── models/
│   └── routes/
├── config/
│   └── config.json
├── go.mod
├── README.md
└── .gitignore
```

## Notes

Several commands show placeholder messages indicating they're not yet implemented:
- `scaffold api/service`
- `dev`, `build`
- `middleware info`
- `config migrate`
- `routes list/test`

These features are planned for future releases.

