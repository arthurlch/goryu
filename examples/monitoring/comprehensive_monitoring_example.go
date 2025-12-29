package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/monitoring"
)

// UserRequest demonstrates the Validator interface
type UserRequest struct {
	Name  string `json:"name" query:"name"`
	Email string `json:"email" query:"email"`
}

// Validate implements the Validator interface
// This will be automatically called by c.BodyParser()
func (u *UserRequest) Validate() error {
	if len(u.Name) < 3 {
		return fmt.Errorf("name must be at least 3 chars")
	}
	if !strings.Contains(u.Email, "@") {
		return fmt.Errorf("invalid email")
	}
	return nil
}

func main() {
	// Create app with "batteries-included" defaults
	// This automatically sets up:
	// - Recovery & Logging Middleware
	// - RequestID tracking
	// - Monitoring (Health, Metrics, Events, Dashboard)
	app := goryu.Default()

	// Add custom health checks
	app.AddHealthCheck("database", &monitoring.HealthCheck{
		Check: func() (monitoring.HealthStatus, error) {
			// Simulate database health check
			return monitoring.StatusHealthy, nil
		},
		Timeout:  5 * time.Second,
		Interval: 30 * time.Second,
		Critical: true,
	})

	app.AddHealthCheck("external_api", &monitoring.HealthCheck{
		Check: func() (monitoring.HealthStatus, error) {
			// Simulate external API health check
			time.Sleep(100 * time.Millisecond) // Simulate some latency
			return monitoring.StatusHealthy, nil
		},
		Timeout:  3 * time.Second,
		Interval: 60 * time.Second,
		Critical: false,
	})

	// Add some test routes
	app.GET("/", func(c *goryu.Ctx) {
		c.JSON(200, map[string]string{"message": "Welcome to monitored Goryu!"})
	})

	app.GET("/slow", func(c *goryu.Ctx) {
		// Simulate slow endpoint
		time.Sleep(200 * time.Millisecond)
		c.JSON(200, map[string]string{"message": "This was a slow response"})
	})

	app.GET("/error", func(c *goryu.Ctx) {
		// Simulate error
		c.JSON(500, map[string]string{"error": "Something went wrong"})
	})

	app.POST("/users", func(c *goryu.Ctx) {
		var req UserRequest

		// Use the new simplified BodyParser with auto-validation
		if err := c.BodyParser(&req); err != nil {
			c.JSON(400, goryu.Map{"error": err.Error()})
			return
		}

		// Emit custom event
		app.EmitEvent(monitoring.EventCustom, "User created", goryu.Map{
			"user_name":  req.Name,
			"user_email": req.Email,
			"action":     "create_user",
		})

		// Use goryu.Map for cleaner JSON construction
		c.JSON(201, goryu.Map{
			"id":      123,
			"created": true,
			"user":    req,
		})
	})

	// Add event handler for custom logging
	app.Monitor.AddEventHandler(func(event monitoring.Event) {
		if event.Type == monitoring.EventError {
			// You could send this to external logging service
			println("[ERROR EVENT]", event.Message)
		}
	})

	// Register the UI dashboard
	app.GET("/_dashboard", app.Monitor.UIHandler("Goryu Monitoring Demo"))

	// The monitoring endpoints are automatically registered:
	// GET /_health     - Health check status
	// GET /_metrics    - Application metrics
	// GET /_events     - Recent events (with ?limit=N parameter)
	// GET /_dashboard  - Visual monitoring dashboard

	println("🚀 Monitoring Dashboard available at: http://localhost:8080/_dashboard")
	app.Listen(":8080")
}
