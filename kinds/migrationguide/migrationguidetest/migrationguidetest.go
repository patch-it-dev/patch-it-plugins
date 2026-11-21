// Package migrationguidetest is what the tests of migration-guide sources
// share: a guide saved as a file, served where the source reads it, and the
// pair to the major it describes, extracted as patch-it's hosted ingestion
// service does.
package migrationguidetest

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
	"github.com/patch-it-dev/patch-it-plugins/kinds/migrationguide"
	"github.com/patch-it-dev/patch-it-plugins/kinds/migrationguide/members"
)

// Release is the pair previous -> rel, the first release of a major, read
// through env, as patch-it's hosted ingestion service extracts it. previous
// isn't a major's first release: it has no guide, and nothing is fetched for
// it.
func Release(t *testing.T, env plugin.Env, g migrationguide.Guide, previous, rel string) []event.Event {
	t.Helper()
	return kittest.Versions(t, env, migrationguide.Plugin{Guide: g}, plugin.Version{ID: previous}, plugin.Version{ID: rel})
}

// GuideEnv serves file as g's guide for rel, at rel's tag.
func GuideEnv(t *testing.T, g members.Guide, file, rel string) plugin.Env {
	t.Helper()
	return kittest.Env(t, kittest.Pages{g.RawURL(g.TagOf(rel), versions.Major(rel)): file})
}

// UntaggedEnv serves file as an Untagged guide's for rel, at commit, and
// commits (a saved file) as GitHub's answer naming commit the newest that
// changed the guide.
func UntaggedEnv(t *testing.T, g members.Guide, file, rel, commits, commit string) plugin.Env {
	t.Helper()
	major := versions.Major(rel)
	return kittest.Env(t, kittest.Pages{g.CommitsURL(major): commits, g.RawURL(commit, major): file})
}
