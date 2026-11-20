package migrate

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
)

// -update rewrites the golden v2 fixture from the v1 ones: go test ./migrate -update
var update = flag.Bool("update", false, "rewrite contract/fixtures/change-events-v2.json")

const fixtures = "../../contract/fixtures/"

// The v1 fixtures the golden v2 file is converted from: the shared contract
// fixture, the hosted service's model retirements and the anthropic SDK release.
var v1Files = []string{
	"change-events.json",
	"change-events-model-retirements-2026-10-13.json",
	"change-events-anthropic-sdk-1.0.0.json",
}

func read(t *testing.T, name string) []event.ChangeEvent {
	t.Helper()
	raw, err := os.ReadFile(fixtures + name)
	if err != nil {
		t.Fatal(err)
	}
	var feed struct {
		Changes []event.ChangeEvent `json:"changes"`
	}
	if err := json.Unmarshal(raw, &feed); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return feed.Changes
}

// The golden file is what the cut-over will store: both loops check it
// (event's v2 tests, patch-it's tests/test_contract_v2.py).
func TestV2FixtureIsTheConversionOfV1(t *testing.T) {
	var out []event.EventV2
	for _, name := range v1Files {
		for _, e := range read(t, name) {
			v2, err := ToV2(e)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if err := v2.Validate(); err != nil {
				t.Fatalf("%s event %d converts to an invalid v2 event: %v", name, e.ID, err)
			}
			out = append(out, v2)
		}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(map[string]any{"changes": out}); err != nil {
		t.Fatal(err)
	}
	golden := fixtures + "change-events-v2.json"
	if *update {
		if err := os.WriteFile(golden, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("%v (run with -update to create it)", err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("%s isn't the conversion of the v1 fixtures: run go test ./migrate -update", golden)
	}
}

// A Stripe event keeps its fingerprint, so the 901 stored events of a release
// aren't re-keyed. Checked on the whole endive release, not only the samples.
func TestAStripeEventKeepsItsFingerprint(t *testing.T) {
	for _, name := range []string{"change-events.json", "change-events-stripe-2026-09-30.json"} {
		for _, e := range read(t, name) {
			v2, err := ToV2(e)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if v2.Fingerprint != e.Fingerprint || v2.Supersedes != "" {
				t.Errorf("%s event %d: fingerprint %s → %s, supersedes %q", name, e.ID, e.Fingerprint, v2.Fingerprint, v2.Supersedes)
			}
			if err := v2.Validate(); err != nil {
				t.Errorf("%s event %d: %v", name, e.ID, err)
			}
		}
	}
}

// A re-keyed event names the fingerprint it replaces: the patch-it client
// finds its open issue or pull request by that one, and opens nothing twice (found in
// review: without it, every repo with an open retirement PR gets a second).
func TestARekeyedEventSupersedesItsV1Fingerprint(t *testing.T) {
	for _, name := range v1Files[1:] {
		for _, e := range read(t, name) {
			v2, err := ToV2(e)
			if err != nil {
				t.Fatal(err)
			}
			if v2.Fingerprint == e.Fingerprint || v2.Supersedes != e.Fingerprint {
				t.Errorf("%s event %d: fingerprint %s, supersedes %q, v1 %s", name, e.ID, v2.Fingerprint, v2.Supersedes, e.Fingerprint)
			}
		}
	}
}

func TestReplacementIsOnlyOnePlainModel(t *testing.T) {
	for hint, want := range map[string]string{
		"Use `gpt-5.6-sol` instead: OpenAI's recommended replacement.":                                                                  "gpt-5.6-sol",
		"Use `gpt-5.6-terra` instead: OpenAI's recommended replacement. (OpenAI first named `gpt-3.5-turbo`, which is itself retired.)": "gpt-5.6-terra",
		"Use `gpt-5.6-sol` (`reasoning.mode: pro`) instead: OpenAI's recommended replacement.":                                          "",
		"OpenAI names no replacement model: The most capable cyber model available to you.":                                             "",
		"Use `a` or `b` instead: OpenAI's recommended replacement.":                                                                     "",
	} {
		if got := replacement(hint); got != want {
			t.Errorf("replacement(%q) = %q, want %q", hint, got, want)
		}
	}
}

func TestAnEventItCantPlaceIsRefused(t *testing.T) {
	e := read(t, "change-events.json")[0]
	e.Affected.Value = "gpt-4"
	e.Affected.Property = "model"
	if _, err := ToV2(e); err == nil || !strings.Contains(err.Error(), "isn't a model retirement") {
		t.Errorf("a value on a Stripe event converted: %v", err)
	}
}
