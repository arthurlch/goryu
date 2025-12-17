# 🗄️ Goryu DB Package

> **Secure, robust, and driver-agnostic database connections.**

The `db` package provides a unified way to connect to various databases while enforcing strict security policies. It handles connection pooling, DSN construction, and security validation automatically.

---

## ✨ Key Features

- **🛡️ Secure by Default**:
    - **Anti-Injection**: Validates all connection parameters against SQL injection patterns.
    - **Path Protection**: Prevents directory traversal attacks for SQLite databases.
    - **Sanitization**: Automatically URL-encodes credentials and parameters.
- **🔌 Multi-Driver Support**: First-class support for PostgreSQL, MySQL, and SQLite3.
- **🏊 Connection Pooling**: Built-in configuration for `MaxOpenConns`, `MaxIdleConns`, and connection lifetimes.
- **✅ Auto-Validation**: Pings the database immediately upon connection to ensure reachability.

---

## 🚀 Usage

### Configuration

Configure your database in your application's config.

```go
import "github.com/arthurlch/goryu/config"

cfg := &config.Config{
    Database: config.DatabaseConfig{
        Driver:          "postgres", // or "mysql", "sqlite3"
        Host:            "localhost",
        Port:            5432,
        Database:        "myapp",
        Username:        "user",
        Password:        "secret",
        SSLMode:         "disable",
        
        // Connection Pool Settings
        MaxOpenConns:    25,
        MaxIdleConns:    5,
        ConnMaxLifetime: 5 * time.Minute,
    },
}
```

### Connecting

Use `db.Connect` to establish a secure connection.

```go
import (
    "log"
    "github.com/arthurlch/goryu/db"
)

func main() {
    // ... load config ...

    conn, err := db.Connect(cfg)
    if err != nil {
        log.Fatalf("Could not connect to database: %v", err)
    }
    defer conn.Close()

    // Access the underlying *sql.DB
    // conn.DB.Query(...)
    
    log.Println("Successfully connected to", conn.Driver)
}
```

---

## 🔒 Security Details

The `db` package implements several layers of security checks before even attempting a connection:

### SQL Injection Prevention
All inputs (username, password, database name, etc.) are scanned for dangerous characters (e.g., `;`, `--`, `/*`) that could be used for injection attacks in the connection string.

### SQLite Path Traversal
If you use SQLite, the path is rigorously checked to prevent:
- Directory traversal (`../../etc/passwd`)
- Access to system directories (`/etc/`, `/var/`)
- Hidden file access (optional based on config)

### Parameter Sanitization
All parameters are URL-encoded when building the Data Source Name (DSN) to prevent parameter injection attacks.

---

## 📦 Supported Drivers

| Driver | Config Key | Notes |
|--------|------------|-------|
| **PostgreSQL** | `postgres` / `pgx` | Supports SSL modes (`disable`, `require`, `verify-full`). |
| **MySQL** | `mysql` | Supports charsets (default `utf8mb4`). |
| **SQLite3** | `sqlite3` | In-memory support via `:memory:`. |
