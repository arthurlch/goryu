package structlog

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
	"github.com/google/uuid"
)

type Config struct {
	base.BaseConfig
	Logger          *slog.Logger
	Output          io.Writer
	Level           slog.Level
	TimeFormat      string
	RequestIDHeader string
	CustomFields    func(c *context.Context) map[string]any
}

const (
	RequestIDKey = "request_id"
	LoggerKey    = "logger"
	StartTimeKey = "start_time"
)

func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if c.Output == nil {
		c.Output = os.Stdout
	}
	if c.Level == 0 {
		c.Level = slog.LevelInfo
	}
	if c.TimeFormat == "" {
		c.TimeFormat = time.RFC3339
	}
	if c.RequestIDHeader == "" {
		c.RequestIDHeader = "X-Request-ID"
	}
	if c.Logger == nil {
		opts := &slog.HandlerOptions{
			Level: c.Level,
		}
		handler := slog.NewJSONHandler(c.Output, opts)
		c.Logger = slog.New(handler)
	}
	return nil
}
func New(config ...Config) func(next context.HandlerFunc) context.HandlerFunc {
	cfg := Config{}
	if len(config) > 0 {
		cfg = config[0]
	}

	if err := cfg.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "StructLog")
			}
		}
	}
	preHandler := func(c *context.Context) error {
		requestID := c.Request.Header.Get(cfg.RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set(RequestIDKey, requestID)
		c.Set(LoggerKey, cfg.Logger)
		c.Set(StartTimeKey, time.Now())
		c.Writer.Header().Set(cfg.RequestIDHeader, requestID)
		rw := base.NewStandardResponseWriter(c.Writer)
		c.Writer = rw
		c.Set("structlog.response_writer", rw)
		return nil
	}
	postHandler := func(c *context.Context) error {
		startTimeVal, exists := c.Get(StartTimeKey)
		if !exists {
			return nil
		}
		startTime := startTimeVal.(time.Time)
		duration := time.Since(startTime)
		requestIDVal, _ := c.Get(RequestIDKey)
		requestID := requestIDVal.(string)
		rwVal, exists := c.Get("structlog.response_writer")
		if !exists {
			return nil
		}
		rw := rwVal.(*base.StandardResponseWriter)
		method := c.Request.Method
		path := c.Request.URL.Path
		statusCode := rw.Status()
		durationMs := fmt.Sprintf("%.2f", float64(duration.Nanoseconds())/1e6)
		userAgent := c.Request.Header.Get("User-Agent")
		remoteIP := getClientIP(c)
		responseSize := rw.Size()
		message := fmt.Sprintf("%s %s %d", method, path, statusCode)
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
		if cfg.CustomFields != nil {
			customFields := cfg.CustomFields(c)
			attrs = append(attrs, slog.Any("custom", customFields))
		}
		var level slog.Level
		switch {
		case statusCode >= 500:
			level = slog.LevelError
		case statusCode >= 400:
			level = slog.LevelWarn
		default:
			level = slog.LevelInfo
		}
		cfg.Logger.LogAttrs(c.Request.Context(), level, message, attrs...)
		return nil
	}
	return base.PostProcessMiddleware("StructLog", cfg.BaseConfig, preHandler, postHandler)
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New()
}
func getClientIP(c *context.Context) string {
	if xff := c.Request.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xri := c.Request.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	remoteAddr := c.Request.RemoteAddr
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		return remoteAddr[:idx]
	}
	return remoteAddr
}
func GetRequestID(c *context.Context) string {
	if id, exists := c.Get(RequestIDKey); exists {
		if idStr, ok := id.(string); ok {
			return idStr
		}
	}
	return ""
}
func GetLogger(c *context.Context) *slog.Logger {
	if logger, exists := c.Get(LoggerKey); exists {
		if slogger, ok := logger.(*slog.Logger); ok {
			return slogger
		}
	}
	return slog.Default()
}
func LogInfo(c *context.Context, msg string, fields ...any) {
	logger := GetLogger(c)
	requestID := GetRequestID(c)
	args := []any{slog.String("request_id", requestID)}
	args = append(args, fields...)
	logger.Info(msg, args...)
}
func LogError(c *context.Context, msg string, err error, fields ...any) {
	logger := GetLogger(c)
	requestID := GetRequestID(c)
	args := []any{
		slog.String("request_id", requestID),
		slog.String("error", err.Error()),
	}
	args = append(args, fields...)
	logger.Error(msg, args...)
}
func LogWarn(c *context.Context, msg string, fields ...any) {
	logger := GetLogger(c)
	requestID := GetRequestID(c)
	args := []any{slog.String("request_id", requestID)}
	args = append(args, fields...)
	logger.Warn(msg, args...)
}
func CorrelatedLogger(c *context.Context) *slog.Logger {
	logger := GetLogger(c)
	requestID := GetRequestID(c)
	return logger.With(slog.String("request_id", requestID))
}
