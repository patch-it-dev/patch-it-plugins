package datadog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/adapter"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/openapitest"
	"gopkg.in/yaml.v3"
)

// fakeDatadogAdapter serves the real, trimmed spec excerpts saved under
// testdata/ (a saved copy of the source, not invented JSON), converted from
// YAML to JSON the same way Datadog.Fetch does.
type fakeDatadogAdapter struct{}

func (fakeDatadogAdapter) API() string { return "datadog/api" }

func (fakeDatadogAdapter) LatestTag(context.Context) (string, error) { return "2.61.0", nil }

func (fakeDatadogAdapter) Fetch(_ context.Context, tag string) (adapter.Spec, error) {
	sourceURL := "https://raw.githubusercontent.com/DataDog/datadog-api-client-python/" +
		tag + "/.generator/schemas/v2/openapi.yaml"
	body, err := os.ReadFile(fmt.Sprintf("testdata/datadog-%s.yaml", tag))
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("no saved copy for tag %s: %w", tag, err)
	}
	var doc any
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return adapter.Spec{}, err
	}
	content, err := json.Marshal(doc)
	if err != nil {
		return adapter.Spec{}, err
	}
	return adapter.Spec{Content: content, Version: tag, SourceURL: sourceURL}, nil
}

// The golden test and the pair fixture in one: testdata/datadog-2.60.0.yaml
// and datadog-2.61.0.yaml are real, trimmed excerpts of
// datadog-api-client-python's bundled spec at those two tags (checked
// 2026-11-13). Between them, PR #4045 removed queried_tags_window_seconds and
// related_asset_tags from TagIndexingRuleDynamicTags — a real historical
// breaking change, not an invented one. This is the pair
// the pair 2.60.0 → 2.61.0 recreates.
func TestDatadogPairRecreatesTheRealPropertyRemoval(t *testing.T) {
	ctx := context.Background()
	s := kittest.NewMem()

	rejected, err := kittest.Pair(ctx, openapitest.Env(s), s, openapi.Plugin{Adapter: fakeDatadogAdapter{}}, "2.60.0", "2.61.0")
	if err != nil || rejected {
		t.Fatalf("pair: rejected=%v err=%v", rejected, err)
	}

	events, err := s.All()
	if err != nil {
		t.Fatalf("reading events: %v", err)
	}
	got := map[string]string{}
	for _, e := range events {
		got[e.Subject.Property] = string(e.Change)
	}
	want := map[string]string{
		"queried_tags_window_seconds": "breaking",
		"related_asset_tags":          "breaking",
	}
	if len(got) != len(want) {
		t.Fatalf("stored %v, want exactly %v", got, want)
	}
	for prop, changeType := range want {
		if got[prop] != changeType {
			t.Errorf("%s: got %q, want %q", prop, got[prop], changeType)
		}
	}

	// Acceptance criteria: every event carries source_url and a non-empty
	// source_excerpt, and is placed on the real tags, not stand-ins.
	for _, e := range events {
		if e.API != "datadog/api" || e.Version != "2.61.0" || e.PreviousVersion != "2.60.0" {
			t.Errorf("event %s: api=%s version=%s previous_version=%s, want datadog/api 2.61.0 from 2.60.0",
				e.Subject.Property, e.API, e.Version, e.PreviousVersion)
		}
		if e.SourceURL == "" || e.SourceExcerpt == "" {
			t.Errorf("event %s: missing provenance", e.Subject.Property)
		}
	}

	// Reading the same pair again must not duplicate the events.
	if _, err := kittest.Pair(ctx, openapitest.Env(s), s, openapi.Plugin{Adapter: fakeDatadogAdapter{}}, "2.60.0", "2.61.0"); err != nil {
		t.Fatalf("second read of the pair: %v", err)
	}
	if again, _ := s.All(); len(again) != len(events) {
		t.Errorf("reading the pair again added events: %d -> %d", len(events), len(again))
	}
}
