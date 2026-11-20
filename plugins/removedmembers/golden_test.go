package removedmembers

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
)

// The guide at tag v2.0.0 (saved 2026-11-04) in, every event out.
func TestGolden(t *testing.T) {
	got := kittest.Ingest(t, Plugin{Guide: NumPy2}, kittest.Pages{
		NumPy2.URL(): "testdata/numpy_2_0_migration_guide-v2.0.0.rst",
	})
	kittest.Golden(t, "testdata/golden/numpy-2.0.0.json", got)
}
