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

	"github.com/arthurlch/goryu"
)

// --- ANSI Color Codes ---
const (
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorReset  = "\033[0m"
)

// --- Logger Configuratione ---

type Config struct {
	Next func(c *goryu.Context) bool

	// Default: os.Stdout
	Output io.Writer

	// TimeFormat defines the format for the timestamp in the log.
	// Default: time.RFC3339
	TimeFormat string

	// TimeZone defines the time zone for the timestamp.
	// Default: "Local"
	TimeZone string

	// DisableColors disables colored output.
	// Default: false (colors are enabled by default)
	DisableColors bool

	// Format is the log format string.
	// It supports the following tags:
	//
	// - {time}: Timestamp
	// - {request_id}: Unique ID for the request
	// - {status}: HTTP status code
	// - {latency}: Time taken to process the request
	// - {ip}: Client IP address
	// - {method}: HTTP method
	// - {path}: Request path
	// - {proto}: HTTP protocol
	// - {size}: Response size in bytes
	// - {user_agent}: Client's User-Agent
	// - {error}: Error message, if any
	//
	// Default: [GORYU] ${time} | ${status} | ${latency} | ${ip} | ${method} ${path}
	Format string
}

// --- Middleware Implementation ---

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	n, err := lrw.ResponseWriter.Write(b)
	lrw.size += n
	return n, err
}

func New(config ...Config) goryu.Middleware {
	cfg := Config{
		Format:        "[GORYU] ${time} | ${status} | ${latency} | ${ip} | ${method} ${path}\n",
		TimeFormat:    time.RFC3339,
		TimeZone:      "Local",
		Output:        os.Stdout,
		DisableColors: false, // Colors are enabled by default coz colors are cool!
	}

	if len(config) > 0 {
		userCfg := config[0]
		if userCfg.Format != "" {
			cfg.Format = userCfg.Format
		}
		if userCfg.TimeFormat != "" {
			cfg.TimeFormat = userCfg.TimeFormat
		}
		if userCfg.TimeZone != "" {
			cfg.TimeZone = userCfg.TimeZone
		}
		if userCfg.Output != nil {
			cfg.Output = userCfg.Output
		}
		if userCfg.Next != nil {
			cfg.Next = userCfg.Next
		}
		cfg.DisableColors = userCfg.DisableColors
	}

	logger := log.New(cfg.Output, "", 0)
	var mu sync.Mutex

	return func(next goryu.HandlerFunc) goryu.HandlerFunc {
		return func(c *goryu.Context) {
			if cfg.Next != nil && cfg.Next(c) {
				next(c)
				return
			}

			start := time.Now()
			lrw := newLoggingResponseWriter(c.Writer)
			c.Writer = lrw

			requestID := c.Request.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}

			next(c)

			stop := time.Now()
			latency := stop.Sub(start)
			clientIP := c.RemoteIP()
			method := c.Request.Method
			path := c.Request.URL.Path
			proto := c.Request.Proto
			statusCode := lrw.statusCode
			size := lrw.size
			userAgent := c.Request.UserAgent()

			err, _ := c.Get("error")
			errMsg := ""
			if err != nil {
				if e, ok := err.(error); ok {
					errMsg = e.Error()
				}
			}

			isColorEnabled := !cfg.DisableColors
			statusColor := colorForStatus(statusCode, isColorEnabled)
			methodColor := colorForMethod(method, isColorEnabled)
			resetColor := colorReset
			if !isColorEnabled {
				resetColor = ""
			}

			var buf bytes.Buffer
			template := cfg.Format

			replacer := strings.NewReplacer(
				"${time}", stop.Format(cfg.TimeFormat),
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
		}
	}
}

// --- Helper Functions ---

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
