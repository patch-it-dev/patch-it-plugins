package changelog

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// Two real changelogs at their release tags (testdata/README.md) in, every
// event out: each the pair from the release before (on PyPI) to the pinned one.
func TestGolden(t *testing.T) {
	for _, c := range []struct {
		file     File
		saved    string
		previous string
	}{
		{Textual0860, "testdata/textual-v0.86.0-CHANGELOG.md", "0.85.2"},
		{Urllib3200, "testdata/urllib3-2.0.0-CHANGES.rst", "1.26.15"},
	} {
		t.Run(c.file.Package, func(t *testing.T) {
			got := release(t, c.file, c.saved, c.previous)
			kittest.Golden(t, "testdata/golden/"+c.file.Package+"-"+c.file.Release+".json", got)
		})
	}
}

// release is the pair previous -> f's pinned release, as patch-it's hosted
// ingestion service extracts it. A changelog's Extract reads only the newer
// release's own section, so previous is named, not read.
func release(t *testing.T, f File, saved, previous string) []event.Event {
	t.Helper()
	env := kittest.Env(t, kittest.Pages{f.URL(): saved})
	p := Plugin{File: f}
	to, err := p.ReadAt(context.Background(), env, plugin.Version{ID: f.Release, Ref: f.Tag})
	if err != nil {
		t.Fatal(err)
	}
	from := plugin.Snapshot{Version: plugin.Version{ID: previous, Ref: f.TagPrefix + previous}}
	return kittest.Extracted(t, env, p, from, to)
}

// Every bullet of each release that is breaking or a deprecation but isn't
// plain enough is dropped, not guessed: pinned here with why.
func TestUnsureBulletsAreDropped(t *testing.T) {
	for _, c := range []struct {
		file    File
		saved   string
		dropped []string // the start of each dropped bullet, in order
	}{
		{Textual0860, "testdata/textual-v0.86.0-CHANGELOG.md", []string{
			"Breaking change: To improve consistency", // a look change, no names
		}},
		{Urllib3200, "testdata/urllib3-2.0.0-CHANGES.rst", []string{
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
		var drops []Finding
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

// A Go module's events name the module as code imports it at that major.
func TestAGoModuleIsNamedByItsMajor(t *testing.T) {
	f := File{Package: "github.com/acme/acme-go", Ecosystem: "go"}
	for release, want := range map[string]string{"73.0.0": "github.com/acme/acme-go/v73", "1.4.0": "github.com/acme/acme-go", "0.31.0": "github.com/acme/acme-go"} {
		if got := f.At(release).installed(); got != want {
			t.Errorf("%s: %s, want %s", release, got, want)
		}
	}
}

// A release's notes on GitHub are read whole: any heading level starts a
// section, a code fence is skipped, and notes for another tag or none at
// all are never read as "no breaking changes".
func TestReleaseNotesAreReadWhole(t *testing.T) {
	f := File{Package: "acme", Repo: "acme/acme", TagPrefix: "v", Notes: true}.At("2.0.0")
	body := "## BREAKING CHANGES\\n\\n* `acme.old()` has been removed\\n\\n```\\n- `acme.example()` has been removed\\n```\\n\\n### Features\\n\\n* add `acme.new()`\\n"
	_, fs, err := f.Read([]byte(`{"tag_name": "v2.0.0", "name": "v2.0.0", "body": "` + body + `"}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 || fs[0].Dropped != "" || strings.Join(fs[0].Change.Symbols, ",") != "acme.old" {
		t.Errorf("findings %+v", fs)
	}
	if f.Cite() != "https://github.com/acme/acme/releases/tag/v2.0.0" || f.URL() != "https://api.github.com/repos/acme/acme/releases/tags/v2.0.0" {
		t.Errorf("cite %s, url %s", f.Cite(), f.URL())
	}
	if _, _, err := f.Read([]byte(`{"tag_name": "v1.9.0", "body": "x"}`)); err == nil {
		t.Error("another release's notes: want an error")
	}
	if _, _, err := f.Read([]byte(`{"tag_name": "v2.0.0", "body": ""}`)); !errors.Is(err, plugin.ErrNothingToRead) {
		t.Errorf("no notes: %v, want plugin.ErrNothingToRead", err)
	}
}

// A rename whose new name the same release renames again or removes points
// to where the chain ends: never to a name the release took away.
func TestRenamesAreFollowedToTheirEnd(t *testing.T) {
	f := File{Package: "acme", Repo: "acme/acme", TagPrefix: "v", Notes: true}.At("2.0.0")
	body := "## BREAKING CHANGES\\n\\n" +
		"* `acme.a()` has been renamed to `acme.b()`\\n* `acme.b()` has been renamed to `acme.c()`\\n" +
		"* `acme.x()` has been renamed to `acme.y()`\\n* `acme.y()` has been removed\\n" +
		"* `acme.p()` has been renamed to `acme.q()`\\n* `acme.q()` has been renamed to `acme.p()`\\n"
	_, fs, err := f.Read([]byte(`{"tag_name": "v2.0.0", "body": "` + body + `"}`))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range fs {
		got = append(got, strings.Join(f.Change.Symbols, ",")+">"+f.Change.Replacement)
	}
	want := "acme.a>acme.c acme.b>acme.c acme.x> acme.y> acme.p> acme.q>"
	if strings.Join(got, " ") != want {
		t.Errorf("got  %s\nwant %s", strings.Join(got, " "), want)
	}
}
