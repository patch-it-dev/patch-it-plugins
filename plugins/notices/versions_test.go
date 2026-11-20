package notices

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
	"github.com/patch-it-dev/patch-it-plugins/plugins/notices/retire"
)

// A year of a living page from its Web Archive captures alone: four
// captures, two of them the same content, are three versions; reading each
// and extracting each consecutive pair gives every retirement once, each
// citing the capture where it first appeared. Nothing reaches the network:
// the CDX answer and the captures are served from memory.
func TestBackfillAYearFromCaptures(t *testing.T) {
	full, err := os.ReadFile("retire/testdata/openai-deprecations.md")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.SplitAfter(string(full), "\n")
	header := strings.Join(lines[:2], "")
	// The page as it grew: the oldest capture has the sections from the
	// 2026-04-22 announcement down, the next adds three, the newest is today's.
	older := header + strings.Join(lines[34:], "")
	middle := header + strings.Join(lines[10:], "")
	pg := retire.OpenAI
	capture := func(ts string) string { return "https://web.archive.org/web/" + ts + "id_/" + pg.URL }
	captures := map[string]string{
		capture("20250115000000"): older,
		capture("20250601000000"): middle,
		capture("20251201000000"): string(full),
	}
	cdx := fmt.Sprintf(`[["timestamp","original","digest"],
["20250115000000",%[1]q,"A"],["20250301000000",%[1]q,"A"],["20250601000000",%[1]q,"B"],["20251201000000",%[1]q,"C"]]`, pg.URL)
	env := plugin.Env{
		Baselines: kittest.NewMem(),
		Now:       func() time.Time { return kittest.Clock },
		Get: func(_ context.Context, u string) ([]byte, error) {
			if strings.HasPrefix(u, versions.Wayback+"/cdx/search/cdx?") {
				return []byte(cdx), nil
			}
			if body, ok := captures[u]; ok {
				return []byte(body), nil
			}
			t.Errorf("fetched %s", u)
			return nil, fmt.Errorf("no saved copy of %s", u)
		},
	}
	ctx := context.Background()
	p := Plugin{Page: pg}
	vs, err := p.ListVersions(ctx, env)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 3 || vs[0].ID != "2025-01-15" || vs[2].ID != "2025-12-01" {
		t.Fatalf("versions %v", vs)
	}
	var snaps []plugin.Snapshot
	for _, v := range vs {
		s, err := p.ReadAt(ctx, env, v)
		if err != nil {
			t.Fatal(err)
		}
		snaps = append(snaps, s)
	}
	firstSeen := map[string]string{} // fingerprint -> the capture it's cited from
	for i := range snaps {
		var from plugin.Snapshot
		if i > 0 {
			from = snaps[i-1]
		}
		res, err := p.Extract(ctx, env, from, snaps[i])
		if err != nil || len(res.Rejected) > 0 {
			t.Fatalf("%v %v", err, res.Rejected)
		}
		if len(res.Events) == 0 {
			t.Errorf("%s: no new retirement", vs[i].ID)
		}
		for _, e := range res.Events {
			if _, dup := firstSeen[e.Fingerprint]; dup {
				t.Errorf("%s: %s again", vs[i].ID, e.Subject.Name)
			}
			if e.SourceURL != vs[i].URL {
				t.Errorf("%s cites %s, want %s", e.Subject.Name, e.SourceURL, vs[i].URL)
			}
			firstSeen[e.Fingerprint] = e.SourceURL
		}
	}
	// Today's page, read in one go, holds every retirement the history found
	// (a retirement can only change its fingerprint, never vanish).
	today, err := p.Extract(ctx, env, plugin.Snapshot{}, snaps[2])
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range today.Events {
		if _, ok := firstSeen[e.Fingerprint]; !ok {
			t.Errorf("%s on today's page, never seen in the history", e.Subject.Name)
		}
	}
	if strings.Contains(firstSeen[today.Events[0].Fingerprint], "20250115") {
		t.Errorf("today's newest retirement is cited from the oldest capture")
	}
}
