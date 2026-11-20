package feed

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
)

// Code written against the old path still builds and runs the same feed
// with the same events.
func TestTheOldPathStillWorks(t *testing.T) {
	saved := kittest.Pages{GitHubActions.URL: "../notices/feed/testdata/github-changelog-actions.rss.xml"}
	archive := &kittest.Archive{T: t, Pages: saved}
	got := kittest.IngestWith(t, kittest.Env(t, saved), Plugin{Feed: GitHubActions, Archiver: archive})
	kittest.CompareGolden(t, "../notices/testdata/golden/github-actions.json", got)
}
