// Package changelogtest is what the tests of providers' changelog sources
// share: a release replayed as patch-it's hosted ingestion service extracts
// it, and an SDK's golden, drop test and registration in one call.
package changelogtest

import (
	"context"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/changelog"
)

// Release is the pair previous -> f's pinned release, as patch-it's hosted
// ingestion service extracts it. A changelog's Extract reads only the newer
// release's own section, so previous is named, not read.
func Release(t *testing.T, f changelog.File, saved, previous string) []event.Event {
	t.Helper()
	env := kittest.Env(t, kittest.Pages{f.URL(): saved})
	p := changelog.Plugin{File: f}
	to, err := p.ReadAt(context.Background(), env, plugin.Version{ID: f.Release, Ref: f.Tag})
	if err != nil {
		t.Fatal(err)
	}
	from := plugin.Snapshot{Version: plugin.Version{ID: previous, Ref: f.TagPrefix + previous}}
	return kittest.Extracted(t, env, p, from, to)
}

// SDKCase is one SDK's changelog, saved at a release's tag
// (testdata/README.md), and the release its golden pins: the pair from the
// release before (on the registry, or among the repo's tags) to it. drop is
// another release in the same saved file (for Notes, in dropped, its own
// saved release) whose notes name nothing removed: it must read as no event.
type SDKCase struct {
	File           changelog.File
	Release, Prev  string
	Saved, Dropped string
	Drop           string
}

// CheckSDKs runs each case's golden, its drop test and its registration.
func CheckSDKs(t *testing.T, cases []SDKCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.File.API, func(t *testing.T) {
			kittest.Registered(t, changelog.Plugin{File: c.File})
			got := Release(t, c.File.At(c.Release), "testdata/"+c.Saved, c.Prev)
			if len(got) == 0 {
				t.Fatal("no events: the golden must pin a release with some")
			}
			kittest.Golden(t, "testdata/golden/"+strings.ReplaceAll(c.File.API, "/", "-")+".json", got)

			saved := c.Saved
			if c.Dropped != "" {
				saved = c.Dropped
			}
			_, fs, err := c.File.At(c.Drop).Read(kittest.Read(t, "testdata/"+saved))
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range fs {
				if f.Dropped == "" {
					t.Errorf("drop test %s: an event from %q", c.Drop, f.Bullet.Text)
				}
			}
		})
	}
}
