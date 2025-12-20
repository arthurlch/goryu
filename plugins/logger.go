package plugins

import (
	"fmt"
	"io"
	"os"
	"time"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/logger"
)

type LoggerBuilder struct {
	*BaseBuilder
	config logger.Config
}

func NewLoggerBuilder() *LoggerBuilder {
	return &LoggerBuilder{
		BaseBuilder: NewBaseBuilder("logger"),
		config: logger.Config{
			Format:     "[GORYU] ${time} | ${status} | ${latency} | ${ip} | ${method} ${path}\n",
			TimeFormat: time.RFC3339,
			Output:     os.Stdout,
			DisableColors: false,
		},
	}
}

func (b *LoggerBuilder) Format(format string) *LoggerBuilder {
	b.config.Format = format
	return b
}

func (b *LoggerBuilder) TimeFormat(format string) *LoggerBuilder {
	b.config.TimeFormat = format
	return b
}

func (b *LoggerBuilder) Output(w io.Writer) *LoggerBuilder {
	b.config.Output = w
	return b
}

func (b *LoggerBuilder) Colors(enabled bool) *LoggerBuilder {
	b.config.DisableColors = !enabled
	return b
}

func (b *LoggerBuilder) DisableColors() *LoggerBuilder {
	b.config.DisableColors = true
	return b
}

func (b *LoggerBuilder) EnableColors() *LoggerBuilder {
	b.config.DisableColors = false
	return b
}

func (b *LoggerBuilder) TimeZone(tz string) *LoggerBuilder {
	b.config.TimeZone = tz
	return b
}

func (b *LoggerBuilder) JSON() *LoggerBuilder {
	b.config.Format = `{"time":"${time}","status":${status},"latency":"${latency}","ip":"${ip}","method":"${method}","path":"${path}"}`
	b.config.DisableColors = true
	return b
}

func (b *LoggerBuilder) CommonLog() *LoggerBuilder {
	b.config.Format = "${ip} - - [${time}] \"${method} ${path} ${proto}\" ${status} ${size}"
	return b
}

func (b *LoggerBuilder) CombinedLog() *LoggerBuilder {
	b.config.Format = "${ip} - - [${time}] \"${method} ${path} ${proto}\" ${status} ${size} \"${user_agent}\""
	return b
}

func (b *LoggerBuilder) Production() *LoggerBuilder {
	b.config.Format = `{"time":"${time}","status":${status},"latency":"${latency}","ip":"${ip}","method":"${method}","path":"${path}"}`
	b.config.DisableColors = true
	b.config.TimeFormat = time.RFC3339
	return b
}

func (b *LoggerBuilder) Development() *LoggerBuilder {
	b.config.Format = "[GORYU] ${time} | ${status} | ${latency} | ${ip} | ${method} ${path}"
	b.config.DisableColors = false
	b.config.TimeFormat = "15:04:05"
	return b
}

func (b *LoggerBuilder) Build() context.Middleware {
	if err := b.Validate(); err != nil {
		panic(fmt.Sprintf("Logger configuration invalid: %v", err))
	}
	return logger.New(b.config)
}

func (b *LoggerBuilder) Validate() error {
	b.ClearErrors()
	
	if b.config.Output == nil {
		b.AddError(fmt.Errorf("output writer cannot be nil"))
	}
	
	if b.config.Format == "" {
		b.AddError(fmt.Errorf("format cannot be empty"))
	}
	
	if b.config.TimeFormat == "" {
		b.AddError(fmt.Errorf("time format cannot be empty"))
	}
	
	return b.BaseBuilder.Validate()
}

func (b *LoggerBuilder) Reset() Builder {
	return NewLoggerBuilder()
}

func (b *LoggerBuilder) Clone() Builder {
	clone := NewLoggerBuilder()
	clone.config = b.config
	return clone
}

func init() {
	Register("logger", func() Builder {
		return NewLoggerBuilder()
	})
}