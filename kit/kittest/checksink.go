package kittest

import (
	"fmt"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/event"
	"github.com/patch-it-dev/patch-it-plugins/plugin"
)

// CheckSink checks that the Sinks open returns store as ingest does: dedupe by
// fingerprint, ids in save order, an invalid event saving nothing, and which
// baseline is the last. Mem passes it; ingest runs it against its Postgres
// store too, so the two can't drift apart. open must return an empty Sink.
func CheckSink(t *testing.T, open func(t *testing.T) Sink) {
	at := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	lastTag := func(t *testing.T, s Sink, api string) string {
		t.Helper()
		last, err := s.LastSpec(api)
		if err != nil {
			t.Fatal(err)
		}
		if last == nil {
			return "(none)"
		}
		return last.Tag
	}
	spec := func(tag, content string, fetched time.Time, replay bool) plugin.Spec {
		return plugin.Baseline("x", tag, tag, "https://example.test/"+tag, []byte(content), replay, fetched)
	}
	saveSpecs := func(t *testing.T, s Sink, specs ...plugin.Spec) {
		t.Helper()
		for _, sp := range specs {
			if err := s.SaveSpec(sp); err != nil {
				t.Fatal(err)
			}
		}
	}

	t.Run("dedupe and ids", func(t *testing.T) {
		s := open(t)
		first := []event.EventV2{sinkEvent(1), sinkEvent(2), sinkEvent(1)}
		if n, err := s.SaveEvents(first); err != nil || n != 2 {
			t.Fatalf("first save: %d new, %v; want 2 (the repeat in the batch is skipped)", n, err)
		}
		if n, err := s.SaveEvents([]event.EventV2{sinkEvent(2), sinkEvent(3)}); err != nil || n != 1 {
			t.Fatalf("second save: %d new, %v; want 1", n, err)
		}
		all, err := s.All()
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for i, e := range all {
			got = append(got, e.Subject.Property)
			if i > 0 && e.ID <= all[i-1].ID {
				t.Errorf("id %d after %d: ids must grow in save order", e.ID, all[i-1].ID)
			}
		}
		if want := "[field_1 field_2 field_3]"; fmt.Sprint(got) != want {
			t.Errorf("stored %v, want %s", got, want)
		}
	})
	t.Run("an invalid event saves nothing", func(t *testing.T) {
		s := open(t)
		bad := sinkEvent(2)
		bad.SourceExcerpt = ""
		if _, err := s.SaveEvents([]event.EventV2{sinkEvent(1), bad}); err == nil {
			t.Fatal("saved an event without provenance")
		}
		if all, _ := s.All(); len(all) != 0 {
			t.Errorf("%d events stored from a rejected batch", len(all))
		}
	})
	t.Run("first run", func(t *testing.T) {
		if got := lastTag(t, open(t), "x"); got != "(none)" {
			t.Errorf("LastSpec = %s, want none", got)
		}
	})
	t.Run("latest instant across offsets", func(t *testing.T) {
		s := open(t)
		saveSpecs(t, s, spec("older", "1", at.Add(-time.Hour), false), spec("newer", "2", at.In(time.FixedZone("UTC-11", -11*3600)), false))
		if got := lastTag(t, s, "x"); got != "newer" {
			t.Errorf("baseline %s, want newer", got)
		}
	})
	t.Run("equal times go to the newest save", func(t *testing.T) {
		s := open(t)
		saveSpecs(t, s, spec("first", "1", at, false), spec("second", "2", at, false))
		if got := lastTag(t, s, "x"); got != "second" {
			t.Errorf("baseline %s, want second", got)
		}
	})
	t.Run("same content is one baseline", func(t *testing.T) {
		s := open(t)
		saveSpecs(t, s, spec("a", "same", at, false), spec("b", "other", at.Add(time.Hour), false), spec("c", "same", at.Add(2*time.Hour), false))
		if got := lastTag(t, s, "x"); got != "b" {
			t.Errorf("baseline %s, want b: the same bytes again aren't a new baseline", got)
		}
	})
	t.Run("replays never move the baseline", func(t *testing.T) {
		s := open(t)
		saveSpecs(t, s, spec("real", "1", at, false), spec("replayed", "2", at.Add(time.Hour), true))
		if got := lastTag(t, s, "x"); got != "real" {
			t.Errorf("baseline %s, want real", got)
		}
	})
	t.Run("a normal fetch claims what a replay stored", func(t *testing.T) {
		s := open(t)
		saveSpecs(t, s, spec("old", "1", at, false), spec("replayed", "same", at.Add(-time.Hour), true), spec("fetched", "same", at.Add(time.Hour), false))
		if got := lastTag(t, s, "x"); got != "fetched" {
			t.Errorf("baseline %s, want fetched", got)
		}
	})
	t.Run("a baseline keeps every field", func(t *testing.T) {
		s := open(t)
		want := spec("v7", "body", at.Add(123456789), false)
		saveSpecs(t, s, want)
		got, err := s.LastSpec("x")
		if err != nil || got == nil {
			t.Fatalf("LastSpec: %v, %v", got, err)
		}
		want.FetchedAt = want.FetchedAt.Truncate(time.Microsecond)
		if got.API != want.API || got.Version != want.Version || got.Tag != want.Tag || got.SHA256 != want.SHA256 || got.SourceURL != want.SourceURL ||
			!got.FetchedAt.Equal(want.FetchedAt) || got.FetchedAt.Location() != time.UTC || got.Replay || string(got.Content) != "body" {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})
}

func sinkEvent(n int) event.EventV2 {
	subject := event.Subject{
		Type: event.SubjectAPIOperation, Kind: event.AffectedKindSchemaProperty, Side: event.SideResponse,
		Schema: "thing", Property: fmt.Sprintf("field_%d", n),
	}
	return event.EventV2{
		Fingerprint:   event.FingerprintV2("x", "2", event.ChangeTypeBreaking, subject, ""),
		API:           "x",
		Version:       "2",
		ChangeType:    event.ChangeTypeBreaking,
		Breaking:      true,
		Subject:       subject,
		Description:   fmt.Sprintf("Property field_%d removed.", n),
		SourceURL:     "https://example.test/2",
		SourceExcerpt: `{"type":"string"}`,
		Confidence:    1,
		DetectedAt:    Clock,
	}
}
