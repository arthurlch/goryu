package context

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func newTestContextForResponse() (*Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	return NewContext(rr, req), rr
}

func TestJSON(t *testing.T) {
	ctx, rr := newTestContextForResponse()

	type user struct {
		Name string `json:"name"`
	}

	data := user{Name: "Goryu"}
	err := ctx.JSON(http.StatusOK, data)
	if err != nil {
		t.Fatalf("JSON failed: %v", err)
	}

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
	}

	var u user
	if err := json.NewDecoder(rr.Body).Decode(&u); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if u.Name != "Goryu" {
		t.Errorf("expected body to contain user 'Goryu', got '%s'", u.Name)
	}
}

func TestText(t *testing.T) {
	ctx, rr := newTestContextForResponse()
	err := ctx.Text(http.StatusOK, "Hello, World!")
	if err != nil {
		t.Fatalf("Text failed: %v", err)
	}

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if contentType := rr.Header().Get("Content-Type"); contentType != "text/plain" {
		t.Errorf("expected Content-Type 'text/plain', got '%s'", contentType)
	}
	if body := rr.Body.String(); body != "Hello, World!" {
		t.Errorf("expected body 'Hello, World!', got '%s'", body)
	}
}

func TestStatus(t *testing.T) {
	ctx, rr := newTestContextForResponse()

	if err := ctx.Status(http.StatusTeapot).Text(http.StatusTeapot, "I'm a teapot"); err != nil {
		t.Fatalf("Text failed: %v", err)
	}

	if rr.Code != http.StatusTeapot {
		t.Errorf("expected status %d, got %d", http.StatusTeapot, rr.Code)
	}
}
func TestRedirect(t *testing.T) {
	ctx, rr := newTestContextForResponse()
	ctx.Redirect(http.StatusFound, "/new-location")

	if rr.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, rr.Code)
	}
	if location := rr.Header().Get("Location"); location != "/new-location" {
		t.Errorf("expected Location '/new-location', got '%s'", location)
	}
}

func TestClearCookie(t *testing.T) {
	ctx, rr := newTestContextForResponse()
	ctx.ClearCookie("session")

	cookieHeader := rr.Header().Get("Set-Cookie")
	if !strings.Contains(cookieHeader, "session=;") {
		t.Errorf("expected cookie to be cleared, got header: %s", cookieHeader)
	}
	if !strings.Contains(cookieHeader, "Max-Age=0") {
		t.Errorf("expected Max-Age=0, got header: %s", cookieHeader)
	}
}

func TestAttachment(t *testing.T) {
	// Test without filename
	ctx, rr := newTestContextForResponse()
	ctx.Attachment()
	if disposition := rr.Header().Get("Content-Disposition"); disposition != "attachment" {
		t.Errorf("expected 'attachment', got '%s'", disposition)
	}

	ctx, rr = newTestContextForResponse()
	ctx.Attachment("report.pdf")
	expected := `attachment; filename="report.pdf"`
	if disposition := rr.Header().Get("Content-Disposition"); disposition != expected {
		t.Errorf("expected '%s', got '%s'", expected, disposition)
	}
}

func TestSendFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-send-file-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }() // Clean up

	content := "hello from a file"
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	ctx, rr := newTestContextForResponse()
	ctx.SendFile(tmpFile.Name())

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if contentType := rr.Header().Get("Content-Type"); contentType != "text/plain; charset=utf-8" {
		t.Errorf("expected content type 'text/plain; charset=utf-8', got '%s'", contentType)
	}
	body, _ := io.ReadAll(rr.Body)
	if string(body) != content {
		t.Errorf("expected body '%s', got '%s'", content, string(body))
	}
}

func TestType(t *testing.T) {
	ctx, rr := newTestContextForResponse()
	ctx.Type("html")

	if contentType := rr.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Errorf("expected 'text/html; charset=utf-8', got '%s'", contentType)
	}
}

func TestAppend(t *testing.T) {
	ctx, rr := newTestContextForResponse()
	ctx.Append("Link", "<http://localhost/>; rel=\"home\"")
	ctx.Append("Link", "<http://localhost/about>; rel=\"about\"")

	headers := rr.Header().Values("Link")
	if len(headers) != 2 {
		t.Fatalf("expected 2 Link headers, got %d", len(headers))
	}
	if headers[0] != "<http://localhost/>; rel=\"home\"" {
		t.Errorf("unexpected first header value: %s", headers[0])
	}
	if headers[1] != "<http://localhost/about>; rel=\"about\"" {
		t.Errorf("unexpected second header value: %s", headers[1])
	}
}
