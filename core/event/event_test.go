package event

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fixtures is the copy of patch-it's contract fixtures (CI_contract keeps it
// in sync).
const fixtures = "../../contract/fixtures/"

var operationAdded = Event{
	ID:              1,
	API:             "stripe/api",
	Version:         "2026-08-26.dahlia",
	PreviousVersion: "2026-07-29.dahlia",
	Change:          ChangeAdded,
	Subject:         Subject{Type: SubjectAPIOperation, Kind: KindOperation, Side: SideRequest, Operation: "POST /v1/billing/feedback_options"},
	Description:     "New operation POST /v1/billing/feedback_options (PostBillingFeedbackOptions).",
	SourceURL:       "https://raw.githubusercontent.com/stripe/openapi/v2468/latest/openapi.spec3.json",
	SourcePointer:   "/paths/~1v1~1billing~1feedback_options/post",
	SourceExcerpt:   `{"operationId":"PostBillingFeedbackOptions","summary":"Create a feedback option"}`,
	Confidence:      1.0,
	DetectedAt:      time.Date(2026, 9, 13, 15, 0, 0, 0, time.UTC),
}

func init() {
	operationAdded.Fingerprint = Fingerprint(operationAdded.API, operationAdded.Version, operationAdded.Change, operationAdded.Subject, "")
}

func rawEvents(t *testing.T, name string) []json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile(fixtures + name)
	if err != nil {
		t.Fatal(err)
	}
	var feed struct {
		Changes []json.RawMessage `json:"changes"`
	}
	if err := json.Unmarshal(raw, &feed); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if len(feed.Changes) == 0 {
		t.Fatalf("%s has no events", name)
	}
	return feed.Changes
}

// Every event in the contract's fixtures, the same ones the patch-it client
// checks, is accepted, with the fingerprint the formula gives (an API event's
// detail isn't stored, so only the other types are recomputed).
func TestFixturesAreValid(t *testing.T) {
	names, err := filepath.Glob(fixtures + "change-events*.json")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, path := range names {
		name := filepath.Base(path)
		if strings.HasSuffix(name, "-invalid.json") {
			continue
		}
		n++
		for _, raw := range rawEvents(t, name) {
			e, err := Decode(raw)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if err := e.Validate(); err != nil {
				t.Errorf("%s event %d: %v", name, e.ID, err)
			}
			if e.Subject.Type == SubjectAPIOperation {
				continue
			}
			if want := Fingerprint(e.API, e.Version, e.Change, e.Subject, ""); e.Fingerprint != want {
				t.Errorf("%s event %d: fingerprint %s, the formula gives %s", name, e.ID, e.Fingerprint, want)
			}
		}
	}
	if n < 10 {
		t.Errorf("read %d fixture files; the contract has more", n)
	}
}

// Every case in the contract's invalid fixture breaks one rule, and is
// rejected while decoding or validating.
func TestInvalidFixtureIsRejected(t *testing.T) {
	raw, err := os.ReadFile(fixtures + "change-events-v2-invalid.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases []struct {
			WhyInvalid string          `json:"why_invalid"`
			Event      json.RawMessage `json:"event"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Cases) == 0 {
		t.Fatal("no cases")
	}
	for _, c := range fixture.Cases {
		e, err := Decode(c.Event)
		if err != nil {
			continue // a field the contract doesn't define: rejected while decoding
		}
		if err := e.Validate(); err == nil {
			t.Errorf("accepted an invalid event: %s", c.WhyInvalid)
		}
	}
}

func TestValidateRejectsContractViolations(t *testing.T) {
	if err := operationAdded.Validate(); err != nil {
		t.Fatalf("the valid event: %v", err)
	}
	cases := map[string]func(e *Event){
		"missing source_url":       func(e *Event) { e.SourceURL = "" },
		"missing source_excerpt":   func(e *Event) { e.SourceExcerpt = "" },
		"missing detected_at":      func(e *Event) { e.DetectedAt = time.Time{} },
		"the old name new_feature": func(e *Event) { e.Change = "new_feature" },
		"no change":                func(e *Event) { e.Change = "" },
		"a package addition that isn't added": func(e *Event) {
			e.Subject = Subject{Type: SubjectPackageRelease, Ecosystem: "pypi", Package: "x", Change: PackageAddedSymbol, Symbols: []string{"a"}}
			e.Change = ChangeBreaking
		},
		"a package removal filed as added": func(e *Event) {
			e.Subject = Subject{Type: SubjectPackageRelease, Ecosystem: "pypi", Package: "x", Change: PackageRemovedSymbol, Symbols: []string{"a"}}
		},
		"added_parameter without the parameter": func(e *Event) {
			e.Subject = Subject{Type: SubjectPackageRelease, Ecosystem: "pypi", Package: "x", Change: PackageAddedParameter, Symbols: []string{"a"}}
		},
		"confidence above 1":                  func(e *Event) { e.Confidence = 1.5 },
		"a bad fingerprint":                   func(e *Event) { e.Fingerprint = "sha256:abc" },
		"operation kind on the response side": func(e *Event) { e.Subject.Side = SideResponse },
		"operation kind without an operation": func(e *Event) { e.Subject.Operation = "" },
		"schema kind on the request side": func(e *Event) {
			e.Subject = Subject{Type: SubjectAPIOperation, Kind: KindSchema, Side: SideRequest, Schema: "invoice"}
		},
		"a model field on an api_operation": func(e *Event) { e.Subject.Name = "gpt-4" },
		"a pair without previous_version":   func(e *Event) { e.PreviousVersion = "" },
		"an old slug as api":                func(e *Event) { e.API = "stripe" },
		"an uppercase id as api":            func(e *Event) { e.API = "Stripe/API" },
	}
	for name, breakIt := range cases {
		e := operationAdded
		breakIt(&e)
		if err := e.Validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// Regression: the excerpt cap is 2,000 characters, not bytes. Spec text is full of
// non-ASCII, and a byte count rejected valid events.
func TestExcerptLimitCountsCharacters(t *testing.T) {
	ok := operationAdded
	ok.SourceExcerpt = strings.Repeat("é", 2000) // 2,000 characters, 4,000 bytes
	if err := ok.Validate(); err != nil {
		t.Errorf("2000-character excerpt rejected: %v", err)
	}
	tooLong := operationAdded
	tooLong.SourceExcerpt = strings.Repeat("é", 2001)
	if err := tooLong.Validate(); err == nil {
		t.Error("2001-character excerpt accepted")
	}
}

// The same release listed in another order is the same event (found in
// review: otherwise the store keeps it twice).
func TestSymbolOrderDoesntChangeTheFingerprint(t *testing.T) {
	s := Subject{Type: SubjectPackageRelease, Ecosystem: "pypi", Package: "anthropic",
		Change: PackageRemovedParameter, Parameter: "top_k", Symbols: []string{"messages.stream", "messages.create"}}
	r := s
	r.Symbols = []string{"messages.create", "messages.stream"}
	if Fingerprint("anthropic/python-sdk", "1.0.0", ChangeBreaking, s, "") != Fingerprint("anthropic/python-sdk", "1.0.0", ChangeBreaking, r, "") {
		t.Error("symbol order changed the fingerprint")
	}
	if s.Symbols[0] != "messages.stream" {
		t.Error("Fingerprint reordered the caller's symbols")
	}
}

// Two kinds of change that share every other field still get two fingerprints.
func TestTheTypeIsPartOfTheFingerprint(t *testing.T) {
	m := Fingerprint("x/y", "1", ChangeBreaking, Subject{Type: SubjectModel, Provider: "p", Name: "n"}, "")
	c := Fingerprint("x/y", "1", ChangeBreaking, Subject{Type: SubjectCIRuntime, Platform: "p", Runtime: "n"}, "")
	if m == c {
		t.Error("a model and a CI runtime with the same values share a fingerprint")
	}
}
