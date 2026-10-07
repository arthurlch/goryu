package rbac_test

import (
	"net/http/httptest"
	"testing"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/rbac"
)

func ctxWithRoles(roles ...string) (*context.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin", nil)
	c := context.NewContext(w, req)
	if roles != nil {
		rbac.SetRoles(c, roles...)
	}
	return c, w
}

func run(mw func(context.HandlerFunc) context.HandlerFunc, c *context.Context) bool {
	reached := false
	mw(func(*context.Context) { reached = true })(c)
	return reached
}

func TestRequireAny(t *testing.T) {
	c, _ := ctxWithRoles("editor", "admin")
	if !run(rbac.Require("admin"), c) {
		t.Fatal("caller with the role should pass")
	}
	c2, w2 := ctxWithRoles("viewer")
	if run(rbac.Require("admin"), c2) {
		t.Fatal("caller without the role must be blocked")
	}
	if w2.Code != 403 {
		t.Fatalf("expected 403, got %d", w2.Code)
	}
}

func TestRequireAll(t *testing.T) {
	c, _ := ctxWithRoles("admin", "billing")
	if !run(rbac.RequireAll("admin", "billing"), c) {
		t.Fatal("caller with all roles should pass")
	}
	c2, _ := ctxWithRoles("admin")
	if run(rbac.RequireAll("admin", "billing"), c2) {
		t.Fatal("missing one required role must be blocked")
	}
}

func TestNoRolesBlocked(t *testing.T) {
	c, w := ctxWithRoles()
	if run(rbac.Require("admin"), c) {
		t.Fatal("caller with no roles must be blocked")
	}
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestCustomExtractorAndForbidden(t *testing.T) {
	e := rbac.New(rbac.Config{
		RolesFunc: func(c *context.Context) []string { return []string{"root"} },
		Forbidden: func(c *context.Context) { c.Status(418).Text(418, "teapot") },
	})
	c, _ := ctxWithRoles()
	if !run(e.Require("root"), c) {
		t.Fatal("custom extractor should supply the role")
	}
	c2, w2 := ctxWithRoles()
	if run(e.Require("nope"), c2) || w2.Code != 418 {
		t.Fatalf("custom forbidden handler expected 418, got %d", w2.Code)
	}
}
