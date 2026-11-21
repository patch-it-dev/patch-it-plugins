package anthropic

import (
	"os"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/noticestest"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/retire"
)

func TestAnthropicsRetirements(t *testing.T) {
	got := byModel(t, Models, "anthropic-deprecations.md")
	want := map[string][2]string{
		"claude-sonnet-4-5-20250929": {"2026-11-30", "claude-sonnet-5-5"},
		"claude-opus-4-1-20250805":   {"2026-08-05", "claude-opus-4-8"},
		"claude-3-opus-20240229":     {"2026-01-05", "claude-opus-4-8"},
		"claude-3-5-sonnet-20241022": {"2025-10-28", "claude-sonnet-5-5"},
		"claude-3-haiku-20240307":    {"2026-04-20", "claude-haiku-4-5-20251001"},
	}
	for m, w := range want {
		if r := got[m]; r.Date != w[0] || retire.First(r.Replacement) != w[1] {
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

// Anthropic's saved page (2026-10-13, testdata/README.md) in, every event out.
func TestModelsGolden(t *testing.T) {
	noticestest.Golden(t, Models, "testdata/anthropic-deprecations.md")
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

func TestEveryRealRetirementBecomesValidEvents(t *testing.T) {
	for _, c := range []struct {
		p    retire.Page
		file string
	}{{Models, "anthropic-deprecations.md"}} {
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
