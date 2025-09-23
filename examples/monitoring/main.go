package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/monitoring"
)

func main() {
	// Create a new Goryu app
	app := goryu.New(goryu.Config{
		AppName: "Goryu Monitoring Dashboard",
	})

	// Enable monitoring endpoints at /_monitor
	app.EnableMonitoring("/_monitor")

	// Add some custom health checks
	addHealthChecks(app)

	// Add some sample routes
	setupRoutes(app)

	// Add custom event handler for logging
	app.Monitor.AddEventHandler(func(event monitoring.Event) {
		if event.Type == monitoring.EventError {
			log.Printf("ERROR EVENT: %s - %s", event.Message, event.Data)
		}
	})

	// Emit a custom startup event
	app.EmitEvent(monitoring.EventCustom, "Application fully configured", map[string]interface{}{
		"version": "1.0.0",
		"env":     "development",
	})

	fmt.Println("🔍 Monitoring endpoints available at:")
	fmt.Println("   Web UI:    http://localhost:8080/_monitor/ui")
	fmt.Println("   Health:    http://localhost:8080/_monitor/health")
	fmt.Println("   Metrics:   http://localhost:8080/_monitor/metrics")
	fmt.Println("   Events:    http://localhost:8080/_monitor/events")
	fmt.Println("   Dashboard: http://localhost:8080/_monitor/dashboard")
	fmt.Println()
	fmt.Println("🎯 Visit http://localhost:8080/_monitor/ui for the full dashboard!")

	log.Fatal(app.Listen(":8080"))
}

func addHealthChecks(app *goryu.App) {
	// Database health check (simulated)
	app.AddHealthCheck("database", &monitoring.HealthCheck{
		Check: func() (monitoring.HealthStatus, error) {
			// Simulate database connection check
			// In real app, you'd check actual database connection
			if simulateDatabaseCheck() {
				return monitoring.StatusHealthy, nil
			}
			return monitoring.StatusUnhealthy, fmt.Errorf("database connection failed")
		},
		Timeout:  5 * time.Second,
		Interval: 30 * time.Second,
		Critical: true,
	})

	// External API health check (simulated)
	app.AddHealthCheck("external_api", &monitoring.HealthCheck{
		Check: func() (monitoring.HealthStatus, error) {
			// Simulate external API check
			resp, err := http.Get("https://httpstat.us/200")
			if err != nil {
				return monitoring.StatusUnhealthy, err
			}
			defer func() {
				if err := resp.Body.Close(); err != nil {
					return
				}
			}()

			if resp.StatusCode == 200 {
				return monitoring.StatusHealthy, nil
			}
			return monitoring.StatusDegraded, fmt.Errorf("external API returned status %d", resp.StatusCode)
		},
		Timeout:  10 * time.Second,
		Interval: 60 * time.Second,
		Critical: false, // Non-critical - won't fail overall health
	})

	// Memory usage health check
	app.AddHealthCheck("memory", &monitoring.HealthCheck{
		Check: func() (monitoring.HealthStatus, error) {
			metrics := app.Monitor.GetMetrics()
			memoryMB := metrics.MemoryUsage / 1024 / 1024

			if memoryMB > 500 {
				return monitoring.StatusUnhealthy, fmt.Errorf("memory usage too high: %d MB", memoryMB)
			} else if memoryMB > 200 {
				return monitoring.StatusDegraded, fmt.Errorf("memory usage elevated: %d MB", memoryMB)
			}
			return monitoring.StatusHealthy, nil
		},
		Timeout:  1 * time.Second,
		Interval: 15 * time.Second,
		Critical: false,
	})
}

func setupRoutes(app *goryu.App) {
	// Normal endpoint
	app.GET("/", func(c *goryu.Context) {
		_ = c.JSON(http.StatusOK, map[string]interface{}{
			"message": "Hello from Goryu!",
			"time":    time.Now(),
		})
	})

	// Endpoint that emits custom events
	app.POST("/orders", func(c *goryu.Context) {
		// Simulate order processing
		app.EmitEvent(monitoring.EventCustom, "New order received", map[string]interface{}{
			"user_id":    "user123",
			"order_id":   "order456",
			"amount":     99.99,
			"ip_address": c.Request.RemoteAddr,
		})

		_ = c.JSON(http.StatusCreated, map[string]interface{}{
			"status":   "success",
			"order_id": "order456",
		})
	})

	// Endpoint that sometimes fails (for testing error monitoring)
	app.GET("/flaky", func(c *goryu.Context) {
		// Randomly fail 30% of the time
		if time.Now().Unix()%3 == 0 {
			app.EmitEvent(monitoring.EventError, "Flaky endpoint failed", map[string]interface{}{
				"reason": "random failure for testing",
			})
			_ = c.Status(http.StatusInternalServerError).JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Something went wrong!",
			})
			return
		}

		_ = c.JSON(http.StatusOK, map[string]interface{}{
			"message": "Everything is fine!",
		})
	})

	// Slow endpoint (for performance monitoring)
	app.GET("/slow", func(c *goryu.Context) {
		// Simulate slow processing
		time.Sleep(2 * time.Second)

		app.EmitEvent(monitoring.EventCustom, "Slow operation completed", map[string]interface{}{
			"operation": "data_processing",
			"duration":  "2s",
		})

		_ = c.JSON(http.StatusOK, map[string]interface{}{
			"message": "Slow operation completed",
		})
	})

	// Admin endpoint that requires monitoring
	adminGroup := app.Group("/admin")
	adminGroup.GET("/stats", func(c *goryu.Context) {
		metrics := app.Monitor.GetMetrics()
		healthStatus := app.Monitor.GetHealthStatus()
		recentEvents := app.Monitor.GetEvents(5)

		_ = c.JSON(http.StatusOK, map[string]interface{}{
			"health":        string(healthStatus),
			"metrics":       metrics,
			"recent_events": recentEvents,
		})
	})
}

// simulateDatabaseCheck simulates a database health check
func simulateDatabaseCheck() bool {
	// In a real application, you would do something like:
	// db, err := sql.Open("postgres", "your-connection-string")
	// if err != nil {
	//     return false
	// }
	// defer func() { _ = db.Close() }()
	// return db.Ping() == nil

	// For this example, we'll just return true
	// You can change this to false to see unhealthy status
	return true
}

// Example of more advanced database health check
func realDatabaseHealthCheck(db *sql.DB) monitoring.HealthCheck {
	return monitoring.HealthCheck{
		Check: func() (monitoring.HealthStatus, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			if err := db.PingContext(ctx); err != nil {
				return monitoring.StatusUnhealthy, fmt.Errorf("database ping failed: %w", err)
			}

			// Check if we can execute a simple query
			var result int
			err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
			if err != nil {
				return monitoring.StatusDegraded, fmt.Errorf("database query failed: %w", err)
			}

			return monitoring.StatusHealthy, nil
		},
		Timeout:  5 * time.Second,
		Interval: 30 * time.Second,
		Critical: true,
	}
}
