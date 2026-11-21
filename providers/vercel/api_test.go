package vercel

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/adapter"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/openapitest"
)

// fakeVercelAdapter serves the trimmed, unedited excerpts of vercel/sdk's
// vercel-spec.json saved under testdata/ (checked 2026-11-18), cited by the
// real URL each one was copied from, the way Vercel.Fetch cites it.
type fakeVercelAdapter struct{ latest string }

func (fakeVercelAdapter) API() string { return "vercel/api" }

func (f fakeVercelAdapter) LatestTag(context.Context) (string, error) { return f.latest, nil }

func (fakeVercelAdapter) Fetch(_ context.Context, tag string) (adapter.Spec, error) {
	body, err := os.ReadFile(vercelFile(tag))
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("no saved copy for tag %s: %w", tag, err)
	}
	return adapter.Spec{
		Content:   body,
		Version:   strings.TrimPrefix(tag, "v"),
		SourceURL: vercelURL(tag),
	}, nil
}

func vercelFile(tag string) string { return "testdata/vercel-" + tag + ".json" }

func vercelURL(tag string) string {
	return "https://raw.githubusercontent.com/vercel/sdk/" + tag + "/vercel-spec.json"
}

// vercelPlugin is the plugin exactly as registered, over the saved copies.
func vercelPlugin(latest string) openapi.Plugin {
	return openapi.Plugin{Adapter: fakeVercelAdapter{latest: latest}, Diff: vercelDiff, Keep: vercelKeep}
}

// The golden test: the whole life of Vercel's Legacy Domains API in its own
// spec. v1.13.5 (2025-10-08) marked four operations deprecated in their
// summaries, the day Vercel announced the Domains Registrar API: "The Legacy
// Domains API is now deprecated. ... Sunset date: 2025-11-09."
// (vercel.com/changelog/new-domains-registrar-api-for-domain-search-pricing-purchase-and-management).
// v1.18.7 (2026-01-09) removed two of them; both now answer "This endpoint has
// been sunsetted on November 9, 2025." The second pair is the pair fixture:
// one real historical breaking change. The same release added the five
// /v1/connect/networks operations, which the filter drops, saved here as one.
func TestVercelGolden(t *testing.T) {
	ctx := context.Background()
	s := kittest.NewMem()
	env := openapitest.Env(s)

	// The deprecations: v1.13.4 to the next day's release, v1.13.5.
	if rejected, err := kittest.Pair(ctx, env, s, vercelPlugin(""), "v1.13.4", "v1.13.5"); err != nil || rejected {
		t.Fatalf("pair: rejected=%v err=%v", rejected, err)
	}
	// The removal, another pair.
	if rejected, err := kittest.Pair(ctx, env, s, vercelPlugin(""), "v1.18.6", "v1.18.7"); err != nil || rejected {
		t.Fatalf("pair: rejected=%v err=%v", rejected, err)
	}

	events := kittest.Stored(t, s)
	got := map[string]string{}
	for _, e := range events {
		got[e.Subject.Operation+" "+e.Version] = string(e.ChangeType)
	}
	want := map[string]string{
		"POST /v5/domains/buy 1.13.5":              "deprecation",
		"GET /v1/domains/{domain}/registry 1.13.5": "deprecation",
		"GET /v4/domains/status 1.13.5":            "deprecation",
		"GET /v4/domains/price 1.13.5":             "deprecation",
		"GET /v4/domains/status 1.18.7":            "breaking",
		"GET /v4/domains/price 1.18.7":             "breaking",
	}
	if len(got) != len(want) || len(events) != len(want) {
		t.Fatalf("stored %v, want exactly %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: got %q, want %q", k, got[k], v)
		}
	}

	// Acceptance criteria: every event carries its source URL and an exact
	// excerpt of it, the node its pointer names in the saved copy.
	for _, e := range events {
		tag := "v" + e.Version
		if e.ChangeType == event.ChangeTypeBreaking {
			tag = "v" + e.PreviousVersion // a removal is evidenced by the old spec
		}
		if e.SourceURL != vercelURL(tag) {
			t.Errorf("%s: source_url %s, want %s", e.Description, e.SourceURL, vercelURL(tag))
		}
		if node := openapitest.JSONNode(t, vercelFile(tag), e.SourcePointer); node != e.SourceExcerpt && !openapitest.Truncated(e.SourceExcerpt, node) {
			t.Errorf("%s: excerpt %.120s…, but %s#%s holds %.120s…", e.Description, e.SourceExcerpt, tag, e.SourcePointer, node)
		}
	}

	kittest.Golden(t, "testdata/golden/vercel.json", events)

	// Reading the same pair again must not duplicate the events.
	if _, err := kittest.Pair(ctx, env, s, vercelPlugin(""), "v1.18.6", "v1.18.7"); err != nil {
		t.Fatalf("second read of the pair: %v", err)
	}
	if again := kittest.Stored(t, s); len(again) != len(events) {
		t.Errorf("reading the pair again added events: %d -> %d", len(events), len(again))
	}
}

// A deprecation said only in words is evidenced by the words: the summary.
func TestVercelSummaryDeprecationQuotesTheSummary(t *testing.T) {
	ctx := context.Background()
	s := kittest.NewMem()
	if _, err := kittest.Pair(ctx, openapitest.Env(s), s, vercelPlugin(""), "v1.13.4", "v1.13.5"); err != nil {
		t.Fatal(err)
	}
	for _, e := range kittest.Stored(t, s) {
		if e.Subject.Operation == "GET /v4/domains/price" {
			if e.SourcePointer != "/paths/~1v4~1domains~1price/get/summary" ||
				e.SourceExcerpt != `"Check the price for a domain (deprecated)"` {
				t.Errorf("cited %s: %s", e.SourcePointer, e.SourceExcerpt)
			}
			return
		}
	}
	t.Error("no deprecation event for GET /v4/domains/price")
}

// The pinned drop test. v1.6.3 (2025-04-17) dropped five operations from the
// spec without marking them deprecated: /v1/log-drains (three operations) and
// two /v1/integrations endpoints. All five came back unchanged in name:
// /v1/log-drains in v1.15.0 (2025-10-23), the other two in v1.18.6
// (2026-01-06). The same release reshuffled the AuthUser response schema.
// Nothing here is a change a caller had to fix.
func TestVercelDropsRemovalsThatWerentDeprecated(t *testing.T) {
	ctx := context.Background()
	s := kittest.NewMem()
	if rejected, err := kittest.Pair(ctx, openapitest.Env(s), s, vercelPlugin(""), "v1.6.2", "v1.6.3"); err != nil || rejected {
		t.Fatalf("pair: rejected=%v err=%v", rejected, err)
	}
	if events := kittest.Stored(t, s); len(events) != 0 {
		for _, e := range events {
			t.Logf("got: %s", e.Description)
		}
		t.Fatalf("stored %d events, want none", len(events))
	}

	// Without the filter, the same pair reports the removals: the fixture still
	// shows what vercelKeep is for.
	s = kittest.NewMem()
	p := vercelPlugin("")
	p.Keep = nil
	if _, err := kittest.Pair(ctx, openapitest.Env(s), s, p, "v1.6.2", "v1.6.3"); err != nil {
		t.Fatal(err)
	}
	var removals int
	for _, e := range kittest.Stored(t, s) {
		if strings.HasPrefix(e.Subject.Operation, "GET /v1/log-drains") && e.ChangeType == event.ChangeTypeBreaking {
			removals++
		}
	}
	if removals == 0 {
		t.Error("unfiltered pair removes no /v1/log-drains operation; the fixture no longer shows the noise")
	}
}

// v1.28.65 (2026-11-17) renamed the path parameter of two DELETEs, {name} to
// {nameOrId}: /v1/drives/... and /v2/sandboxes/drives/..., the second marked
// deprecated. Callers send the same URL as before, so neither is removed.
func TestVercelPathParameterRenameIsNoChange(t *testing.T) {
	ctx := context.Background()
	s := kittest.NewMem()
	if rejected, err := kittest.Pair(ctx, openapitest.Env(s), s, vercelPlugin(""), "v1.28.64", "v1.28.65"); err != nil || rejected {
		t.Fatalf("pair: rejected=%v err=%v", rejected, err)
	}
	if events := kittest.Stored(t, s); len(events) != 0 {
		for _, e := range events {
			t.Logf("got: %s", e.Description)
		}
		t.Fatalf("stored %d events, want none", len(events))
	}

	// Matching by exact path instead, the deprecated one reads as removed.
	s = kittest.NewMem()
	p := vercelPlugin("")
	p.Diff.PathTemplates = false
	if _, err := kittest.Pair(ctx, openapitest.Env(s), s, p, "v1.28.64", "v1.28.65"); err != nil {
		t.Fatal(err)
	}
	events := kittest.Stored(t, s)
	if len(events) != 1 || events[0].Subject.Operation != "DELETE /v2/sandboxes/drives/{name}" {
		t.Errorf("by exact path got %d events, want the one false removal of DELETE /v2/sandboxes/drives/{name}", len(events))
	}
}

func TestVercelDeprecatedSummary(t *testing.T) {
	// Every word-only deprecation form in vercel/sdk's history.
	for _, summary := range []string{
		"Check the price for a domain (deprecated)",
		"Retrieves a list of all the Log Drains (deprecated)",
		"Deprecated: Use DELETE /v1/drives/:name. Requires ...",
		"Deprecated: true. Update Resource Secrets (Deprecated)",
	} {
		if !vercelDeprecatedSummary(summary) {
			t.Errorf("%q: not read as deprecated", summary)
		}
	}
	// Mentions of the word that aren't a deprecation, and no summary at all.
	for _, summary := range []string{
		"Get a list of deprecated Node.js versions",
		"Undeprecated thing",
		"Deprecated runtimes for a project",
		"Update a project (deprecated fields are ignored)",
		"",
	} {
		if vercelDeprecatedSummary(summary) {
			t.Errorf("%q: read as deprecated", summary)
		}
	}
}
