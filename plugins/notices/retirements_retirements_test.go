package notices

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/plugins/notices/retire"
)

const deprecations = `### 2026-04-22: Legacy GPT model snapshots

| Shutdown date    | Model snapshot             | Substitute model |
| ---------------- | -------------------------- | ---------------- |
| October 23, 2026 | ` + "`gpt-4-0613` \\| `gpt-4`" + ` | ` + "`gpt-5.6-sol`" + ` |
`

// fakeArchive is the Wayback Machine without the network: it serves what it
// was given and counts the snapshots it was asked for.
type fakeArchive struct {
	content string
	err     error
	calls   *int
}

func (f fakeArchive) Snapshot(_ context.Context, page string) (string, []byte, error) {
	*f.calls++
	if f.err != nil {
		return "", nil, f.err
	}
	return "https://web.archive.org/web/20261013115538/" + page, []byte(f.content), nil
}

// page is what the provider's page says now, and what the archive holds.
type page struct {
	live    string
	archive fakeArchive
}

var current page

func withPage(t *testing.T, live string, archive fakeArchive) {
	t.Helper()
	current = page{live, archive}
}

// store is the events stored and the snapshot kept, as the hosted ingestion
// service keeps them between reads.
type store struct {
	*kittest.Mem
	kept plugin.Snapshot
}

func openStore(*testing.T) *store { return &store{Mem: kittest.NewMem()} }

// retirements reads the current page with OpenAI's plugin against the
// snapshot kept from the read before, as the hosted ingestion service does.
func retirements(ctx context.Context, s *store, p retire.Page) (bool, error) {
	env := plugin.Env{
		Get: func(context.Context, string) ([]byte, error) { return []byte(current.live), nil },
		Now: time.Now,
	}
	kept, rejected, err := kittest.ReadPage(ctx, env, s.Mem, Plugin{Page: p, Archiver: current.archive}, s.kept)
	s.kept = kept
	return rejected, err
}

func TestRetirementsStoreOneEventPerModel(t *testing.T) {
	ctx, s, calls := context.Background(), openStore(t), 0
	withPage(t, deprecations, fakeArchive{content: deprecations, calls: &calls})

	if rejected, err := retirements(ctx, s, retire.OpenAI); err != nil || rejected {
		t.Fatalf("rejected=%v err=%v", rejected, err)
	}
	events, err := s.All()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, e := range events {
		got[e.Subject.Provider+" "+e.Version+" "+e.Subject.Name] = true
		if !strings.HasPrefix(e.SourceURL, "https://web.archive.org/web/20261013115538/") {
			t.Errorf("cites %s, want the snapshot", e.SourceURL)
		}
	}
	if len(events) != 2 || !got["openai 2026-10-23 gpt-4-0613"] || !got["openai 2026-10-23 gpt-4"] {
		t.Errorf("want one event per model name, got %v", got)
	}

	// Same page again: no snapshot, nothing new.
	if _, err := retirements(ctx, s, retire.OpenAI); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("%d snapshots for an unchanged page, want 1", calls)
	}
	if again, _ := s.All(); len(again) != 2 {
		t.Errorf("%d events after a rerun, want 2", len(again))
	}
}

func TestRetirementsStoreNothingWithoutASnapshotThatHoldsThem(t *testing.T) {
	for name, archive := range map[string]fakeArchive{
		"the archive refused":  {err: fmt.Errorf("429 too many requests")},
		"the page moved since": {content: strings.Replace(deprecations, "October 23", "November 23", 1)},
	} {
		ctx, s, calls := context.Background(), openStore(t), 0
		archive.calls = &calls
		withPage(t, deprecations, archive)
		if _, err := retirements(ctx, s, retire.OpenAI); err == nil {
			t.Errorf("%s: want an error", name)
		}
		if events, _ := s.All(); len(events) != 0 {
			t.Errorf("%s: stored %d events with nothing to cite", name, len(events))
		}
		if !s.kept.IsZero() {
			t.Errorf("%s: the page was kept, so the next read would skip it", name)
		}
	}
}

// A renamed column hides a table: the run must fail, not store a shorter list.
func TestRetirementsRefuseAPageThatLostMostOfItsTables(t *testing.T) {
	ctx, s, calls := context.Background(), openStore(t), 0
	withPage(t, deprecations, fakeArchive{content: deprecations, calls: &calls})
	if _, err := retirements(ctx, s, retire.OpenAI); err != nil {
		t.Fatal(err)
	}
	renamed := strings.Replace(deprecations, "Substitute model", "Successor", 1) + strings.Replace(strings.Replace(deprecations, "gpt-4-0613` \\| `gpt-4", "gpt-4o-old", 1), "Substitute model", "Recommended replacement", 1)
	withPage(t, renamed, fakeArchive{content: renamed, calls: &calls})
	if _, err := retirements(ctx, s, retire.OpenAI); err == nil || !strings.Contains(err.Error(), "shape may have changed") {
		t.Fatalf("want the shape error, got %v", err)
	}
	if events, _ := s.All(); len(events) != 2 {
		t.Errorf("%d events, want only the first run's 2", len(events))
	}
}
