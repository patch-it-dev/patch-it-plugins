package migrationguide

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/sdkguide"
)

// The guide at tag v1.0.0 (saved 2026-10-17) in, every event out: the pair
// from 0.75.0, the release before on PyPI.
func TestGolden(t *testing.T) {
	kittest.Golden(t, "testdata/golden/anthropic-1.0.0.json", anthropicV1(t, anthropicGuideV1))
}

// anthropicV1 is g's pair 0.75.0 -> 1.0.0 over the saved v1.0.0 guide, as
// patch-it's hosted ingestion service extracts it. 0.75.0 isn't a new major:
// it has no guide, and nothing is fetched for it.
func anthropicV1(t *testing.T, g sdkguide.Guide, more ...kittest.Pages) []event.Event {
	t.Helper()
	pages := kittest.Pages{anthropicGuideV1.URL(): "sdkguide/testdata/anthropic-MIGRATION-v1.0.0.md"}
	for _, m := range more {
		for k, v := range m {
			pages[k] = v
		}
	}
	return kittest.Versions(t, kittest.Env(t, pages), Plugin{Guide: g},
		plugin.Version{ID: "0.75.0", Ref: "v0.75.0"}, plugin.Version{ID: "1.0.0", Ref: "v1.0.0"})
}

// anthropicGuideV1 is anthropic-sdk-python's guide pinned to 1.0.0: what the
// golden pins. Only a test names a release.
var anthropicGuideV1 = sdkguide.Anthropic.At(1)
