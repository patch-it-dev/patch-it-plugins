// Package noticestest is what the tests of providers' notices sources
// share: a saved page through the whole plugin, against its golden file.
package noticestest

import (
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices"
)

// Golden reads the page saved as file through the whole plugin, guarded by
// its registered source's Shape (so the anchors it declares must be on the
// saved page too), and compares every event with
// testdata/golden/<source id, / as ->.json. The archive
// serves the same copy as the snapshot, so the events cite it; kittest's
// second read of the unchanged page asks for no second snapshot and stores
// nothing new.
func Golden(t *testing.T, page notices.Page, file string) {
	t.Helper()
	GoldenFile(t, page, file, "testdata/golden/"+strings.ReplaceAll(page.Slug(), "/", "-")+".json")
}

// GoldenFile is Golden against another golden file: a page's earlier layout,
// kept as a case while it still reads (testdata/layouts/), since a source has
// one golden under testdata/golden/.
func GoldenFile(t *testing.T, page notices.Page, file, golden string) {
	t.Helper()
	saved := kittest.Pages{page.PageURL(): file}
	archive := &kittest.Archive{T: t, Pages: saved}
	src, ok := plugin.LookupSource(page.Slug())
	if !ok {
		t.Fatalf("%s isn't registered", page.Slug())
	}
	p := notices.Plugin{Page: page, Archiver: archive}.WithShape(src.Source.Shape).(plugin.Living)
	got := kittest.Page(t, kittest.Env(t, saved), p)
	if archive.Calls != 1 {
		t.Errorf("%d snapshots, want 1: the second read's page is unchanged", archive.Calls)
	}
	kittest.Golden(t, golden, got)
}
