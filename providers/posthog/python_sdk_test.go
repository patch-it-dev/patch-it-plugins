package posthog

import (
	"context"
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

// The guide as published at tag posthog-v8.0.0 (fetched 2026-11-19), whole.
const posthogSource = "testdata/migration-7.x-to-8.0-posthog-v8.0.0.md"

func TestPostHogRegistersItself(t *testing.T) {
	kittest.Registered(t, migrationguide.Plugin{Guide: PythonSDK})
}

// The golden test: the saved guide in, every event out.
func TestPostHogGolden(t *testing.T) {
	kittest.Golden(t, "testdata/golden/posthog-8.0.0.json", posthogIngest(t))
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
		if e.API != "posthog/python-sdk" || e.Version != "8.0.0" || e.Subject.Ecosystem != "pypi" || e.Subject.Package != "posthog" ||
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
	rs, err := members.Parse(kittest.Read(t, posthogSource), PythonSDK)
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
		g := PythonSDK
		g.Section, g.Header = other.section, other.header
		rs, err := members.Parse(kittest.Read(t, posthogSource), g)
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

// posthogIngest is the pair from 7.91.4, the release before on PyPI, over
// the saved guide.
func posthogIngest(t *testing.T) []event.Event {
	t.Helper()
	return migrationguidetest.Release(t, migrationguidetest.GuideEnv(t, PythonSDK, posthogSource, "8.0.0"), PythonSDK, "7.91.4", "8.0.0")
}

// The guide is read at the major's first tag (posthog-v8.0.0); a release
// that isn't one has none.
func TestPythonSDKGuideIsReadAtTheMajorsTag(t *testing.T) {
	ctx := context.Background()
	want := "https://raw.githubusercontent.com/PostHog/posthog-python/posthog-v8.0.0/docs/migration-7.x-to-8.0.md"
	if got, err := PythonSDK.GuideURL(ctx, migrationguidetest.GuideEnv(t, PythonSDK, "testdata/README.md", "8.0.0"), plugin.Version{ID: "8.0.0"}); err != nil || got != want {
		t.Errorf("%q, %v; want %q", got, err, want)
	}
	if got, err := PythonSDK.GuideURL(ctx, plugin.Env{}, plugin.Version{ID: "8.0.1"}); got != "" || err != nil {
		t.Errorf("a release that isn't a major's first has no guide: %q, %v", got, err)
	}
}
