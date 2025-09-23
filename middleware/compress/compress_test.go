package compress_test

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arthurlch/goryu"
	"github.com/arthurlch/goryu/context"
	"github.com/arthurlch/goryu/middleware/compress"
)

func newTestContext(req *http.Request) (*goryu.Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return context.NewContext(rr, req), rr
}

func TestCompressMiddleware(t *testing.T) {
	t.Run("GzipCompression", func(t *testing.T) {
		middleware := compress.New()

		handler := func(c *goryu.Context) {
			c.SetHeader("Content-Type", "text/html")
			// Large enough content to trigger compression
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

		// Check compression headers
		if rr.Header().Get("Content-Encoding") != "gzip" {
			t.Errorf("Expected Content-Encoding: gzip, got %s", rr.Header().Get("Content-Encoding"))
		}

		// Verify content is actually compressed
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

		handler := func(c *goryu.Context) {
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

		handler := func(c *goryu.Context) {
			c.SetHeader("Content-Type", "text/html")
			content := strings.Repeat("Hello, World! ", 100)
			c.Text(http.StatusOK, content)
		}

		req := httptest.NewRequest("GET", "/", nil)
		// No Accept-Encoding header
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

		handler := func(c *goryu.Context) {
			c.Writer.Header().Set("Content-Type", "image/jpeg")
			content := strings.Repeat("Binary data ", 100)
			c.Writer.WriteHeader(http.StatusOK)
			c._, _ = Writer.Write([]byte(content))
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
			MinLength:         10, // Low threshold for testing
		}
		middleware := compress.New(config)

		handler := func(c *goryu.Context) {
			c.Writer.Header().Set("Content-Type", "application/custom")
			c.Writer.WriteHeader(http.StatusOK)
			c._, _ = Writer.Write([]byte("Custom content type"))
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

		handler := func(c *goryu.Context) {
			c.SetHeader("Content-Type", "text/html")
			content := strings.Repeat("Hello, World! ", 100)
			c.Text(http.StatusOK, content)
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Encoding", "deflate") // Only deflate, no gzip
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Header().Get("Content-Encoding") != "deflate" {
			t.Errorf("Expected Content-Encoding: deflate, got %s", rr.Header().Get("Content-Encoding"))
		}
	})

	t.Run("GzipPreferredOverDeflate", func(t *testing.T) {
		middleware := compress.New()

		handler := func(c *goryu.Context) {
			c.SetHeader("Content-Type", "text/html")
			content := strings.Repeat("Hello, World! ", 100)
			c.Text(http.StatusOK, content)
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Encoding", "deflate, gzip") // Both supported
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Header().Get("Content-Encoding") != "gzip" {
			t.Errorf("Expected gzip to be preferred, got %s", rr.Header().Get("Content-Encoding"))
		}
	})

	t.Run("SkipMiddleware", func(t *testing.T) {
		config := compress.Config{
			Next: func(c *goryu.Context) bool {
				return c.Request.URL.Path == "/skip"
			},
		}
		middleware := compress.New(config)

		handler := func(c *goryu.Context) {
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

		handler := func(c *goryu.Context) {
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
			CompressibleTypes: []string{"text/*", "application/*"},
			MinLength:         10,
		}
		middleware := compress.New(config)

		handler := func(c *goryu.Context) {
			c.SetHeader("Content-Type", "text/custom; charset=utf-8")
			c.Text(http.StatusOK, "Custom text content")
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		ctx, rr := newTestContext(req)

		middleware(handler)(ctx)

		if rr.Header().Get("Content-Encoding") != "gzip" {
			t.Error("Wildcard content type matching should work")
		}
	})
}

func TestWithGzip(t *testing.T) {
	middleware := compress.WithGzip(compress.LevelBestCompression)

	handler := func(c *goryu.Context) {
		c.SetHeader("Content-Type", "text/html")
		content := strings.Repeat("Hello, World! ", 100)
		c.Text(http.StatusOK, content)
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	ctx, rr := newTestContext(req)

	middleware(handler)(ctx)

	if rr.Header().Get("Content-Encoding") != "gzip" {
		t.Error("WithGzip should compress with gzip")
	}
}

func BenchmarkCompression(b *testing.B) {
	middleware := compress.New()
	content := strings.Repeat("Hello, World! This is a test of compression performance. ", 100)

	handler := func(c *goryu.Context) {
		c.SetHeader("Content-Type", "text/html")
		c.Text(http.StatusOK, content)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		ctx, _ := newTestContext(req)
		middleware(handler)(ctx)
	}
}

func BenchmarkNoCompression(b *testing.B) {
	middleware := compress.New()
	content := strings.Repeat("Hello, World! This is a test of compression performance. ", 100)

	handler := func(c *goryu.Context) {
		c.SetHeader("Content-Type", "text/html")
		c.Text(http.StatusOK, content)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		// No Accept-Encoding header
		ctx, _ := newTestContext(req)
		middleware(handler)(ctx)
	}
}
