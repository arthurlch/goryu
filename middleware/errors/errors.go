package errors
import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"github.com/arthurlch/goryu"
)
type Config struct {
	ShowDetails bool
	ShowStackTrace bool
	LogErrors bool
	CustomHandler func(c *goryu.Ctx, err error)
	ErrorTransformer func(err error) error
	ResponseFormatter func(c *goryu.Ctx, err *AppError) interface{}
	DevMode bool
}
type ErrorHandlerFunc func(c *goryu.Ctx) error
func New() goryu.Middleware {
	return NewWithConfig(Config{
		ShowDetails:    true,
		ShowStackTrace: false,
		LogErrors:      true,
		DevMode:        false,
	})
}
func NewWithConfig(config Config) goryu.Middleware {
	return func(next goryu.Handler) goryu.Handler {
		return func(c *goryu.Ctx) {
			defer func() {
				if r := recover(); r != nil {
					handlePanic(c, r, config)
				}
			}()
			next(c)
			if c.HasErrors() {
				handleContextErrors(c, config)
			}
		}
	}
}
func Handle(fn ErrorHandlerFunc) goryu.Handler {
	return func(c *goryu.Ctx) {
		if err := fn(c); err != nil {
			SendError(c, err)
		}
	}
}
func SendError(c *goryu.Ctx, err error) {
	config, ok := c.Get("error.config")
	if !ok {
		config = Config{
			ShowDetails: true,
			LogErrors:   true,
		}
	}
	cfg := config.(Config)
	if cfg.ErrorTransformer != nil {
		err = cfg.ErrorTransformer(err)
	}
	var appErr *AppError
	switch e := err.(type) {
	case *AppError:
		appErr = e
	default:
		appErr = InternalError(e)
	}
	if reqID, exists := c.Get("request_id"); exists {
		appErr.RequestID = reqID.(string)
	}
	if cfg.LogErrors {
		logError(appErr, cfg)
	}
	if cfg.CustomHandler != nil {
		cfg.CustomHandler(c, appErr)
		if c.IsResponseSent() {
			return
		}
	}
	sendErrorResponse(c, appErr, cfg)
}
func sendErrorResponse(c *goryu.Ctx, err *AppError, config Config) {
	if c.IsResponseSent() {
		return
	}
	var response interface{}
	if config.ResponseFormatter != nil {
		response = config.ResponseFormatter(c, err)
	} else {
		response = formatErrorResponse(err, config)
	}
	if jsonErr := c.JSON(err.Status, response); jsonErr != nil {
		log.Printf("Error encoding error response: %v", jsonErr)
	}
}
func formatErrorResponse(err *AppError, config Config) map[string]interface{} {
	response := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    err.Code,
			"message": err.Message,
		},
	}
	errorData := response["error"].(map[string]interface{})
	if config.ShowDetails && len(err.Details) > 0 {
		errorData["details"] = err.Details
	}
	if config.DevMode {
		if err.Source != "" {
			errorData["source"] = err.Source
		}
		if err.Internal != nil {
			errorData["internal"] = err.Internal.Error()
		}
		if config.ShowStackTrace && len(err.StackTrace) > 0 {
			errorData["stack_trace"] = err.StackTrace
		}
	}
	if err.RequestID != "" {
		errorData["request_id"] = err.RequestID
	}
	errorData["timestamp"] = err.Timestamp
	return response
}
func handlePanic(c *goryu.Ctx, r interface{}, config Config) {
	err := NewError("PANIC", "Internal server error").
		Status(http.StatusInternalServerError).
		Internal(fmt.Errorf("panic: %v", r)).
		Build()
	if config.DevMode && config.ShowStackTrace {
		err.StackTrace = strings.Split(string(debug.Stack()), "\n")
	}
	if config.LogErrors {
		log.Printf("PANIC: %v\nStack: %s", r, debug.Stack())
	}
	sendErrorResponse(c, err, config)
}
func handleContextErrors(c *goryu.Ctx, config Config) {
	errors := c.Errors()
	if len(errors) == 0 {
		return
	}
	if len(errors) > 1 {
		handleMultipleErrors(c, errors, config)
		return
	}
	SendError(c, errors[0])
}
func handleMultipleErrors(c *goryu.Ctx, errors []error, config Config) {
	appErrors := make([]*AppError, 0, len(errors))
	for _, err := range errors {
		var appErr *AppError
		if ae, ok := err.(*AppError); ok {
			appErr = ae
		} else {
			appErr = InternalError(err)
		}
		appErrors = append(appErrors, appErr)
	}
	status := http.StatusInternalServerError
	for _, err := range appErrors {
		if err.Status < status {
			status = err.Status
		}
	}
	response := map[string]interface{}{
		"errors": appErrors,
	}
	if jsonErr := c.JSON(status, response); jsonErr != nil {
		log.Printf("Error encoding multi-error response: %v", jsonErr)
	}
}
func logError(err *AppError, config Config) {
	if !config.LogErrors {
		return
	}
	logMsg := fmt.Sprintf("ERROR [%s] %s: %s", err.Code, err.Source, err.Message)
	if err.Internal != nil {
		logMsg += fmt.Sprintf(" (internal: %v)", err.Internal)
	}
	if err.RequestID != "" {
		logMsg = fmt.Sprintf("[%s] %s", err.RequestID, logMsg)
	}
	log.Println(logMsg)
	if config.DevMode && len(err.Details) > 0 {
		log.Printf("  Details: %+v", err.Details)
	}
}
func Must(err error) {
	if err != nil {
		panic(err)
	}
}
func MustGet[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}
func Try(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	err = fn()
	return
}
func TryGet[T any](fn func() (T, error)) (value T, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	value, err = fn()
	return
}