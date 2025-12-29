# Envvar Middleware

A middleware that exposes the application's environment variables via an HTTP endpoint. This is useful for debugging configuration issues and verifying deployment environments.

## Features

- **JSON Output**: Returns environment variables as a JSON object.
- **Filtering**: Whitelist (`Expose`) or blacklist (`Exclude`) specific variables.
- **Security**: Configurable endpoint path.

## Usage

### Basic Setup

Expose all environment variables at `/envvar` (Use with caution!):

```go
app.Use(envvar.Default())
```

### Whitelisting (Recommended)

Only expose specific variables:

```go
app.Use(envvar.WithExpose([]string{
    "APP_ENV",
    "REGION",
    "VERSION",
}))
```

### Blacklisting

Expose everything *except* sensitive keys:

```go
app.Use(envvar.WithExclude([]string{
    "DB_PASSWORD",
    "AWS_SECRET_KEY",
    "API_TOKEN",
}))
```

### Advanced Configuration

```go
app.Use(envvar.New(envvar.Config{
    Path:   "/debug/env",
    Expose: []string{"HOSTNAME", "PORT"},
}))
```

## Security Warning

**NEVER** expose sensitive information (passwords, API keys, secrets) via this middleware in a production environment.
- Always use the `Expose` option to whitelist safe variables.
- Alternatively, use `Exclude` to block known secrets, but whitelisting is safer.
- Protect the endpoint with authentication middleware.
