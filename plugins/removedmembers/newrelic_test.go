package removedmembers

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
)

// An excerpt of the release note at commit f0b4f817 (fetched 2026-11-19): the
// "Removed features" heading and its two tables, every kept line unchanged.
const newrelicSource = "testdata/python-agent-110000-f0b4f817.mdx"

func TestNewRelicRegistersItself(t *testing.T) {
	kittest.Registered(t, Plugin{Guide: NewRelic11})
}

// The golden test: the saved excerpt in, every event out.
func TestNewRelicGolden(t *testing.T) {
	kittest.Golden(t, "testdata/golden/newrelic-11.0.0.json", newrelicIngest(t))
}

// Each of the "Legacy API" table's 8 rows, decided: a rename only for the two
// drop-in replacements (same arguments, and the old function only called the
// new one at v10.17.0); a removal, with the replacement in the hint, for the
// rest.
func TestNewRelicEachRowOfTheTable(t *testing.T) {
	type want struct {
		change      event.PackageChange
		replacement string
	}
	got := map[string]want{}
	hints := map[string]string{}
	for _, e := range newrelicIngest(t) {
		if len(e.Subject.Symbols) != 1 {
			t.Fatalf("%v: one symbol per event", e.Subject.Symbols)
		}
		if e.API != "newrelic" || e.Version != "11.0.0" || e.Subject.Ecosystem != "pypi" || e.Subject.Package != "newrelic" ||
			e.ChangeType != event.ChangeTypeBreaking || !strings.HasPrefix(e.SourceExcerpt, "Removed features\n    | Legacy API ") ||
			!strings.HasSuffix(e.MigrationHint, "(newrelic's release notes for 11.0.0)") {
			t.Errorf("%+v", e)
		}
		got[e.Subject.Symbols[0]] = want{e.Subject.Change, e.Subject.Replacement}
		hints[e.Subject.Symbols[0]] = e.MigrationHint
	}
	removed := want{change: event.PackageRemovedSymbol}
	expect := map[string]want{
		"agent.add_custom_parameter":  {event.PackageRenamedSymbol, "agent.add_custom_attribute"},
		"agent.add_custom_parameters": {event.PackageRenamedSymbol, "agent.add_custom_attributes"},
		// One name each, but not drop-ins: different data or arguments.
		"agent.accept_distributed_trace_payload": removed,
		"agent.create_distributed_trace_payload": removed,
		"agent.record_exception":                 removed,
		"api.log.NewRelicLogHandler":             removed,
		// Prose: "wrapped into get_browser_timing_header()".
		"agent.get_browser_timing_footer": removed,
	}
	if len(got) != len(expect) {
		t.Errorf("%d events, want %d: %v", len(got), len(expect), got)
	}
	for sym, w := range expect {
		if g, ok := got[sym]; !ok || g != w {
			t.Errorf("%s: %+v, want %+v", sym, g, w)
		}
	}
	// The removals still say what to use, in the guide's own words.
	for sym, use := range map[string]string{
		"agent.record_exception":     "`notice_error()`",
		"api.log.NewRelicLogHandler": "`NewRelicLogForwardingHandler`",
	} {
		if !strings.Contains(hints[sym], use) {
			t.Errorf("%s: hint %q doesn't name %s", sym, hints[sym], use)
		}
	}
	// Every DropIns entry is a member the table removes: a typo would quietly
	// turn its rename into a removal.
	for _, d := range NewRelic11.DropIns {
		if got[d].change != event.PackageRenamedSymbol {
			t.Errorf("DropIns %q: not a rename (%+v)", d, got[d])
		}
	}
}

// The pinned drop test. The lambda row (two names joined by "&", moved to
// another package) and the section's settings table must give no event. Each
// is shown to be there and to read as names once its rule is off, so the test
// isn't vacuous.
func TestNewRelicDropsWhatIsNotARemovedMember(t *testing.T) {
	var symbols []string
	for _, e := range newrelicIngest(t) {
		symbols = append(symbols, e.Subject.Symbols...)
		symbols = append(symbols, e.Subject.Replacement)
	}
	page := string(kittest.Read(t, newrelicSource))

	// The lambda row: read, and dropped.
	rs, err := Parse([]byte(page), NewRelic11)
	if err != nil {
		t.Fatal(err)
	}
	var dropped []string
	for _, r := range rs {
		if r.Dropped != "" {
			dropped = append(dropped, r.Member)
		}
	}
	if !slices.Equal(dropped, []string{"`lambda_handler` & `LambdaHandlerWrapper`"}) {
		t.Errorf("dropped %q, want only the lambda row", dropped)
	}
	// With "&" read as "and", it would be two removals.
	asList := strings.Replace(page, "`lambda_handler` & `LambdaHandlerWrapper`", "`lambda_handler` and `LambdaHandlerWrapper`", 1)
	if asList == page {
		t.Fatal("no lambda row in the saved release note")
	}
	if names := readNames(t, asList, NewRelic11); !slices.Contains(names, "lambda_handler") || !slices.Contains(names, "LambdaHandlerWrapper") {
		t.Errorf("with \"and\", the lambda row reads as %v: its drop proves nothing", names)
	}

	// The settings table, read as if it were the APIs table (the APIs table
	// cut out, its header named): its rows would be renames such as
	// strip_exception_messages.whitelist -> .allowlist, wrong events about
	// newrelic.ini keys. Only "the section's first table" keeps it out.
	start := strings.Index(page, "    | Legacy API")
	end := strings.Index(page, "* Removes various deprecated settings")
	if start < 0 || end < start {
		t.Fatal("the saved release note's shape changed")
	}
	g := NewRelic11
	g.Header = "Legacy Setting"
	settings, err := Parse([]byte(page[:start]+page[end:]), g)
	if err != nil {
		t.Fatal(err)
	}
	read := map[string]string{}
	for _, r := range settings {
		if r.Dropped == "" {
			read[r.Member] = r.Replacement
		}
	}
	if read["strip_exception_messages.whitelist"] != "strip_exception_messages.allowlist" || len(read) != 10 {
		t.Errorf("the settings table reads as %v: its drop proves nothing", read)
	}
	for name := range read {
		if slices.Contains(symbols, name) {
			t.Errorf("an event names %q, a setting", name)
		}
	}
	for _, name := range []string{"lambda_handler", "LambdaHandlerWrapper", "agent.lambda_handler", "agent.LambdaHandlerWrapper", "ssl"} {
		if slices.Contains(symbols, name) {
			t.Errorf("an event names %q", name)
		}
	}
}

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
		if got := soleName(cell, "acme"); got != want {
			t.Errorf("soleName(%q) = %q, want %q", cell, got, want)
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
	g := Guide{Package: "acme", Module: "acme", Release: "2.0.0", Repo: "acme/acme", Path: "MIGRATION.md",
		Section: "Removed", Format: FormatMarkdown, Header: "Removed",
		Paths: map[string]string{"old_a": "mod.old_a", "new_a": "mod.new_a"}}
	run := func(g Guide) map[string]string {
		out := map[string]string{}
		for _, e := range kittest.Ingest(t, Plugin{Guide: g}, kittest.Pages{g.URL(): writeTemp(t, page)}) {
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

func readNames(t *testing.T, page string, g Guide) []string {
	t.Helper()
	rs, err := Parse([]byte(page), g)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, r := range rs {
		if r.Dropped == "" {
			names = append(names, r.Member)
		}
	}
	return names
}

// newrelicIngest runs the plugin over the saved excerpt.
func newrelicIngest(t *testing.T) []event.EventV2 {
	t.Helper()
	return kittest.Ingest(t, Plugin{Guide: NewRelic11}, kittest.Pages{NewRelic11.URL(): newrelicSource})
}

// The hint is shown in a pull request or an issue, where a site-relative link
// breaks: LinkBase makes the guide's links absolute, and leaves the rest.
func TestAbsoluteLinks(t *testing.T) {
	g := Guide{LinkBase: "https://docs.newrelic.com/"}
	in := "Replacement API: [`notice_error()`](/docs/x/notice_error), see [here](https://example.test/y)"
	want := "Replacement API: [`notice_error()`](https://docs.newrelic.com/docs/x/notice_error), see [here](https://example.test/y)"
	if got := g.absoluteLinks(in); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := (Guide{}).absoluteLinks(in); got != in {
		t.Errorf("no LinkBase changed the hint: %q", got)
	}
}
