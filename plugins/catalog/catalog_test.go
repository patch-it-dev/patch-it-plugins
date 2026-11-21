package catalog

import (
	"context"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// The two commits redis-doc's own history shows the removal happen between:
// PR #1697 dropped FUNCTION CREATE and FUNCTION INFO from commands.json on
// 2022-01-24 (checked 2026-11-18), replacing the single-function CREATE with
// a whole-library LOAD. Neither command ever reached a tagged Redis release
// (not even 7.0's first release candidate, which shipped a week later with
// neither in its own src/commands/): a real removal, but one that happened
// before any user could depend on it, not a production break.
const (
	oldRef = "cc0427b3c3671e90b92f1ad2fe4e835fd32cd3c9"
	newRef = "06f3598f39c7ca86dbdb0aa1668cfdd1b82b908d"
)

func env(s *kittest.Mem, pages kittest.Pages, t *testing.T) plugin.Env {
	return plugin.Env{Baselines: s, Get: pages.Get(t), Now: time.Now}
}

func fixturePages() kittest.Pages {
	return kittest.Pages{
		Redis.URL(oldRef): "testdata/redis-doc-cc0427b.json",
		Redis.URL(newRef): "testdata/redis-doc-06f3598.json",
	}
}

// The golden test and the pair fixture in one: testdata/redis-doc-cc0427b.json
// and redis-doc-06f3598.json are real, trimmed excerpts of redis-doc's
// commands.json at those two commits (checked 2026-11-18). Between them,
// FUNCTION CREATE and FUNCTION INFO disappear, FUNCTION LOAD appears, and GET
// (an unrelated, long-lived command) is unchanged. This is the pair the
// version pair oldRef → newRef (ReadAt both, Extract) recreates.
func TestPairRecreatesTheRealFunctionCommandRemoval(t *testing.T) {
	ctx := context.Background()
	s := kittest.NewMem()
	e := env(s, fixturePages(), t)

	rejected, err := kittest.Pair(ctx, e, s, Plugin{Catalog: Redis}, oldRef, newRef)
	if err != nil || rejected {
		t.Fatalf("pair: rejected=%v err=%v", rejected, err)
	}

	events, err := s.All()
	if err != nil {
		t.Fatalf("reading events: %v", err)
	}
	// Precision over recall: GET (unchanged) and FUNCTION LOAD (added, not
	// removed) must produce nothing. Only the two real removals should.
	got := map[string]bool{}
	for _, ev := range events {
		if len(ev.Subject.Symbols) != 1 {
			t.Fatalf("event with %d symbols, want exactly 1: %+v", len(ev.Subject.Symbols), ev)
		}
		got[ev.Subject.Symbols[0]] = true
	}
	want := map[string]bool{"FUNCTION CREATE": true, "FUNCTION INFO": true}
	if len(got) != len(want) {
		t.Fatalf("stored %v, want exactly %v (GET and FUNCTION LOAD must drop silently)", got, want)
	}
	for name := range want {
		if !got[name] {
			t.Errorf("missing an event for %s", name)
		}
	}

	// Acceptance criteria: every event carries source_url and a non-empty
	// source_excerpt, is marked breaking, and is placed on the real refs.
	for _, ev := range events {
		if ev.API != "redis/commands" || ev.Version != newRef || ev.PreviousVersion != oldRef {
			t.Errorf("event %v: api=%s version=%s previous_version=%s, want redis/commands %s from %s",
				ev.Subject.Symbols, ev.API, ev.Version, ev.PreviousVersion, newRef, oldRef)
		}
		if ev.ChangeType != event.ChangeTypeBreaking || !ev.Breaking {
			t.Errorf("event %v: change_type=%s breaking=%v, want breaking", ev.Subject.Symbols, ev.ChangeType, ev.Breaking)
		}
		if ev.SourceURL != Redis.URL(oldRef) {
			t.Errorf("event %v: source_url=%s, want the old snapshot, where the removed command is still evidenced", ev.Subject.Symbols, ev.SourceURL)
		}
		if ev.SourceExcerpt == "" {
			t.Errorf("event %v: missing source_excerpt", ev.Subject.Symbols)
		}
	}

	// Reading the same pair again must not duplicate the events.
	if _, err := kittest.Pair(ctx, e, s, Plugin{Catalog: Redis}, oldRef, newRef); err != nil {
		t.Fatalf("second read of the pair: %v", err)
	}
	if again, _ := s.All(); len(again) != len(events) {
		t.Errorf("reading the pair again added events: %d -> %d", len(events), len(again))
	}
}

// Run's own path: looking up the newest commit, then diffing it against
// whatever was last stored. A first run has no baseline to diff against, so
// it reports nothing (same as an unchanged one) and just saves one.
// kittest.Ingest reruns this, so a second run that found anything new would
// fail it too.
func TestRunFindsNothingOnAFirstRunAndStaysQuietOnARerun(t *testing.T) {
	pages := kittest.Pages{
		Redis.RefsURL:     "testdata/commits-master.json",
		Redis.URL(newRef): "testdata/redis-doc-06f3598.json",
	}
	events := kittest.Ingest(t, Plugin{Catalog: Redis}, pages)
	if len(events) != 0 {
		t.Errorf("first run found %d events, want 0: there is nothing yet to diff against", len(events))
	}
}
