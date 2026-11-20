package members_test

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"

	"github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide"
	. "github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/members" // the reader under test
)

// The guide at tag v2.0.0 (saved 2026-11-04) in, every event out.
func TestGolden(t *testing.T) {
	got := kittest.Ingest(t, migrationguide.Plugin{Guide: NumPy2}, kittest.Pages{
		NumPy2.URL(): "testdata/numpy_2_0_migration_guide-v2.0.0.rst",
	})
	kittest.Golden(t, "../testdata/golden/numpy-2.0.0.json", got)
}
