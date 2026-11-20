package eol

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/fetch"
	"github.com/patch-it-dev/patch-it-plugins/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/plugin"
)

func TestRegistersItself(t *testing.T) {
	kittest.Registered(t, Plugin{Page: Salesforce})
}

// The golden test: Salesforce's own end-of-life policy page (checked
// 2026-11-20, saved 2026-10-31), real and unedited but for dropping the
// surrounding navigation chrome. It already holds two real, past retirements
// (versions 7.0 through 20.0, retired as of Summer '22; versions 21.0
// through 30.0, retired as of Summer '25) and one real, scheduled one
// (versions 31.0 through 40.0, deprecated from Summer '27, retired from
// Summer '28): 34 events, and the still-"Supported." 41.0 through 67.0 band
// names none, the pinned drop test.
func TestGolden(t *testing.T) {
	saved := kittest.Pages{Salesforce.URL: "testdata/salesforce-api-rest-eol-2026-10-31.html"}
	archive := &kittest.Archive{T: t, Pages: saved}
	got := kittest.IngestWith(t, kittest.Env(t, saved), Plugin{Page: Salesforce, Archiver: archive})
	if archive.Calls != 1 {
		t.Errorf("%d snapshots, want 1: the second run's page is unchanged", archive.Calls)
	}
	kittest.Golden(t, "testdata/golden/salesforce.json", got)
}

// The pinned drop test: "Versions 41.0 through 67.0: Supported." is a real
// row on the page, and it must name no version.
func TestDropsAStillSupportedBand(t *testing.T) {
	page := kittest.Read(t, "testdata/salesforce-api-rest-eol-2026-10-31.html")
	found, err := Findings(page)
	if err != nil {
		t.Fatal(err)
	}
	supported := map[string]bool{}
	for v := 41; v <= 67; v++ {
		supported[fmt.Sprintf("%d.0", v)] = true
	}
	var dropped bool
	for _, f := range found {
		if f.Dropped == "" && supported[f.Version] {
			t.Errorf("version %s got an event, want none: it's still Supported", f.Version)
		}
		if f.Dropped != "" && f.Row == "Versions 41.0 through 67.0" {
			dropped = true
		}
	}
	if !dropped {
		t.Error("the still-supported band wasn't found and dropped")
	}
}

// Salesforce's docs site answers 403 to a plain GET; the live page is never
// reachable through env.Get. Run must fall back to the Wayback Machine's own
// fetch and use its snapshot directly, not fail outright.
func TestFallsBackToTheArchiveWhenTheLiveFetchIs403d(t *testing.T) {
	saved := kittest.Pages{Salesforce.URL: "testdata/salesforce-api-rest-eol-2026-10-31.html"}
	archive := &kittest.Archive{T: t, Pages: saved}
	mem := kittest.NewMem()
	env := plugin.Env{
		Baselines: mem,
		Get: func(context.Context, string) ([]byte, error) {
			return nil, &fetch.StatusError{URL: Salesforce.URL, StatusCode: 403}
		},
		Now: func() time.Time { return kittest.Clock },
	}
	got := kittest.IngestWith(t, env, Plugin{Page: Salesforce, Archiver: archive})
	if len(got) == 0 {
		t.Fatal("no events: the archive fallback didn't run")
	}
	// Unlike the normal path, a permanently blocked live page has no cheap
	// way to check "did it change": liveOrArchived needs the archive's own
	// fetch just to get bytes to hash, on every run, not only a changed one.
	// The second run still stores nothing new: the hash matches.
	if archive.Calls != 2 {
		t.Errorf("%d snapshots, want 2: the live page is blocked every run", archive.Calls)
	}
	for _, e := range got {
		if e.SourceURL == Salesforce.URL {
			t.Errorf("%s: source_url is the blocked live page, want the archive's snapshot", e.Fingerprint)
		}
	}
}

// A real failure (not the provider's own 403) must stay a failure: falling
// back to the archive for a timeout or a 5xx would hide it as a real page.
func TestDoesNotFallBackOnATransientError(t *testing.T) {
	env := plugin.Env{
		Baselines: kittest.NewMem(),
		Get: func(context.Context, string) ([]byte, error) {
			return nil, &fetch.StatusError{URL: Salesforce.URL, StatusCode: 503}
		},
		Now: func() time.Time { return kittest.Clock },
	}
	_, err := Plugin{Page: Salesforce, Archiver: &kittest.Archive{T: t}}.Run(context.Background(), env)
	if err == nil {
		t.Fatal("want an error for a 503, got none")
	}
}
