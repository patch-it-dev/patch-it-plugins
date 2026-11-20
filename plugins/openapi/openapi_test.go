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
	"github.com/patch-it-dev/patch-it-plugins/plugins/openapi/adapter"
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

func env(s *kittest.Mem) plugin.Env {
	return plugin.Env{Baselines: s, Now: time.Now}
}

func ingest(ctx context.Context, s *kittest.Mem, a adapter.Adapter) (bool, error) {
	return kittest.Run(ctx, env(s), s, newPlugin(a, noChangelog))
}

func replay(ctx context.Context, s *kittest.Mem, a adapter.Adapter, from, to string) (bool, error) {
	return replayWith(ctx, s, newPlugin(a, noChangelog), from, to)
}

func replayWith(ctx context.Context, s *kittest.Mem, p Plugin, from, to string) (bool, error) {
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

func openStore(*testing.T) *kittest.Mem { return kittest.NewMem() }

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

// End to end: a real change is detected AND STORED as a change event.
func TestIngestStoresTheEventsItFinds(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)

	// First run: nothing to diff against, so the spec becomes the baseline.
	if rejected, err := ingest(ctx, s, fakeAdapter{latest: "v1", specs: specs}); err != nil || rejected {
		t.Fatalf("first run: rejected=%v err=%v", rejected, err)
	}
	if got := storedEvents(t, s); len(got) != 0 {
		t.Fatalf("first run stored %d events, want 0: %v", len(got), got)
	}

	// Second run: v1 -> v2. The events must be in the store, not just printed.
	if rejected, err := ingest(ctx, s, fakeAdapter{latest: "v2", specs: specs}); err != nil || rejected {
		t.Fatalf("second run: rejected=%v err=%v", rejected, err)
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

	// The baseline moved to v2 only after the events were safely stored.
	last, _ := s.LastSpec("stripe/api")
	if last == nil || last.Version != "version-v2" {
		t.Errorf("baseline after the second run = %+v, want version-v2", last)
	}

	// Third run, nothing new: no change, and no duplicate events.
	if _, err := ingest(ctx, s, fakeAdapter{latest: "v2", specs: specs}); err != nil {
		t.Fatalf("third run: %v", err)
	}
	if again := storedEvents(t, s); len(again) != len(got) {
		t.Errorf("unchanged spec added events: %d -> %d", len(got), len(again))
	}
}

// Replay (--from/--to) is how the demo change gets recreated: it must store the
// events, and it must never move the normal-run baseline.
func TestReplayStoresEventsWithoutMovingTheBaseline(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)

	if _, err := replay(ctx, s, fakeAdapter{specs: specs}, "v1", "v2"); err != nil {
		t.Fatalf("replay: %v", err)
	}
	if got := storedEvents(t, s); len(got) != 2 {
		t.Errorf("replay stored %d events, want 2: %v", len(got), got)
	}
	if last, _ := s.LastSpec("stripe/api"); last != nil {
		t.Errorf("replay moved the baseline to %s; replays must never set it", last.Version)
	}

	// Replaying the same pair again stores nothing new.
	before := len(storedEvents(t, s))
	if _, err := replay(ctx, s, fakeAdapter{specs: specs}, "v1", "v2"); err != nil {
		t.Fatalf("second replay: %v", err)
	}
	if after := len(storedEvents(t, s)); after != before {
		t.Errorf("repeating a replay added events: %d -> %d", before, after)
	}
}

// A changelog that can't be reached costs the hints, never the events (the
// hint source in these tests always errors: noChangelog).
func TestIngestStoresEventsWhenTheChangelogIsUnreachable(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)
	if _, err := replay(ctx, s, fakeAdapter{specs: specs}, "v1", "v2"); err != nil {
		t.Fatalf("replay: %v", err)
	}
	events, _ := s.All()
	if len(events) != 2 {
		t.Fatalf("stored %d events, want 2", len(events))
	}
}

// With a changelog, the entry's text and link become the event's migration hint.
func TestReplayAttachesMigrationHintsFromTheChangelog(t *testing.T) {
	withHints := fakeHints{entries: []hint.Entry{{
		Title:    "Removes the charges attribute from the PaymentIntent object",
		URL:      "https://docs.stripe.com/changelog/2022-11-15/removes-charges-attribute-paymentintent",
		WhatsNew: "Use latest_charge instead.",
		Rows:     []hint.Row{{Parameter: "charges", Change: "Removed", Targets: []hint.Target{{Schema: "payment_intent", Path: "charges"}}}},
	}}}

	ctx := context.Background()
	s := openStore(t)
	if _, err := replayWith(ctx, s, newPlugin(fakeAdapter{specs: specs}, withHints), "v1", "v2"); err != nil {
		t.Fatalf("replay: %v", err)
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
