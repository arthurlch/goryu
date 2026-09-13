package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	ctx "github.com/arthurlch/goryu/goryuctx"
)

func mkReq(path string) (req *http.Request) {
	defer func() {
		if recover() != nil {
			req = nil
		}
	}()
	return httptest.NewRequest("GET", "http://x"+path, nil)
}

func buildTrickyRouter(t *testing.T) *Router {
	r := New()
	routes := []string{
		"/users/:id",
		"/users/:id/edit",
		"/users/:id/delete",
		"/users/:id/posts/:pid",
		"/users/new",
		"/files/*path",
		"/a/b/c",
		"/a/:x/c",
		"/a/b/:y",
		"/",
		"/health",
	}
	for _, p := range routes {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					t.Fatalf("panic registering %q: %v", p, rec)
				}
			}()
			r.GET(p, func(c *ctx.Context) { _ = c.Text(200, "ok") })
		}()
	}
	return r
}

func FuzzRouterServe(f *testing.F) {
	seeds := []string{
		"/", "/users/5", "/users/5/edit", "/users/5/delete", "/users/new",
		"/files/a/b/c", "/a/b/c", "/a/x/c", "/a/b/z", "//", "/users//edit",
		"/users/5/", "/users/%2e%2e", "/\x00", "/users/:id", "/files/",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	r := buildTrickyRouter(&testing.T{})
	f.Fuzz(func(t *testing.T, path string) {
		if len(path) == 0 || path[0] != '/' {
			path = "/" + path
		}
		if len(path) > 2000 {
			return
		}
		var req = mkReq(path)
		if req == nil {
			return // malformed target rejected before reaching the router
		}
		panicked := make(chan any, 1)
		done := make(chan struct{})
		go func() {
			defer func() {
				if rec := recover(); rec != nil {
					panicked <- rec
				}
				close(done)
			}()
			r.ServeHTTP(httptest.NewRecorder(), req)
		}()
		select {
		case <-done:
			select {
			case rec := <-panicked:
				t.Fatalf("router panicked on path %q: %v", path, rec)
			default:
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("router hung on path %q", path)
		}
	})
}
