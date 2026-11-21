package members_test

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"

	"github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide"
	. "github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/members" // the reader under test
)

// The guide at tag v2.0.0 (saved 2026-11-04) in, every event out: the pair
// from 1.26.4, the release before on PyPI.
func TestGolden(t *testing.T) {
	got := release(t, NumPy, "testdata/numpy_2_0_migration_guide-v2.0.0.rst", "1.26.4", "2.0.0")
	kittest.Golden(t, "../testdata/golden/numpy-2.0.0.json", got)
}

// release is the pair previous -> rel, the first release of a major, over
// the guide saved as file, as patch-it's hosted ingestion service extracts
// it. previous isn't a major's first release: it has no guide, and nothing
// is fetched for it.
func release(t *testing.T, g Guide, file, previous, rel string) []event.Event {
	t.Helper()
	return kittest.Versions(t, guideEnv(t, g, file, rel), migrationguide.Plugin{Guide: g}, plugin.Version{ID: previous}, plugin.Version{ID: rel})
}

// newrelicCommit is the docs-website commit New Relic's saved release note
// was fetched at, and newrelicCommits GitHub's answer naming it as the
// newest commit that changed the note.
const (
	newrelicCommit  = "f0b4f817bf9a7b76a78e26609369a8a7ce764c8e"
	newrelicCommits = "testdata/github-commits-python-agent-110000.json"
)

// guideEnv serves file as g's guide for rel: at rel's tag, or (Untagged) at
// the commit GitHub names for it.
func guideEnv(t *testing.T, g Guide, file, rel string) plugin.Env {
	t.Helper()
	major := versions.Major(rel)
	pages := kittest.Pages{g.RawURL(g.TagOf(rel), major): file}
	if g.Untagged {
		pages = kittest.Pages{g.CommitsURL(major): newrelicCommits, g.RawURL(newrelicCommit, major): file}
	}
	return kittest.Env(t, pages)
}
