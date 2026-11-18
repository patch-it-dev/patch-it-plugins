package diff

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// A small spec pair that exercises every classification rule in
// README.md, the $ref rule, and the one-event-per-change rule.
// The expected output below was written from the docs, not copied from a run.
const oldSpec = `{
  "openapi": "3.0.0", "info": {"version": "v1"},
  "paths": {
    "/v1/charges": {"post": {
      "parameters": [{"name": "expand", "in": "query", "schema": {"type": "string"}}],
      "requestBody": {"content": {"application/x-www-form-urlencoded": {"schema": {
        "type": "object",
        "properties": {
          "amount": {"type": "integer"},
          "method": {"type": "string", "enum": ["card", "bank"]}
        },
        "required": ["amount"]
      }}}}
    }},
    "/v1/orders": {"get": {}}
  },
  "components": {"schemas": {
    "charge": {
      "type": "object",
      "properties": {
        "id": {"type": "string"},
        "customer": {"$ref": "#/components/schemas/customer"},
        "source": {"$ref": "#/components/schemas/customer"},
        "status": {"type": "string", "enum": ["paid", "failed"]}
      },
      "required": ["id"]
    },
    "customer": {"type": "object", "properties": {"email": {"type": "string"}}},
    "order": {"type": "object"}
  }}
}`

const newSpec = `{
  "openapi": "3.0.0", "info": {"version": "v2"},
  "paths": {
    "/v1/charges": {"post": {
      "deprecated": true,
      "parameters": [
        {"name": "expand", "in": "query", "schema": {"type": "string"}},
        {"name": "idempotency", "in": "query", "required": true, "schema": {"type": "string"}}
      ],
      "requestBody": {"content": {"application/x-www-form-urlencoded": {"schema": {
        "type": "object",
        "properties": {
          "amount": {"type": "integer"},
          "currency": {"type": "string"},
          "method": {"type": "string", "enum": ["card"]}
        },
        "required": ["amount", "currency"]
      }}}}
    }},
    "/v1/refunds": {"post": {}}
  },
  "components": {"schemas": {
    "charge": {
      "type": "object",
      "properties": {
        "id": {"type": "string", "deprecated": true},
        "customer": {"$ref": "#/components/schemas/customer"},
        "source": {"$ref": "#/components/schemas/card"},
        "status": {"type": "string", "enum": ["paid", "refunded"]},
        "receipt": {"type": "string"}
      },
      "required": ["id", "receipt"]
    },
    "customer": {"type": "object", "properties": {}},
    "card": {"type": "object"}
  }}
}`

func mustParse(t *testing.T, s string) *Spec {
	t.Helper()
	spec, err := Parse([]byte(s))
	if err != nil {
		t.Fatalf("parsing spec: %v", err)
	}
	return spec
}

func describe(c Change) string {
	where := c.Schema
	if c.Operation != "" {
		where = c.Operation
	}
	s := fmt.Sprintf("%s %s %s | %s | %s", c.Side, c.Kind, c.ChangeType, where, c.Property)
	if c.Detail != "" {
		s += " | " + c.Detail
	}
	return strings.TrimSpace(s)
}

func TestChangesClassification(t *testing.T) {
	changes := Changes(mustParse(t, oldSpec), mustParse(t, newSpec))

	want := []string{
		// Request side (paths).
		"request became_required breaking | POST /v1/charges | currency | added as required",
		"request became_required breaking | POST /v1/charges | idempotency | added as required",
		"request enum_value_removed breaking | POST /v1/charges | method | bank",
		"request operation_removed breaking | GET /v1/orders |",
		"request operation_deprecated deprecation | POST /v1/charges |",
		"request operation_added new_feature | POST /v1/refunds |",
		// Response side (components.schemas).
		"response property_added new_feature | charge | receipt",
		"response property_deprecated deprecation | charge | id",
		"response enum_value_added new_feature | charge | status | refunded",
		"response type_changed breaking | charge | source | customer → card",
		"response property_removed breaking | customer | email",
		"response schema_added new_feature | card |",
		"response schema_removed breaking | order |",
	}

	var got []string
	for _, c := range changes {
		got = append(got, describe(c))
	}
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("changes differ from the classification rules\n--- got\n%s\n--- want\n%s",
			strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// Regression: a new required request field used to emit a new_feature
// "property added" next to the breaking event, so the same field looked both
// harmless and harmful.
func TestNewRequiredRequestFieldIsOneBreakingEvent(t *testing.T) {
	for _, c := range Changes(mustParse(t, oldSpec), mustParse(t, newSpec)) {
		if c.Side == "request" && (c.Property == "currency" || c.Property == "idempotency") && c.Kind != BecameRequired {
			t.Errorf("new required request field %q also emitted %s (%s)", c.Property, c.Kind, c.ChangeType)
		}
	}
}

// The $ref rule: a change inside a referenced schema is reported once, where the
// schema is defined, never again under each property that points at it.
func TestRefsAreNeverFollowed(t *testing.T) {
	for _, c := range Changes(mustParse(t, oldSpec), mustParse(t, newSpec)) {
		if c.Schema == "charge" && strings.HasPrefix(c.Property, "customer") {
			t.Errorf("change inside the referenced customer schema leaked into charge: %s", describe(c))
		}
	}
}

// Removals quote the old spec, because the evidence no longer exists in the new one.
func TestRemovalsPointAtTheOldSpec(t *testing.T) {
	for _, c := range Changes(mustParse(t, oldSpec), mustParse(t, newSpec)) {
		removal := c.Kind == OperationRemoved || c.Kind == SchemaRemoved ||
			c.Kind == PropertyRemoved || c.Kind == EnumValueRemoved || c.Kind == MemberRemoved
		if removal != c.InOld {
			t.Errorf("%s: InOld=%v, want %v", describe(c), c.InOld, removal)
		}
		if c.Pointer == "" || c.Node == nil {
			t.Errorf("%s: missing evidence (pointer %q, node %v)", describe(c), c.Pointer, c.Node)
		}
	}
}

func TestEdgeCases(t *testing.T) {
	spec := mustParse(t, newSpec)
	if got := Changes(spec, spec); len(got) != 0 {
		t.Errorf("identical specs: got %d changes, want 0", len(got))
	}
	if got := Changes(nil, spec); len(got) != 0 {
		t.Errorf("first run (no old spec): got %d changes, want 0", len(got))
	}
	if _, err := Parse([]byte("{not json")); err == nil {
		t.Error("malformed spec: want an error, got nil")
	}
	empty := mustParse(t, `{}`)
	_ = Changes(empty, spec) // must not panic on a spec with no paths or components
}

func TestOutputIsDeterministic(t *testing.T) {
	first := Changes(mustParse(t, oldSpec), mustParse(t, newSpec))
	for i := 0; i < 5; i++ {
		again := Changes(mustParse(t, oldSpec), mustParse(t, newSpec))
		for j := range first {
			if describe(first[j]) != describe(again[j]) || first[j].Pointer != again[j].Pointer {
				t.Fatalf("run %d differs at %d: %s vs %s", i, j, describe(first[j]), describe(again[j]))
			}
		}
	}
}
