package google

import (
	"os"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/noticestest"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/retire"
)

// section is the saved page from start up to end (to the end of the page
// when end is ""), as a page of its own.
func section(t *testing.T, file, start, end string) string {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	i := strings.Index(s, start)
	if i < 0 {
		t.Fatalf("%s: the saved page's shape changed", file)
	}
	if end == "" {
		return s[i:]
	}
	j := strings.Index(s[i+1:], end)
	if j < 0 {
		t.Fatalf("%s: the saved page's shape changed", file)
	}
	return s[i : i+1+j]
}

// readGoogle is a first read of page, served as content, through the kind.
func readGoogle(t *testing.T, page retire.Page, content string) int {
	t.Helper()
	f := t.TempDir() + "/page"
	if err := os.WriteFile(f, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	saved := kittest.Pages{page.URL: f}
	return len(kittest.Page(t, kittest.Env(t, saved), notices.Plugin{Page: page, Archiver: &kittest.Archive{T: t, Pages: saved}}))
}

// The pinned drop test for the Gemini API page: the Gemini 4 models (no
// shutdown date announced) and the managed agents, whose deprecated table
// has real dates but names agents, not models, next to the two Gemini 3
// models shut down already. Only those two are events; the agents' table
// renamed to "Model" would give two more, so the header is what keeps it
// out.
func TestGeminiDropsModelsWithNoDateAndAgents(t *testing.T) {
	const file = "testdata/gemini-deprecations.md"
	current := section(t, file, "## Gemini 4 models", "## Gemini 3 models")
	retired := section(t, file, "### Deprecated models  {% id=\"gemini-3-deprecated-models\" %}", "## Gemini 2.5 Pro models")
	agents := section(t, file, "## Managed agents", "")
	if n := readGoogle(t, GeminiModels, current+retired+agents); n != 2 {
		t.Errorf("%d events, want the 2 shut-down Gemini 3 models only", n)
	}
	if _, err := retire.Parse(GeminiModels.Format.Pipes([]byte(agents)), GeminiModels.Keep); err == nil {
		t.Error("the agents' tables read as retirements")
	}
	asModels := strings.ReplaceAll(agents, "  * Agent\n", "  * Model\n")
	if rs, err := retire.Parse(GeminiModels.Format.Pipes([]byte(asModels)), GeminiModels.Keep); err != nil || len(rs) != 2 {
		t.Errorf("the agents' table with a Model header: %d, %v; the drop proves nothing", len(rs), err)
	}
}

// The pinned drop test for Vertex AI: a GA model's "June 28, 2027 or
// later" is how long it's kept at least, not a retirement. The Gemini image
// models' table has four such rows and one dated retirement
// (gemini-2.5-flash-image, March 15, 2027): one event. The same table with
// the dates plain would give five.
func TestVertexDropsAMinimumLifetime(t *testing.T) {
	ga := section(t, "testdata/vertex-model-versions.html", `<h3 id="gemini-image-models"`, `<h3 id="gemini-omni-models"`)
	if n := readGoogle(t, VertexModelRetirements, ga); n != 1 {
		t.Errorf("%d events, want the one dated retirement only", n)
	}
	plain := strings.ReplaceAll(ga, " or later", "")
	if rs, err := retire.Parse(VertexModelRetirements.Format.Pipes([]byte(plain)), VertexModelRetirements.Keep); err != nil || len(rs) < 2 {
		t.Errorf("the same table with plain dates: %d, %v; the drop proves nothing", len(rs), err)
	}
}

// Google's Gemini dates are the earliest a model may go: the description
// says so, and an exact table (Vertex AI's) doesn't.
func TestGeminiDatesAreTheEarliest(t *testing.T) {
	saved := kittest.Pages{GeminiModels.URL: "testdata/gemini-deprecations.md"}
	for _, e := range kittest.Page(t, kittest.Env(t, saved), notices.Plugin{Page: GeminiModels, Archiver: &kittest.Archive{T: t, Pages: saved}}) {
		if !strings.Contains(e.Description, "the earliest date Google gives") {
			t.Fatalf("%s: %q", e.Subject.Name, e.Description)
		}
	}
}

// Google's saved pages (2026-11-21, testdata/README.md) in, every event out.
func TestModelsGolden(t *testing.T) {
	noticestest.Golden(t, GeminiModels, "testdata/gemini-deprecations.md")
	noticestest.Golden(t, VertexModelRetirements, "testdata/vertex-model-versions.html")
}
