// Package rbac provides per-route role checks. Populate the caller's roles with
// SetRoles after authentication, then guard routes with Require / RequireAll.
package rbac

import (
	"net/http"

	context "github.com/arthurlch/goryu/goryuctx"
	"github.com/arthurlch/goryu/middleware/base"
)

const RolesKey = "roles"

func SetRoles(c *context.Context, roles ...string) {
	c.Set(RolesKey, roles)
}

func GetRoles(c *context.Context) []string {
	if v, ok := c.Get(RolesKey); ok {
		if roles, ok := v.([]string); ok {
			return roles
		}
	}
	return nil
}

type Config struct {
	base.BaseConfig
	RolesFunc func(c *context.Context) []string
	Forbidden func(c *context.Context)
}

type Enforcer struct {
	roles     func(c *context.Context) []string
	forbidden func(c *context.Context)
}

func New(config ...Config) *Enforcer {
	cfg := Config{}
	if len(config) > 0 {
		cfg = config[0]
	}
	if cfg.RolesFunc == nil {
		cfg.RolesFunc = GetRoles
	}
	if cfg.Forbidden == nil {
		cfg.Forbidden = func(c *context.Context) {
			_ = c.Status(http.StatusForbidden).JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
		}
	}
	return &Enforcer{roles: cfg.RolesFunc, forbidden: cfg.Forbidden}
}

func (e *Enforcer) Require(roles ...string) func(next context.HandlerFunc) context.HandlerFunc {
	return e.guard(roles, false)
}

func (e *Enforcer) RequireAll(roles ...string) func(next context.HandlerFunc) context.HandlerFunc {
	return e.guard(roles, true)
}

func (e *Enforcer) guard(required []string, all bool) func(next context.HandlerFunc) context.HandlerFunc {
	return func(next context.HandlerFunc) context.HandlerFunc {
		return func(c *context.Context) {
			if hasRoles(e.roles(c), required, all) {
				next(c)
				return
			}
			e.forbidden(c)
		}
	}
}

var defaultEnforcer = New()

func Require(roles ...string) func(next context.HandlerFunc) context.HandlerFunc {
	return defaultEnforcer.Require(roles...)
}

func RequireAll(roles ...string) func(next context.HandlerFunc) context.HandlerFunc {
	return defaultEnforcer.RequireAll(roles...)
}

func hasRoles(have, required []string, all bool) bool {
	if len(required) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(have))
	for _, r := range have {
		set[r] = struct{}{}
	}
	for _, r := range required {
		_, ok := set[r]
		if all && !ok {
			return false
		}
		if !all && ok {
			return true
		}
	}
	return all
}
