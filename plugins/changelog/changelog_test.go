package changelog

import (
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
)

// Two real changelogs at their release tags (testdata/README.md) in, every
// event out.
func TestGolden(t *testing.T) {
	for _, c := range []struct {
		file  File
		saved string
	}{
		{Textual0860, "testdata/textual-v0.86.0-CHANGELOG.md"},
		{Urllib3200, "testdata/urllib3-2.0.0-CHANGES.rst"},
	} {
		t.Run(c.file.Package, func(t *testing.T) {
			got := kittest.Ingest(t, Plugin{File: c.file}, kittest.Pages{c.file.URL(): c.saved})
			kittest.Golden(t, "testdata/golden/"+c.file.Package+"-"+c.file.Release+".json", got)
		})
	}
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
