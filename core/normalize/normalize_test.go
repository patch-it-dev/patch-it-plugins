package normalize

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/patch-it-dev/patch-it-plugins/core/diff"
	"github.com/patch-it-dev/patch-it-plugins/core/event"
)

const (
	oldURL = "https://raw.githubusercontent.com/stripe/openapi/v207/openapi/spec3.json"
	newURL = "https://raw.githubusercontent.com/stripe/openapi/v208/openapi/spec3.json"
)

var source = Source{
	API:             "stripe",
	Version:         "2022-11-15",
	PreviousVersion: "2022-08-01",
	NewSpecURL:      newURL,
	OldSpecURL:      oldURL,
	DetectedAt:      time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC),
}

// The demo change itself, as the differ reports it.
func chargesRemoved() diff.Change {
	return diff.Change{
		Kind:       diff.PropertyRemoved,
		ChangeType: event.ChangeTypeBreaking,
		Breaking:   true,
		Side:       event.SideResponse,
		Schema:     "payment_intent",
		Property:   "charges",
		Pointer:    "/components/schemas/payment_intent/properties/charges",
		InOld:      true,
		Node:       map[string]any{"description": "Charges that were created by this PaymentIntent, if any."},
	}
}

func latestChargeAdded() diff.Change {
	return diff.Change{
		Kind:       diff.PropertyAdded,
		ChangeType: event.ChangeTypeNewFeature,
		Side:       event.SideResponse,
		Schema:     "payment_intent",
		Property:   "latest_charge",
		Pointer:    "/components/schemas/payment_intent/properties/latest_charge",
		Node:       map[string]any{"description": "The latest charge created by this payment intent."},
	}
}

func TestEventsAreValidWithFullProvenance(t *testing.T) {
	events, errs := Events([]diff.Change{chargesRemoved(), latestChargeAdded()}, source)
	if len(errs) != 0 {
		t.Fatalf("unexpected rejections: %v", errs)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}

	removed, added := events[0], events[1]
	if removed.Affected.Kind != event.AffectedKindSchemaProperty || removed.Affected.Schema != "payment_intent" ||
		removed.Affected.Property != "charges" || !removed.Breaking {
		t.Errorf("charges removal mapped wrong: %+v", removed)
	}
	// Removals cite the old spec: the evidence is gone from the new one.
	if removed.SourceURL != oldURL || added.SourceURL != newURL {
		t.Errorf("source_url: removal=%s addition=%s, want old then new", removed.SourceURL, added.SourceURL)
	}
	for _, e := range events {
		if e.Confidence != 1.0 {
			t.Errorf("%s: confidence %v, want 1.0 for a spec diff", e.Affected.Property, e.Confidence)
		}
		if e.Version != "2022-11-15" || e.PreviousVersion != "2022-08-01" || !e.DetectedAt.Equal(source.DetectedAt) {
			t.Errorf("%s: version/time not carried over: %+v", e.Affected.Property, e)
		}
		if e.SourcePointer == "" || e.SourceExcerpt == "" {
			t.Errorf("%s: missing pointer or excerpt", e.Affected.Property)
		}
	}
}

// Re-running ingest over the same specs must update nothing, which relies on
// fingerprints being stable across runs and distinct between changes.
func TestFingerprintsAreStableAndDistinct(t *testing.T) {
	changes := []diff.Change{chargesRemoved(), latestChargeAdded()}
	first, _ := Events(changes, source)
	again, _ := Events(changes, source)
	if first[0].Fingerprint != again[0].Fingerprint || first[1].Fingerprint != again[1].Fingerprint {
		t.Error("fingerprints changed between identical runs")
	}
	if first[0].Fingerprint == first[1].Fingerprint {
		t.Error("two different changes got the same fingerprint")
	}
}

// Regression: the excerpt is capped at 2,000 characters, not bytes. Spec text is
// full of non-ASCII, and cutting by characters while validating by bytes once
// rejected 10 of 38 real Stripe events.
func TestLongExcerptIsCutToTheLimitInCharacters(t *testing.T) {
	long := chargesRemoved()
	long.Node = strings.Repeat("é", 3000) // 3,000 characters, 6,000 bytes

	events, errs := Events([]diff.Change{long}, source)
	if len(errs) != 0 {
		t.Fatalf("long excerpt was rejected: %v", errs)
	}
	excerpt := events[0].SourceExcerpt
	if n := utf8.RuneCountInString(excerpt); n != 2000 {
		t.Errorf("excerpt is %d characters, want exactly 2000", n)
	}
	if !strings.HasSuffix(excerpt, "…") {
		t.Error("a cut excerpt should end with …")
	}
}

// A change with nothing to quote is rejected and reported, never stored and
// never a panic: no provenance, no event.
func TestChangeWithoutEvidenceIsRejected(t *testing.T) {
	bare := chargesRemoved()
	bare.Node = nil
	events, errs := Events([]diff.Change{bare, latestChargeAdded()}, source)
	if len(errs) != 1 || len(events) != 1 {
		t.Errorf("got %d events and %d errors, want 1 and 1 (the bad one rejected, the good one kept)", len(events), len(errs))
	}
}
