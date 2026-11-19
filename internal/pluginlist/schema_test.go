package pluginlist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"
)

// docs/plugins.json validates against docs/plugins.schema.json. The checker
// below knows only the keywords that schema uses (type, required,
// properties, additionalProperties, items, enum, pattern, minLength): the
// module takes no dependency for a JSON Schema library.
func TestPluginsJSONMatchesTheSchema(t *testing.T) {
	var schema, doc any
	for file, v := range map[string]*any{"docs/plugins.schema.json": &schema, JSON: &doc} {
		b, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, v); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
	}
	for _, err := range validate(schema.(map[string]any), doc, "$") {
		t.Error(err)
	}
}

// The schema's enums are the values the generator writes.
func TestSchemaEnumsMatchTheGenerator(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(root, "docs/plugins.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var s struct {
		Properties struct {
			Plugins struct {
				Items struct {
					Properties map[string]struct {
						Enum []string `json:"enum"`
					} `json:"properties"`
				} `json:"items"`
			} `json:"plugins"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	props := s.Properties.Plugins.Items.Properties
	var layers []string
	for _, l := range Layers {
		layers = append(layers, l)
	}
	for name, want := range map[string][]string{"layer": layers, "status": Statuses()} {
		got := slices.Sorted(slices.Values(props[name].Enum))
		if !slices.Equal(got, slices.Sorted(slices.Values(want))) {
			t.Errorf("schema enum for %s = %v, want %v", name, got, want)
		}
	}
	formats, err := filepath.Glob(filepath.Join(root, "plugins", "*"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range formats {
		if name := filepath.Base(f); name != "all" && !slices.Contains(props["format"].Enum, name) {
			t.Errorf("format plugin %s is missing from the schema's format enum", name)
		}
	}
}

func TestValidateCatchesABadDocument(t *testing.T) {
	schema := map[string]any{
		"type": "object", "required": []any{"a"}, "additionalProperties": false,
		"properties": map[string]any{"a": map[string]any{"type": "string", "enum": []any{"x"}}},
	}
	for _, doc := range []any{
		map[string]any{},
		map[string]any{"a": "y"},
		map[string]any{"a": "x", "b": 1.0},
		[]any{},
	} {
		if len(validate(schema, doc, "$")) == 0 {
			t.Errorf("%v: valid, want an error", doc)
		}
	}
	if errs := validate(schema, map[string]any{"a": "x"}, "$"); len(errs) != 0 {
		t.Errorf("valid document: %v", errs)
	}
}

func validate(s map[string]any, v any, at string) []error {
	var errs []error
	fail := func(format string, args ...any) { errs = append(errs, fmt.Errorf(at+": "+format, args...)) }
	switch s["type"] {
	case "object":
		obj, ok := v.(map[string]any)
		if !ok {
			fail("want an object")
			return errs
		}
		props, _ := s["properties"].(map[string]any)
		for _, r := range asList(s["required"]) {
			if _, ok := obj[r.(string)]; !ok {
				fail("missing %q", r)
			}
		}
		for k, child := range obj {
			ps, ok := props[k].(map[string]any)
			if !ok {
				if s["additionalProperties"] == false {
					fail("unexpected property %q", k)
				}
				continue
			}
			errs = append(errs, validate(ps, child, at+"."+k)...)
		}
	case "array":
		arr, ok := v.([]any)
		if !ok {
			fail("want an array")
			return errs
		}
		items, _ := s["items"].(map[string]any)
		for i, child := range arr {
			errs = append(errs, validate(items, child, fmt.Sprintf("%s[%d]", at, i))...)
		}
	case "integer":
		if n, ok := v.(float64); !ok || n != float64(int64(n)) {
			fail("want an integer")
		}
	case "string":
		str, ok := v.(string)
		if !ok {
			fail("want a string")
			return errs
		}
		if enum := asList(s["enum"]); enum != nil && !slices.Contains(enum, any(str)) {
			fail("%q is not one of %v", str, enum)
		}
		if p, ok := s["pattern"].(string); ok && !regexp.MustCompile(p).MatchString(str) {
			fail("%q doesn't match %s", str, p)
		}
		if n, ok := s["minLength"].(float64); ok && len(str) < int(n) {
			fail("shorter than %v", n)
		}
	}
	return errs
}

func asList(v any) []any {
	l, _ := v.([]any)
	return l
}
