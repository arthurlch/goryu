package plugins

import (
	"fmt"
	"net/http"
	"time"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/recovery"
)

type RecoveryBuilder struct {
	*BaseBuilder
	config recovery.Config
}

func NewRecoveryBuilder() *RecoveryBuilder {
	return &RecoveryBuilder{
		BaseBuilder: NewBaseBuilder("recovery"),
		config: recovery.Config{
			EnableStackTrace: true,
		},
	}
}

func (b *RecoveryBuilder) EnableStackTrace(enabled bool) *RecoveryBuilder {
	b.config.EnableStackTrace = enabled
	return b
}

func (b *RecoveryBuilder) DisableStackTrace() *RecoveryBuilder {
	b.config.EnableStackTrace = false
	return b
}

func (b *RecoveryBuilder) Handler(handler func(c *context.Context, err interface{})) *RecoveryBuilder {
	b.config.CustomRecoveryHandler = handler
	return b
}

func (b *RecoveryBuilder) JSONResponse() *RecoveryBuilder {
	b.config.CustomRecoveryHandler = func(c *context.Context, err interface{}) {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":     "Internal Server Error",
			"message":   "An unexpected error occurred",
			"timestamp": time.Now().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return b
}

func (b *RecoveryBuilder) Production() *RecoveryBuilder {
	b.config.EnableStackTrace = false
	return b
}

func (b *RecoveryBuilder) Development() *RecoveryBuilder {
	b.config.EnableStackTrace = true
	return b
}

func (b *RecoveryBuilder) Silent() *RecoveryBuilder {
	b.config.EnableStackTrace = false
	return b
}

func (b *RecoveryBuilder) Build() context.Middleware {
	if err := b.Validate(); err != nil {
		panic(fmt.Sprintf("Recovery configuration invalid: %v", err))
	}
	return recovery.New(b.config)
}

func (b *RecoveryBuilder) Validate() error {
	b.ClearErrors()

	return b.BaseBuilder.Validate()
}

func (b *RecoveryBuilder) Reset() Builder {
	return NewRecoveryBuilder()
}

func (b *RecoveryBuilder) Clone() Builder {
	clone := NewRecoveryBuilder()
	clone.config = b.config
	return clone
}

func init() {
	Register("recovery", func() Builder {
		return NewRecoveryBuilder()
	})
}
