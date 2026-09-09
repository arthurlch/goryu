package goryuctx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	goryuctx "github.com/arthurlch/goryu/goryuctx"
)

func safeReq(method, target, body string) (req *http.Request) {
	defer func() {
		if recover() != nil {
			req = nil
		}
	}()
	return httptest.NewRequest(method, target, strings.NewReader(body))
}

type fuzzTarget struct {
	Name  string `query:"name"`
	Age   int8   `query:"age"`
	Big   int64  `query:"big"`
	Admin bool   `query:"admin"`
}

func FuzzQueryParser(f *testing.F) {
	f.Add("name=a&age=5&admin=true")
	f.Add("age=99999999999999999999")
	f.Add("admin=maybe")
	f.Add("age=-300")
	f.Add("%zz=%")
	f.Fuzz(func(t *testing.T, raw string) {
		req := safeReq("GET", "http://x/?"+strings.ReplaceAll(raw, " ", "%20"), "")
		if req == nil {
			return
		}
		rr := httptest.NewRecorder()
		c := goryuctx.NewContext(rr, req)
		var out fuzzTarget
		_ = c.QueryParser(&out) // must never panic; error is fine
	})
}

func FuzzBindJSON(f *testing.F) {
	f.Add(`{"name":"a","age":5}`)
	f.Add(`{`)
	f.Add(`[1,2,3]`)
	f.Add(`{"age": 1e400}`)
	f.Add("")
	f.Fuzz(func(t *testing.T, body string) {
		req := safeReq("POST", "http://x/", body)
		if req == nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		c := goryuctx.NewContext(rr, req)
		var out fuzzTarget
		_ = c.BindJSON(&out)
	})
}
