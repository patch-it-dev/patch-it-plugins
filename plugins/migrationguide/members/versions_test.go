package members_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"

	"github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide"
	. "github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/members" // the reader under test
)

// The package's releases come from PyPI; only the pair into the release the
// guide documents has events (the golden's, with previous_version set).
func TestOnlyTheGuidesReleaseHasEvents(t *testing.T) {
	env := kittest.Env(t, kittest.Pages{
		"https://pypi.org/pypi/numpy/json": "testdata/pypi-numpy-excerpt.json",
		NumPy2.URL():                       "testdata/numpy_2_0_migration_guide-v2.0.0.rst",
	})
	p := migrationguide.Plugin{Guide: NumPy2}
	ctx := context.Background()
	vs, err := p.ListVersions(ctx, env)
	if err != nil {
		t.Fatal(err)
	}
	pinned := release(t, NumPy2, "testdata/numpy_2_0_migration_guide-v2.0.0.rst", "1.26.4")
	var got []string
	for _, pair := range versions.Pairs(vs) {
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
		for i, e := range res.Events {
			if e.PreviousVersion != pair[0].ID || e.Fingerprint != pinned[i].Fingerprint {
				t.Errorf("event %d: previous %q, fingerprint differs from the golden's: %v", i, e.PreviousVersion, e.Fingerprint != pinned[i].Fingerprint)
			}
		}
		got = append(got, fmt.Sprintf("%s->%s:%d", pair[0].ID, pair[1].ID, len(res.Events)))
	}
	want := fmt.Sprintf("1.26.4->2.0.0:%d 2.0.0->2.0.1:0", len(pinned))
	if strings.Join(got, " ") != want || len(pinned) == 0 {
		t.Errorf("got %s, want %s", strings.Join(got, " "), want)
	}
}

// Each release's ref follows the guide's tag shape.
func TestTagsFollowTheGuide(t *testing.T) {
	for g, want := range map[*Guide]string{&NumPy2: "v2.0.0", &PostHog8: "posthog-v8.0.0", &NewRelic11: "f0b4f817bf9a7b76a78e26609369a8a7ce764c8e"} {
		if got := g.Tag(); got != want {
			t.Errorf("%s: tag %q, want %q", g.Package, got, want)
		}
	}
}
