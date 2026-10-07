package goryu_test

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/arthurlch/goryu"
)

func TestMountSecurityTxt(t *testing.T) {
	app := goryu.New(goryu.Config{DisableStartupMessage: true})
	app.MountSecurityTxt(goryu.SecurityTxt{
		Contact:            []string{"mailto:security@example.com", "https://example.com/report"},
		Expires:            time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		Policy:             "https://example.com/security-policy",
		PreferredLanguages: "en",
	})

	for _, path := range []string{"/.well-known/security.txt", "/security.txt"} {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 {
			t.Fatalf("%s: expected 200, got %d", path, w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, "Contact: mailto:security@example.com") {
			t.Fatalf("%s: missing contact: %q", path, body)
		}
		if !strings.Contains(body, "Expires: 2030-01-01T00:00:00Z") {
			t.Fatalf("%s: missing/!RFC3339 expires: %q", path, body)
		}
		if !strings.Contains(body, "Policy: https://example.com/security-policy") {
			t.Fatalf("%s: missing policy: %q", path, body)
		}
		if strings.Contains(body, "Hiring:") {
			t.Fatalf("%s: empty field must be omitted: %q", path, body)
		}
	}
}
