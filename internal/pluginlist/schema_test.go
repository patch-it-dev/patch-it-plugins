package pluginlist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// What list-plugins prints validates against docs/plugins.schema.json. The
// checker below knows only the keywords that schema uses (type, required,
// properties, additionalProperties, items, enum, pattern, minLength): the
// module takes no dependency for a JSON Schema library.
func TestTheListMatchesTheSchema(t *testing.T) {
	var schema, doc any
	b, err := os.ReadFile(filepath.Join(root, "docs/plugins.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &schema); err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"main", "v9.9.9"} {
		out, err := Marshal(Build(ref))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(out, &doc); err != nil {
			t.Fatal(err)
		}
		for _, err := range validate(schema.(map[string]any), doc, "$") {
			t.Error(err)
		}
	}
}

// The schema's enums are the values the registry allows, and every kind.
func TestSchemaEnumsMatchTheRegistry(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(root, "docs/plugins.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var s struct {
		Properties struct {
			Categories struct {
				Items struct {
					Enum []string `json:"enum"`
				} `json:"items"`
			} `json:"categories"`
			Providers struct {
				Items struct {
					Properties struct {
						Category struct {
							Enum []string `json:"enum"`
						} `json:"category"`
						Sources struct {
							Items struct {
								Properties map[string]struct {
									Enum []string `json:"enum"`
								} `json:"properties"`
							} `json:"items"`
						} `json:"sources"`
					} `json:"properties"`
				} `json:"items"`
			} `json:"providers"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	var categories, layers []string
	for _, c := range plugin.Categories() {
		categories = append(categories, string(c))
	}
	for _, l := range plugin.Layers() {
		layers = append(layers, string(l))
	}
	src := s.Properties.Providers.Items.Properties.Sources.Items.Properties
	for name, c := range map[string]struct{ got, want []string }{
		"categories":         {s.Properties.Categories.Items.Enum, categories},
		"providers.category": {s.Properties.Providers.Items.Properties.Category.Enum, categories},
		"sources.layer":      {src["layer"].Enum, layers},
	} {
		if !slices.Equal(slices.Sorted(slices.Values(c.got)), slices.Sorted(slices.Values(c.want))) {
			t.Errorf("schema enum %s = %v, want %v", name, c.got, c.want)
		}
	}
	kinds, err := filepath.Glob(filepath.Join(root, "plugins", "*"))
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range kinds {
		if name := filepath.Base(k); name != "all" && !slices.Contains(src["kind"].Enum, name) {
			t.Errorf("kind %s is missing from the schema's kind enum", name)
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
