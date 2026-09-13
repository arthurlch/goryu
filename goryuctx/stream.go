package goryuctx

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type SSEvent struct {
	ID    string
	Event string
	Data  string
	Retry int
}

func (c *Context) SSE(producer func(send func(SSEvent) error) error) error {
	if !c.markResponseSent() {
		return errors.New("response already started")
	}
	h := c.Writer.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	flusher, _ := c.Writer.(http.Flusher)
	if flusher != nil {
		flusher.Flush()
	}

	send := func(e SSEvent) error {
		if err := c.Request.Context().Err(); err != nil {
			return err
		}
		var b strings.Builder
		if id := stripNewlines(e.ID); id != "" {
			b.WriteString("id: ")
			b.WriteString(id)
			b.WriteByte('\n')
		}
		if ev := stripNewlines(e.Event); ev != "" {
			b.WriteString("event: ")
			b.WriteString(ev)
			b.WriteByte('\n')
		}
		if e.Retry > 0 {
			b.WriteString("retry: ")
			b.WriteString(strconv.Itoa(e.Retry))
			b.WriteByte('\n')
		}
		for _, line := range strings.Split(normalizeNewlines(e.Data), "\n") {
			b.WriteString("data: ")
			b.WriteString(line)
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
		if _, err := io.WriteString(c.Writer, b.String()); err != nil {
			return err
		}
		if flusher != nil {
			flusher.Flush()
		}
		return nil
	}
	return producer(send)
}

func (c *Context) Stream(fn func(w io.Writer) error) error {
	if !c.markResponseSent() {
		return errors.New("response already started")
	}
	c.Writer.WriteHeader(http.StatusOK)
	flusher, _ := c.Writer.(http.Flusher)
	return fn(&flushWriter{w: c.Writer, f: flusher})
}

type flushWriter struct {
	w io.Writer
	f http.Flusher
}

func (fw *flushWriter) Write(p []byte) (int, error) {
	n, err := fw.w.Write(p)
	if fw.f != nil {
		fw.f.Flush()
	}
	return n, err
}

func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

func stripNewlines(s string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}
