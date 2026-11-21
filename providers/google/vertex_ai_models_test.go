package google

import (
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/feed"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/noticestest"
)

// The feed as saved on 2026-11-14 (testdata/README.md) through the whole
// plugin.
func TestVertexAIModelsGolden(t *testing.T) {
	noticestest.Golden(t, VertexAIModels, "testdata/vertex-ai-generative-ai.atom.xml")
}

// What the feed's marked entries came to, kept and dropped: the golden files
// pin the events, this pins that the unsure ones were dropped, not guessed.
func TestVertexAIModelsUnsureEntriesAreDropped(t *testing.T) {
	for _, c := range []struct {
		feed    feed.Feed
		file    string
		dropped []string // a substring of each drop reason, in order
	}{
		{VertexAIModels, "testdata/vertex-ai-generative-ai.atom.xml", []string{
			"Vertex AI Extensions deprecation",
			"Anthropic's Claude 3 Haiku", // a product name in prose, not a model id
			"Anthropic's Claude 3.5 Haiku",
		}},
	} {
		fs, err := c.feed.Findings(kittest.Read(t, c.file))
		if err != nil {
			t.Fatal(err)
		}
		var drops []string
		for _, f := range fs {
			if f.Dropped != "" {
				drops = append(drops, f.Entry+" "+f.Dropped)
			}
		}
		if len(drops) != len(c.dropped) {
			t.Errorf("%s: dropped %d, want %d:\n%s", c.feed.API, len(drops), len(c.dropped), strings.Join(drops, "\n"))
			continue
		}
		for i, want := range c.dropped {
			if !strings.Contains(drops[i], want) {
				t.Errorf("%s: drop %d is %q, want %q", c.feed.API, i, drops[i], want)
			}
		}
	}
}
