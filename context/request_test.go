package context

import (
	"bytes"
	"crypto/tls"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func newTestContext(req *http.Request) (*Context, *httptest.ResponseRecorder) {
	rr := httptest.NewRecorder()
	return NewContext(rr, req), rr
}

func TestQuery(t *testing.T) {
	req := httptest.NewRequest("GET", "/?name=goryu&version=1", nil)
	ctx, _ := newTestContext(req)

	if ctx.Query("name") != "goryu" {
		t.Errorf("expected 'goryu', got '%s'", ctx.Query("name"))
	}
	if ctx.Query("version") != "1" {
		t.Errorf("expected '1', got '%s'", ctx.Query("version"))
	}
	if ctx.Query("nonexistent") != "" {
		t.Errorf("expected empty string, got '%s'", ctx.Query("nonexistent"))
	}
}

func TestBindJSON(t *testing.T) {
	type user struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	body := `{"name":"Goryu","age":1}`
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx, _ := newTestContext(req)

	var u user
	if err := ctx.BindJSON(&u); err != nil {
		t.Fatalf("BindJSON failed: %v", err)
	}

	if u.Name != "Goryu" || u.Age != 1 {
		t.Errorf("expected Goryu/1, got %s/%d", u.Name, u.Age)
	}

	req = httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")
	ctx, _ = newTestContext(req)
	if err := ctx.BindJSON(&u); err != http.ErrNotSupported {
		t.Errorf("expected ErrNotSupported, got %v", err)
	}
}

func TestRemoteIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	req.Header.Set("X-Forwarded-For", "1.1.1.1, 2.2.2.2")
	ctx, _ := newTestContext(req)
	if ip := ctx.RemoteIP(); ip != "1.1.1.1" {
		t.Errorf("expected '1.1.1.1', got '%s'", ip)
	}

	req.Header.Del("X-Forwarded-For")
	req.Header.Set("X-Real-IP", "3.3.3.3")
	if ip := ctx.RemoteIP(); ip != "3.3.3.3" {
		t.Errorf("expected '3.3.3.3', got '%s'", ip)
	}

	req.Header.Del("X-Real-IP")
	req.RemoteAddr = "4.4.4.4:12345"
	if ip := ctx.RemoteIP(); ip != "4.4.4.4" {
		t.Errorf("expected '4.4.4.4', got '%s'", ip)
	}
}

func TestBaseURL(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Host = "example.com"
	ctx, _ := newTestContext(req)
	if url := ctx.BaseURL(); url != "http://example.com" {
		t.Errorf("expected 'http://example.com', got '%s'", url)
	}

	req.TLS = &tls.ConnectionState{}
	if url := ctx.BaseURL(); url != "https://example.com" {
		t.Errorf("expected 'https://example.com', got '%s'", url)
	}
}

func TestQueryParser(t *testing.T) {
	type searchParams struct {
		Query   string `query:"q"`
		Page    int    `query:"page"`
		IsAdmin bool   `query:"admin"`
	}

	req := httptest.NewRequest("GET", "/search?q=goryu&page=2&admin=true", nil)
	ctx, _ := newTestContext(req)

	var params searchParams
	if err := ctx.QueryParser(&params); err != nil {
		t.Fatalf("QueryParser failed: %v", err)
	}

	if params.Query != "goryu" {
		t.Errorf("expected query 'goryu', got '%s'", params.Query)
	}
	if params.Page != 2 {
		t.Errorf("expected page 2, got %d", params.Page)
	}
	if !params.IsAdmin {
		t.Errorf("expected admin to be true")
	}
}

func TestIs(t *testing.T) {
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	ctx, _ := newTestContext(req)

	if !ctx.Is("json") {
		t.Error("expected Is('json') to be true")
	}
	if !ctx.Is(".json") {
		t.Error("expected Is('.json') to be true")
	}
	if !ctx.Is("application/json") {
		t.Error("expected Is('application/json') to be true")
	}
	if ctx.Is("xml") {
		t.Error("expected Is('xml') to be false")
	}
}

func TestSaveUploadedFile(t *testing.T) {
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	fw, err := writer.CreateFormFile("upload", "test.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.WriteString(fw, "hello world")
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/upload", &b)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	ctx, _ := newTestContext(req)

	err = req.ParseMultipartForm(32 << 20) // 32 MB
	if err != nil {
		t.Fatal(err)
	}

	file, header, err := ctx.FormFile("upload")
	if err != nil {
		t.Fatalf("FormFile failed: %v", err)
	}
	defer func() { _ = file.Close() }()

	err = ctx.SaveUploadedFile(header, header.Filename)
	if err != nil {
		t.Fatalf("SaveUploadedFile with valid name failed: %v", err)
	}

	filePath := "./uploads/test.txt"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("file was not saved to '%s'", filePath)
	}

	t.Cleanup(func() {
		_ = os.RemoveAll("./uploads")
	})

	err = ctx.SaveUploadedFile(header, "../../../etc/passwd")
	expectedErr := "invalid destination filename: contains path separators"
	if err == nil || err.Error() != expectedErr {
		t.Errorf("expected '%s' error for path traversal, got %v", expectedErr, err)
	}
}
