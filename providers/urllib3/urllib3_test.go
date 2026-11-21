package urllib3

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
	"github.com/patch-it-dev/patch-it-plugins/kinds/changelog"
	"github.com/patch-it-dev/patch-it-plugins/kinds/changelog/changelogtest"
)

// The real changelog at a release tag (testdata/README.md) in, every
// event out: the pair from the release before (on PyPI) to the pinned one.
func TestUrllib3Golden(t *testing.T) {
	for _, c := range []struct {
		file     changelog.File
		saved    string
		previous string
	}{
		{Urllib3.At("2.0.0"), "testdata/urllib3-2.0.0-CHANGES.rst", "1.26.15"},
	} {
		t.Run(c.file.Package, func(t *testing.T) {
			got := changelogtest.Release(t, c.file, c.saved, c.previous)
			kittest.Golden(t, "testdata/golden/"+c.file.Package+"-"+c.file.Release+".json", got)
		})
	}
}

// Every bullet of each release that is breaking or a deprecation but isn't
// plain enough is dropped, not guessed: pinned here with why.
func TestUrllib3UnsureBulletsAreDropped(t *testing.T) {
	for _, c := range []struct {
		file    changelog.File
		saved   string
		dropped []string // the start of each dropped bullet, in order
	}{
		{Urllib3.At("2.0.0"), "testdata/urllib3-2.0.0-CHANGES.rst", []string{
			"Removed support for Python 2.7",
			"Removed fallback on certificate",
			"Removed support for Python with an",
			"Removed support for OpenSSL versions",
			"Removed the list of default ciphers",
			"Removed ``urllib3.contrib.appengine.AppEngineManager`` and support", // names and an environment
			"Removed deprecated ``Retry`` options",                               // parameters
			"Removed default value of ``None``",                                  // a default, of a parameter
			"Removed the ``urllib3.request`` module",                             // the release adds urllib3.request back
			"Removed support for SSLv3.0",
			"Removed ``DEFAULT_CIPHERS``, ``HAS_SNI``", // a private module
			"Deprecated ``ssl_version`` option",        // a parameter of what?
			"Deprecated the ``strict`` parameter",
			"Deprecated ``RequestField.header_formatter`` parameter",
		}},
	} {
		_, fs, err := c.file.Read(kittest.Read(t, c.saved))
		if err != nil {
			t.Fatal(err)
		}
		var drops []changelog.Finding
		for _, f := range fs {
			if f.Dropped != "" {
				drops = append(drops, f)
			}
		}
		if len(drops) != len(c.dropped) {
			for _, d := range drops {
				t.Logf("dropped %q: %s", d.Bullet.Text, d.Dropped)
			}
			t.Errorf("%s: dropped %d, want %d", c.file.Package, len(drops), len(c.dropped))
			continue
		}
		for i, want := range c.dropped {
			if !strings.HasPrefix(drops[i].Bullet.Text, want) {
				t.Errorf("%s: drop %d is %q, want %q…", c.file.Package, i, drops[i].Bullet.Text, want)
			}
		}
	}
}

// Unpinned, a changelog's versions are the package's final releases on PyPI,
// each read at its tag; the 1.26.15 -> 2.0.0 pair gives the same events as
// the pinned golden, with previous_version set.
func TestReleasesAreListedAndExtractedPairByPair(t *testing.T) {
	pages := kittest.Pages{
		"https://pypi.org/pypi/urllib3/json": "testdata/pypi-urllib3-excerpt.json",
		Urllib3.At("1.26.15").URL():          "testdata/urllib3-2.0.0-CHANGES.rst", // only 2.0.0's section is read
		Urllib3.At("2.0.0").URL():            "testdata/urllib3-2.0.0-CHANGES.rst",
	}
	env := kittest.Env(t, pages)
	p := changelog.Plugin{File: Urllib3}
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
	pinned := changelogtest.Release(t, Urllib3.At("2.0.0"), "testdata/urllib3-2.0.0-CHANGES.rst", "1.26.15")
	if len(res.Events) == 0 || len(res.Events) != len(pinned) {
		t.Fatalf("pair gave %d events, the pinned release %d", len(res.Events), len(pinned))
	}
	for i, e := range res.Events {
		if e.PreviousVersion != "1.26.15" || e.Version != "2.0.0" || e.Fingerprint != pinned[i].Fingerprint {
			t.Errorf("event %d: %s<-%s %s, pinned %s", i, e.Version, e.PreviousVersion, e.Fingerprint, pinned[i].Fingerprint)
		}
	}
}

// A release whose changelog has no section for it is nothing to read, not a
// failure: patch-it's hosted ingestion service skips the pair.
func TestAReleaseWithNoSectionIsNothingToRead(t *testing.T) {
	pages := kittest.Pages{
		"https://pypi.org/pypi/urllib3/json": "testdata/pypi-urllib3-excerpt.json",
		Urllib3.At("2.0.1").URL():            "testdata/urllib3-2.0.0-CHANGES.rst",
	}
	env := kittest.Env(t, pages)
	p := changelog.Plugin{File: Urllib3}
	ctx := context.Background()
	to, err := p.ReadAt(ctx, env, plugin.Version{ID: "2.0.1", Ref: "2.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	// The saved file is 2.0.0's, which has no 2.0.1 section: Extract says so
	// rather than inventing events.
	_, err = p.Extract(ctx, env, plugin.Snapshot{Version: plugin.Version{ID: "2.0.0", Ref: "2.0.0"}}, to)
	if !errors.Is(err, plugin.ErrNothingToRead) || !strings.Contains(err.Error(), "2.0.1") {
		t.Errorf("err = %v, want plugin.ErrNothingToRead naming 2.0.1", err)
	}
}
