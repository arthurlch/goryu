package goryu

import (
	"strings"
	"time"
)

// SecurityTxt is a security disclosure policy served per RFC 9116.
type SecurityTxt struct {
	Contact            []string
	Expires            time.Time
	Encryption         []string
	Acknowledgments    string
	Policy             string
	Hiring             string
	PreferredLanguages string
	CanonicalURL       string
}

func (s SecurityTxt) render() string {
	var b strings.Builder
	for _, contact := range s.Contact {
		writeField(&b, "Contact", contact)
	}
	if !s.Expires.IsZero() {
		writeField(&b, "Expires", s.Expires.UTC().Format(time.RFC3339))
	}
	for _, enc := range s.Encryption {
		writeField(&b, "Encryption", enc)
	}
	writeField(&b, "Acknowledgments", s.Acknowledgments)
	writeField(&b, "Policy", s.Policy)
	writeField(&b, "Hiring", s.Hiring)
	writeField(&b, "Preferred-Languages", s.PreferredLanguages)
	writeField(&b, "Canonical", s.CanonicalURL)
	return b.String()
}

func writeField(b *strings.Builder, name, value string) {
	if value == "" {
		return
	}
	b.WriteString(name)
	b.WriteString(": ")
	b.WriteString(value)
	b.WriteByte('\n')
}

// MountSecurityTxt serves the policy at /.well-known/security.txt and /security.txt.
func (app *App) MountSecurityTxt(s SecurityTxt) {
	body := s.render()
	handler := func(c *Ctx) {
		_ = c.Data(200, "text/plain; charset=utf-8", []byte(body))
	}
	app.GET("/.well-known/security.txt", handler)
	app.GET("/security.txt", handler)
}
