package handlers

import (
	"net/http"
	"time"

	"github.com/arthurlch/goryu"
)

// Health returns the health status
func Health(c *goryu.Context) {
	c.JSON(http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"service":   "goryu-app",
		"timestamp": time.Now().UTC(),
	})
}
