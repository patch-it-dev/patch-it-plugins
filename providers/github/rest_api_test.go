package github

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/adapter"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/openapitest"
)

// fakeGitHubAdapter serves the real, trimmed spec excerpts saved under
// testdata/ (a saved copy of the source, not invented JSON). Unlike
// Datadog's, github's spec is already JSON, so there's no conversion step.
type fakeGitHubAdapter struct{}

func (fakeGitHubAdapter) API() string { return "github/rest-api" }

func (fakeGitHubAdapter) LatestTag(context.Context) (string, error) { return "2026-03-10", nil }

func (fakeGitHubAdapter) Fetch(_ context.Context, tag string) (adapter.Spec, error) {
	sourceURL := "https://raw.githubusercontent.com/github/rest-api-description/main/" +
		"descriptions/api.github.com/api.github.com." + tag + ".json"
	body, err := os.ReadFile(fmt.Sprintf("testdata/github-%s.json", tag))
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("no saved copy for tag %s: %w", tag, err)
	}
	return adapter.Spec{Content: body, Version: tag, SourceURL: sourceURL}, nil
}

// The golden test and the pair fixture in one: testdata/github-2022-11-28.json
// and github-2026-03-10.json are real, trimmed excerpts of github's own bundled
// REST API description at those two dated API versions (checked 2026-11-14).
// Between them, GitHub's documented breaking change for 2026-03-10 ("Remove
// deprecated rate property from rate limit endpoint") dropped the required
// `rate` property from the `rate-limit-overview` schema — a real historical
// breaking change, not an invented one. This is the pair
// the pair 2022-11-28 → 2026-03-10 recreates.
func TestGitHubPairRecreatesTheRealPropertyRemoval(t *testing.T) {
	ctx := context.Background()
	s := kittest.NewMem()

	rejected, err := kittest.Pair(ctx, openapitest.Env(s), s, openapi.Plugin{Adapter: fakeGitHubAdapter{}}, "2022-11-28", "2026-03-10")
	if err != nil || rejected {
		t.Fatalf("pair: rejected=%v err=%v", rejected, err)
	}

	events, err := s.All()
	if err != nil {
		t.Fatalf("reading events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("stored %d events, want exactly 1 (the rate property removal)", len(events))
	}
	e := events[0]
	if e.Subject.Property != "rate" || string(e.ChangeType) != "breaking" {
		t.Errorf("got property=%q change_type=%q, want rate/breaking", e.Subject.Property, e.ChangeType)
	}

	// Acceptance criteria: the event carries source_url and a non-empty
	// source_excerpt, and is placed on the real dated versions, not stand-ins.
	if e.API != "github/rest-api" || e.Version != "2026-03-10" || e.PreviousVersion != "2022-11-28" {
		t.Errorf("event %s: api=%s version=%s previous_version=%s, want github/rest-api 2026-03-10 from 2022-11-28",
			e.Subject.Property, e.API, e.Version, e.PreviousVersion)
	}
	if e.SourceURL == "" || e.SourceExcerpt == "" {
		t.Errorf("event %s: missing provenance", e.Subject.Property)
	}

	// Reading the same pair again must not duplicate the event.
	if _, err := kittest.Pair(ctx, openapitest.Env(s), s, openapi.Plugin{Adapter: fakeGitHubAdapter{}}, "2022-11-28", "2026-03-10"); err != nil {
		t.Fatalf("second read of the pair: %v", err)
	}
	if again, _ := s.All(); len(again) != len(events) {
		t.Errorf("reading the pair again added events: %d -> %d", len(events), len(again))
	}
}
