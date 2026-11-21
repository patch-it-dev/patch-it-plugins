package newrelic

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"

	"github.com/patch-it-dev/patch-it-plugins/kinds/migrationguide"
	"github.com/patch-it-dev/patch-it-plugins/kinds/migrationguide/members"
	"github.com/patch-it-dev/patch-it-plugins/kinds/migrationguide/migrationguidetest"
)

// An excerpt of the release note at commit f0b4f817 (fetched 2026-11-19): the
// "Removed features" heading and its two tables, every kept line unchanged.
const newrelicSource = "testdata/python-agent-110000-f0b4f817.mdx"

func TestNewRelicRegistersItself(t *testing.T) {
	kittest.Registered(t, migrationguide.Plugin{Guide: PythonAgent})
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
		if e.API != "newrelic/python-agent" || e.Version != "11.0.0" || e.Subject.Ecosystem != "pypi" || e.Subject.Package != "newrelic" ||
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
	for _, d := range PythonAgent.DropIns {
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
	rs, err := members.Parse([]byte(page), PythonAgent)
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
	if names := readNames(t, asList, PythonAgent); !slices.Contains(names, "lambda_handler") || !slices.Contains(names, "LambdaHandlerWrapper") {
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
	g := PythonAgent
	g.Header = "Legacy Setting"
	settings, err := members.Parse([]byte(page[:start]+page[end:]), g)
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

func readNames(t *testing.T, page string, g members.Guide) []string {
	t.Helper()
	rs, err := members.Parse([]byte(page), g)
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

// newrelicIngest is the pair from 10.17.0, the release before on PyPI, over
// the saved excerpt.
func newrelicIngest(t *testing.T) []event.Event {
	t.Helper()
	return migrationguidetest.Release(t, migrationguidetest.UntaggedEnv(t, PythonAgent, newrelicSource, "11.0.0", newrelicCommits, newrelicCommit), PythonAgent, "10.17.0", "11.0.0")
}

// newrelicCommit is the docs-website commit New Relic's saved release note
// was fetched at, and newrelicCommits GitHub's answer naming it as the
// newest commit that changed the note.
const (
	newrelicCommit  = "f0b4f817bf9a7b76a78e26609369a8a7ce764c8e"
	newrelicCommits = "testdata/github-commits-python-agent-110000.json"
)

// The docs repo has no tags: a major's note is read at the newest commit
// that changed it. A release that isn't a major's first has none.
func TestPythonAgentNoteIsReadAtTheNewestCommit(t *testing.T) {
	ctx := context.Background()
	want := "https://raw.githubusercontent.com/newrelic/docs-website/" + newrelicCommit + "/src/content/docs/release-notes/agent-release-notes/python-release-notes/python-agent-110000.mdx"
	env := migrationguidetest.UntaggedEnv(t, PythonAgent, "testdata/README.md", "11.0.0", newrelicCommits, newrelicCommit)
	if got, err := PythonAgent.GuideURL(ctx, env, plugin.Version{ID: "11.0.0"}); err != nil || got != want {
		t.Errorf("%q, %v; want %q", got, err, want)
	}
	if got, err := PythonAgent.GuideURL(ctx, plugin.Env{}, plugin.Version{ID: "11.0.1"}); got != "" || err != nil {
		t.Errorf("a release that isn't a major's first has no guide: %q, %v", got, err)
	}
}

// A major whose note no commit ever changed has nothing to read: the pair is
// skipped, not failed, and not a release without changes.
func TestPythonAgentMajorWithNoNoteHasNothingToRead(t *testing.T) {
	env := kittest.Env(t, kittest.Pages{PythonAgent.CommitsURL(14): "testdata/github-commits-none.json"})
	if _, err := (migrationguide.Plugin{Guide: PythonAgent}).ReadAt(context.Background(), env, plugin.Version{ID: "14.0.0"}); !errors.Is(err, plugin.ErrNothingToRead) {
		t.Errorf("no commit for the note: %v, want plugin.ErrNothingToRead", err)
	}
}
