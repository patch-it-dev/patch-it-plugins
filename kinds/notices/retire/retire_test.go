package retire

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

func read(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func byModel(t *testing.T, p Page, file string) map[string]Retirement {
	t.Helper()
	rs, err := Parse(read(t, file), p.Keep)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]Retirement{}
	for _, r := range rs {
		if _, dup := out[r.Model]; dup {
			t.Fatalf("%s listed twice", r.Model)
		}
		out[r.Model] = r
	}
	return out
}

// The batch OpenAI shuts down on 2026-10-23, every alias its own name, each
// with the replacement the page names (saved copy of the page, 2026-10-13).
func TestOpenAIsOctober23Batch(t *testing.T) {
	got := byModel(t, OpenAI, "openai-deprecations.md")
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
		if !ok || r.Date != "2026-10-23" || First(r.Replacement) != repl {
			t.Errorf("%s: got %+v, want 2026-10-23 -> %s", m, r, repl)
		}
	}
	if !strings.Contains(got["o1-pro"].Replacement, "reasoning.mode: pro") {
		t.Errorf("the replacement's note is lost: %q", got["o1-pro"].Replacement)
	}
	gpt4 := got["gpt-4"].Excerpt
	if !strings.Contains(gpt4, "### 2026-04-22: Legacy GPT model snapshots") || !strings.Contains(gpt4, "| Shutdown date") || !strings.Contains(gpt4, "`gpt-4-0613` \\| `gpt-4`") {
		t.Errorf("the excerpt should be the heading, the header and the row:\n%s", gpt4)
	}
}

// Not text models, or not names code sends: image, audio, embeddings,
// fine-tuned bases, the legacy "-completions" aliases, endpoints.
func TestOnlyTextModelsCodeSendsAreKept(t *testing.T) {
	got := byModel(t, OpenAI, "openai-deprecations.md")
	for _, m := range []string{"gpt-image-1", "ft-gpt-4", "gpt-4-0613-completions", "gpt-3.5-turbo-completions", "tts-1-hd-1106", "dall-e-2", "/v1/fine-tunes", "babbage-002", "text-embedding-ada-002"} {
		if _, ok := got[m]; ok {
			t.Errorf("%s should not be an event", m)
		}
	}
}

// A replacement that is itself retired is followed to one that works:
// gpt-3.5-turbo-0613 was told to move to gpt-3.5-turbo, which goes on 2026-10-23.
func TestARetiredReplacementIsFollowed(t *testing.T) {
	got := byModel(t, OpenAI, "openai-deprecations.md")
	r := got["gpt-3.5-turbo-0613"]
	if First(r.Replacement) != "gpt-5.6-terra" || len(r.Via) != 1 || r.Via[0] != "gpt-3.5-turbo" {
		t.Errorf("got %+v", r)
	}
	if h := hint("OpenAI", r); !strings.Contains(h, "gpt-5.6-terra") || !strings.Contains(h, "first named gpt-3.5-turbo") {
		t.Errorf("hint: %q", h)
	}
}

func TestAnthropicsRetirements(t *testing.T) {
	got := byModel(t, Anthropic, "anthropic-deprecations.md")
	want := map[string][2]string{
		"claude-sonnet-4-5-20250929": {"2026-11-30", "claude-sonnet-5-5"},
		"claude-opus-4-1-20250805":   {"2026-08-05", "claude-opus-4-8"},
		"claude-3-opus-20240229":     {"2026-01-05", "claude-opus-4-8"},
		"claude-3-5-sonnet-20241022": {"2025-10-28", "claude-sonnet-5-5"},
		"claude-3-haiku-20240307":    {"2026-04-20", "claude-haiku-4-5-20251001"},
	}
	for m, w := range want {
		if r := got[m]; r.Date != w[0] || First(r.Replacement) != w[1] {
			t.Errorf("%s: got %+v, want %v", m, r, w)
		}
	}
	if len(got) != 20 {
		t.Errorf("%d retired Claude models, want 20", len(got))
	}
	if _, ok := got["claude-mythos-preview"]; ok {
		t.Error("a model with no retirement date yet is not an event")
	}
}

func TestTableEdgeCases(t *testing.T) {
	page := strings.Join([]string{
		"### 2026-05-01: newer announcement",
		"| Shutdown date | Model | Recommended replacement |",
		"| --- | --- | --- |",
		"| Jun 1, 2026 | `gpt-4-x` \\| `gpt-4-x-alias` | `gpt-5.6-sol` |",
		"| To be announced | `gpt-4-later` | `gpt-5.6-sol` |",
		"",
		"### 2026-01-01: older announcement",
		"| Shutdown date | Model | Recommended replacement |",
		"| --- | --- | --- |",
		"| 2026-03-01 | `gpt-4-x` | `gpt-4o` |",
		"| 2026-03-01 | `gpt-4-later` | `gpt-4o` |",
		"",
		"| Shutdown date | System | Recommended replacement |",
		"| --- | --- | --- |",
		"| 2024-01-04 | `/v1/fine-tunes` | `/v1/fine_tuning/jobs` |",
	}, "\n")
	rs, err := Parse([]byte(page), openAIText)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]Retirement{}
	for _, r := range rs {
		got[r.Model] = r
	}
	if len(got) != 2 || got["gpt-4-x"].Date != "2026-06-01" || got["gpt-4-x-alias"].Date != "2026-06-01" {
		t.Errorf("the newest announcement wins, aliases split on the escaped pipe, no date means no event: %+v", got)
	}
	if _, ok := got["gpt-4-later"]; ok {
		t.Error("a newer \"To be announced\" withdraws an older row's date: no stale event")
	}
	if _, err := Parse([]byte("# Deprecations\n\nnothing here\n"), openAIText); !errors.Is(err, plugin.ErrNothingToRead) {
		t.Errorf("a page with no retirement tables must be plugin.ErrNothingToRead, not zero events: %v", err)
	}
}

func TestEventsOnePerModel(t *testing.T) {
	rs := []Retirement{{Model: "gpt-4", Date: "2026-10-23", Replacement: "`gpt-5.6-sol`", Excerpt: "### h\n| Shutdown date | … |\n| October 23, 2026 | `gpt-4` | `gpt-5.6-sol` |"}}
	src := "https://web.archive.org/web/20261013115538/https://developers.openai.com/api/docs/deprecations.md"
	events, errs := Events(OpenAI, rs, src, time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC))
	if len(errs) != 0 || len(events) != 1 {
		t.Fatalf("events %d, errs %v", len(events), errs)
	}
	e := events[0]
	want := event.Subject{Type: event.SubjectModel, Provider: "openai", Name: "gpt-4", Replacement: "gpt-5.6-sol"}
	if e.Subject.Type != want.Type || e.Subject.Provider != want.Provider || e.Subject.Name != want.Name || e.Subject.Replacement != want.Replacement {
		t.Errorf("subject: %+v", e.Subject)
	}
	if e.API != "openai/models" || e.Version != "2026-10-23" || e.PreviousVersion != "" || !e.Breaking ||
		e.SourceURL != src || e.SourceExcerpt != rs[0].Excerpt || e.Confidence != 1.0 {
		t.Errorf("event: %+v", e)
	}
	if e.Fingerprint != event.Fingerprint("openai/models", "2026-10-23", event.ChangeTypeBreaking, e.Subject, "") {
		t.Error("the fingerprint is the contract's formula over the model subject")
	}
	// The replacement is set only when the provider names exactly one model.
	for cell, want := range map[string]string{"`gpt-5.6-sol`": "gpt-5.6-sol", "`a` or `b`": "", "`gpt-5.6-sol` (`reasoning.mode: pro`)": "", "none": ""} {
		if got := replacement(cell); got != want {
			t.Errorf("replacement(%q) = %q, want %q", cell, got, want)
		}
	}
	if !strings.Contains(e.Description, `"gpt-4"`) || !strings.Contains(e.MigrationHint, "gpt-5.6-sol") {
		t.Errorf("%q / %q", e.Description, e.MigrationHint)
	}
	moved, _ := Events(OpenAI, []Retirement{{Model: "gpt-4", Date: "2026-11-23", Replacement: "`gpt-5.6-sol`", Excerpt: "x"}}, src, time.Now())
	if moved[0].Fingerprint == e.Fingerprint {
		t.Error("a moved shutdown date is a new event")
	}
}

func TestEveryRealRetirementBecomesValidEvents(t *testing.T) {
	for _, c := range []struct {
		p    Page
		file string
	}{{OpenAI, "openai-deprecations.md"}, {Anthropic, "anthropic-deprecations.md"}} {
		rs, err := Parse(read(t, c.file), c.p.Keep)
		if err != nil {
			t.Fatal(err)
		}
		events, errs := Events(c.p, rs, "https://web.archive.org/web/1/x", time.Now())
		if len(errs) != 0 || len(events) != len(rs) {
			t.Errorf("%s: %d events from %d retirements, errors %v", c.p.API, len(events), len(rs), errs)
		}
	}
}
