# 🎓 Goryu Tutorial: Zero to Hero

Build a production-ready Todo API in 5 minutes using Goryu.

## Prerequisites
- Go 1.24+
- The Goryu CLI installed:
  ```bash
  go install github.com/arthurlch/goryu/cmd/goryu@latest
  ```

---

## 1. Create a Project
Use the CLI to scaffold a best-practice project structure.

```bash
goryu init todo-app
cd todo-app
go mod tidy
```

## 2. The Server
Open `cmd/server/main.go`. Goryu's `Default()` gives you a "batteries-included" server with logging, recovery, and monitoring.

```go
package main

import (
    "github.com/arthurlch/goryu"
)

func main() {
    // 1. Create App (Logger, Recovery, Monitoring enabled)
    app := goryu.Default()

    // 2. Start (Prints Route Table)
    app.Listen(":8080")
}
```

## 3. The Model
Define your data structure. Implement the `Validator` interface to get **automatic validation**.

```go
// internal/models/todo.go
package models

import (
    "fmt"
    "time"
)

type Todo struct {
    ID        int       `json:"id"`
    Title     string    `json:"title"`
    Completed bool      `json:"completed"`
    CreatedAt time.Time `json:"created_at"`
}

// Validate is called automatically by BodyParser!
func (t *Todo) Validate() error {
    if len(t.Title) < 3 {
        return fmt.Errorf("title must be at least 3 chars")
    }
    return nil
}
```

## 4. The Handler
Let's add a route to create a Todo. We'll use:
- `BodyParser` for smart binding & validation.
- `goryu.Map` for clean JSON responses.

```go
// cmd/server/main.go

import (
    "net/http"
    "github.com/arthurlch/goryu"
    "todo-app/internal/models"
)

func main() {
    app := goryu.Default()

    // Simulated DB
    todos := []models.Todo{}

    app.POST("/todos", func(c *goryu.Ctx) {
        var todo models.Todo

        // 🧠 Smart Binding + Auto-Validation
        if err := c.BodyParser(&todo); err != nil {
            // Returns 400 if JSON is invalid OR Validate() fails
            c.Status(400).JSON(goryu.Map{"error": err.Error()})
            return
        }

        todo.ID = len(todos) + 1
        todo.CreatedAt = time.Now()
        todos = append(todos, todo)

        // ✨ Clean JSON with goryu.Map
        c.Status(201).JSON(goryu.Map{
            "status": "created",
            "data":   todo,
        })
    })

    app.Listen(":8080")
}
```

## 5. Run & Monitor
Start your dev server with hot-reload:

```bash
goryu dev
```

### Try it out
**Create a Todo**:
```bash
curl -X POST http://localhost:8080/todos \
  -H "Content-Type: application/json" \
  -d '{"title": "Learn Goryu"}'
```

**Try Invalid Data** (Auto-Validation):
```bash
curl -X POST http://localhost:8080/todos \
  -d '{"title": "Hi"}'
# Output: {"error": "title must be at least 3 chars"}
```

### 👁️ Real-time Dashboard
Visit **[http://localhost:8080/_dashboard](http://localhost:8080/_dashboard)**.
You'll see:
- Real-time event log of your requests.
- Health status.
- Metrics graph.

---

## Next Steps
- [Middleware Guide](./middleware/README.md)
- [CLI References](./CLI.md)
