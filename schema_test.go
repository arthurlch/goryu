package goryu_test

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/arthurlch/goryu"
)

type schemaUser struct {
	Email    string    `json:"email" jsonschema:"description=User email,format=email"`
	Age      int       `json:"age" jsonschema:"minimum=0,maximum=120"`
	Role     string    `json:"role" jsonschema:"enum=admin|user|guest"`
	Tags     []string  `json:"tags,omitempty"`
	Nickname *string   `json:"nickname"`
	Created  time.Time `json:"created"`
	secret   string    //nolint:unused // exercises unexported skip
}

func TestSchemaTopLevel(t *testing.T) {
	s := goryu.Schema[schemaUser]()
	if s["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
		t.Fatalf("missing dialect: %v", s["$schema"])
	}
	if s["type"] != "object" {
		t.Fatalf("expected object, got %v", s["type"])
	}
}

func TestSchemaProperties(t *testing.T) {
	s := goryu.Schema[schemaUser]()
	props, ok := s["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties missing")
	}

	if _, ok := props["secret"]; ok {
		t.Fatal("unexported field must be skipped")
	}

	email := props["email"].(map[string]any)
	if email["type"] != "string" || email["format"] != "email" {
		t.Fatalf("email schema wrong: %v", email)
	}
	if email["description"] != "User email" {
		t.Fatalf("description missing: %v", email)
	}

	age := props["age"].(map[string]any)
	if age["type"] != "integer" || age["minimum"] != float64(0) || age["maximum"] != float64(120) {
		t.Fatalf("age constraints wrong: %v", age)
	}

	role := props["role"].(map[string]any)
	enum, ok := role["enum"].([]string)
	if !ok || len(enum) != 3 || enum[0] != "admin" {
		t.Fatalf("enum wrong: %v", role["enum"])
	}

	created := props["created"].(map[string]any)
	if created["type"] != "string" || created["format"] != "date-time" {
		t.Fatalf("time.Time schema wrong: %v", created)
	}

	tags := props["tags"].(map[string]any)
	if tags["type"] != "array" {
		t.Fatalf("tags should be array: %v", tags)
	}
}

func TestSchemaRequired(t *testing.T) {
	s := goryu.Schema[schemaUser]()
	req, _ := s["required"].([]string)
	has := func(name string) bool {
		return slices.Contains(req, name)
	}
	if !has("email") || !has("age") {
		t.Fatalf("email/age should be required: %v", req)
	}
	if has("tags") {
		t.Fatal("omitempty field must not be required")
	}
	if has("nickname") {
		t.Fatal("pointer field must not be required")
	}
}

func TestSchemaOfPrimitiveAndSlice(t *testing.T) {
	if got := goryu.SchemaOf(reflect.TypeFor[int]()); got["type"] != "integer" {
		t.Fatalf("int schema wrong: %v", got)
	}
	arr := goryu.SchemaOf(reflect.TypeFor[[]schemaUser]())
	if arr["type"] != "array" {
		t.Fatalf("expected array: %v", arr)
	}
	if _, ok := arr["items"].(map[string]any); !ok {
		t.Fatalf("array items missing: %v", arr)
	}
}
