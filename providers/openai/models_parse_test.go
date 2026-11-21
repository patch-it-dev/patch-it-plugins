package openai

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/retire"
)

// The batch OpenAI shuts down on 2026-10-23, every alias its own name, each
// with the replacement the page names, in both layouts of the page (saved
// 2026-10-13 and 2026-11-21).
func TestOpenAIsOctober23Batch(t *testing.T) {
	for _, c := range []struct{ file, heading, row string }{
		{"openai-deprecations-2026-10-13.md", "### 2026-04-22: Legacy GPT model snapshots", "`gpt-4-0613` \\| `gpt-4`"},
		{"openai-deprecations.md", "## Past deprecations", "| **Oct 23, 2026**"},
	} {
		t.Run(c.file, func(t *testing.T) { october23(t, c.file, c.heading, c.row) })
	}
}

func october23(t *testing.T, file, heading, row string) {
	got := byModel(t, Models, file)
	want := map[string]string{
		"gpt-3.5-turbo-0125": "gpt-5.6-terra", "gpt-3.5-turbo": "gpt-5.6-terra",
		"gpt-4-0613": "gpt-5.6-sol", "gpt-4": "gpt-5.6-sol",
		"gpt-4-1106-preview": "gpt-5.6-sol",
		"gpt-4-turbo":        "gpt-5.6-sol", "gpt-4-turbo-2024-04-09": "gpt-5.6-sol",
		"gpt-4.1-nano": "gpt-5.6-luna", "gpt-4.1-nano-2025-04-14": "gpt-5.6-luna",
		"gpt-4o-2024-05-13": "gpt-5.6-sol",
		"o1-2024-12-17":     "gpt-5.6-sol", "o1": "gpt-5.6-sol",
		"o1-pro-2025-03-19": "gpt-5.6-sol", "o1-pro": "gpt-5.6-sol",
		"o3-mini-2025-01-31": "gpt-5.6-sol", "o3-mini": "gpt-5.6-sol",
		"o4-mini-2025-04-16": "gpt-5.6-terra", "o4-mini": "gpt-5.6-terra",
	}
	n := 0
	for _, r := range got {
		if r.Date == "2026-10-23" {
			n++
			if want[r.Model] == "" {
				t.Errorf("unexpected in the batch: %s", r.Model)
			}
		}
	}
	if n != len(want) {
		t.Errorf("%d models on 2026-10-23, want %d", n, len(want))
	}
	for m, repl := range want {
		r, ok := got[m]
		if !ok || r.Date != "2026-10-23" || retire.First(r.Replacement) != repl {
			t.Errorf("%s: got %+v, want 2026-10-23 -> %s", m, r, repl)
		}
	}
	if !strings.Contains(got["o1-pro"].Replacement, "reasoning.mode: pro") {
		t.Errorf("the replacement's note is lost: %q", got["o1-pro"].Replacement)
	}
	gpt4 := got["gpt-4"].Excerpt
	if !strings.Contains(gpt4, heading) || !strings.Contains(gpt4, "| Shutdown date") || !strings.Contains(gpt4, row) {
		t.Errorf("the excerpt should be the heading, the header and the row:\n%s", gpt4)
	}
}

// Not text models, or not names code sends: image, audio, embeddings,
// fine-tuned bases, the legacy "-completions" aliases, endpoints.
func TestOnlyTextModelsCodeSendsAreKept(t *testing.T) {
	got := byModel(t, Models, "openai-deprecations.md")
	for _, m := range []string{"gpt-image-1", "ft-gpt-4", "gpt-4-0613-completions", "gpt-3.5-turbo-completions", "tts-1-hd-1106", "dall-e-2", "/v1/fine-tunes", "babbage-002", "text-embedding-ada-002"} {
		if _, ok := got[m]; ok {
			t.Errorf("%s should not be an event", m)
		}
	}
}

// A replacement that is itself retired is followed to one that works:
// gpt-3.5-turbo-0613 was told to move to gpt-3.5-turbo, which goes on 2026-10-23.
func TestARetiredReplacementIsFollowed(t *testing.T) {
	got := byModel(t, Models, "openai-deprecations.md")
	r := got["gpt-3.5-turbo-0613"]
	if retire.First(r.Replacement) != "gpt-5.6-terra" || len(r.Via) != 1 || r.Via[0] != "gpt-3.5-turbo" {
		t.Errorf("got %+v", r)
	}
	es, errs := retire.Events(Models, []retire.Retirement{r}, "https://web.archive.org/web/20261013115538/https://developers.openai.com/api/docs/deprecations.md", time.Now())
	if len(es) != 1 {
		t.Fatalf("events %d, errs %v", len(es), errs)
	}
	if h := es[0].MigrationHint; !strings.Contains(h, "gpt-5.6-terra") || !strings.Contains(h, "first named gpt-3.5-turbo") {
		t.Errorf("hint: %q", h)
	}
}

func TestEveryRealRetirementBecomesValidEvents(t *testing.T) {
	for _, c := range []struct {
		p    retire.Page
		file string
	}{{Models, "openai-deprecations.md"}, {Models, "openai-deprecations-2026-10-13.md"}} {
		rs, err := retire.Parse(read(t, c.file), c.p.Keep)
		if err != nil {
			t.Fatal(err)
		}
		events, errs := retire.Events(c.p, rs, "https://web.archive.org/web/1/x", time.Now())
		if len(errs) != 0 || len(events) != len(rs) {
			t.Errorf("%s: %d events from %d retirements, errors %v", c.p.API, len(events), len(rs), errs)
		}
	}
}

func read(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func byModel(t *testing.T, p retire.Page, file string) map[string]retire.Retirement {
	t.Helper()
	rs, err := retire.Parse(read(t, file), p.Keep)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]retire.Retirement{}
	for _, r := range rs {
		if _, dup := out[r.Model]; dup {
			t.Fatalf("%s listed twice", r.Model)
		}
		out[r.Model] = r
	}
	return out
}

// Since 2026-11 the page has a Status and a Notice date column. A model
// listed twice keeps the row with the latest notice, whatever the order, and
// a status that isn't a retirement (a postponed one) is not read as one.
func TestTheLatestNoticeWinsAndOnlyARetiringStatusCounts(t *testing.T) {
	const page = `## Upcoming deprecations

| Shutdown date    | Affected model   | Status     | Recommended replacement | Notice date  |
| ---------------- | ---------------- | ---------- | ----------------------- | ------------ |
| **Dec 11, 2026** | ` + "`gpt-5`" + `          | Deprecated | ` + "`gpt-6.1-sol`" + `           | Jun 11, 2026 |
| **Jan 6, 2027**  | ` + "`o3`" + `             | Postponed  | ` + "`gpt-6.1-sol`" + `           | Nov 1, 2026  |

## Past deprecations

| Shutdown date    | Affected model or feature | Listed replacement | Notice date  |
| ---------------- | ------------------------- | ------------------ | ------------ |
| **Mar 1, 2027**  | ` + "`gpt-5`" + `                   | ` + "`gpt-6-sol`" + `        | Aug 1, 2026  |
| **Dec 11, 2026** | ` + "`o3`" + `                      | ` + "`gpt-6.1-sol`" + `      | Jun 11, 2026 |
| **Oct 23, 2026** | ` + "`o1`" + `                      | ` + "`gpt-5.6-sol`" + `      | Apr 22, 2026 |
`
	rs, err := retire.Parse([]byte(page), Models.Keep)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]retire.Retirement{}
	for _, r := range rs {
		got[r.Model] = r
	}
	if r := got["gpt-5"]; r.Date != "2027-03-01" || retire.First(r.Replacement) != "gpt-6-sol" {
		t.Errorf("gpt-5: the August notice moved it to 2027-03-01, got %+v", r)
	}
	if r, ok := got["o3"]; ok {
		t.Errorf("o3: the latest notice postponed it, got %+v", r)
	}
	if r := got["o1"]; r.Date != "2026-10-23" {
		t.Errorf("o1: got %+v", r)
	}
}
