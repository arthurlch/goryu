package compress_test
import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
	"github.com/arthurlch/goryu/middleware/compress"
)
func newTestContext(req *http.Request) (*context.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}
func TestCompressMiddleware(t *testing.T) {
	t.Run("GzipCompression", func(t *testing.T) {
		middleware := compress.New()
		handler := func(c *context.Context) {
			c.SetHeader("Content-Type", "text/html")
			content := strings.Repeat("Hello, World! ", 100)
			c.Text(http.StatusOK, content)
		}
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		if rr.Header().Get("Content-Encoding") != "gzip" {
			t.Errorf("Expected Content-Encoding: gzip, got %s", rr.Header().Get("Content-Encoding"))
		}
		reader, err := gzip.NewReader(rr.Body)
		if err != nil {
			t.Fatalf("Failed to create gzip reader: %v", err)
		}
		defer func() { _ = reader.Close() }()
		decompressed, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("Failed to decompress: %v", err)
		}
		expected := strings.Repeat("Hello, World! ", 100)
		if string(decompressed) != expected {
			t.Errorf("Decompressed content doesn't match expected")
		}
	})
	t.Run("NoCompressionForSmallContent", func(t *testing.T) {
		middleware := compress.New()
		handler := func(c *context.Context) {
			c.SetHeader("Content-Type", "text/html")
			c.Text(http.StatusOK, "Small content")
		}
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Header().Get("Content-Encoding") != "" {
			t.Error("Small content should not be compressed")
		}
		if rr.Body.String() != "Small content" {
			t.Errorf("Expected 'Small content', got %s", rr.Body.String())
		}
	})
	t.Run("NoCompressionWithoutAcceptEncoding", func(t *testing.T) {
		middleware := compress.New()
		handler := func(c *context.Context) {
			c.SetHeader("Content-Type", "text/html")
			content := strings.Repeat("Hello, World! ", 100)
			c.Text(http.StatusOK, content)
		}
		req := httptest.NewRequest("GET", "/", nil)
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Header().Get("Content-Encoding") != "" {
			t.Error("Content should not be compressed without Accept-Encoding")
		}
		expected := strings.Repeat("Hello, World! ", 100)
		if rr.Body.String() != expected {
			t.Error("Content should be uncompressed")
		}
	})
	t.Run("NoCompressionForNonCompressibleTypes", func(t *testing.T) {
		middleware := compress.New()
		handler := func(c *context.Context) {
			c.Writer.Header().Set("Content-Type", "image/jpeg")
			content := strings.Repeat("Binary data ", 100)
			c.Writer.WriteHeader(http.StatusOK)
			_, _ = c.Writer.Write([]byte(content))
		}
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Header().Get("Content-Encoding") != "" {
			t.Error("JPEG images should not be compressed")
		}
	})
	t.Run("CustomCompressibleTypes", func(t *testing.T) {
		config := compress.Config{
			CompressibleTypes: []string{"application/custom"},
			MinLength:         10, 
		}
		middleware := compress.New(config)
		handler := func(c *context.Context) {
			c.Writer.Header().Set("Content-Type", "application/custom")
			c.Writer.WriteHeader(http.StatusOK)
			_, _ = c.Writer.Write([]byte("Custom content type"))
		}
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Header().Get("Content-Encoding") != "gzip" {
			t.Error("Custom content type should be compressed")
		}
	})
	t.Run("DeflateCompression", func(t *testing.T) {
		middleware := compress.New()
		handler := func(c *context.Context) {
			c.SetHeader("Content-Type", "text/html")
			content := strings.Repeat("Hello, World! ", 100)
			c.Text(http.StatusOK, content)
		}
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Encoding", "deflate") 
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Header().Get("Content-Encoding") != "deflate" {
			t.Errorf("Expected Content-Encoding: deflate, got %s", rr.Header().Get("Content-Encoding"))
		}
	})
	t.Run("GzipPreferredOverDeflate", func(t *testing.T) {
		middleware := compress.New()
		handler := func(c *context.Context) {
			c.SetHeader("Content-Type", "text/html")
			content := strings.Repeat("Hello, World! ", 100)
			c.Text(http.StatusOK, content)
		}
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Encoding", "deflate, gzip") 
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Header().Get("Content-Encoding") != "gzip" {
			t.Errorf("Expected gzip to be preferred, got %s", rr.Header().Get("Content-Encoding"))
		}
	})
	t.Run("SkipMiddleware", func(t *testing.T) {
		config := compress.Config{
			BaseConfig: base.BaseConfig{
				Skip: func(c *context.Context) bool {
					return c.Request.URL.Path == "/skip"
				},
			},
		}
		middleware := compress.New(config)
		handler := func(c *context.Context) {
			c.SetHeader("Content-Type", "text/html")
			content := strings.Repeat("Hello, World! ", 100)
			c.Text(http.StatusOK, content)
		}
		req := httptest.NewRequest("GET", "/skip", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Header().Get("Content-Encoding") != "" {
			t.Error("Compression should be skipped for /skip path")
		}
	})
	t.Run("NoCompressionForErrorResponses", func(t *testing.T) {
		middleware := compress.New()
		handler := func(c *context.Context) {
			c.SetHeader("Content-Type", "text/html")
			content := strings.Repeat("Error message ", 100)
			c.Text(http.StatusInternalServerError, content)
		}
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		ctx, rr := newTestContext(req)
		middleware(handler)(ctx)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", rr.Code)
		}
		if rr.Header().Get("Content-Encoding") != "" {
			t.Error("Error responses should not be compressed")
		}
	})
	t.Run("WildcardContentTypeMatching", func(t *testing.T) {
		config := compress.Config{
			CompressibleTypes: []string{"text/*"},
			MinLength:         10,
		}
		middleware := compress.New(config)
		
		handler := func(c *context.Context) {
			c.SetHeader("Content-Type", "text/plain")
			content := strings.Repeat("Hello, World! ", 100)
			c.Text(http.StatusOK, content)
		}
		
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		ctx, rr := newTestContext(req)
		
		middleware(handler)(ctx)
		
		if rr.Header().Get("Content-Encoding") != "gzip" {
			t.Error("text/plain should be compressed with text/* wildcard")
		}
	})
}
