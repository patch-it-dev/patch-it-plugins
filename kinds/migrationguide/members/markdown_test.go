package members_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	. "github.com/patch-it-dev/patch-it-plugins/kinds/migrationguide/members" // the reader under test
	"github.com/patch-it-dev/patch-it-plugins/kinds/migrationguide/migrationguidetest"
)

// A replacement cell that is a Markdown link around one code span is read as
// that name; a link inside prose, or around more than a code span, isn't.
func TestSoleNameReadsALinkedName(t *testing.T) {
	for cell, want := range map[string]string{
		"`new_fn`":                              "new_fn",
		"[`new_fn`](/docs/new-fn)":              "new_fn",
		" [`acme.new_fn()`](https://x.test/a) ": "new_fn",
		"[`new_fn`](/docs/new-fn#anchor)":       "new_fn",
		"Use [`new_fn`](/docs/new-fn)":          "",
		"[`new_fn`](/docs/new-fn) instead":      "",
		"[the `new_fn` API](/docs/new-fn)":      "",
		"[`new_fn`]":                            "",
		"[`x=1`](/docs/x)":                      "",
		"[`a`](/a) and [`b`](/b)":               "",
	} {
		if got := SoleName(cell, "acme"); got != want {
			t.Errorf("SoleName(%q) = %q, want %q", cell, got, want)
		}
	}
}

// Without DropIns every one-replacement row is a rename, as NumPy's are; with
// it, only the listed ones. Paths applies to the replacement as to the name.
func TestDropInsLimitsRenames(t *testing.T) {
	page := strings.Join([]string{
		"## Removed",
		"| Removed | Use instead |",
		"| --- | --- |",
		"| `old_a` | [`new_a`](/a) |",
		"| `old_b` | `new_b` |",
	}, "\n")
	g := Guide{API: "acme/sdk", Package: "acme", Module: "acme", Repo: "acme/acme", Path: "MIGRATION.md",
		Section: "Removed", Format: FormatMarkdown, Header: "Removed",
		Paths: map[string]string{"old_a": "mod.old_a", "new_a": "mod.new_a"}}
	run := func(g Guide) map[string]string {
		out := map[string]string{}
		for _, e := range migrationguidetest.Release(t, migrationguidetest.GuideEnv(t, g, writeTemp(t, page), "2.0.0"), g, "1.9.0", "2.0.0") {
			out[e.Subject.Symbols[0]] = string(e.Subject.Change) + " " + e.Subject.Replacement
		}
		return out
	}
	if got := run(g); got["mod.old_a"] != "renamed_symbol mod.new_a" || got["old_b"] != "renamed_symbol new_b" {
		t.Errorf("no DropIns: %v", got)
	}
	g.DropIns = []string{"mod.old_a"}
	if got := run(g); got["mod.old_a"] != "renamed_symbol mod.new_a" || got["old_b"] != "removed_symbol " {
		t.Errorf("DropIns mod.old_a: %v", got)
	}
}

func writeTemp(t *testing.T, page string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "guide.md")
	if err := os.WriteFile(f, []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	return f
}

// The hint is shown in a pull request or an issue, where a site-relative link
// breaks: LinkBase makes the guide's links absolute, and leaves the rest.
func TestAbsoluteLinks(t *testing.T) {
	g := Guide{LinkBase: "https://docs.newrelic.com/"}
	in := "Replacement API: [`notice_error()`](/docs/x/notice_error), see [here](https://example.test/y)"
	want := "Replacement API: [`notice_error()`](https://docs.newrelic.com/docs/x/notice_error), see [here](https://example.test/y)"
	if got := g.AbsoluteLinks(in); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := (Guide{}).AbsoluteLinks(in); got != in {
		t.Errorf("no LinkBase changed the hint: %q", got)
	}
}

func TestPipeCells(t *testing.T) {
	for row, want := range map[string][]string{
		"| Removed | Use instead |":                   {"Removed", "Use instead"},
		"| --- | --- |":                               {"---", "---"},
		"| `a`, `b` | Nothing |":                      {"`a`, `b`", "Nothing"},
		"| `x | y` | a \\| b |":                       {"`x | y`", "a | b"},
		"|`a`|`b`|":                                   {"`a`", "`b`"},
		"| `request.batch_post`, `EVENTS_ENDPOINT` |": {"`request.batch_post`, `EVENTS_ENDPOINT`"},
		"| one | two | three |":                       {"one", "two", "three"},
	} {
		if got := PipeCells(row); !slices.Equal(got, want) {
			t.Errorf("PipeCells(%q) = %q, want %q", row, got, want)
		}
	}
}

func TestMarkdownTable(t *testing.T) {
	g := Guide{Package: "acme", Module: "acme", Section: "Removed", Format: FormatMarkdown, Header: "Removed", EnvVars: []string{"ACME_MODE"}}
	page := strings.Join([]string{
		"# Guide",
		"## Removed",
		"```python",
		"# not a heading",
		"```",
		"Text before the table.",
		"",
		"| Removed | Use instead |",
		"| :--- | ---: |",
		"| `acme.old_fn()` | `acme.new_fn` |",
		"| `A`, `B`, and `ACME_MODE` | Nothing |",
		"| `C` | `D` and `E` |",
		"| `F` | Use `G` instead |",
		"| `same` | `same` |",
		"| `x=1` | `y` |",
		"| The `dep` dependency | Nothing |",
		"| `short` |",
		"",
		"| Removed | Use instead |",
		"| --- | --- |",
		"| `not_read` | `the second table` |",
	}, "\n")
	rs, err := Parse([]byte(page), g)
	if err != nil {
		t.Fatal(err)
	}
	type row struct{ member, replacement, dropped string }
	var got []row
	for _, r := range rs {
		d := ""
		if r.Dropped != "" {
			d = "dropped"
		}
		got = append(got, row{r.Member, r.Replacement, d})
	}
	want := []row{
		{"old_fn", "new_fn", ""},
		{"A", "", ""},
		{"B", "", ""},
		{"ACME_MODE", "", "dropped"},
		{"C", "", ""},
		{"F", "", ""},
		{"same", "", ""},
		{"`x=1`", "", "dropped"},
		{"The `dep` dependency", "", "dropped"},
		{"| `short` |", "", "dropped"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("got\n%v\nwant\n%v", got, want)
	}
	if rs[0].Guideline != "Use instead: `acme.new_fn`" ||
		rs[0].Excerpt != "Removed\n| Removed | Use instead |\n| :--- | ---: |\n| `acme.old_fn()` | `acme.new_fn` |" {
		t.Errorf("%+v", rs[0])
	}

	for name, c := range map[string]struct {
		page    string
		nothing bool // plugin.ErrNothingToRead: no part to read, not a guide that changed shape
	}{
		"no section":    {"## Other\n| Removed | Use instead |\n| --- | --- |\n| `a` | b |", true},
		"no table":      {"## Removed\ntext\n## Next\n| Removed | Use instead |\n| --- | --- |\n| `a` | b |", true},
		"another table": {"## Removed\n| Method | Endpoint |\n| --- | --- |\n| `a` | b |", false},
	} {
		_, err := Parse([]byte(c.page), g)
		if err == nil || errors.Is(err, plugin.ErrNothingToRead) != c.nothing {
			t.Errorf("%s: err %v, want an error that is plugin.ErrNothingToRead: %v", name, err, c.nothing)
		}
	}
}
