package sdkguide

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

func guide(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/anthropic-MIGRATION-v1.0.0.md")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// anthropic-sdk-python's MIGRATION.md at v1.0.0 (saved 2026-10-17): the three
// sampling parameters, not the beta-only `output_format` row.
func TestAnthropicV1RemovesTheSamplingParameters(t *testing.T) {
	rs, err := Parse(guide(t), AnthropicV1)
	if err != nil {
		t.Fatal(err)
	}
	var params []string
	for _, r := range rs {
		params = append(params, r.Param)
		if !strings.HasPrefix(r.Instead, "Remove them.") || !strings.Contains(r.Instead, "`extra_body`") {
			t.Errorf("%s: instead %q", r.Param, r.Instead)
		}
		if !strings.HasPrefix(r.Excerpt, "## Removed: deprecated request parameters") {
			t.Errorf("excerpt starts %q", r.Excerpt[:40])
		}
	}
	if strings.Join(params, ",") != "temperature,top_p,top_k" {
		t.Errorf("params %v", params)
	}
}

func TestAnotherMajorsGuideOrNoRowIsAnError(t *testing.T) {
	v2 := AnthropicV1
	v2.Major = 2
	if _, err := Parse(guide(t), v2); !errors.Is(err, plugin.ErrNothingToRead) {
		t.Errorf("the v1 guide read as v2's: %v, want plugin.ErrNothingToRead", err)
	}
	if _, err := Parse([]byte("# Migrating to v1\n\nNothing removed.\n"), AnthropicV1); !errors.Is(err, plugin.ErrNothingToRead) {
		t.Errorf("no table must be plugin.ErrNothingToRead, not zero events: %v", err)
	}
}

func TestEventsAreValidAndCiteTheTag(t *testing.T) {
	rs, _ := Parse(guide(t), AnthropicV1)
	events, errs := Events(AnthropicV1, rs, "0.75.0", time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC))
	if len(errs) != 0 || len(events) != 3 {
		t.Fatalf("%d events, %v", len(events), errs)
	}
	e := events[0]
	if e.API != "anthropic/python-sdk" || e.Version != "1.0.0" || !e.Breaking ||
		e.Subject.Type != event.SubjectPackageRelease || e.Subject.Package != "anthropic" || e.Subject.Change != event.PackageRemovedParameter ||
		e.Subject.Parameter != "temperature" || strings.Join(e.Subject.Symbols, ",") != "messages.create,messages.stream,messages.parse" ||
		e.SourceURL != "https://raw.githubusercontent.com/anthropics/anthropic-sdk-python/v1.0.0/MIGRATION.md" {
		t.Errorf("%+v", e)
	}
	if !strings.Contains(e.MigrationHint, "extra_body") || !strings.HasSuffix(e.MigrationHint, "/blob/v1.0.0/MIGRATION.md") {
		t.Errorf("hint %q", e.MigrationHint)
	}
}
