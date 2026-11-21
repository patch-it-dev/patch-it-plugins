package openai

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices"
)

// growing is a living page that changes between reads, and the archive
// that snapshots it: each read sees the page as it is at that point.
type growing struct {
	page  string
	at    time.Time
	calls int
}

func (g *growing) Snapshot(_ context.Context, url string) (string, []byte, error) {
	g.calls++
	return "https://web.archive.org/web/" + g.at.Format("20060102150405") + "/" + url, []byte(g.page), nil
}

// Three days of a page that grows, read once a day (and once more on the
// third, unchanged): each read that changed is a new dated version, its
// events are the retirements the read before didn't have, each citing the
// snapshot of the day it first appeared; the unchanged read costs one GET
// and keeps the snapshot. Nothing reaches the network.
func TestReadsOfALivingPageStoreEachNoticeOnce(t *testing.T) {
	full, err := os.ReadFile("testdata/openai-deprecations.md")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.SplitAfter(string(full), "\n")
	header := strings.Join(lines[:2], "")
	days := []string{
		header + strings.Join(lines[34:], ""), // the sections from the 2026-04-22 announcement down
		header + strings.Join(lines[10:], ""), // three more
		string(full),                          // today's
		string(full),                          // today's, read again
	}
	pg := Models
	archive := &growing{}
	env := plugin.Env{
		Get: func(_ context.Context, u string) ([]byte, error) {
			if u != pg.URL {
				t.Errorf("fetched %s", u)
				return nil, fmt.Errorf("no saved copy of %s", u)
			}
			return []byte(archive.page), nil
		},
		Now: func() time.Time { return archive.at },
	}
	ctx, sink, p := context.Background(), kittest.NewMem(), notices.Plugin{Page: pg, Archiver: archive}
	var kept plugin.Snapshot
	stored := 0
	for i, page := range days {
		archive.page, archive.at = page, kittest.Clock.AddDate(0, 0, i)
		before := kept
		var rejected bool
		if kept, rejected, err = kittest.ReadPage(ctx, env, sink, p, kept); err != nil || rejected {
			t.Fatalf("day %d: rejected=%v err=%v", i, rejected, err)
		}
		all := kittest.Stored(t, sink)
		if i == 3 {
			if kept.Version != before.Version || len(all) != stored || archive.calls != 3 {
				t.Errorf("an unchanged read: version %s (was %s), %d new events, %d snapshots; want the kept one, none, 3", kept.Version.ID, before.Version.ID, len(all)-stored, archive.calls)
			}
			continue
		}
		if want := archive.at.Format(time.RFC3339); kept.Version.ID != want {
			t.Errorf("day %d: version %s, want %s", i, kept.Version.ID, want)
		}
		if i < 2 && len(all) == stored {
			t.Errorf("day %d: no new retirement", i)
		}
		for _, e := range all[stored:] {
			if e.SourceURL != kept.SourceURL || e.PreviousVersion != "" {
				t.Errorf("day %d: %s cites %s (previous_version %q), want %s and none", i, e.Subject.Name, e.SourceURL, e.PreviousVersion, kept.SourceURL)
			}
		}
		stored = len(all)
	}
	// Today's page, read in one go, holds exactly what the reads stored.
	today, err := p.Extract(ctx, env, plugin.Snapshot{}, kept)
	if err != nil {
		t.Fatal(err)
	}
	if len(today.Events) != stored {
		t.Errorf("today's page has %d retirements, the reads stored %d", len(today.Events), stored)
	}
}
