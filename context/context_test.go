package context_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/arthurlch/goryu/context"
)

func TestContext_Query(t *testing.T) {
	req, _ := http.NewRequest("GET", "/test?name=goryu&version=1", nil)
	rr := httptest.NewRecorder()
	ctx := context.NewContext(rr, req)

	if name := ctx.Query("name"); name != "goryu" {
		t.Errorf("Query(\"name\") got %s, want goryu", name)
	}
	if version := ctx.Query("version"); version != "1" {
		t.Errorf("Query(\"version\") got %s, want 1", version)
	}
	if nonExistent := ctx.Query("nonexistent"); nonExistent != "" {
		t.Errorf("Query(\"nonexistent\") got %s, want \"\"", nonExistent)
	}
}

func TestContext_Form(t *testing.T) {
	form := url.Values{}
	form.Add("username", "tester")
	form.Add("email", "test@example.com")

	req, _ := http.NewRequest("POST", "/submit", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	if err := req.ParseForm(); err != nil {
		t.Fatalf("Failed to parse form: %v", err)
	}
	ctx := context.NewContext(rr, req)

	if username := ctx.Form("username"); username != "tester" {
		t.Errorf("Form(\"username\") got %s, want tester", username)
	}
	if email := ctx.Form("email"); email != "test@example.com" {
		t.Errorf("Form(\"email\") got %s, want test@example.com", email)
	}
	if nonExistent := ctx.Form("nonexistent"); nonExistent != "" {
		t.Errorf("Form(\"nonexistent\") got %s, want \"\"", nonExistent)
	}
}

func TestContext_JSON(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	ctx := context.NewContext(rr, req)

	data := map[string]string{"message": "hello"}
	ctx.JSON(http.StatusOK, data)

	if rr.Code != http.StatusOK {
		t.Errorf("JSON() status code got %d, want %d", rr.Code, http.StatusOK)
	}
	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("JSON() Content-Type got %s, want application/json", contentType)
	}
	expectedBody := "{\"message\":\"hello\"}\n" // json.Encoder adds a newline
	if rr.Body.String() != expectedBody {
		t.Errorf("JSON() body got %s, want %s", rr.Body.String(), expectedBody)
	}
}
