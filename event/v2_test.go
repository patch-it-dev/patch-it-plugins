package event

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"testing"
)

// -update writes the computed fingerprints into the hand-written v2 fixture:
// go test ./internal/event -update
var update = flag.Bool("update", false, "rewrite fingerprints in change-events-v2-types.json")

const v2Fixtures = "../contract/fixtures/"

func rawEvents(t *testing.T, name string) []json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile(v2Fixtures + name)
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

// Contract test for v2, the same fixtures the patch-it client checks
// (patch-it's tests/test_contract_v2.py): the converted events and one example per
// type are accepted, with fingerprints the formula gives.
func TestV2FixturesAreValid(t *testing.T) {
	for _, name := range []string{"change-events-v2.json", "change-events-v2-types.json"} {
		for _, raw := range rawEvents(t, name) {
			e, err := DecodeV2(raw)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if err := e.Validate(); err != nil {
				t.Errorf("%s event %d: %v", name, e.ID, err)
			}
			// An API event's detail isn't stored, so only the other types are
			// recomputed here; TestAPIOperationKeepsV1Fingerprint covers it.
			if e.Subject.Type == SubjectAPIOperation && name == "change-events-v2.json" {
				continue
			}
			if want := FingerprintV2(e.API, e.Version, e.ChangeType, e.Subject, ""); e.Fingerprint != want && !*update {
				t.Errorf("%s event %d: fingerprint %s, the formula gives %s", name, e.ID, e.Fingerprint, want)
			}
		}
	}
}

func TestUpdateV2TypeFingerprints(t *testing.T) {
	if !*update {
		t.Skip("only with -update")
	}
	var out []EventV2
	for _, raw := range rawEvents(t, "change-events-v2-types.json") {
		e, err := DecodeV2(raw)
		if err != nil {
			t.Fatal(err)
		}
		e.Fingerprint = FingerprintV2(e.API, e.Version, e.ChangeType, e.Subject, "")
		out = append(out, e)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(map[string]any{"changes": out}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(v2Fixtures+"change-events-v2-types.json", buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Every case breaks one rule, and both loops must reject all of them.
func TestV2InvalidFixtureIsRejected(t *testing.T) {
	raw, err := os.ReadFile(v2Fixtures + "change-events-v2-invalid.json")
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
		e, err := DecodeV2(c.Event)
		if err != nil {
			continue // a field v2 doesn't define: rejected while decoding
		}
		if err := e.Validate(); err == nil {
			t.Errorf("accepted an invalid v2 event: %s", c.WhyInvalid)
		}
	}
}

// Stripe's stored events keep their fingerprints: v2's api_operation formula
// is v1's, byte for byte.
func TestAPIOperationKeepsV1Fingerprint(t *testing.T) {
	a := Affected{Kind: AffectedKindSchemaProperty, Side: SideResponse, Schema: "invoice", Property: "subscription"}
	s := Subject{Type: SubjectAPIOperation, Kind: a.Kind, Side: a.Side, Schema: a.Schema, Property: a.Property}
	v1 := Fingerprint("stripe", "2026-09-30.endive", ChangeTypeBreaking, a, "removed")
	if v2 := FingerprintV2("stripe", "2026-09-30.endive", ChangeTypeBreaking, s, "removed"); v2 != v1 {
		t.Errorf("v2 %s, v1 %s", v2, v1)
	}
}

// The same release listed in another order is the same event (the PR #124
// review: otherwise the store keeps it twice).
func TestSymbolOrderDoesntChangeTheFingerprint(t *testing.T) {
	s := Subject{Type: SubjectPackageRelease, Ecosystem: "pypi", Package: "anthropic",
		Change: PackageRemovedParameter, Parameter: "top_k", Symbols: []string{"messages.stream", "messages.create"}}
	r := s
	r.Symbols = []string{"messages.create", "messages.stream"}
	if FingerprintV2("anthropic", "1.0.0", ChangeTypeBreaking, s, "") != FingerprintV2("anthropic", "1.0.0", ChangeTypeBreaking, r, "") {
		t.Error("symbol order changed the fingerprint")
	}
	if s.Symbols[0] != "messages.stream" {
		t.Error("FingerprintV2 reordered the caller's symbols")
	}
}

// Two kinds of change that share every other field still get two fingerprints.
func TestTheTypeIsPartOfTheFingerprint(t *testing.T) {
	m := FingerprintV2("x", "1", ChangeTypeBreaking, Subject{Type: SubjectModel, Provider: "p", Name: "n"}, "")
	c := FingerprintV2("x", "1", ChangeTypeBreaking, Subject{Type: SubjectCIRuntime, Platform: "p", Runtime: "n"}, "")
	if m == c {
		t.Error("a model and a CI runtime with the same values share a fingerprint")
	}
}
