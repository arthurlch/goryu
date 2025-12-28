package errors

import (
	"fmt"
	"net/http"
	"time"
)

type AppError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Status     int                    `json:"-"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Internal   error                  `json:"-"`
	Timestamp  time.Time              `json:"timestamp"`
	RequestID  string                 `json:"request_id,omitempty"`
	StackTrace []string               `json:"-"`
	Source     string                 `json:"source,omitempty"`
}

func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %s (internal: %v)", e.Code, e.Message, e.Internal)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	e.Details = details
	return e
}
func (e *AppError) WithDetail(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}
func (e *AppError) WithInternal(err error) *AppError {
	e.Internal = err
	return e
}
func (e *AppError) WithRequestID(id string) *AppError {
	e.RequestID = id
	return e
}
func (e *AppError) WithSource(source string) *AppError {
	e.Source = source
	return e
}

type ErrorBuilder struct {
	err *AppError
}

func NewError(code string, message string) *ErrorBuilder {
	return &ErrorBuilder{
		err: &AppError{
			Code:      code,
			Message:   message,
			Status:    http.StatusInternalServerError,
			Timestamp: time.Now(),
		},
	}
}
func (b *ErrorBuilder) Status(status int) *ErrorBuilder {
	b.err.Status = status
	return b
}
func (b *ErrorBuilder) Details(details map[string]interface{}) *ErrorBuilder {
	b.err.Details = details
	return b
}
func (b *ErrorBuilder) Detail(key string, value interface{}) *ErrorBuilder {
	if b.err.Details == nil {
		b.err.Details = make(map[string]interface{})
	}
	b.err.Details[key] = value
	return b
}
func (b *ErrorBuilder) Internal(err error) *ErrorBuilder {
	b.err.Internal = err
	return b
}
func (b *ErrorBuilder) RequestID(id string) *ErrorBuilder {
	b.err.RequestID = id
	return b
}
func (b *ErrorBuilder) Source(source string) *ErrorBuilder {
	b.err.Source = source
	return b
}
func (b *ErrorBuilder) Build() *AppError {
	return b.err
}
func BadRequest(message string) *AppError {
	return NewError("BAD_REQUEST", message).Status(http.StatusBadRequest).Build()
}
func Unauthorized(message string) *AppError {
	return NewError("UNAUTHORIZED", message).Status(http.StatusUnauthorized).Build()
}
func Forbidden(message string) *AppError {
	return NewError("FORBIDDEN", message).Status(http.StatusForbidden).Build()
}
func NotFound(resource string) *AppError {
	return NewError("NOT_FOUND", fmt.Sprintf("%s not found", resource)).
		Status(http.StatusNotFound).
		Detail("resource", resource).
		Build()
}
func Conflict(message string) *AppError {
	return NewError("CONFLICT", message).Status(http.StatusConflict).Build()
}
func ValidationError(field string, message string) *AppError {
	return NewError("VALIDATION_ERROR", "Validation failed").
		Status(http.StatusBadRequest).
		Detail("field", field).
		Detail("error", message).
		Build()
}
func InternalError(err error) *AppError {
	return NewError("INTERNAL_ERROR", "An internal error occurred").
		Status(http.StatusInternalServerError).
		Internal(err).
		Build()
}
func Wrap(err error, code string, message string) *AppError {
	if err == nil {
		return nil
	}
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return NewError(code, message).Internal(err).Build()
}
func WrapWithStatus(err error, status int, code string, message string) *AppError {
	if err == nil {
		return nil
	}
	return NewError(code, message).Status(status).Internal(err).Build()
}
