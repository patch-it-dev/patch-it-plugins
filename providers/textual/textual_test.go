package textual

import (
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/kinds/changelog"
	"github.com/patch-it-dev/patch-it-plugins/kinds/changelog/changelogtest"
)

// The real changelog at a release tag (testdata/README.md) in, every
// event out: the pair from the release before (on PyPI) to the pinned one.
func TestTextualGolden(t *testing.T) {
	for _, c := range []struct {
		file     changelog.File
		saved    string
		previous string
	}{
		{Textual.At("0.86.0"), "testdata/textual-v0.86.0-CHANGELOG.md", "0.85.2"},
	} {
		t.Run(c.file.Package, func(t *testing.T) {
			got := changelogtest.Release(t, c.file, c.saved, c.previous)
			kittest.Golden(t, "testdata/golden/"+c.file.Package+"-"+c.file.Release+".json", got)
		})
	}
}

// Every bullet of each release that is breaking or a deprecation but isn't
// plain enough is dropped, not guessed: pinned here with why.
func TestTextualUnsureBulletsAreDropped(t *testing.T) {
	for _, c := range []struct {
		file    changelog.File
		saved   string
		dropped []string // the start of each dropped bullet, in order
	}{
		{Textual.At("0.86.0"), "testdata/textual-v0.86.0-CHANGELOG.md", []string{
			"Breaking change: To improve consistency", // a look change, no names
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
