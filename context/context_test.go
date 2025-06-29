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
}

func TestContext_Form(t *testing.T) {
	form := url.Values{}
	form.Add("username", "tester")
	req, _ := http.NewRequest("POST", "/submit", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	ctx := context.NewContext(rr, req)

	if username := ctx.Form("username"); username != "tester" {
		t.Errorf("Form(\"username\") got %s, want tester", username)
	}
}

func TestContext_JSON(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	ctx := context.NewContext(rr, req)

	data := map[string]string{"message": "hello"}
	if err := ctx.JSON(http.StatusOK, data); err != nil {
		t.Fatalf("JSON write failed: %v", err)
	}

	if rr.Code != http.StatusOK {
		t.Errorf("JSON() status code got %d, want %d", rr.Code, http.StatusOK)
	}
	expectedBody := "{\"message\":\"hello\"}\n"
	if rr.Body.String() != expectedBody {
		t.Errorf("JSON() body got %s, want %s", rr.Body.String(), expectedBody)
	}
}

func TestContext_SetAndGet(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	ctx := context.NewContext(httptest.NewRecorder(), req)

	ctx.Set("testKey", "testValue")
	value, exists := ctx.Get("testKey")

	if !exists {
		t.Error("Expected key 'testKey' to exist, but it didn't")
	}
	if value.(string) != "testValue" {
		t.Errorf("Expected value 'testValue', got '%v'", value)
	}

	_, exists = ctx.Get("nonExistentKey")
	if exists {
		t.Error("Expected key 'nonExistentKey' to not exist, but it did")
	}
}

func TestContext_Status(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	ctx := context.NewContext(rr, req)

	ctx.Status(http.StatusTeapot)
	if rr.Code != http.StatusTeapot {
		t.Errorf("Status() got status %d, want %d", rr.Code, http.StatusTeapot)
	}
}

func TestContext_Data(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	ctx := context.NewContext(rr, req)

	testData := []byte("this is raw data")
	if err := ctx.Data(http.StatusOK, "application/octet-stream", testData); err != nil {
		t.Fatalf("Data write failed: %v", err)
	}

	if rr.Code != http.StatusOK {
		t.Errorf("Data() status code got %d, want %d", rr.Code, http.StatusOK)
	}
	if contentType := rr.Header().Get("Content-Type"); contentType != "application/octet-stream" {
		t.Errorf("Data() Content-Type got %s, want application/octet-stream", contentType)
	}
	if rr.Body.String() != string(testData) {
		t.Errorf("Data() body got %s, want %s", rr.Body.String(), string(testData))
	}
}

func TestContext_Redirect(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	ctx := context.NewContext(rr, req)

	ctx.Redirect(http.StatusFound, "/new-location")

	if rr.Code != http.StatusFound {
		t.Errorf("Redirect() status code got %d, want %d", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/new-location" {
		t.Errorf("Redirect() Location header got %s, want /new-location", location)
	}
}

func TestContext_RemoteIP(t *testing.T) {
	t.Run("No Proxy", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", nil)
		req.RemoteAddr = "123.123.123.123:12345"
		ctx := context.NewContext(nil, req)
		if ctx.RemoteIP() != "123.123.123.123" {
			t.Errorf("Expected IP 123.123.123.123, got %s", ctx.RemoteIP())
		}
	})

	t.Run("X-Real-IP", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("X-Real-IP", "192.168.1.1")
		ctx := context.NewContext(nil, req)
		if ctx.RemoteIP() != "192.168.1.1" {
			t.Errorf("Expected IP 192.168.1.1, got %s", ctx.RemoteIP())
		}
	})

	t.Run("X-Forwarded-For", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")
		ctx := context.NewContext(nil, req)
		if ctx.RemoteIP() != "203.0.113.195" {
			t.Errorf("Expected IP 203.0.113.195, got %s", ctx.RemoteIP())
		}
	})
}
