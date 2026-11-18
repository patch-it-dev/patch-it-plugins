package migrationguide

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/sdkguide"
)

// The guide at tag v1.0.0 (saved 2026-10-17) in, every event out.
func TestGolden(t *testing.T) {
	got := kittest.Ingest(t, Plugin{Guide: sdkguide.AnthropicV1}, kittest.Pages{
		sdkguide.AnthropicV1.URL(): "../../sdkguide/testdata/anthropic-MIGRATION-v1.0.0.md",
	})
	kittest.Golden(t, "testdata/golden/anthropic-1.0.0.json", got)
}
