package structlog

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/arthurlch/goryu"
	"github.com/google/uuid"
)

// Config defines the configuration for structured logging middleware
type Config struct {
	// Logger is the structured logger instance. If nil, a default JSON logger is created
	Logger *slog.Logger
	// Output is where logs are written. Default: os.Stdout
	Output io.Writer
	// Level is the minimum log level. Default: slog.LevelInfo
	Level slog.Level
	// TimeFormat for log timestamps. Default: RFC3339
	TimeFormat string
	// RequestIDHeader is the header name for request correlation ID. Default: X-Request-ID
	RequestIDHeader string
	// Skip defines when to skip logging
	Skip func(c *goryu.Context) bool
	// CustomFields allows adding custom fields to every log entry
	CustomFields func(c *goryu.Context) map[string]any
}

const (
	RequestIDKey = "request_id"
	LoggerKey    = "logger"
	StartTimeKey = "start_time"
)

// New creates a new structured logging middleware
func New(config ...Config) goryu.Middleware {
	cfg := Config{
		Output:          os.Stdout,
		Level:           slog.LevelInfo,
		TimeFormat:      time.RFC3339,
		RequestIDHeader: "X-Request-ID",
	}

	if len(config) > 0 {
		provided := config[0]
		if provided.Output != nil {
			cfg.Output = provided.Output
		}
		if provided.Level != 0 {
			cfg.Level = provided.Level
		}
		if provided.TimeFormat != "" {
			cfg.TimeFormat = provided.TimeFormat
		}
		if provided.RequestIDHeader != "" {
			cfg.RequestIDHeader = provided.RequestIDHeader
		}
		cfg.Logger = provided.Logger
		cfg.Skip = provided.Skip
		cfg.CustomFields = provided.CustomFields
	}

	// Create default logger if not provided
	if cfg.Logger == nil {
		opts := &slog.HandlerOptions{
			Level: cfg.Level,
		}
		handler := slog.NewJSONHandler(cfg.Output, opts)
		cfg.Logger = slog.New(handler)
	}

	return func(next goryu.HandlerFunc) goryu.HandlerFunc {
		return func(c *goryu.Context) {
			if cfg.Skip != nil && cfg.Skip(c) {
				next(c)
				return
			}

			// Generate or extract request ID
			requestID := c.GetHeader(cfg.RequestIDHeader)
			if requestID == "" {
				requestID = uuid.New().String()
			}

			// Store request ID and logger in context
			c.Set(RequestIDKey, requestID)
			c.Set(LoggerKey, cfg.Logger)
			c.Set(StartTimeKey, time.Now())

			// Set request ID header for response
			c.SetHeader(cfg.RequestIDHeader, requestID)

			// Create a response writer wrapper to capture status code and size
			rw := &responseWriter{
				ResponseWriter: c.Writer,
				statusCode:     200, // Default to 200
			}
			c.Writer = rw

			// Process request
			next(c)

			// Calculate duration
			startTime, _ := c.Get(StartTimeKey)
			duration := time.Since(startTime.(time.Time))

			// Prepare log data
			method := c.Request.Method
			path := c.Request.URL.Path
			statusCode := rw.statusCode
			durationMs := fmt.Sprintf("%.2f", float64(duration.Nanoseconds())/1e6)
			userAgent := c.GetHeader("User-Agent")
			remoteIP := c.RemoteIP()
			responseSize := rw.size
			message := fmt.Sprintf("%s %s %d", method, path, statusCode)

			// Prepare log attributes
			attrs := []slog.Attr{
				slog.String("request_id", requestID),
				slog.String("method", method),
				slog.String("path", path),
				slog.Int("status_code", statusCode),
				slog.String("duration_ms", durationMs),
				slog.String("user_agent", userAgent),
				slog.String("remote_ip", remoteIP),
				slog.Int("response_size", responseSize),
			}

			// Add custom fields if provided
			if cfg.CustomFields != nil {
				customFields := cfg.CustomFields(c)
				attrs = append(attrs, slog.Any("custom", customFields))
			}

			// Determine log level based on status code
			var level slog.Level
			switch {
			case statusCode >= 500:
				level = slog.LevelError
			case statusCode >= 400:
				level = slog.LevelWarn
			default:
				level = slog.LevelInfo
			}

			// Log the request
			cfg.Logger.LogAttrs(c.Request.Context(), level, message, attrs...)
		}
	}
}

// responseWriter wraps http.ResponseWriter to capture status code and response size
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw._, _ = ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// GetRequestID retrieves the request ID from context
func GetRequestID(c *goryu.Context) string {
	if id, exists := c.Get(RequestIDKey); exists {
		if idStr, ok := id.(string); ok {
			return idStr
		}
	}
	return ""
}

// GetLogger retrieves the structured logger from context
func GetLogger(c *goryu.Context) *slog.Logger {
	if logger, exists := c.Get(LoggerKey); exists {
		if slogger, ok := logger.(*slog.Logger); ok {
			return slogger
		}
	}
	// Return default logger if not found
	return slog.Default()
}

// LogInfo logs an info message with request correlation
func LogInfo(c *goryu.Context, msg string, fields ...any) {
	logger := GetLogger(c)
	requestID := GetRequestID(c)

	args := []any{slog.String("request_id", requestID)}
	args = append(args, fields...)

	logger.Info(msg, args...)
}

// LogError logs an error message with request correlation
func LogError(c *goryu.Context, msg string, err error, fields ...any) {
	logger := GetLogger(c)
	requestID := GetRequestID(c)

	args := []any{
		slog.String("request_id", requestID),
		slog.String("error", err.Error()),
	}
	args = append(args, fields...)

	logger.Error(msg, args...)
}

// LogWarn logs a warning message with request correlation
func LogWarn(c *goryu.Context, msg string, fields ...any) {
	logger := GetLogger(c)
	requestID := GetRequestID(c)

	args := []any{slog.String("request_id", requestID)}
	args = append(args, fields...)

	logger.Warn(msg, args...)
}

// CorrelatedLogger returns a logger that automatically includes the request ID
func CorrelatedLogger(c *goryu.Context) *slog.Logger {
	logger := GetLogger(c)
	requestID := GetRequestID(c)

	return logger.With(slog.String("request_id", requestID))
}
