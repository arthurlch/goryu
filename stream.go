package goryu

// Typed streaming JSON helpers. These build on Ctx.Stream / Ctx.SSE and give a
// generic, type-safe way to stream a sequence of values — the common shape for
// LLM token/chunk streaming and long-running list endpoints.
// Stream is dedicated to llm token streaming,
// and SSE is for general-purpose server-sent events.

import (
	"io"

	goryujson "github.com/arthurlch/goryu/internal/json"
)

// The StreamJSON streams a sequence of T values as newline-delimited JSON (NDJSON,
// Content-Type application/x-ndjson), flushing after each value. The producer
// receives an emit function; returning an error from emit or the producer stops
// the stream. Please check eth example below !
//
//	goryu.StreamJSON(c, func(emit func(Chunk) error) error {
//	    for chunk := range chunks {
//	        if err := emit(chunk); err != nil {
//	            return err
//	        }
//	    }
//	    return nil
//	})
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
