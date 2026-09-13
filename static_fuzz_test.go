package goryu

import (
	"path/filepath"
	"strings"
	"testing"
)

func FuzzSanitizeStaticPath(f *testing.F) {
	root := f.TempDir()
	rootAbs, _ := filepath.Abs(root)

	seeds := []string{
		"/index.html", "/../secret", "/..%2fsecret", "/a/b/../../etc/passwd",
		"/%2e%2e/%2e%2e/x", "/a/./b", "//evil", "/.git/config", "/\x00",
		"/" + strings.Repeat("../", 40) + "etc/passwd",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, reqPath string) {
		got, err := sanitizeStaticPath(root, reqPath)
		if err != nil {
			return
		}
		gotAbs, aerr := filepath.Abs(got)
		if aerr != nil {
			t.Fatalf("returned unresolvable path %q", got)
		}
		if gotAbs != rootAbs && !strings.HasPrefix(gotAbs, rootAbs+string(filepath.Separator)) {
			t.Fatalf("path %q escaped root: got %q (root %q)", reqPath, gotAbs, rootAbs)
		}
	})
}
