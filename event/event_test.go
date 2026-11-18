package event

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

var fixtureOperationAdded = ChangeEvent{
	ID:              1,
	Fingerprint:     "sha256:3b1f000000000000000000000000000000000000000000000000000000000000",
	API:             "stripe",
	Version:         "2026-08-26.dahlia",
	PreviousVersion: "2026-07-29.dahlia",
	ChangeType:      ChangeTypeNewFeature,
	Breaking:        false,
	Affected: Affected{
		Kind:      AffectedKindOperation,
		Side:      SideRequest,
		Operation: "POST /v1/billing/feedback_options",
	},
	Description:   "New operation POST /v1/billing/feedback_options (PostBillingFeedbackOptions).",
	SourceURL:     "https://raw.githubusercontent.com/stripe/openapi/v2468/latest/openapi.spec3.json",
	SourcePointer: "/paths/~1v1~1billing~1feedback_options/post",
	SourceExcerpt: `{"operationId":"PostBillingFeedbackOptions","summary":"Create a feedback option"}`,
	Confidence:    1.0,
	DetectedAt:    time.Date(2026, 9, 13, 15, 0, 0, 0, time.UTC),
}

var fixtureEnumValueAdded = ChangeEvent{
	ID:              2,
	Fingerprint:     "sha256:9f2c000000000000000000000000000000000000000000000000000000000000",
	API:             "stripe",
	Version:         "2026-08-26.dahlia",
	PreviousVersion: "2026-07-29.dahlia",
	ChangeType:      ChangeTypeNewFeature,
	Breaking:        false,
	Affected: Affected{
		Kind:     AffectedKindSchemaProperty,
		Side:     SideResponse,
		Schema:   "invoices_payment_settings",
		Property: "payment_method_types[]",
	},
	Description:   "Enum value 'billie' added to invoices_payment_settings.payment_method_types[].",
	SourceURL:     "https://raw.githubusercontent.com/stripe/openapi/v2468/latest/openapi.spec3.json",
	SourcePointer: "/components/schemas/invoices_payment_settings/properties/payment_method_types/items/enum",
	SourceExcerpt: `["ach_credit_transfer","ach_debit","billie"]`,
	Confidence:    1.0,
	DetectedAt:    time.Date(2026, 9, 13, 15, 0, 0, 0, time.UTC),
}

var fixturePropertyRemoved = ChangeEvent{
	ID:              3,
	Fingerprint:     "sha256:c07a000000000000000000000000000000000000000000000000000000000000",
	API:             "stripe",
	Version:         "2026-08-26.dahlia",
	PreviousVersion: "2026-07-29.dahlia",
	ChangeType:      ChangeTypeBreaking,
	Breaking:        true,
	Affected: Affected{
		Kind:     AffectedKindSchemaProperty,
		Side:     SideResponse,
		Schema:   "payments_primitives_payment_records_resource_payment_method_card_details_resource_three_d_secure",
		Property: "cryptogram",
	},
	Description:   "Property cryptogram removed from payments_primitives_payment_records_resource_payment_method_card_details_resource_three_d_secure.",
	SourceURL:     "https://raw.githubusercontent.com/stripe/openapi/v2460/latest/openapi.spec3.json",
	SourcePointer: "/components/schemas/payments_primitives_payment_records_resource_payment_method_card_details_resource_three_d_secure/properties/cryptogram",
	SourceExcerpt: `{"description":"The 3D Secure cryptogram.","maxLength":5000,"nullable":true,"type":"string"}`,
	Confidence:    1.0,
	DetectedAt:    time.Date(2026, 9, 13, 15, 0, 0, 0, time.UTC),
}

var fixtureDeprecation = ChangeEvent{
	ID:              4,
	Fingerprint:     "sha256:a1e5000000000000000000000000000000000000000000000000000000000000",
	API:             "stripe",
	Version:         "2026-08-26.dahlia",
	PreviousVersion: "2026-07-29.dahlia",
	ChangeType:      ChangeTypeDeprecation,
	Breaking:        false,
	Affected: Affected{
		Kind:      AffectedKindOperation,
		Side:      SideRequest,
		Operation: "POST /v1/charges",
	},
	Description:   "Operation POST /v1/charges marked deprecated in favor of POST /v1/payment_intents.",
	MigrationHint: "Use POST /v1/payment_intents instead.",
	SourceURL:     "https://raw.githubusercontent.com/stripe/openapi/v2468/latest/openapi.spec3.json",
	SourcePointer: "/paths/~1v1~1charges/post/deprecated",
	SourceExcerpt: `{"deprecated":true}`,
	Confidence:    1.0,
	DetectedAt:    time.Date(2026, 9, 13, 15, 0, 0, 0, time.UTC),
}

func TestFixturesAreValid(t *testing.T) {
	fixtures := []ChangeEvent{
		fixtureOperationAdded,
		fixtureEnumValueAdded,
		fixturePropertyRemoved,
		fixtureDeprecation,
	}
	for i, e := range fixtures {
		if err := e.Validate(); err != nil {
			t.Errorf("fixture %d failed validation: %v", i, err)
		}
	}
}

func TestValidateRejectsMissingProvenance(t *testing.T) {
	bad := fixtureOperationAdded
	bad.SourceURL = ""
	if err := bad.Validate(); err == nil {
		t.Error("expected validation error for missing source_url, got nil")
	}
}

// Regression: the excerpt cap is 2,000 characters, not bytes. Spec text is full of
// non-ASCII, and a byte count rejected valid events.
func TestExcerptLimitCountsCharacters(t *testing.T) {
	ok := fixtureOperationAdded
	ok.SourceExcerpt = strings.Repeat("é", 2000) // 2,000 characters, 4,000 bytes
	if err := ok.Validate(); err != nil {
		t.Errorf("2000-character excerpt rejected: %v", err)
	}
	tooLong := fixtureOperationAdded
	tooLong.SourceExcerpt = strings.Repeat("é", 2001)
	if err := tooLong.Validate(); err == nil {
		t.Error("2001-character excerpt accepted")
	}
}

func TestValidateRejectsContractViolations(t *testing.T) {
	cases := map[string]func(e *ChangeEvent){
		"missing source_excerpt":              func(e *ChangeEvent) { e.SourceExcerpt = "" },
		"missing detected_at":                 func(e *ChangeEvent) { e.DetectedAt = time.Time{} },
		"breaking disagrees with change_type": func(e *ChangeEvent) { e.Breaking = true },
		"unknown change_type":                 func(e *ChangeEvent) { e.ChangeType = "maybe" },
		"confidence above 1":                  func(e *ChangeEvent) { e.Confidence = 1.5 },
		"operation kind on the response side": func(e *ChangeEvent) { e.Affected.Side = SideResponse },
		"operation kind without an operation": func(e *ChangeEvent) { e.Affected.Operation = "" },
	}
	for name, breakIt := range cases {
		e := fixtureOperationAdded
		breakIt(&e)
		if err := e.Validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}

	schemaOnRequest := fixtureEnumValueAdded
	schemaOnRequest.Affected.Side = SideRequest
	if err := schemaOnRequest.Validate(); err == nil {
		t.Error("schema_property kind on the request side: accepted")
	}
}

func TestAValueNeedsTheProperty(t *testing.T) {
	e := fixtureOperationAdded
	e.Affected = Affected{Kind: AffectedKindOperation, Side: SideRequest, Operation: "POST /v1/chat/completions", Value: "gpt-4"}
	if err := e.Validate(); err == nil {
		t.Error("a value with no property it belongs to: accepted")
	}
	e.Affected = Affected{Kind: AffectedKindRequestParameter, Side: SideRequest, Operation: "POST /v1/chat/completions", Property: "model", Value: "gpt-4"}
	if err := e.Validate(); err != nil {
		t.Errorf("a model retirement: %v", err)
	}
}

// Contract test between ingest and the patch-it client.
// contract/fixtures/change-events.json is the file the client builds against
// in place of the real feed. The client's tests check the same file against its
// own model, so if either side's idea of a change event drifts from the
// contract, one of the two tests fails. Reading a shared fixture is not an
// import: the two still share only the contract.
func TestSharedFixtureMatchesTheContract(t *testing.T) {
	raw, err := os.ReadFile("../contract/fixtures/change-events.json")
	if err != nil {
		t.Fatalf("reading the shared fixture: %v", err)
	}
	var feed struct {
		Changes []ChangeEvent `json:"changes"`
	}
	if err := json.Unmarshal(raw, &feed); err != nil {
		t.Fatalf("the fixture doesn't decode as change events: %v", err)
	}
	if len(feed.Changes) == 0 {
		t.Fatal("the fixture has no events")
	}
	for _, e := range feed.Changes {
		if err := e.Validate(); err != nil {
			t.Errorf("fixture event %d fails ingest's validation: %v", e.ID, err)
		}
	}
}

// The model retirements fixture is ingest's real output, the first to
// carry affected.value. Ingest and the client both check it.
func TestRetirementFixtureMatchesTheContract(t *testing.T) {
	raw, err := os.ReadFile("../contract/fixtures/change-events-model-retirements-2026-10-13.json")
	if err != nil {
		t.Fatalf("reading the fixture: %v", err)
	}
	var feed struct {
		Changes []ChangeEvent `json:"changes"`
	}
	if err := json.Unmarshal(raw, &feed); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if len(feed.Changes) == 0 {
		t.Fatal("no events")
	}
	for _, e := range feed.Changes {
		if err := e.Validate(); err != nil || e.Affected.Value == "" {
			t.Errorf("event %d: %v (value %q)", e.ID, err, e.Affected.Value)
		}
	}
}

// The negative half of the contract test: every event in the shared invalid
// fixture breaks one rule, and both loops must reject all of them.
func TestSharedInvalidFixtureIsRejected(t *testing.T) {
	raw, err := os.ReadFile("../contract/fixtures/change-events-invalid.json")
	if err != nil {
		t.Fatalf("reading the shared invalid fixture: %v", err)
	}
	var fixture struct {
		Cases []struct {
			WhyInvalid string          `json:"why_invalid"`
			Event      json.RawMessage `json:"event"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decoding the invalid fixture: %v", err)
	}
	for _, c := range fixture.Cases {
		var e ChangeEvent
		if err := json.Unmarshal(c.Event, &e); err != nil {
			continue // can't even decode, e.g. a malformed timestamp: rejected
		}
		if err := e.Validate(); err == nil {
			t.Errorf("accepted an invalid event: %s", c.WhyInvalid)
		}
	}
}
