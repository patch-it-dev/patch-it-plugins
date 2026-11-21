package numpy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
	"github.com/patch-it-dev/patch-it-plugins/kinds/migrationguide"
	"github.com/patch-it-dev/patch-it-plugins/kinds/migrationguide/members"
	"github.com/patch-it-dev/patch-it-plugins/kinds/migrationguide/migrationguidetest"
)

// The guide as published at tag v2.0.0 (fetched 2026-11-04).
func guide(t *testing.T) []byte {
	t.Helper()
	return kittest.Read(t, "testdata/numpy_2_0_migration_guide-v2.0.0.rst")
}

func byMember(t *testing.T) map[string]members.Removal {
	t.Helper()
	rs, err := members.Parse(guide(t), Numpy)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]members.Removal{}
	for _, r := range rs {
		out[r.Member] = r
	}
	return out
}

func TestEveryRemovedMemberOfTheMainNamespaceIsRead(t *testing.T) {
	got := byMember(t)
	if len(got) != 56 {
		t.Errorf("%d removed members, want the table's 56", len(got))
	}
	// A guideline continued over several lines is read whole.
	if r := got["find_common_type"]; !strings.Contains(r.Guideline, "``0.0``, or ``0j``") {
		t.Errorf("find_common_type: %q", r.Guideline)
	}
	// The table's escaped underscore is the member's own.
	if _, ok := got["float_"]; !ok {
		t.Error(`no "float_": the \_ escape wasn't undone`)
	}
}

// Only a guideline naming exactly one member of np is a rename; anything else
// (a builtin, an expression, advice) is a removal the patch-it client
// opens as an issue.
func TestOnlyOnePlainReplacementInNpIsARename(t *testing.T) {
	got := byMember(t)
	for member, want := range map[string]string{
		"float_":     "float64",
		"NaN":        "nan",
		"product":    "prod",
		"cumproduct": "cumprod",
		"string_":    "bytes_",
		"Inf":        "inf",
		"alltrue":    "", // "Use ``all`` instead.": the builtin, or np.all? Not said
		"NINF":       "", // "Use ``-np.inf`` instead.": an expression
		"asfarray":   "", // "Use ``np.asarray`` with a float dtype instead."
		"compat":     "", // no replacement
		"who":        "",
	} {
		if r, ok := got[member]; !ok || r.Replacement != want {
			t.Errorf("%s: replacement %q, want %q", member, r.Replacement, want)
		}
	}
}

func TestTheGuideStoresOneEventPerMemberAndARerunNothing(t *testing.T) {
	// The guide at the v2.0.0 tag, and nothing else; kittest extracts the
	// pair again and fails on anything new.
	events := migrationguidetest.Release(t, migrationguidetest.GuideEnv(t, Numpy, "testdata/numpy_2_0_migration_guide-v2.0.0.rst", "2.0.0"), Numpy, "1.26.4", "2.0.0")
	if len(events) != 56 {
		t.Fatalf("%d events, want 56", len(events))
	}
	var float *event.Event
	for i := range events {
		if events[i].Subject.Symbols[0] == "float_" {
			float = &events[i]
		}
	}
	if float == nil || float.Subject.Change != event.PackageRenamedSymbol || float.Subject.Replacement != "float64" ||
		float.API != "numpy/numpy" || float.Version != "2.0.0" || !strings.Contains(float.SourceExcerpt, "float\\_") {
		t.Errorf("float_: %+v", float)
	}
}

// The package's releases come from PyPI; only the pair into the release the
// guide documents has events (the golden's, with previous_version set).
func TestOnlyTheGuidesReleaseHasEvents(t *testing.T) {
	env := kittest.Env(t, kittest.Pages{
		"https://pypi.org/pypi/numpy/json": "testdata/pypi-numpy-excerpt.json",
		Numpy.RawURL("v2.0.0", 2):          "testdata/numpy_2_0_migration_guide-v2.0.0.rst",
	})
	p := migrationguide.Plugin{Guide: Numpy}
	ctx := context.Background()
	vs, err := p.ListVersions(ctx, env)
	if err != nil {
		t.Fatal(err)
	}
	pinned := migrationguidetest.Release(t, migrationguidetest.GuideEnv(t, Numpy, "testdata/numpy_2_0_migration_guide-v2.0.0.rst", "2.0.0"), Numpy, "1.26.4", "2.0.0")
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

// The guide at tag v2.0.0 (saved 2026-11-04) in, every event out: the pair
// from 1.26.4, the release before on PyPI.
func TestNumpyGolden(t *testing.T) {
	got := migrationguidetest.Release(t, migrationguidetest.GuideEnv(t, Numpy, "testdata/numpy_2_0_migration_guide-v2.0.0.rst", "2.0.0"), Numpy, "1.26.4", "2.0.0")
	kittest.Golden(t, "testdata/golden/numpy-2.0.0.json", got)
}

// The guide is read at the major's first tag; a release that isn't one has
// none.
func TestNumpyGuideIsReadAtTheMajorsTag(t *testing.T) {
	ctx := context.Background()
	want := "https://raw.githubusercontent.com/numpy/numpy/v2.0.0/doc/source/numpy_2_0_migration_guide.rst"
	if got, err := Numpy.GuideURL(ctx, migrationguidetest.GuideEnv(t, Numpy, "testdata/README.md", "2.0.0"), plugin.Version{ID: "2.0.0"}); err != nil || got != want {
		t.Errorf("%q, %v; want %q", got, err, want)
	}
	if got, err := Numpy.GuideURL(ctx, plugin.Env{}, plugin.Version{ID: "2.0.1"}); got != "" || err != nil {
		t.Errorf("a release that isn't a major's first has no guide: %q, %v", got, err)
	}
}

// A major with no guide at its tag (a 404) has nothing to read: the pair is
// skipped, not failed, and not a release without changes.
func TestNumpyMajorWithNoGuideHasNothingToRead(t *testing.T) {
	notFound := plugin.Env{Get: func(context.Context, string) ([]byte, error) { return nil, &fetch.StatusError{StatusCode: 404} }}
	if _, err := (migrationguide.Plugin{Guide: Numpy}).ReadAt(context.Background(), notFound, plugin.Version{ID: "3.0.0"}); !errors.Is(err, plugin.ErrNothingToRead) {
		t.Errorf("no guide at the tag: %v, want plugin.ErrNothingToRead", err)
	}
}
