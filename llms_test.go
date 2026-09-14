package goryu_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arthurlch/goryu"
)

func llmsApp() *goryu.App {
	app := goryu.New(goryu.Config{AppName: "Test API", DisableStartupMessage: true})
	app.GET("/users", func(c *goryu.Ctx) { _ = c.Text(200, "ok") })
	app.POST("/users", func(c *goryu.Ctx) { _ = c.Text(200, "ok") })
	return app
}

func TestAPIReference(t *testing.T) {
	ref := llmsApp().APIReference()
	if ref.Name != "Test API" {
		t.Fatalf("name wrong: %q", ref.Name)
	}
	if len(ref.Routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(ref.Routes))
	}
	if !strings.Contains(ref.Summary, "2 endpoints") {
		t.Fatalf("summary wrong: %q", ref.Summary)
	}
}

func TestLLMsText(t *testing.T) {
	txt := llmsApp().LLMsText()
	if !strings.HasPrefix(txt, "# Test API") {
		t.Fatalf("missing title: %q", txt)
	}
	if !strings.Contains(txt, "## Endpoints") {
		t.Fatalf("missing endpoints section: %q", txt)
	}
	if !strings.Contains(txt, "`GET /users`") || !strings.Contains(txt, "`POST /users`") {
		t.Fatalf("missing routes: %q", txt)
	}
}

func TestMountLLMs(t *testing.T) {
	app := llmsApp()
	app.MountLLMs()

	req := httptest.NewRequest("GET", "/llms.txt", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "# Test API") {
		t.Fatalf("/llms.txt failed: %d %q", w.Code, w.Body.String())
	}

	req = httptest.NewRequest("GET", "/llms.json", nil)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"name":"Test API"`) {
		t.Fatalf("/llms.json failed: %d %q", w.Code, w.Body.String())
	}
}
