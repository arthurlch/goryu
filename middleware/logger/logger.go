package logger
import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/base"
)
const (
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorReset  = "\033[0m"
)
type Config struct {
	base.BaseConfig
	Output io.Writer
	TimeFormat string
	TimeZone string
	DisableColors bool
	Format string
}
func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if c.Format == "" {
		c.Format = "[GORYU] ${time} | ${status} | ${latency} | ${ip} | ${method} ${path}\n"
	}
	if c.TimeFormat == "" {
		c.TimeFormat = time.RFC3339
	}
	if c.TimeZone == "" {
		c.TimeZone = "Local"
	}
	if c.Output == nil {
		c.Output = os.Stdout
	}
	return nil
}
func New(config Config) func(next context.HandlerFunc) context.HandlerFunc {
	if err := config.Validate(); err != nil {
		return func(next context.HandlerFunc) context.HandlerFunc {
			return func(c *context.Context) {
				base.DefaultErrorHandler(c, err, "Logger")
			}
		}
	}
	logger := log.New(config.Output, "", 0)
	var mu sync.Mutex
	handler := func(c *context.Context) error {
		start := time.Now()
		lrw := base.NewStandardResponseWriter(c.Writer)
		c.Writer = lrw
		requestID := c.Request.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("logger.start_time", start)
		c.Set("logger.request_id", requestID)
		c.Set("logger.response_writer", lrw)
		return nil
	}
	postHandler := func(c *context.Context) error {
		startVal, exists := c.Get("logger.start_time")
		if !exists {
			return nil
		}
		start := startVal.(time.Time)
		requestIDVal, _ := c.Get("logger.request_id")
		requestID := requestIDVal.(string)
		lrwVal, exists := c.Get("logger.response_writer")
		if !exists {
			return nil
		}
		lrw := lrwVal.(*base.StandardResponseWriter)
		stop := time.Now()
		latency := stop.Sub(start)
		clientIP := getClientIP(c)
		method := c.Request.Method
		path := c.Request.URL.Path
		proto := c.Request.Proto
		statusCode := lrw.Status()
		size := lrw.Size()
		userAgent := c.Request.UserAgent()
		err, _ := c.Get("error")
		errMsg := ""
		if err != nil {
			if e, ok := err.(error); ok {
				errMsg = e.Error()
			}
		}
		isColorEnabled := !config.DisableColors
		statusColor := colorForStatus(statusCode, isColorEnabled)
		methodColor := colorForMethod(method, isColorEnabled)
		resetColor := colorReset
		if !isColorEnabled {
			resetColor = ""
		}
		var buf bytes.Buffer
		template := config.Format
		replacer := strings.NewReplacer(
			"${time}", stop.Format(config.TimeFormat),
			"${request_id}", requestID,
			"${status}", fmt.Sprintf("%s%d%s", statusColor, statusCode, resetColor),
			"${latency}", latency.String(),
			"${ip}", clientIP,
			"${method}", fmt.Sprintf("%s%s%s", methodColor, method, resetColor),
			"${path}", path,
			"${proto}", proto,
			"${size}", strconv.Itoa(size),
			"${user_agent}", userAgent,
			"${error}", errMsg,
		)
		buf.WriteString(replacer.Replace(template))
		mu.Lock()
		defer mu.Unlock()
		logger.Print(buf.String())
		return nil
	}
	return base.PostProcessMiddleware("Logger", config.BaseConfig, handler, postHandler)
}
func Default() func(next context.HandlerFunc) context.HandlerFunc {
	return New(Config{})
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
func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
func colorForStatus(code int, enable bool) string {
	if !enable {
		return ""
	}
	switch {
	case code >= http.StatusOK && code < http.StatusMultipleChoices:
		return colorGreen
	case code >= http.StatusMultipleChoices && code < http.StatusBadRequest:
		return colorBlue
	case code >= http.StatusBadRequest && code < http.StatusInternalServerError:
		return colorYellow
	default:
		return colorRed
	}
}
func colorForMethod(method string, enable bool) string {
	if !enable {
		return ""
	}
	switch method {
	case "GET":
		return colorBlue
	case "POST":
		return colorCyan
	case "PUT":
		return colorYellow
	case "DELETE":
		return colorRed
	case "PATCH":
		return colorPurple
	default:
		return colorReset
	}
}
