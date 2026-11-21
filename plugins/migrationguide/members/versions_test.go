package members_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"

	"github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide"
	. "github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/members" // the reader under test
)

// The package's releases come from PyPI; only the pair into the release the
// guide documents has events (the golden's, with previous_version set).
func TestOnlyTheGuidesReleaseHasEvents(t *testing.T) {
	env := kittest.Env(t, kittest.Pages{
		"https://pypi.org/pypi/numpy/json": "testdata/pypi-numpy-excerpt.json",
		NumPy.RawURL("v2.0.0", 2):          "testdata/numpy_2_0_migration_guide-v2.0.0.rst",
	})
	p := migrationguide.Plugin{Guide: NumPy}
	ctx := context.Background()
	vs, err := p.ListVersions(ctx, env)
	if err != nil {
		t.Fatal(err)
	}
	pinned := release(t, NumPy, "testdata/numpy_2_0_migration_guide-v2.0.0.rst", "1.26.4", "2.0.0")
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

// Each guide is read where its package publishes it: at the major's first
// tag (TagPrefix + version), or, for a docs repo with no tags, at the newest
// commit that changed the major's note.
func TestEachGuideIsReadWhereItsPublished(t *testing.T) {
	ctx := context.Background()
	for _, c := range []struct {
		g    Guide
		rel  string
		want string
	}{
		{NumPy, "2.0.0", "https://raw.githubusercontent.com/numpy/numpy/v2.0.0/doc/source/numpy_2_0_migration_guide.rst"},
		{PostHog, "8.0.0", "https://raw.githubusercontent.com/PostHog/posthog-python/posthog-v8.0.0/docs/migration-7.x-to-8.0.md"},
		{NewRelic, "11.0.0", "https://raw.githubusercontent.com/newrelic/docs-website/" + newrelicCommit + "/src/content/docs/release-notes/agent-release-notes/python-release-notes/python-agent-110000.mdx"},
	} {
		got, err := c.g.GuideURL(ctx, guideEnv(t, c.g, "testdata/README.md", c.rel), plugin.Version{ID: c.rel})
		if err != nil || got != c.want {
			t.Errorf("%s %s: %q, %v; want %q", c.g.Package, c.rel, got, err, c.want)
		}
		if got, err := c.g.GuideURL(ctx, plugin.Env{}, plugin.Version{ID: c.rel[:len(c.rel)-1] + "1"}); got != "" || err != nil {
			t.Errorf("%s: a release that isn't a major's first has no guide: %q, %v", c.g.Package, got, err)
		}
	}
}

// A major with no guide has nothing to read: a 404 at its tag, or (docs
// repo) no commit ever changed its note's path. The pair is skipped, not
// failed, and not a release without changes.
func TestAMajorWithNoGuideHasNothingToRead(t *testing.T) {
	ctx := context.Background()
	env := kittest.Env(t, kittest.Pages{NewRelic.CommitsURL(14): "testdata/github-commits-none.json"})
	if _, err := (migrationguide.Plugin{Guide: NewRelic}).ReadAt(ctx, env, plugin.Version{ID: "14.0.0"}); !errors.Is(err, plugin.ErrNothingToRead) {
		t.Errorf("no commit for the note: %v, want plugin.ErrNothingToRead", err)
	}
	notFound := plugin.Env{Get: func(context.Context, string) ([]byte, error) { return nil, &fetch.StatusError{StatusCode: 404} }}
	if _, err := (migrationguide.Plugin{Guide: NumPy}).ReadAt(ctx, notFound, plugin.Version{ID: "3.0.0"}); !errors.Is(err, plugin.ErrNothingToRead) {
		t.Errorf("no guide at the tag: %v, want plugin.ErrNothingToRead", err)
	}
}
