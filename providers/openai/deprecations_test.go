package openai

import (
	"os"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/noticestest"
)

const deprecationsPage = "testdata/openai-deprecations-2026-11-21.md"

// The golden test: every endpoint the page names by its path, upcoming
// (/v1/prompts, /v1/audio/translations) and past (/v1/edits, /v1/engines, …).
func TestDeprecationsGolden(t *testing.T) {
	noticestest.Golden(t, Deprecations, deprecationsPage)
}

// The pinned drop test: the "Feature and endpoint deprecations" table's rows
// that name a feature in words ("Agent Builder", "Evals dashboard, API, and
// graders", "Self-serve fine-tuning job creation", "`service_tier: fast`"),
// and a model row, are no endpoint. Only the two rows that start with a
// path are read.
func TestDeprecationsDropsFeaturesInWords(t *testing.T) {
	b, err := os.ReadFile(deprecationsPage)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	i, j := strings.Index(s, "## Feature and endpoint deprecations"), strings.Index(s, "## Model deprecation notice periods")
	if i < 0 || j < i {
		t.Fatal("the saved page's shape changed")
	}
	found, err := Deprecations.Find([]byte(s[i:j]))
	if err != nil {
		t.Fatal(err)
	}
	got := found.(Endpoints)
	if len(got) != 2 || got[0].Path != "/v1/prompts" || got[1].Path != "/v1/audio/translations" {
		t.Fatalf("%+v, want /v1/prompts and /v1/audio/translations only", got)
	}
	if got[0].Replacement != "" || got[1].Replacement != "/v1/realtime/translations" {
		t.Errorf("replacements %q, %q: want none (prose) and /v1/realtime/translations", got[0].Replacement, got[1].Replacement)
	}
}
