package retire

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

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
	rs, err := Parse([]byte(page), gptOnly)
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
	if _, err := Parse([]byte("# Deprecations\n\nnothing here\n"), gptOnly); !errors.Is(err, plugin.ErrNothingToRead) {
		t.Errorf("a page with no retirement tables must be plugin.ErrNothingToRead, not zero events: %v", err)
	}
}

func TestEventsOnePerModel(t *testing.T) {
	rs := []Retirement{{Model: "gpt-4", Date: "2026-10-23", Replacement: "`gpt-5.6-sol`", Excerpt: "### h\n| Shutdown date | … |\n| October 23, 2026 | `gpt-4` | `gpt-5.6-sol` |"}}
	src := "https://web.archive.org/web/20261013115538/https://developers.openai.com/api/docs/deprecations.md"
	events, errs := Events(acme, rs, src, time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC))
	if len(errs) != 0 || len(events) != 1 {
		t.Fatalf("events %d, errs %v", len(events), errs)
	}
	e := events[0]
	want := event.Subject{Type: event.SubjectModel, Provider: "openai", Name: "gpt-4", Replacement: "gpt-5.6-sol"}
	if e.Subject.Type != want.Type || e.Subject.Provider != want.Provider || e.Subject.Name != want.Name || e.Subject.Replacement != want.Replacement {
		t.Errorf("subject: %+v", e.Subject)
	}
	if e.API != "openai/models" || e.Version != "2026-10-23" || e.PreviousVersion != "" || e.Change != event.ChangeBreaking ||
		e.SourceURL != src || e.SourceExcerpt != rs[0].Excerpt || e.Confidence != 1.0 {
		t.Errorf("event: %+v", e)
	}
	if e.Fingerprint != event.Fingerprint("openai/models", "2026-10-23", event.ChangeBreaking, e.Subject, "") {
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
	moved, _ := Events(acme, []Retirement{{Model: "gpt-4", Date: "2026-11-23", Replacement: "`gpt-5.6-sol`", Excerpt: "x"}}, src, time.Now())
	if moved[0].Fingerprint == e.Fingerprint {
		t.Error("a moved shutdown date is a new event")
	}
}

// acme is a page like OpenAI's, for the tests of the reader itself: the
// providers' own pages are tested with their providers.
var acme = Page{API: "openai/models", Provider: "openai", Vendor: "OpenAI", URL: "https://developers.openai.com/api/docs/deprecations.md", Keep: gptOnly}

func gptOnly(name string) bool { return strings.HasPrefix(name, "gpt-") }
