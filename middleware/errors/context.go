package errors
import (
	"github.com/arthurlch/goryu"
)
type ContextError struct {
	*goryu.Ctx
}
func Error(c *goryu.Ctx) *ContextError {
	return &ContextError{Ctx: c}
}
func (ce *ContextError) Send(err error) {
	SendError(ce.Ctx, err)
}
func (ce *ContextError) BadRequest(message string) {
	ce.Send(BadRequest(message))
}
func (ce *ContextError) Unauthorized(message string) {
	ce.Send(Unauthorized(message))
}
func (ce *ContextError) Forbidden(message string) {
	ce.Send(Forbidden(message))
}
func (ce *ContextError) NotFound(resource string) {
	ce.Send(NotFound(resource))
}
func (ce *ContextError) Conflict(message string) {
	ce.Send(Conflict(message))
}
func (ce *ContextError) Validation(field string, message string) {
	ce.Send(ValidationError(field, message))
}
func (ce *ContextError) Internal(err error) {
	ce.Send(InternalError(err))
}
func (ce *ContextError) Custom(code string, message string, status int) {
	ce.Send(NewError(code, message).Status(status).Build())
}
func HandleResult[T any](c *goryu.Ctx, result T, err error, successHandler func(T)) {
	if err != nil {
		SendError(c, Wrap(err, "OPERATION_FAILED", "Operation failed"))
		return
	}
	successHandler(result)
}
func HandleError(c *goryu.Ctx, err error, code string, message string) bool {
	if err != nil {
		SendError(c, Wrap(err, code, message))
		return true
	}
	return false
}
func ValidateAndHandle(c *goryu.Ctx, validator func() error) bool {
	if err := validator(); err != nil {
		SendError(c, ValidationError("input", err.Error()))
		return false
	}
	return true
}
type Chain struct {
	c   *goryu.Ctx
	err error
}
func NewChain(c *goryu.Ctx) *Chain {
	return &Chain{c: c}
}
func (ch *Chain) Do(fn func() error) *Chain {
	if ch.err != nil {
		return ch
	}
	ch.err = fn()
	return ch
}
func (ch *Chain) DoWithResult(fn func() (interface{}, error)) *Chain {
	if ch.err != nil {
		return ch
	}
	result, err := fn()
	if err != nil {
		ch.err = err
		return ch
	}
	ch.c.Set("chain.result", result)
	return ch
}
func (ch *Chain) Result() (interface{}, bool) {
	return ch.c.Get("chain.result")
}
func (ch *Chain) OnError(handler func(error)) *Chain {
	if ch.err != nil {
		handler(ch.err)
	}
	return ch
}
func (ch *Chain) OnSuccess(handler func()) *Chain {
	if ch.err == nil {
		handler()
	}
	return ch
}
func (ch *Chain) SendError(code string, message string) bool {
	if ch.err != nil {
		SendError(ch.c, Wrap(ch.err, code, message))
		return true
	}
	return false
}
func (ch *Chain) Error() error {
	return ch.err
}