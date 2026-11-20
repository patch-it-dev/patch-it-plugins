package members_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"

	"github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide"
	. "github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/members" // the reader under test
)

// The guide as published at tag posthog-v8.0.0 (fetched 2026-11-19), whole.
const posthogSource = "testdata/migration-7.x-to-8.0-posthog-v8.0.0.md"

func TestPostHogRegistersItself(t *testing.T) {
	kittest.Registered(t, migrationguide.Plugin{Guide: PostHog8})
}

// The golden test: the saved guide in, every event out.
func TestPostHogGolden(t *testing.T) {
	kittest.Golden(t, "../testdata/golden/posthog-8.0.0.json", posthogIngest(t))
}

// Each of the "Fix removed APIs" table's 7 rows, decided: every name is a
// removal, its "Use instead" in the hint (each replacement is new in 8.0.0,
// so none is a rename), and what isn't a removed member of the package is no
// event.
func TestPostHogEachRowOfTheTable(t *testing.T) {
	type want struct {
		change      event.PackageChange
		replacement string
	}
	got := map[string]want{}
	for _, e := range posthogIngest(t) {
		if len(e.Subject.Symbols) != 1 {
			t.Fatalf("%v: one symbol per event", e.Subject.Symbols)
		}
		if e.API != "posthog" || e.Version != "8.0.0" || e.Subject.Ecosystem != "pypi" || e.Subject.Package != "posthog" ||
			e.ChangeType != event.ChangeTypeBreaking || !strings.HasPrefix(e.SourceExcerpt, "Fix removed APIs\n| Removed | Use instead |") {
			t.Errorf("%+v", e)
		}
		got[e.Subject.Symbols[0]] = want{e.Subject.Change, e.Subject.Replacement}
	}
	removed := want{change: event.PackageRemovedSymbol}
	expect := map[string]want{
		"CaptureMode": removed, // | Nothing (capture_mode, in the same row, still works: dropped)
		// | `CaptureError`, new in 8.0.0 and in another module: in the hint,
		// not a rename. It's posthog.capture_v1's, written in full.
		"capture_v1.CaptureV1Error": removed,
		"capture_v1":                removed, // two replacements: `posthog.capture_event` and `posthog.capture_send`
		// | Send events through a client. The cell spells out `request.` only
		// for the first name; the others' real modules, from the code.
		"request.batch_post":              removed,
		"_async_request.async_batch_post": removed,
		"request.EVENTS_ENDPOINT":         removed,
		"request.AI_EVENTS_ENDPOINT":      removed,
	}
	if len(got) != len(expect) {
		t.Errorf("%d events, want %d: %v", len(got), len(expect), got)
	}
	for sym, w := range expect {
		if g, ok := got[sym]; !ok || g != w {
			t.Errorf("%s: %+v, want %+v", sym, g, w)
		}
	}
}

// The pinned drop test. The guide has eight more tables, and this one has
// four rows (and one name) that aren't removed members; none may become an
// event. Each is
// shown to be there, and to read as names, so the test isn't vacuous.
func TestPostHogDropsWhatIsNotARemovedMember(t *testing.T) {
	events := posthogIngest(t)
	var symbols []string
	for _, e := range events {
		symbols = append(symbols, e.Subject.Symbols...)
		symbols = append(symbols, e.Subject.Replacement)
	}

	// The table's own rows: an environment variable, an argument value, and
	// two rows of prose (a parameter, a dependency). With the rules off, each
	// is a row of the table, with names in code spans.
	rs, err := Parse(kittest.Read(t, posthogSource), PostHog8)
	if err != nil {
		t.Fatal(err)
	}
	dropped := map[string]string{}
	for _, r := range rs {
		if r.Dropped != "" {
			dropped[r.Member] = r.Dropped
		}
	}
	for _, member := range []string{
		"POSTHOG_CAPTURE_MODE",
		"capture_mode",
		"`gzip=True`",
		"The `gzip` parameter of `request.post` and `request.flags`",
		"The `backoff` dependency",
	} {
		if dropped[member] == "" {
			t.Errorf("%q: not read and dropped (%v)", member, dropped)
		}
	}
	if len(dropped) != 5 {
		t.Errorf("%d dropped, want 5: %v", len(dropped), dropped)
	}
	if !strings.Contains(dropped["capture_mode"], "Rename deprecated settings") {
		t.Errorf("capture_mode: dropped for %q, want the guide's own reason", dropped["capture_mode"])
	}
	for _, name := range []string{"POSTHOG_CAPTURE_MODE", "capture_mode", "gzip", "gzip=True", "request.post", "request.flags", "backoff", "capture_compression"} {
		if slices.Contains(symbols, name) {
			t.Errorf("an event names %q", name)
		}
	}

	// The guide's other tables, read as if each were the removals table. The
	// "Rename deprecated settings" one would rename _use_ai_lane to
	// enable_full_ai_capture, a wrong breaking change (the old names still
	// work in 8.x.x); the "Prepare your proxy" one would remove capture. Only
	// the Section heading keeps them out.
	for _, other := range []struct{ section, header, name string }{
		{"Rename deprecated settings", "Old", "_use_ai_lane"},
		{"Rename deprecated settings", "Old", "trust_tracing_headers"},
		{"Prepare your proxy", "Method", "capture"},
		{"Route AI events", "Argument", "capture_ai_compression"},
		{"Update error handling", "Field", "drops"},
		{"Move to event options", "Option", "cookieless_mode"},
	} {
		g := PostHog8
		g.Section, g.Header = other.section, other.header
		rs, err := Parse(kittest.Read(t, posthogSource), g)
		if err != nil {
			t.Fatal(err)
		}
		read := false
		for _, r := range rs {
			read = read || (r.Dropped == "" && r.Member == other.name)
		}
		if !read {
			t.Errorf("%q isn't read from the %q table: its drop proves nothing", other.name, other.section)
		}
		if slices.Contains(symbols, other.name) {
			t.Errorf("an event names %q, from the %q table", other.name, other.section)
		}
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

	for name, page := range map[string]string{
		"no section":    "## Other\n| Removed | Use instead |\n| --- | --- |\n| `a` | b |",
		"no table":      "## Removed\ntext\n## Next\n| Removed | Use instead |\n| --- | --- |\n| `a` | b |",
		"another table": "## Removed\n| Method | Endpoint |\n| --- | --- |\n| `a` | b |",
	} {
		if _, err := Parse([]byte(page), g); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

// posthogIngest runs the plugin over the saved guide.
func posthogIngest(t *testing.T) []event.EventV2 {
	t.Helper()
	return kittest.Ingest(t, migrationguide.Plugin{Guide: PostHog8}, kittest.Pages{PostHog8.URL(): posthogSource})
}
