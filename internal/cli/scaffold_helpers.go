package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arthurlch/goryu/internal/utils"
)

type Field struct {
	Name string
	Type string
	Tag  string
}

func parseFields(fields string) []Field {
	var result []Field
	fieldDefs := strings.Split(fields, ",")

	for _, fieldDef := range fieldDefs {
		parts := strings.Split(strings.TrimSpace(fieldDef), ":")
		if len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			typ := strings.TrimSpace(parts[1])

			goType := convertToGoType(typ)
			jsonTag := fmt.Sprintf(`json:"%s"`, name)

			result = append(result, Field{
				Name: utils.ToGoIdentifier(name),
				Type: goType,
				Tag:  jsonTag,
			})
		}
	}

	return result
}

func convertToGoType(typ string) string {
	switch strings.ToLower(typ) {
	case "string", "text":
		return "string"
	case "int", "integer", "number":
		return "int"
	case "int64", "bigint":
		return "int64"
	case "float", "float64", "decimal":
		return "float64"
	case "bool", "boolean":
		return "bool"
	case "time", "datetime", "timestamp":
		return "time.Time"
	case "uuid":
		return "uuid.UUID"
	case "[]string":
		return "[]string"
	case "[]int":
		return "[]int"
	default:
		return "string"
	}
}

func generateRepository(resource string, fields []Field, moduleName string) error {
	repoDir := "internal/repository"
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return err
	}

	resourceName := utils.ToGoIdentifier(resource)
	filename := filepath.Join(repoDir, strings.ToLower(resource)+"_repository.go")

	content := fmt.Sprintf(`package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	
	"%s/internal/models"
)

type %sRepository struct {
	db *sql.DB
}

func New%sRepository(db *sql.DB) *%sRepository {
	return &%sRepository{db: db}
}

func (r *%sRepository) Create(ctx context.Context, %s *models.%s) error {
	query := %s
	INSERT INTO %ss (%s) VALUES (%s) RETURNING id, created_at, updated_at%s
	
	err := r.db.QueryRowContext(ctx, query, %s).Scan(
		&%s.ID, &%s.CreatedAt, &%s.UpdatedAt,
	)
	
	return err
}

func (r *%sRepository) GetByID(ctx context.Context, id int64) (*models.%s, error) {
	query := %s
	SELECT %s FROM %ss WHERE id = $1%s
	
	%s := &models.%s{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(%s)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%s not found")
	}
	
	return %s, err
}

func (r *%sRepository) Update(ctx context.Context, %s *models.%s) error {
	query := %s
	UPDATE %ss SET %s, updated_at = $%d WHERE id = $%d%s
	
	result, err := r.db.ExecContext(ctx, query, %s, %s.ID)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("%s not found")
	}
	
	return nil
}

func (r *%sRepository) Delete(ctx context.Context, id int64) error {
	query := "DELETE FROM %ss WHERE id = $1"
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("%s not found")
	}
	
	return nil
}

func (r *%sRepository) List(ctx context.Context, offset, limit int) ([]*models.%s, error) {
	query := %s
	SELECT %s FROM %ss ORDER BY created_at DESC LIMIT $1 OFFSET $2%s
	
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var items []*models.%s
	for rows.Next() {
		%s := &models.%s{}
		if err := rows.Scan(%s); err != nil {
			return nil, err
		}
		items = append(items, %s)
	}
	
	return items, rows.Err()
}
`, moduleName, resourceName, resourceName, resourceName, resourceName,
		resourceName, strings.ToLower(resource), resourceName,
		"`", strings.ToLower(resource), generateFieldList(fields, false), generatePlaceholders(len(fields)), "`",
		generateFieldValues(fields, strings.ToLower(resource)),
		strings.ToLower(resource), strings.ToLower(resource), strings.ToLower(resource),
		resourceName, resourceName,
		"`", generateFieldList(fields, true), strings.ToLower(resource), "`",
		strings.ToLower(resource), resourceName, generateScanFields(fields, strings.ToLower(resource)),
		strings.ToLower(resource),
		strings.ToLower(resource),
		resourceName, strings.ToLower(resource), resourceName,
		"`", strings.ToLower(resource), generateUpdateFields(fields), len(fields)+1, len(fields)+2, "`",
		generateUpdateValues(fields, strings.ToLower(resource)), strings.ToLower(resource),
		strings.ToLower(resource),
		resourceName, strings.ToLower(resource),
		strings.ToLower(resource),
		resourceName, resourceName,
		"`", generateFieldList(fields, true), strings.ToLower(resource), "`",
		resourceName, strings.ToLower(resource), resourceName, generateScanFields(fields, strings.ToLower(resource)),
		strings.ToLower(resource))

	return os.WriteFile(filename, []byte(content), 0644)
}

func generateValidation(resource string, fields []Field, moduleName string) error {
	validationDir := "internal/validation"
	if err := os.MkdirAll(validationDir, 0755); err != nil {
		return err
	}

	resourceName := utils.ToGoIdentifier(resource)
	filename := filepath.Join(validationDir, strings.ToLower(resource)+"_validation.go")

	content := fmt.Sprintf(`package validation

import (
	"regexp"
	"strings"
	
	"%s/internal/models"
)

var emailRegex = regexp.MustCompile(%s^[a-zA-Z0-9._%s+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$%s)

func Validate%s(model *models.%s) []string {
	var errors []string
	
%s
	
	return errors
}

func Validate%sUpdate(model *models.%s) []string {
	// For updates, we might have different validation rules
	return Validate%s(model)
}
`, moduleName, "`", "\\", "`", resourceName, resourceName, generateValidationChecks(fields), resourceName, resourceName, resourceName)

	return os.WriteFile(filename, []byte(content), 0644)
}

func generateAPITests(resource string, fields []Field, moduleName string) error {
	testsDir := "internal/handlers"
	resourceName := utils.ToGoIdentifier(resource)
	filename := filepath.Join(testsDir, strings.ToLower(resource)+"_test.go")

	content := fmt.Sprintf(`package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	
	"github.com/arthurlch/goryu"
	"%s/internal/models"
)

func TestCreate%s(t *testing.T) {
	app := goryu.New()
	app.POST("/api/%s", Create%s)
	
	payload := map[string]interface{}{
%s
	}
	
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/%s", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %%d", resp.StatusCode)
	}
}

func TestGet%s(t *testing.T) {
	app := goryu.New()
	app.GET("/api/%s/:id", Get%s)
	
	req := httptest.NewRequest("GET", "/api/%s/1", nil)
	resp, err := app.Test(req)
	
	if err != nil {
		t.Fatal(err)
	}
	
	// In a real test, you'd mock the database
	// For now, we expect 404 since no data exists
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %%d", resp.StatusCode)
	}
}

func TestUpdate%s(t *testing.T) {
	app := goryu.New()
	app.PUT("/api/%s/:id", Update%s)
	
	payload := map[string]interface{}{
%s
	}
	
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("PUT", "/api/%s/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	
	// In a real test, you'd mock the database
	// For now, we expect 404 since no data exists
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %%d", resp.StatusCode)
	}
}

func TestDelete%s(t *testing.T) {
	app := goryu.New()
	app.DELETE("/api/%s/:id", Delete%s)
	
	req := httptest.NewRequest("DELETE", "/api/%s/1", nil)
	resp, err := app.Test(req)
	
	if err != nil {
		t.Fatal(err)
	}
	
	// In a real test, you'd mock the database
	// For now, we expect 404 since no data exists
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %%d", resp.StatusCode)
	}
}

func TestList%s(t *testing.T) {
	app := goryu.New()
	app.GET("/api/%s", List%s)
	
	req := httptest.NewRequest("GET", "/api/%s", nil)
	resp, err := app.Test(req)
	
	if err != nil {
		t.Fatal(err)
	}
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %%d", resp.StatusCode)
	}
}
`, moduleName, resourceName, strings.ToLower(resource), resourceName,
		generateTestPayload(fields),
		strings.ToLower(resource),
		resourceName, strings.ToLower(resource), resourceName, strings.ToLower(resource),
		resourceName, strings.ToLower(resource), resourceName,
		generateTestPayload(fields),
		strings.ToLower(resource),
		resourceName, strings.ToLower(resource), resourceName, strings.ToLower(resource),
		resourceName, strings.ToLower(resource), resourceName, strings.ToLower(resource))

	return os.WriteFile(filename, []byte(content), 0644)
}

func generateAPIDocumentation(resource string, fields []Field, routeGroup string) error {
	docsDir := "docs/api"
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		return err
	}

	resourceName := utils.ToGoIdentifier(resource)
	filename := filepath.Join(docsDir, strings.ToLower(resource)+".md")

	content := fmt.Sprintf(`# %s API

## Overview
REST API endpoints for managing %s resources.

## Base URL
%s

## Endpoints

### Create %s
**POST** %s/

Creates a new %s.

**Request Body:**
%sjson
{
%s
}
%s

**Response:**
- **201 Created**: %s created successfully
- **400 Bad Request**: Invalid request data
- **500 Internal Server Error**: Server error

---

### Get %s by ID
**GET** %s/:id

Retrieves a specific %s by ID.

**Parameters:**
- %sid%s (path): The ID of the %s

**Response:**
- **200 OK**: Returns the %s
- **404 Not Found**: %s not found
- **500 Internal Server Error**: Server error

---

### Update %s
**PUT** %s/:id

Updates an existing %s.

**Parameters:**
- %sid%s (path): The ID of the %s

**Request Body:**
%sjson
{
%s
}
%s

**Response:**
- **200 OK**: %s updated successfully
- **400 Bad Request**: Invalid request data
- **404 Not Found**: %s not found
- **500 Internal Server Error**: Server error

---

### Delete %s
**DELETE** %s/:id

Deletes a %s.

**Parameters:**
- %sid%s (path): The ID of the %s

**Response:**
- **204 No Content**: %s deleted successfully
- **404 Not Found**: %s not found
- **500 Internal Server Error**: Server error

---

### List %s
**GET** %s/

Lists all %s with pagination.

**Query Parameters:**
- %spage%s (optional): Page number (default: 1)
- %slimit%s (optional): Items per page (default: 10)

**Response:**
- **200 OK**: Returns array of %s
- **500 Internal Server Error**: Server error

## Models

### %s
%sjson
{
  "id": "integer",
%s,
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
%s
`, resourceName, strings.ToLower(resource),
		routeGroup,
		resourceName, routeGroup, strings.ToLower(resource),
		"```", generateJSONExample(fields), "```",
		resourceName,
		resourceName, routeGroup, strings.ToLower(resource),
		"`", "`", strings.ToLower(resource),
		strings.ToLower(resource), resourceName,
		resourceName, routeGroup, strings.ToLower(resource),
		"`", "`", strings.ToLower(resource),
		"```", generateJSONExample(fields), "```",
		resourceName, resourceName,
		resourceName, routeGroup, strings.ToLower(resource),
		"`", "`", strings.ToLower(resource),
		resourceName, resourceName,
		resourceName, routeGroup, strings.ToLower(resource),
		"`", "`", "`", "`",
		strings.ToLower(resource),
		resourceName,
		"```", generateModelDoc(fields), "```")

	return os.WriteFile(filename, []byte(content), 0644)
}

func generateFieldList(fields []Field, includeID bool) string {
	var fieldNames []string
	if includeID {
		fieldNames = append(fieldNames, "id", "created_at", "updated_at")
	}
	for _, field := range fields {
		fieldNames = append(fieldNames, strings.ToLower(field.Name))
	}
	return strings.Join(fieldNames, ", ")
}

func generatePlaceholders(count int) string {
	var placeholders []string
	for i := 1; i <= count; i++ {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
	}
	return strings.Join(placeholders, ", ")
}

func generateFieldValues(fields []Field, varName string) string {
	var values []string
	for _, field := range fields {
		values = append(values, fmt.Sprintf("%s.%s", varName, field.Name))
	}
	return strings.Join(values, ", ")
}

func generateScanFields(fields []Field, varName string) string {
	scanFields := []string{
		fmt.Sprintf("&%s.ID", varName),
		fmt.Sprintf("&%s.CreatedAt", varName),
		fmt.Sprintf("&%s.UpdatedAt", varName),
	}
	for _, field := range fields {
		scanFields = append(scanFields, fmt.Sprintf("&%s.%s", varName, field.Name))
	}
	return strings.Join(scanFields, ", ")
}

func generateUpdateFields(fields []Field) string {
	var updateFields []string
	for i, field := range fields {
		updateFields = append(updateFields, fmt.Sprintf("%s = $%d", strings.ToLower(field.Name), i+1))
	}
	return strings.Join(updateFields, ", ")
}

func generateUpdateValues(fields []Field, varName string) string {
	var values []string
	for _, field := range fields {
		values = append(values, fmt.Sprintf("%s.%s", varName, field.Name))
	}
	values = append(values, "time.Now()")
	return strings.Join(values, ", ")
}

func generateValidationChecks(fields []Field) string {
	var checks []string
	for _, field := range fields {
		switch field.Type {
		case "string":
			if strings.ToLower(field.Name) == "email" {
				checks = append(checks, fmt.Sprintf(`	if !emailRegex.MatchString(model.%s) {
		errors = append(errors, "invalid email format")
	}`, field.Name))
			} else {
				checks = append(checks, fmt.Sprintf(`	if strings.TrimSpace(model.%s) == "" {
		errors = append(errors, "%s is required")
	}`, field.Name, strings.ToLower(field.Name)))
			}
		case "int", "int64", "float64":
			checks = append(checks, fmt.Sprintf(`	if model.%s <= 0 {
		errors = append(errors, "%s must be greater than 0")
	}`, field.Name, strings.ToLower(field.Name)))
		}
	}
	return strings.Join(checks, "\n")
}

func generateTestPayload(fields []Field) string {
	var payload []string
	for _, field := range fields {
		switch field.Type {
		case "string":
			payload = append(payload, fmt.Sprintf(`		"%s": "test %s"`, strings.ToLower(field.Name), strings.ToLower(field.Name)))
		case "int", "int64":
			payload = append(payload, fmt.Sprintf(`		"%s": 123`, strings.ToLower(field.Name)))
		case "float64":
			payload = append(payload, fmt.Sprintf(`		"%s": 123.45`, strings.ToLower(field.Name)))
		case "bool":
			payload = append(payload, fmt.Sprintf(`		"%s": true`, strings.ToLower(field.Name)))
		}
	}
	return strings.Join(payload, ",\n")
}

func generateJSONExample(fields []Field) string {
	var examples []string
	for _, field := range fields {
		switch field.Type {
		case "string":
			examples = append(examples, fmt.Sprintf(`  "%s": "example %s"`, strings.ToLower(field.Name), strings.ToLower(field.Name)))
		case "int", "int64":
			examples = append(examples, fmt.Sprintf(`  "%s": 123`, strings.ToLower(field.Name)))
		case "float64":
			examples = append(examples, fmt.Sprintf(`  "%s": 123.45`, strings.ToLower(field.Name)))
		case "bool":
			examples = append(examples, fmt.Sprintf(`  "%s": true`, strings.ToLower(field.Name)))
		case "time.Time":
			examples = append(examples, fmt.Sprintf(`  "%s": "2024-01-01T00:00:00Z"`, strings.ToLower(field.Name)))
		}
	}
	return strings.Join(examples, ",\n")
}

func generateModelDoc(fields []Field) string {
	var docs []string
	for _, field := range fields {
		typeDoc := field.Type
		switch field.Type {
		case "time.Time":
			typeDoc = "timestamp"
		case "int", "int64":
			typeDoc = "integer"
		case "float64":
			typeDoc = "number"
		}
		docs = append(docs, fmt.Sprintf(`  "%s": "%s"`, strings.ToLower(field.Name), typeDoc))
	}
	return strings.Join(docs, ",\n")
}

func generateServiceStructure(name string) error {
	serviceName := strings.ToLower(name)
	directories := []string{
		filepath.Join(serviceName, "cmd", "server"),
		filepath.Join(serviceName, "internal", "service"),
		filepath.Join(serviceName, "internal", "handlers"),
		filepath.Join(serviceName, "internal", "repository"),
		filepath.Join(serviceName, "internal", "models"),
		filepath.Join(serviceName, "api", "proto"),
		filepath.Join(serviceName, "config"),
		filepath.Join(serviceName, "deployments"),
		filepath.Join(serviceName, "scripts"),
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

func generateServiceImplementation(name string, includeHTTP, includeGRPC bool) error {
	serviceName := utils.ToGoIdentifier(name)
	filename := filepath.Join(strings.ToLower(name), "internal", "service", strings.ToLower(name)+".go")

	content := fmt.Sprintf(`package service

import (
	"context"
	"log"
)

// %sService interface defines the service operations
type %sService interface {
	// Add your service methods here
	GetHealth(ctx context.Context) (*HealthResponse, error)
}

// %sServiceImpl implements %sService
type %sServiceImpl struct {
	// Add dependencies like repositories, clients, etc.
}

// New%sService creates a new service instance
func New%sService() %sService {
	return &%sServiceImpl{}
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status  string %smap[string]interface{} %s
	Message string %s
}

// GetHealth returns the health status of the service
func (s *%sServiceImpl) GetHealth(ctx context.Context) (*HealthResponse, error) {
	return &HealthResponse{
		Status:  "ok",
		Message: "%s service is healthy",
	}, nil
}
`, serviceName, serviceName, serviceName, serviceName, serviceName, serviceName, serviceName, serviceName, serviceName,
		"`json:\"status\"`", "`json:\"data,omitempty\"`", "`json:\"message\"`",
		serviceName, serviceName)

	return os.WriteFile(filename, []byte(content), 0644)
}

func generateHTTPService(name string) error {
	serviceName := utils.ToGoIdentifier(name)
	filename := filepath.Join(strings.ToLower(name), "internal", "handlers", "http.go")

	content := fmt.Sprintf(`package handlers

import (
	"net/http"
	
	"github.com/arthurlch/goryu"
	"%s/internal/service"
)

type HTTPHandler struct {
	service service.%sService
}

func NewHTTPHandler(service service.%sService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

// RegisterRoutes registers all HTTP routes
func (h *HTTPHandler) RegisterRoutes(app *goryu.App) {
	api := app.Group("/api/v1")
	
	api.GET("/health", h.GetHealth)
	// Add more routes here
}

// GetHealth handles health check requests
func (h *HTTPHandler) GetHealth(c *goryu.Context) {
	ctx := c.Request().Context()
	
	health, err := h.service.GetHealth(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get health status",
		})
		return
	}
	
	c.JSON(http.StatusOK, health)
}
`, strings.ToLower(name), serviceName, serviceName)

	return os.WriteFile(filename, []byte(content), 0644)
}

func generateGRPCService(name string) error {
	serviceName := utils.ToGoIdentifier(name)
	lowerName := strings.ToLower(name)

	// make proto file
	protoFile := filepath.Join(lowerName, "api", "proto", lowerName+".proto")
	protoContent := fmt.Sprintf(`syntax = "proto3";

package %s;

option go_package = "./%s";

service %sService {
  rpc GetHealth(HealthRequest) returns (HealthResponse);
}

message HealthRequest {}

message HealthResponse {
  string status = 1;
  string message = 2;
}
`, lowerName, lowerName, serviceName)

	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		return err
	}

	// make gRPC handler (but well not fully supported yet)
	grpcFile := filepath.Join(lowerName, "internal", "handlers", "grpc.go")
	grpcContent := fmt.Sprintf(`package handlers

import (
	"context"
	
	"%s/internal/service"
	pb "%s/api/proto"
)

type GRPCHandler struct {
	pb.Unimplemented%sServiceServer
	service service.%sService
}

func NewGRPCHandler(service service.%sService) *GRPCHandler {
	return &GRPCHandler{service: service}
}

func (h *GRPCHandler) GetHealth(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	health, err := h.service.GetHealth(ctx)
	if err != nil {
		return nil, err
	}
	
	return &pb.HealthResponse{
		Status:  health.Status,
		Message: health.Message,
	}, nil
}
`, lowerName, lowerName, serviceName, serviceName, serviceName)

	return os.WriteFile(grpcFile, []byte(grpcContent), 0644)
}

func generateKafkaService(name string) error {
	serviceName := utils.ToGoIdentifier(name)
	filename := filepath.Join(strings.ToLower(name), "internal", "handlers", "kafka.go")

	content := fmt.Sprintf(`package handlers

import (
	"context"
	"encoding/json"
	"log"
	
	"github.com/segmentio/kafka-go"
	"%s/internal/service"
)

type KafkaHandler struct {
	service service.%sService
	reader  *kafka.Reader
	writer  *kafka.Writer
}

func NewKafkaHandler(service service.%sService, brokers []string) *KafkaHandler {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   "%s-events",
		GroupID: "%s-consumer-group",
	})
	
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: brokers,
		Topic:   "%s-responses",
	})
	
	return &KafkaHandler{
		service: service,
		reader:  reader,
		writer:  writer,
	}
}

func (h *KafkaHandler) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			message, err := h.reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading Kafka message: %%v", err)
				continue
			}
			
			// Process message
			if err := h.processMessage(ctx, message); err != nil {
				log.Printf("Error processing message: %%v", err)
			}
		}
	}
}

func (h *KafkaHandler) processMessage(ctx context.Context, message kafka.Message) error {
	// Parse message and handle based on type
	var event map[string]interface{}
	if err := json.Unmarshal(message.Value, &event); err != nil {
		return err
	}
	
	// Add your message processing logic here
	log.Printf("Received event: %%+v", event)
	
	return nil
}

func (h *KafkaHandler) Close() error {
	if err := h.reader.Close(); err != nil {
		return err
	}
	return h.writer.Close()
}
`, strings.ToLower(name), serviceName, serviceName, strings.ToLower(name), strings.ToLower(name), strings.ToLower(name))

	return os.WriteFile(filename, []byte(content), 0644)
}

func generateServiceMonitoring(name string) error {
	filename := filepath.Join(strings.ToLower(name), "internal", "monitoring", "metrics.go")

	// Create monitoring directory
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}

	content := fmt.Sprintf(`package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "%s_requests_total",
			Help: "Total number of requests",
		},
		[]string{"method", "endpoint", "status"},
	)
	
	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "%s_request_duration_seconds",
			Help: "Request duration in seconds",
		},
		[]string{"method", "endpoint"},
	)
	
	activeConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "%s_active_connections",
			Help: "Number of active connections",
		},
	)
)

// RecordRequest records a request metric
func RecordRequest(method, endpoint, status string, duration float64) {
	requestsTotal.WithLabelValues(method, endpoint, status).Inc()
	requestDuration.WithLabelValues(method, endpoint).Observe(duration)
}

// SetActiveConnections sets the number of active connections
func SetActiveConnections(count float64) {
	activeConnections.Set(count)
}
`, strings.ToLower(name), strings.ToLower(name), strings.ToLower(name))

	return os.WriteFile(filename, []byte(content), 0644)
}

func generateServiceDeployment(name string, includeGRPC, includeHTTP, includeKafka, includeMonitoring bool) error {
	serviceName := strings.ToLower(name)

	dockerFile := filepath.Join(serviceName, "Dockerfile")
	dockerContent := fmt.Sprintf(`FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o %s cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/%s .
COPY --from=builder /app/config ./config

CMD ["./%s"]
`, serviceName, serviceName, serviceName)

	if err := os.WriteFile(dockerFile, []byte(dockerContent), 0644); err != nil {
		return err
	}

	composeFile := filepath.Join(serviceName, "docker-compose.yml")
	composeContent := fmt.Sprintf(`version: '3.8'

services:
  %s:
    build: .
    ports:
      - "8080:8080"`, serviceName)

	if includeGRPC {
		composeContent += `
      - "9090:9090"`
	}

	if includeMonitoring {
		composeContent += `
      - "8081:8081"  # metrics port`
	}

	composeContent += `
    environment:
      - ENV=development
    depends_on:`

	if includeKafka {
		composeContent += `
      - kafka
    
  kafka:
    image: confluentinc/cp-kafka:latest
    environment:
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    ports:
      - "9092:9092"
    depends_on:
      - zookeeper
  
  zookeeper:
    image: confluentinc/cp-zookeeper:latest
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
      ZOOKEEPER_TICK_TIME: 2000
    ports:
      - "2181:2181"`
	}

	if includeMonitoring {
		composeContent += `
  
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9091:9090"
    volumes:
      - ./deployments/prometheus.yml:/etc/prometheus/prometheus.yml`
	}

	return os.WriteFile(composeFile, []byte(composeContent), 0644)
}

func generateServiceMain(name string, includeGRPC, includeHTTP, includeKafka, includeMonitoring bool) error {
	serviceName := utils.ToGoIdentifier(name)
	lowerName := strings.ToLower(name)
	mainFile := filepath.Join(lowerName, "cmd", "server", "main.go")

	imports := []string{
		`"context"`,
		`"log"`,
		`"os"`,
		`"os/signal"`,
		`"syscall"`,
		`"time"`,
	}

	if includeHTTP {
		imports = append(imports, `"github.com/arthurlch/goryu"`)
		imports = append(imports, fmt.Sprintf(`"%s/internal/handlers"`, lowerName))
	}

	if includeGRPC {
		imports = append(imports, `"google.golang.org/grpc"`)
		imports = append(imports, `"net"`)
		imports = append(imports, fmt.Sprintf(`pb "%s/api/proto"`, lowerName))
	}

	if includeMonitoring {
		imports = append(imports, `"github.com/prometheus/client_golang/prometheus/promhttp"`)
		imports = append(imports, `"net/http"`)
	}

	imports = append(imports, fmt.Sprintf(`"%s/internal/service"`, lowerName))

	content := fmt.Sprintf(`package main

import (
%s
)

func main() {
	// Create service
	svc := service.New%sService()
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
`, strings.Join(imports, "\n\t"), serviceName)

	if includeHTTP {
		content += `	// Start HTTP server
	app := goryu.New()
	httpHandler := handlers.NewHTTPHandler(svc)
	httpHandler.RegisterRoutes(app)
	
	go func() {
		log.Println("Starting HTTP server on :8080")
		if err := app.Listen(":8080"); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()
	
`
	}

	if includeGRPC {
		content += `	// Start gRPC server
	go func() {
		lis, err := net.Listen("tcp", ":9090")
		if err != nil {
			log.Fatalf("Failed to listen on port 9090: %v", err)
		}
		
		grpcServer := grpc.NewServer()
		grpcHandler := handlers.NewGRPCHandler(svc)
		pb.Register` + serviceName + `ServiceServer(grpcServer, grpcHandler)
		
		log.Println("Starting gRPC server on :9090")
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()
	
`
	}

	if includeKafka {
		content += `	// Start Kafka consumer
	go func() {
		kafkaHandler := handlers.NewKafkaHandler(svc, []string{"localhost:9092"})
		defer kafkaHandler.Close()
		
		log.Println("Starting Kafka consumer")
		if err := kafkaHandler.Start(ctx); err != nil {
			log.Printf("Kafka consumer error: %v", err)
		}
	}()
	
`
	}

	if includeMonitoring {
		content += `	// Start metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("Starting metrics server on :8081")
		if err := http.ListenAndServe(":8081", nil); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()
	
`
	}

	content += `	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down...")
	
	// Give services time to shutdown gracefully
	cancel()
	time.Sleep(5 * time.Second)
	
	log.Println("Service stopped")
}
`

	goModFile := filepath.Join(lowerName, "go.mod")
	goModContent := fmt.Sprintf(`module %s

go 1.21

require (
	github.com/arthurlch/goryu v1.0.0`, lowerName)

	if includeGRPC {
		goModContent += `
	google.golang.org/grpc v1.58.0
	google.golang.org/protobuf v1.31.0`
	}

	if includeKafka {
		goModContent += `
	github.com/segmentio/kafka-go v0.4.42`
	}

	if includeMonitoring {
		goModContent += `
	github.com/prometheus/client_golang v1.17.0`
	}

	goModContent += `
)
`

	if err := os.WriteFile(goModFile, []byte(goModContent), 0644); err != nil {
		return err
	}

	return os.WriteFile(mainFile, []byte(content), 0644)
}
