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
goryu init [project-name] [flags]
```

**Flags:**
- `-t, --template`: Project template (`basic`, `api`, `web`, `db`) [default: basic]
- `-p, --path`: Project path [default: .]
- `-m, --module`: Go module name
- `--git`: Initialize git repository [default: true]
- `--docker`: Include Docker files [default: false]
- `--ci`: Include CI/CD configs [default: false]
- `--db-tool`: Database tool (`sqlc`, `ent`, `gorm`) [default: sqlc] (only for db template)

### `goryu generate` (alias: `g`)

Generate boilerplate code for common components.

#### Generate Handlers

```bash
goryu generate handler <name> [flags]
```

**Flags:**
- `-t, --type`: Handler type (`basic`, `crud`, `api`, `websocket`) [default: basic]
- `-p, --path`: Output path [default: internal/handlers]
- `--model`: Associated model name
- `--middleware`: Middleware to apply (comma-separated)
- `--route`: Route pattern [default: /{name}]

#### Generate Middleware

```bash
goryu generate middleware <name> [flags]
```

**Flags:**
- `-t, --type`: Middleware type (`standard`, `builder`, `plugin`) [default: builder]
- `-p, --path`: Output path [default: internal/middleware]
- `--global`: Make middleware global [default: false]

#### Generate Models

```bash
goryu generate model <name> [flags]
```

**Flags:**
- `-t, --type`: Model type (`basic`, `db`) [default: basic]
- `--db-tool`: Database tool (`gorm`, `sqlc`, `ent`) [default: gorm]
- `-f, --fields`: Model fields (comma-separated)
- `-p, --path`: Output path [default: internal/models]

#### Generate Routes

```bash
goryu generate route <name> [flags]
```

**Flags:**
- `-b, --builder`: Use route builder pattern [default: true]
- `-g, --group`: Route group prefix
- `-m, --middleware`: Route middleware (comma-separated)
- `--methods`: HTTP methods [default: GET,POST,PUT,DELETE]

#### Generate Config

```bash
goryu generate config <name> [flags]
```

**Flags:**
- `-b, --builder`: Use config builder pattern [default: true]
- `-t, --type`: Config type (`server`, `database`, `cache`) [default: server]
- `-f, --format`: Config format (`json`, `yaml`, `toml`, `env`) [default: json]

### `goryu scaffold`

Scaffold complete features or services.

#### Scaffold API

Scaffold a complete REST API resource with database, validation, and tests.

```bash
goryu scaffold api <resource> [flags]
```

**Flags:**
- `-f, --fields`: Resource fields [required]
- `--db`: Include database layer [default: true]
- `--auth`: Add authentication [default: false]
- `--validation`: Add validation [default: true]
- `--tests`: Generate tests [default: true]

#### Scaffold Service

Scaffold a microservice.

```bash
goryu scaffold service <name> [flags]
```

**Flags:**
- `--grpc`: Include gRPC support [default: false]
- `--http`: Include HTTP support [default: true]
- `--kafka`: Include Kafka support [default: false]
- `--monitoring`: Include monitoring [default: true]

### `goryu config`

Manage application configuration.

#### Initialize Configuration
```bash
goryu config init [--type=server] [--format=json]
```

#### Validate Configuration
```bash
goryu config validate [--file=config.json]
```

#### Migrate Configuration
Migrate configuration between formats.
```bash
goryu config migrate --from=json --to=yaml
```

### `goryu routes`

Manage and test routes.

#### List Routes
```bash
goryu routes list [--format=table|json] [--filter=pattern]
```

#### Test Route
Test route matching logic.
```bash
goryu routes test <path> [method]
```

### `goryu middleware`

Manage middleware.

#### List Middleware
```bash
goryu middleware list
```

#### Middleware Info
Show detailed information about a middleware.
```bash
goryu middleware info <name>
```

### `goryu dev`

Development tools.

```bash
goryu dev [--port=3000] [--hot-reload] [--debug]
```

### `goryu build`

Build the application.

```bash
goryu build [--output=server] [--target=production] [--compress] [--static]
```

### `goryu validate`

Validate project structure and configuration.

```bash
goryu validate [flags]
```

**Flags:**
- `--config`: Configuration file path [default: config/config.json]
- `--project`: Validate project structure [default: true]
- `--fix`: Auto-fix issues [default: false]

### `goryu version`

Show version information.

```bash
goryu version
```