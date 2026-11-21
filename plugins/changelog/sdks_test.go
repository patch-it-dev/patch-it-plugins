package changelog

import (
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
)

// sdkCase is one SDK's changelog, saved at a release's tag
// (testdata/README.md), and the release its golden pins: the pair from the
// release before (on the registry, or among the repo's tags) to it. drop is
// another release in the same saved file (for Notes, in dropped, its own
// saved release) whose notes name nothing removed: it must read as no event.
type sdkCase struct {
	file           File
	release, prev  string
	saved, dropped string
	drop           string
}

// checkSDKs runs each case's golden, its drop test and its registration.
func checkSDKs(t *testing.T, cases []sdkCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.file.API, func(t *testing.T) {
			kittest.Registered(t, Plugin{File: c.file})
			got := release(t, c.file.At(c.release), "testdata/"+c.saved, c.prev)
			if len(got) == 0 {
				t.Fatal("no events: the golden must pin a release with some")
			}
			kittest.Golden(t, "testdata/golden/"+strings.ReplaceAll(c.file.API, "/", "-")+".json", got)

			saved := c.saved
			if c.dropped != "" {
				saved = c.dropped
			}
			_, fs, err := c.file.At(c.drop).Read(kittest.Read(t, "testdata/"+saved))
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range fs {
				if f.Dropped == "" {
					t.Errorf("drop test %s: an event from %q", c.drop, f.Bullet.Text)
				}
			}
		})
	}
}
