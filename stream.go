package goryu

// Typed streaming JSON helpers built on Ctx.Stream / Ctx.SSE.

import (
	"io"

	goryujson "github.com/arthurlch/goryu/internal/json"
)

// StreamJSON streams T values as newline-delimited JSON (application/x-ndjson),
// flushing after each value.
func StreamJSON[T any](c *Ctx, producer func(emit func(T) error) error) error {
	if err := c.SetHeader("Content-Type", "application/x-ndjson"); err != nil {
		return err
	}
	return c.Stream(func(w io.Writer) error {
		enc := goryujson.Default.NewEncoder(w)
		return producer(func(v T) error {
			return enc.Encode(v)
		})
	})
}

func SSEJSON[T any](c *Ctx, event string, producer func(send func(T) error) error) error {
	return c.SSE(func(send func(SSEvent) error) error {
		return producer(func(v T) error {
			data, err := goryujson.Default.Marshal(v)
			if err != nil {
				return err
			}
			return send(SSEvent{Event: event, Data: string(data)})
		})
	})
}
