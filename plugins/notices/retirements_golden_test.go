package notices

import (
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/plugins/notices/retire"
)

// Both providers' saved pages (2026-10-13) in, every event out. The archive
// serves the same copy as the snapshot, so the events cite it; a second run
// over the unchanged page asks for no snapshot.
func TestRetirementsGolden(t *testing.T) {
	for file, page := range map[string]retire.Page{
		"openai-deprecations.md":    retire.OpenAI,
		"anthropic-deprecations.md": retire.Anthropic,
	} {
		t.Run(page.API, func(t *testing.T) {
			saved := kittest.Pages{page.URL: "retire/testdata/" + file}
			archive := &kittest.Archive{T: t, Pages: saved}
			got := kittest.IngestWith(t, kittest.Env(t, saved), Plugin{Page: page, Archiver: archive})
			if archive.Calls != 1 {
				t.Errorf("%d snapshots, want 1: the second run's page is unchanged", archive.Calls)
			}
			kittest.Golden(t, "testdata/golden/"+strings.ReplaceAll(page.API, "/", "-")+".json", got)
		})
	}
}
