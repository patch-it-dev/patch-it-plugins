package members_test

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"

	"github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide"
	. "github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/members" // the reader under test
)

// The guide at tag v2.0.0 (saved 2026-11-04) in, every event out: the pair
// from 1.26.4, the release before on PyPI.
func TestGolden(t *testing.T) {
	got := release(t, NumPy2, "testdata/numpy_2_0_migration_guide-v2.0.0.rst", "1.26.4")
	kittest.Golden(t, "../testdata/golden/numpy-2.0.0.json", got)
}

// release is the pair previous -> the release g documents, over the guide
// saved as file, as patch-it's hosted ingestion service extracts it.
// previous isn't the guide's release: it has no guide, and nothing is
// fetched for it.
func release(t *testing.T, g Guide, file, previous string) []event.Event {
	t.Helper()
	env := kittest.Env(t, kittest.Pages{g.URL(): file})
	return kittest.Versions(t, env, migrationguide.Plugin{Guide: g}, plugin.Version{ID: previous}, plugin.Version{ID: g.Release})
}
