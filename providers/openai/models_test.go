package openai

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/noticestest"
)

// OpenAI's saved page (2026-10-13, testdata/README.md) in, every event out.
func TestModelsGolden(t *testing.T) {
	noticestest.Golden(t, Models, "testdata/openai-deprecations.md")
}
