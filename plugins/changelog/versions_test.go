package changelog

import (
	"context"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
)

// Unpinned, a changelog's versions are the package's final releases on PyPI,
// each read at its tag; the 1.26.15 -> 2.0.0 pair gives the same events as
// the pinned golden, with previous_version set.
func TestReleasesAreListedAndExtractedPairByPair(t *testing.T) {
	pages := kittest.Pages{
		"https://pypi.org/pypi/urllib3/json": "testdata/pypi-urllib3-excerpt.json",
		Urllib3.At("1.26.15").URL():          "testdata/urllib3-2.0.0-CHANGES.rst", // only 2.0.0's section is read
		Urllib3200.URL():                     "testdata/urllib3-2.0.0-CHANGES.rst",
	}
	env := kittest.Env(t, pages)
	p := Plugin{File: Urllib3}
	ctx := context.Background()
	vs, err := p.ListVersions(ctx, env)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, v := range vs {
		ids = append(ids, v.ID+"@"+v.Ref)
	}
	if got := strings.Join(ids, " "); got != "1.26.15@1.26.15 2.0.0@2.0.0 2.0.1@2.0.1" {
		t.Fatalf("versions %s", got)
	}
	pair := versions.Pairs(vs)[0]
	from, err := p.ReadAt(ctx, env, pair[0])
	if err != nil {
		t.Fatal(err)
	}
	to, err := p.ReadAt(ctx, env, pair[1])
	if err != nil {
		t.Fatal(err)
	}
	res, err := p.Extract(ctx, env, from, to)
	if err != nil || len(res.Rejected) > 0 {
		t.Fatalf("%v %v", err, res.Rejected)
	}
	pinned := kittest.Ingest(t, Plugin{File: Urllib3200}, kittest.Pages{Urllib3200.URL(): "testdata/urllib3-2.0.0-CHANGES.rst"})
	if len(res.Events) == 0 || len(res.Events) != len(pinned) {
		t.Fatalf("pair gave %d events, pinned run %d", len(res.Events), len(pinned))
	}
	for i, e := range res.Events {
		if e.PreviousVersion != "1.26.15" || e.Version != "2.0.0" || e.Fingerprint != pinned[i].Fingerprint {
			t.Errorf("event %d: %s<-%s %s, pinned %s", i, e.Version, e.PreviousVersion, e.Fingerprint, pinned[i].Fingerprint)
		}
	}
}

// Run on an unpinned file reads the newest release on PyPI.
func TestRunReadsTheNewestRelease(t *testing.T) {
	pages := kittest.Pages{
		"https://pypi.org/pypi/urllib3/json": "testdata/pypi-urllib3-excerpt.json",
		Urllib3.At("2.0.1").URL():            "testdata/urllib3-2.0.0-CHANGES.rst",
	}
	_, err := Plugin{File: Urllib3}.Run(context.Background(), kittest.Env(t, pages))
	// The saved file is 2.0.0's, which has no 2.0.1 section: the run says so
	// rather than inventing events.
	if err == nil || !strings.Contains(err.Error(), "2.0.1") {
		t.Errorf("err = %v, want one naming 2.0.1", err)
	}
	var _ plugin.Versioned = Plugin{}
}
