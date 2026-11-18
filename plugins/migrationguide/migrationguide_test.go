package migrationguide

import (
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/sdkguide"
)

func TestAMigrationGuideStoresOneEventPerRemovedParameter(t *testing.T) {
	// Only the v1.0.0 tag is saved: fetching anything else fails the test, and
	// kittest's second run (the tag never changes) must store nothing new.
	events := kittest.Ingest(t, Plugin{Guide: sdkguide.AnthropicV1}, kittest.Pages{
		sdkguide.AnthropicV1.URL(): "../../sdkguide/testdata/anthropic-MIGRATION-v1.0.0.md",
	})
	var params []string
	for _, e := range events {
		params = append(params, e.Subject.Parameter)
	}
	if strings.Join(params, ",") != "temperature,top_p,top_k" {
		t.Errorf("stored %v", params)
	}
}
