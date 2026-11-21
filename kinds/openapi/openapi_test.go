package openapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/hint"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/adapter"
)

// fakeHints stands in for a changelog: an index built from the entries given,
// or an error, so the ingest tests never reach docs.stripe.com.
type fakeHints struct {
	entries []hint.Entry
	err     error
}

func (f fakeHints) Index(context.Context, string) (*hint.Index, error) {
	if f.err != nil {
		return nil, f.err
	}
	return hint.NewIndex(f.entries), nil
}

// noChangelog is the hint source in every test but the one that gives entries:
// the tests never reach docs.stripe.com.
var noChangelog = fakeHints{err: fmt.Errorf("no changelog in tests")}

// plugin is the Stripe plugin over a fake adapter, run through the same core
// the command uses.
func newPlugin(a adapter.Adapter, hints hint.Source) Plugin { return Plugin{Adapter: a, Hints: hints} }

func env(*kittest.Mem) plugin.Env {
	return plugin.Env{Now: time.Now}
}

func pair(ctx context.Context, s *kittest.Mem, a adapter.Adapter, from, to string) (bool, error) {
	return pairWith(ctx, s, newPlugin(a, noChangelog), from, to)
}

func pairWith(ctx context.Context, s *kittest.Mem, p Plugin, from, to string) (bool, error) {
	return kittest.Pair(ctx, env(s), s, p, from, to)
}

// fakeAdapter serves specs from memory, one per tag, so the whole ingest flow
// runs without the network.
type fakeAdapter struct {
	latest string
	specs  map[string]string // tag -> spec JSON
}

func (fakeAdapter) API() string { return "stripe/api" }

func (f fakeAdapter) LatestTag(context.Context) (string, error) { return f.latest, nil }

func (f fakeAdapter) Fetch(_ context.Context, tag string) (adapter.Spec, error) {
	body, ok := f.specs[tag]
	if !ok {
		return adapter.Spec{}, fmt.Errorf("no spec for tag %s", tag)
	}
	return adapter.Spec{
		Content:   []byte(body),
		Version:   "version-" + tag,
		SourceURL: "https://example/" + tag + "/spec.json",
	}, nil
}

// v1 -> v2 removes payment_intent.charges (the demo change) and adds latest_charge.
var specs = map[string]string{
	"v1": `{"openapi":"3.0.0","info":{"version":"version-v1"},"paths":{},
	  "components":{"schemas":{"payment_intent":{"type":"object","properties":{
	    "id":{"type":"string"},"charges":{"type":"object"}}}}}}`,
	"v2": `{"openapi":"3.0.0","info":{"version":"version-v2"},"paths":{},
	  "components":{"schemas":{"payment_intent":{"type":"object","properties":{
	    "id":{"type":"string"},"latest_charge":{"type":"string"}}}}}}`,
}

func storedEvents(t *testing.T, s *kittest.Mem) []string {
	t.Helper()
	events, err := s.All()
	if err != nil {
		t.Fatalf("reading events: %v", err)
	}
	var got []string
	for _, e := range events {
		got = append(got, fmt.Sprintf("%s %s.%s", e.ChangeType, e.Subject.Schema, e.Subject.Property))
	}
	return got
}

// A version pair (ReadAt both sides, Extract): a real change is detected and
// stored as change events, once.
func TestAPairStoresItsEventsOnce(t *testing.T) {
	ctx := context.Background()
	s := kittest.NewMem()

	if _, err := pair(ctx, s, fakeAdapter{specs: specs}, "v1", "v2"); err != nil {
		t.Fatalf("pair: %v", err)
	}
	got := storedEvents(t, s)
	want := map[string]bool{
		"breaking payment_intent.charges":          true,
		"new_feature payment_intent.latest_charge": true,
	}
	if len(got) != len(want) {
		t.Fatalf("stored %v, want exactly %v", got, want)
	}
	for _, g := range got {
		if !want[g] {
			t.Errorf("unexpected stored event %q", g)
		}
	}

	// Reading the same pair again stores nothing new.
	before := len(storedEvents(t, s))
	if _, err := pair(ctx, s, fakeAdapter{specs: specs}, "v1", "v2"); err != nil {
		t.Fatalf("second read of the pair: %v", err)
	}
	if after := len(storedEvents(t, s)); after != before {
		t.Errorf("reading the pair again added events: %d -> %d", before, after)
	}
}

// A changelog that can't be reached costs the hints, never the events (the
// hint source in these tests always errors: noChangelog).
func TestIngestStoresEventsWhenTheChangelogIsUnreachable(t *testing.T) {
	ctx := context.Background()
	s := kittest.NewMem()
	if _, err := pair(ctx, s, fakeAdapter{specs: specs}, "v1", "v2"); err != nil {
		t.Fatalf("pair: %v", err)
	}
	events, _ := s.All()
	if len(events) != 2 {
		t.Fatalf("stored %d events, want 2", len(events))
	}
}

// With a changelog, the entry's text and link become the event's migration hint.
func TestAPairAttachesMigrationHintsFromTheChangelog(t *testing.T) {
	withHints := fakeHints{entries: []hint.Entry{{
		Title:    "Removes the charges attribute from the PaymentIntent object",
		URL:      "https://docs.stripe.com/changelog/2022-11-15/removes-charges-attribute-paymentintent",
		WhatsNew: "Use latest_charge instead.",
		Rows:     []hint.Row{{Parameter: "charges", Change: "Removed", Targets: []hint.Target{{Schema: "payment_intent", Path: "charges"}}}},
	}}}

	ctx := context.Background()
	s := kittest.NewMem()
	if _, err := pairWith(ctx, s, newPlugin(fakeAdapter{specs: specs}, withHints), "v1", "v2"); err != nil {
		t.Fatalf("pair: %v", err)
	}
	events, _ := s.All()
	var removed *event.Event
	for i := range events {
		if events[i].Subject.Property == "charges" {
			removed = &events[i]
		}
	}
	if removed == nil {
		t.Fatal("no charges event stored")
	}
	want := "Use latest_charge instead.\n\nSource: https://docs.stripe.com/changelog/2022-11-15/removes-charges-attribute-paymentintent"
	if removed.MigrationHint != want {
		t.Errorf("hint = %q, want %q", removed.MigrationHint, want)
	}
}
