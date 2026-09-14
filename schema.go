package goryu

// JSON Schema generation for typed I/O. This turns a Go type into a JSON Schema
// (draft 2020-12) document, which is what most LLM providers want for structured
// output / function calling, and doubles as machine-readable API documentation.
//
// It is deliberately dependency-free coz why not and (reflection only) and reads two struct
// tags:
//   - `json`      — property name and `omitempty` (affects `required`).
//   - `jsonschema` — comma-separated constraints, e.g.
//       `jsonschema:"description=User email,format=email,required"`
//       `jsonschema:"enum=red|green|blue"` (enum values are `|`-separated)
//       `jsonschema:"minimum=0,maximum=120"`
//       `jsonschema:"minLength=1,maxLength=280"`
//     Bare `required` / `optional` override the default derived from `omitempty`.

import (
	"reflect"
	"strconv"
	"strings"
)

const jsonSchemaDialect = "https://json-schema.org/draft/2020-12/schema"

func Schema[T any]() map[string]any {
	s := SchemaOf(reflect.TypeFor[T]())
	s["$schema"] = jsonSchemaDialect
	return s
}

func SchemaOf(t reflect.Type) map[string]any {
	if t == nil {
		return map[string]any{}
	}
	return buildSchema(t, map[reflect.Type]bool{})
}

func buildSchema(t reflect.Type, seen map[reflect.Type]bool) map[string]any {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	// time.Time is the one std type worth special-casing !
	if t.PkgPath() == "time" && t.Name() == "Time" {
		return map[string]any{"type": "string", "format": "date-time"}
	}

	switch t.Kind() {
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 { // []byte
			return map[string]any{"type": "string", "contentEncoding": "base64"}
		}
		return map[string]any{"type": "array", "items": buildSchema(t.Elem(), seen)}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": buildSchema(t.Elem(), seen)}
	case reflect.Struct:
		if seen[t] {
			return map[string]any{"type": "object"}
		}
		seen[t] = true
		defer delete(seen, t)
		return structSchema(t, seen)
	default:
		// all other non supported types
		return map[string]any{}
	}
}

func structSchema(t reflect.Type, seen map[reflect.Type]bool) map[string]any {
	props := map[string]any{}
	var required []string

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue
		}
		name, opts := parseJSONTag(f)
		if name == "-" {
			continue
		}

		if f.Anonymous && name == "" {
			sub := buildSchema(f.Type, seen)
			if p, ok := sub["properties"].(map[string]any); ok {
				for k, v := range p {
					props[k] = v
				}
			}
			if r, ok := sub["required"].([]string); ok {
				required = append(required, r...)
			}
			continue
		}
		if name == "" {
			name = f.Name
		}

		fieldSchema := buildSchema(f.Type, seen)
		applyConstraints(fieldSchema, f.Tag.Get("jsonschema"))
		props[name] = fieldSchema

		if fieldRequired(f, opts) {
			required = append(required, name)
		}
	}

	schema := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func parseJSONTag(f reflect.StructField) (name string, opts map[string]bool) {
	opts = map[string]bool{}
	tag := f.Tag.Get("json")
	if tag == "" {
		return "", opts
	}
	parts := strings.Split(tag, ",")
	name = parts[0]
	for _, o := range parts[1:] {
		opts[o] = true
	}
	return name, opts
}

func fieldRequired(f reflect.StructField, opts map[string]bool) bool {
	for _, tok := range strings.Split(f.Tag.Get("jsonschema"), ",") {
		switch strings.TrimSpace(tok) {
		case "required":
			return true
		case "optional":
			return false
		}
	}
	if opts["omitempty"] {
		return false
	}
	return f.Type.Kind() != reflect.Pointer
}

func applyConstraints(schema map[string]any, tag string) {
	if tag == "" {
		return
	}
	for _, tok := range strings.Split(tag, ",") {
		k, v, ok := strings.Cut(tok, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch k {
		case "description", "format", "pattern", "title", "default":
			schema[k] = v
		case "enum":
			schema["enum"] = strings.Split(v, "|")
		case "minimum", "maximum", "multipleOf":
			if n, err := strconv.ParseFloat(v, 64); err == nil {
				schema[k] = n
			}
		case "minLength", "maxLength", "minItems", "maxItems":
			if n, err := strconv.Atoi(v); err == nil {
				schema[k] = n
			}
		}
	}
}
