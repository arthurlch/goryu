package timeout
import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
	goryuContext "github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/base"
)
type Config struct {
	base.BaseConfig
	Timeout time.Duration
	TimeoutHandler goryuContext.HandlerFunc
}
type timeoutWriter struct {
	http.ResponseWriter
	mu         sync.Mutex
	timedOut   int32 
	headerSent int32 
}
func (tw *timeoutWriter) WriteHeader(status int) {
	if atomic.LoadInt32(&tw.timedOut) == 1 {
		return 
	}
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if atomic.CompareAndSwapInt32(&tw.headerSent, 0, 1) {
		tw.ResponseWriter.WriteHeader(status)
	}
}
func (tw *timeoutWriter) Write(data []byte) (int, error) {
	if atomic.LoadInt32(&tw.timedOut) == 1 {
		return 0, http.ErrHandlerTimeout 
	}
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if atomic.LoadInt32(&tw.timedOut) == 1 {
		return 0, http.ErrHandlerTimeout
	}
	return tw.ResponseWriter.Write(data)
}
func (tw *timeoutWriter) markTimedOut() {
	atomic.StoreInt32(&tw.timedOut, 1)
}
func (c *Config) Configure(baseConfig *base.BaseConfig) {
	c.BaseConfig = *baseConfig
}
func (c *Config) Validate() error {
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}
	if c.Timeout > 5*time.Minute {
		return base.NewConfigError("Timeout", "cannot exceed 5 minutes")
	}
	if c.TimeoutHandler == nil {
		c.TimeoutHandler = func(ctx *goryuContext.Context) {
			if ctx.Writer.Header().Get("Content-Type") == "" {
				ctx.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
			}
			ctx.Writer.WriteHeader(http.StatusRequestTimeout)
			_ = ctx.Text(http.StatusRequestTimeout, "Request Timeout")
		}
	}
	return nil
}
func New(config Config) func(next goryuContext.HandlerFunc) goryuContext.HandlerFunc {
	if err := config.Validate(); err != nil {
		return func(next goryuContext.HandlerFunc) goryuContext.HandlerFunc {
			return func(c *goryuContext.Context) {
				base.DefaultErrorHandler(c, err, "Timeout")
			}
		}
	}
	return func(next goryuContext.HandlerFunc) goryuContext.HandlerFunc {
		return func(c *goryuContext.Context) {
			if config.Skip != nil && config.Skip(c) {
				next(c)
				return
			}
			ctx, cancel := context.WithTimeout(c.Request.Context(), config.Timeout)
			defer cancel() 
			timeoutWriter := &timeoutWriter{
				ResponseWriter: c.Writer,
			}
			originalWriter := c.Writer
			c.Writer = timeoutWriter
			c.Request = c.Request.WithContext(ctx)
			type result struct {
				panicked bool
				panicValue interface{}
			}
			done := make(chan result, 1) 
			go func() {
				defer func() {
					res := result{}
					if r := recover(); r != nil {
						res.panicked = true
						res.panicValue = r
					}
					select {
					case done <- res:
					default:
					}
				}()
				select {
				case <-ctx.Done():
					return
				default:
					next(c)
				}
			}()
			select {
			case res := <-done:
				c.Writer = originalWriter 
				if res.panicked {
					panic(res.panicValue) 
				}
				return
			case <-ctx.Done():
				timeoutWriter.markTimedOut()
				c.Writer = originalWriter 
				if ctx.Err() == context.DeadlineExceeded {
					config.TimeoutHandler(c)
				}
				return
			}
		}
	}
}
func Default() func(next goryuContext.HandlerFunc) goryuContext.HandlerFunc {
	return New(Config{})
}
