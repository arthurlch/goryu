package compress_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arthurlch/goryu"
	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/compress"
)

// lockingWriter mimics net/http: headers are frozen at WriteHeader time, so a
// Content-Encoding added afterwards would be lost (as it is on a real server).
type lockingWriter struct {
	header http.Header
	frozen http.Header
	code   int
	body   bytes.Buffer
}

func (w *lockingWriter) Header() http.Header { return w.header }
func (w *lockingWriter) WriteHeader(code int) {
	if w.frozen == nil {
		w.frozen = w.header.Clone()
		w.code = code
	}
}
func (w *lockingWriter) Write(b []byte) (int, error) {
	if w.frozen == nil {
		w.WriteHeader(http.StatusOK)
	}
	return w.body.Write(b)
}

func TestCompressSetsContentEncodingWhenHandlerSetsStatus(t *testing.T) {
	payload := strings.Repeat("compress me please ", 200)
	handler := func(c *goryu.Ctx) {
		c.Writer.Header().Set("Content-Type", "text/plain")
		c.Writer.WriteHeader(http.StatusOK) // handler commits status before body
		_, _ = c.Writer.Write([]byte(payload))
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	lw := &lockingWriter{header: make(http.Header)}
	c := context.NewContext(lw, req)

	// Drive the middleware exactly like the framework's PostProcess chain.
	compress.New()(func(c *goryu.Ctx) { handler(c) })(c)

	if got := lw.frozen.Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding not committed with headers: got %q", got)
	}
	gz, err := gzip.NewReader(bytes.NewReader(lw.body.Bytes()))
	if err != nil {
		t.Fatalf("body is not valid gzip: %v", err)
	}
	out, _ := io.ReadAll(gz)
	if string(out) != payload {
		t.Fatalf("decompressed body mismatch")
	}
}
