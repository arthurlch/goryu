package cli

import (
	"fmt"
	"github.com/arthurlch/goryu/internal/utils"
	"strings"
)

// Handler template generators
func generateBasicHandlerContent(name string) string {
	handlerName := utils.ToGoIdentifier(name)

	return fmt.Sprintf(`package handlers

import (
	"net/http"

	"github.com/arthurlch/goryu"
)

// %s handles %s requests
func %s(c *goryu.Context) {
	c.JSON(http.StatusOK, map[string]interface{}{
		"message": "%s handler",
		"status":  "success",
	})
}
`, handlerName, name, handlerName, name)
}

func generateCRUDHandlerContent(name string) string {
	handlerName := utils.ToGoIdentifier(name)
	lowerName := strings.ToLower(name)

	return fmt.Sprintf(`package handlers

import (
	"net/http"
	"strconv"

	"github.com/arthurlch/goryu"
)

// List%s handles GET /%s - list all %s
func List%s(c *goryu.Context) {
	// TODO: Implement list logic
	c.JSON(http.StatusOK, map[string]interface{}{
		"data":  []interface{}{},
		"count": 0,
	})
}

// Get%s handles GET /%s/:id - get %s by ID
func Get%s(c *goryu.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid ID",
		})
		return
	}

	// TODO: Implement get by ID logic
	c.JSON(http.StatusOK, map[string]interface{}{
		"id":      id,
		"message": "%s found",
	})
}

// Create%s handles POST /%s - create new %s
func Create%s(c *goryu.Context) {
	// TODO: Add struct for request body
	var req map[string]interface{}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	// TODO: Implement create logic
	c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "%s created successfully",
		"data":    req,
	})
}

// Update%s handles PUT /%s/:id - update %s
func Update%s(c *goryu.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid ID",
		})
		return
	}

	var req map[string]interface{}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	// TODO: Implement update logic
	c.JSON(http.StatusOK, map[string]interface{}{
		"id":      id,
		"message": "%s updated successfully",
		"data":    req,
	})
}

// Delete%s handles DELETE /%s/:id - delete %s
func Delete%s(c *goryu.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid ID",
		})
		return
	}

	// TODO: Implement delete logic
	c.JSON(http.StatusOK, map[string]interface{}{
		"message": "%s deleted successfully",
		"id":      id,
	})
}
`, handlerName, lowerName, lowerName, handlerName,
		handlerName, lowerName, lowerName, handlerName, handlerName,
		handlerName, lowerName, lowerName, handlerName, handlerName,
		handlerName, lowerName, lowerName, handlerName, handlerName,
		handlerName, lowerName, lowerName, handlerName, handlerName)
}

func generateAPIHandlerContent(name string) string {
	handlerName := utils.ToGoIdentifier(name)
	lowerName := strings.ToLower(name)

	return fmt.Sprintf(`package handlers

import (
	"net/http"

	"github.com/arthurlch/goryu"
)

// %sRequest represents the request body for %s operations
type %sRequest struct {
	// TODO: Add your request fields
	// Name  string `+"`json:\"name\" validate:\"required\"`"+`
	// Email string `+"`json:\"email\" validate:\"required,email\"`"+`
}

// %sResponse represents the response body for %s operations
type %sResponse struct {
	ID      int    `+"`json:\"id\"`"+`
	Message string `+"`json:\"message\"`"+`
	// TODO: Add your response fields
}

// %sAPI provides API endpoints for %s management
type %sAPI struct {
	// TODO: Add dependencies (services, repositories, etc.)
	// service %sService
}

// New%sAPI creates a new %s API handler
func New%sAPI() *%sAPI {
	return &%sAPI{
		// TODO: Initialize dependencies
	}
}

// Handle%s handles %s API requests
func (h *%sAPI) Handle%s(c *goryu.Context) {
	switch c.Request.Method {
	case http.MethodGet:
		h.get%s(c)
	case http.MethodPost:
		h.create%s(c)
	case http.MethodPut:
		h.update%s(c)
	case http.MethodDelete:
		h.delete%s(c)
	default:
		c.JSON(http.StatusMethodNotAllowed, map[string]string{
			"error": "Method not allowed",
		})
	}
}

func (h *%sAPI) get%s(c *goryu.Context) {
	// TODO: Implement get logic
	c.JSON(http.StatusOK, %sResponse{
		ID:      1,
		Message: "%s retrieved successfully",
	})
}

func (h *%sAPI) create%s(c *goryu.Context) {
	var req %sRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	// TODO: Validate request and create %s
	c.JSON(http.StatusCreated, %sResponse{
		ID:      1,
		Message: "%s created successfully",
	})
}

func (h *%sAPI) update%s(c *goryu.Context) {
	var req %sRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	// TODO: Validate request and update %s
	c.JSON(http.StatusOK, %sResponse{
		ID:      1,
		Message: "%s updated successfully",
	})
}

func (h *%sAPI) delete%s(c *goryu.Context) {
	// TODO: Get ID from path and delete %s
	c.JSON(http.StatusOK, map[string]string{
		"message": "%s deleted successfully",
	})
}
`, handlerName, lowerName, handlerName,
		handlerName, lowerName, handlerName,
		handlerName, lowerName, handlerName, handlerName,
		handlerName, lowerName, handlerName, handlerName, handlerName,
		handlerName, lowerName, handlerName, handlerName,
		handlerName, handlerName, handlerName, handlerName,
		handlerName, handlerName, handlerName, handlerName,
		handlerName, handlerName, handlerName, handlerName,
		handlerName, handlerName, handlerName, handlerName,
		handlerName, handlerName, handlerName, handlerName, handlerName,
		handlerName, handlerName, handlerName)
}

func generateMiddlewareContent(name string) string {
	middlewareName := utils.ToGoIdentifier(name)

	return fmt.Sprintf(`package middleware

import (
	"github.com/arthurlch/goryu"
)

// %s creates a new %s middleware
func %s() goryu.Middleware {
	return func(next goryu.HandlerFunc) goryu.HandlerFunc {
		return func(c *goryu.Context) {
			// TODO: Add middleware logic before request
			
			// Call next handler
			next(c)
			
			// TODO: Add middleware logic after request
		}
	}
}
`, middlewareName, name, middlewareName)
}
