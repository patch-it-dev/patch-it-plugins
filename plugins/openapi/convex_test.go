package openapi

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/adapter"
	"github.com/patch-it-dev/patch-it-plugins/event"
	"github.com/patch-it-dev/patch-it-plugins/kit/kittest"
)

// convexCommits are the spec commits saved under testdata/ (trimmed to the
// operations involved, values unchanged; checked 2026-11-19): short SHA ->
// full SHA and committer date.
var convexCommits = map[string][2]string{
	"f35879bb": {"f35879bb396491942bf8ecbc8ef5e4280499f89f", "2026-10-30"},
	"5e5159cb": {"5e5159cbea5188e9da5400febda1b031a233eea0", "2026-11-05"},
	"3f6edf7a": {"3f6edf7ab8f3f98e027f317633246a19348eabcc", "2026-07-14"},
	"3ead9091": {"3ead909103b7f5897f529c031e80991ad174b07e", "2026-07-14"},
}

// fakeConvexAdapter serves the saved copies, cited by the real URL each was
// copied from, as adapter.Convex.Fetch cites it.
type fakeConvexAdapter struct{ latest string }

func (fakeConvexAdapter) API() string { return "convex" }

func (f fakeConvexAdapter) LatestTag(context.Context) (string, error) { return f.latest, nil }

func (fakeConvexAdapter) Fetch(_ context.Context, tag string) (adapter.Spec, error) {
	c, ok := convexCommits[tag]
	if !ok {
		return adapter.Spec{}, fmt.Errorf("no saved copy for %s", tag)
	}
	body, err := os.ReadFile(convexFile(tag))
	if err != nil {
		return adapter.Spec{}, err
	}
	return adapter.Spec{Content: body, Version: c[1], SourceURL: convexURL(tag)}, nil
}

func convexFile(tag string) string { return "testdata/convex-" + tag + ".json" }

func convexURL(tag string) string {
	return "https://raw.githubusercontent.com/get-convex/convex-backend/" + convexCommits[tag][0] + "/npm-packages/@convex-dev/platform/deployment-openapi.json"
}

// convexPlugin is the plugin exactly as registered, over the saved copies.
func convexPlugin(latest string) Plugin {
	return Plugin{Adapter: fakeConvexAdapter{latest: latest}, Diff: convexDiff, Keep: convexKeep}
}

// convexReplacements are the six log stream operations Convex deprecated on
// 2026-11-05, and the /integrations operation each description names
// instead (added 2026-10-30). The commit that deprecated them says "The
// routes still work, so existing callers and older dashboards keep working."
var convexReplacements = map[string]string{
	"GET /list_log_streams":            "`GET /integrations`",
	"GET /get_log_stream/{id}":         "`GET /integrations/{id}`",
	"POST /create_log_stream":          "`POST /integrations`",
	"POST /update_log_stream/{id}":     "`PATCH /integrations/{id}`",
	"POST /delete_log_stream/{id}":     "`DELETE /integrations/{id}`",
	"POST /rotate_webhook_secret/{id}": "`POST /integrations/{id}/rotate_webhook_secret`",
}

// The golden test: the only deprecations in the Deployment API spec's
// history, replayed from the commit before (f35879bb) to the one that made
// them (5e5159cb). Each is a deprecation, not breaking, cites 5e5159cb, and
// quotes the whole operation, so the flag and the replacement are in the
// excerpt.
func TestConvexGolden(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)
	if rejected, err := kittest.Replay(ctx, vercelEnv(s), s, convexPlugin(""), "f35879bb", "5e5159cb"); err != nil || rejected {
		t.Fatalf("replay: rejected=%v err=%v", rejected, err)
	}
	events := kittest.Stored(t, s)
	if len(events) != len(convexReplacements) {
		for _, e := range events {
			t.Logf("got: %s", e.Description)
		}
		t.Fatalf("stored %d events, want %d", len(events), len(convexReplacements))
	}
	for _, e := range events {
		instead, ok := convexReplacements[e.Subject.Operation]
		if !ok {
			t.Errorf("unexpected event %s", e.Subject.Operation)
			continue
		}
		if e.ChangeType != event.ChangeTypeDeprecation || e.Breaking || e.Version != "2026-11-05" {
			t.Errorf("%s: %s breaking=%v version %s, want a 2026-11-05 deprecation", e.Subject.Operation, e.ChangeType, e.Breaking, e.Version)
		}
		if e.SourceURL != convexURL("5e5159cb") {
			t.Errorf("%s: source_url %s", e.Subject.Operation, e.SourceURL)
		}
		if node := jsonNode(t, convexFile("5e5159cb"), e.SourcePointer); node != e.SourceExcerpt && !truncated(e.SourceExcerpt, node) {
			t.Errorf("%s: excerpt doesn't match 5e5159cb#%s", e.Subject.Operation, e.SourcePointer)
		}
		if !strings.Contains(e.SourceExcerpt, `"deprecated":true`) || !strings.Contains(e.SourceExcerpt, "Deprecated: use "+instead+" instead.") {
			t.Errorf("%s: excerpt lacks the flag or %s: %.300s", e.Subject.Operation, instead, e.SourceExcerpt)
		}
	}
	kittest.Golden(t, "testdata/golden/convex.json", events)

	// As a daily ingest, the same six events.
	s = openStore(t)
	env := vercelEnv(s)
	for _, tag := range []string{"f35879bb", "5e5159cb"} {
		if rejected, err := kittest.Run(ctx, env, s, convexPlugin(tag)); err != nil || rejected {
			t.Fatalf("run %s: rejected=%v err=%v", tag, rejected, err)
		}
	}
	if got := kittest.Stored(t, s); len(got) != len(convexReplacements) {
		t.Errorf("ingest stored %d events, want %d", len(got), len(convexReplacements))
	}
}

// The pinned drop test. On 2026-07-14 (3ead9091, "Remove name from usage
// limits") the optional, nullable `name` left the usage-limit request and
// response, unmarked. Convex's commit says it "was never editable in the UI
// (always `None` on create, only echoed back)", and the request type at that
// commit doesn't deny unknown fields, so a caller still sending it isn't
// rejected. No event.
func TestConvexDropsPropertyChangesThatWerentDeprecations(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)
	if rejected, err := kittest.Replay(ctx, vercelEnv(s), s, convexPlugin(""), "3f6edf7a", "3ead9091"); err != nil || rejected {
		t.Fatalf("replay: rejected=%v err=%v", rejected, err)
	}
	if events := kittest.Stored(t, s); len(events) != 0 {
		t.Fatalf("stored %d events, want none", len(events))
	}
	// Without the filter the same pair reports two breaking removals of
	// `name`, one per schema, and nothing else.
	s = openStore(t)
	p := convexPlugin("")
	p.Keep = nil
	if _, err := kittest.Replay(ctx, vercelEnv(s), s, p, "3f6edf7a", "3ead9091"); err != nil {
		t.Fatal(err)
	}
	unfiltered := kittest.Stored(t, s)
	var removed int
	for _, e := range unfiltered {
		if e.Breaking && strings.HasPrefix(e.Description, "Property name removed from schema UsageLimitConfig") {
			removed++
		}
	}
	if removed != 2 || len(unfiltered) != 2 {
		t.Errorf("unfiltered: %d events, %d of them `name` removals; want 2 and 2", len(unfiltered), removed)
	}
}
