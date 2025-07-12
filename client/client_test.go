package client

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	m.Run()
}

func TestAgent_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		_, _ = fmt.Fprint(w, "hello world")
	}))
	defer server.Close()

	code, body, errs := Get(server.URL).String()
	if len(errs) > 0 {
		t.Fatalf("Expected no errors, got %v", errs)
	}
	if code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", code)
	}
	if body != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", body)
	}
}

func TestAgent_PostJSON(t *testing.T) {
	type requestData struct {
		Name string `json:"name"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}
		var data requestData
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}
		if data.Name != "goryu" {
			t.Errorf("Expected name 'goryu', got '%s'", data.Name)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, "created")
	}))
	defer server.Close()

	reqData := requestData{Name: "goryu"}
	code, body, errs := Post(server.URL).JSON(reqData).String()

	if len(errs) > 0 {
		t.Fatalf("Expected no errors, got %v", errs)
	}
	if code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", code)
	}
	if body != "created" {
		t.Errorf("Expected 'created', got '%s'", body)
	}
}

func TestAgent_Setters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "goryu-client" {
			t.Errorf("Expected User-Agent 'goryu-client', got '%s'", r.Header.Get("User-Agent"))
		}
		if r.Header.Get("X-Custom-Header") != "custom-value" {
			t.Errorf("Expected X-Custom-Header 'custom-value', got '%s'", r.Header.Get("X-Custom-Header"))
		}

		user, pass, ok := r.BasicAuth()
		if !ok || user != "testuser" || pass != "testpass" {
			t.Errorf("Basic auth failed or incorrect. Got user: %s, pass: %s", user, pass)
		}

		cookie, err := r.Cookie("session")
		if err != nil || cookie.Value != "12345" {
			t.Errorf("Cookie 'session' not found or incorrect. Got: %v", cookie)
		}

		if r.URL.Query().Get("search") != "framework" {
			t.Errorf("Expected query param 'search=framework', got '%s'", r.URL.Query().Get("search"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	agent := Get(server.URL).
		UserAgent("goryu-client").
		Set("X-Custom-Header", "custom-value").
		BasicAuth("testuser", "testpass").
		Cookie("session", "12345").
		Query("search", "framework")

	code, _, errs := agent.Bytes()
	if len(errs) > 0 {
		t.Fatalf("Request failed: %v", errs)
	}
	if code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", code)
	}
}

func TestAgent_Form(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("Server failed to parse form: %v", err)
		}
		if r.FormValue("username") != "goryu_user" {
			t.Errorf("Expected username 'goryu_user', got '%s'", r.FormValue("username"))
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Expected Content-Type for form, got '%s'", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	formData := url.Values{}
	formData.Add("username", "goryu_user")

	code, _, errs := Post(server.URL).Form(formData).Bytes()
	if len(errs) > 0 {
		t.Fatalf("Request failed: %v", errs)
	}
	if code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", code)
	}
}

func TestAgent_StructResponse(t *testing.T) {
	type responseData struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"id": 1, "title": "Test Title"}`)
	}))
	defer server.Close()

	var data responseData
	code, _, errs := Get(server.URL).Struct(&data)

	if len(errs) > 0 {
		t.Fatalf("Request failed: %v", errs)
	}
	if code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", code)
	}
	if data.ID != 1 || data.Title != "Test Title" {
		t.Errorf("Struct not populated correctly. Got: %+v", data)
	}
}

func TestAgent_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Sleep longer than the timeout
		_, _ = fmt.Fprint(w, "should not see this")
	}))
	defer server.Close()

	_, _, errs := Get(server.URL).Timeout(50 * time.Millisecond).Bytes()

	if len(errs) == 0 {
		t.Fatal("Expected a timeout error, but got none")
	}
	// Check if the error is a timeout error
	if err, ok := errs[0].(net.Error); !ok || !err.Timeout() {
		t.Errorf("Expected a net.Error with Timeout, got %v", errs[0])
	}
}

func TestAgent_InvalidURL(t *testing.T) {
	// An invalid URL with a control character.
	_, _, errs := Get("http://\x7f").Bytes()
	if len(errs) == 0 {
		t.Fatal("Expected an error for invalid URL, but got none")
	}
	if !strings.Contains(errs[0].Error(), "invalid control character in URL") {
		t.Errorf("Expected URL control character error, got: %v", errs[0])
	}
}

func TestAgent_InsecureSkipVerify(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "success")
	}))
	defer server.Close()

	_, _, errs := Get(server.URL).Bytes()
	if len(errs) == 0 {
		t.Fatal("Expected TLS verification error, but got none")
	}

	code, body, errs := Get(server.URL).InsecureSkipVerify().Bytes()
	if len(errs) > 0 {
		t.Fatalf("Expected no errors with InsecureSkipVerify, got %v", errs)
	}
	if code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", code)
	}
	if string(body) != "success" {
		t.Errorf("Expected body 'success', got '%s'", string(body))
	}
}
