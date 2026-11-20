package notices

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/plugins/notices/feed"
)

// The two feeds as saved on 2026-11-14 (feed/testdata/README.md), each through the
// whole plugin: live feed and snapshot are the same copy, so the events cite
// the snapshot, and kittest's second run over the unchanged feed asks for no
// second snapshot and stores nothing new.
func TestFeedsGolden(t *testing.T) {
	for _, c := range []struct {
		feed feed.Feed
		file string
	}{
		{feed.GitHubActions, "feed/testdata/github-changelog-actions.rss.xml"},
		{feed.VertexAIModels, "feed/testdata/vertex-ai-generative-ai.atom.xml"},
		{feed.CloudflareAPI, "feed/testdata/cloudflare-api-deprecations.rss.xml"},
	} {
		t.Run(c.feed.API, func(t *testing.T) {
			saved := kittest.Pages{c.feed.URL: c.file}
			archive := &kittest.Archive{T: t, Pages: saved}
			got := kittest.IngestWith(t, kittest.Env(t, saved), Plugin{Page: c.feed, Archiver: archive})
			if archive.Calls != 1 {
				t.Errorf("%d snapshots, want 1", archive.Calls)
			}
			kittest.Golden(t, "testdata/golden/"+c.feed.API+".json", got)
		})
	}
}
