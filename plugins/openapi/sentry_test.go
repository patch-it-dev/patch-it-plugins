package openapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/diff"
	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/plugins/openapi/adapter"
)

// The getsentry/sentry-api-schema commits and release tags saved under
// testdata/ (trimmed to the paths involved, every line unchanged; checked
// 2026-11-19), with the dates GitHub gives them: a commit's committer date,
// a tag's release publication.
var (
	sentryCommits = map[string]string{
		"ead7f0cf9ccb8ac817105ac11bb94fe316fcb337": "2024-07-31T20:37:27Z",
		"4650a66a859e015cccef2c171b115633321768e9": "2024-07-31T22:59:10Z",
		"d64149d65723484ca4d34c956ad017a66e1a4e07": "2025-07-25T16:37:23Z",
		"c2f638f601baadcb055aec4ca3c2ba9f2654c087": "2025-07-25T19:03:21Z",
		"47231e6180b34f222ed462b1d0cb3cdb951cc4e7": "2026-01-21T18:50:44Z",
		"1da1f31fac696201b1a7d20046fc199c8b1324b6": "2026-01-22T21:11:05Z",
		"94bc6e593eaf8a3800341abd429d2949895926fd": "2022-06-15T23:41:53Z",
		"d15557ff7472748887043ab8a72687b20214a9bd": "2022-06-16T18:26:31Z",
		"e8ec23b4c9601dea2d51205983ec1d7b89550b85": "2026-10-14T07:33:16Z",
		"1ef7f21a0501fae9490d69e3ae29f0d5acbfb61c": "2026-10-15T14:07:38Z",
	}
	sentryTags = map[string]string{
		"0.223.0": "2026-06-16T21:16:09Z",
		"0.224.0": "2026-06-16T23:16:45Z",
		"2.0.1":   "2026-11-13T14:42:22Z",
		"2.1.0":   "2026-11-13T15:52:17Z",
	}
)

// sentryFile is the saved copy of a commit or tag.
func sentryFile(ref string) string {
	if _, ok := sentryTags[ref]; ok {
		return "testdata/sentry-" + ref + ".json"
	}
	return "testdata/sentry-" + ref[:8] + ".json"
}

// sentryPlugin is the plugin as registered for Sentry, its adapter reading
// GitHub's answers and the saved copies from a local server. It returns the
// base of the spec URLs: <base>/<ref>/openapi-derefed.json.
func sentryPlugin(t *testing.T) (Plugin, string) {
	t.Helper()
	const schema = "/api/repos/getsentry/sentry-api-schema"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if sha, ok := strings.CutPrefix(p, schema+"/commits/"); ok && sentryCommits[sha] != "" {
			fmt.Fprintf(w, `{"sha":%q,"commit":{"committer":{"date":%q}}}`, sha, sentryCommits[sha])
			return
		}
		if tag, ok := strings.CutPrefix(p, schema+"/releases/tags/"); ok && sentryTags[tag] != "" {
			fmt.Fprintf(w, `{"tag_name":%q,"published_at":%q}`, tag, sentryTags[tag])
			return
		}
		if rest, ok := strings.CutPrefix(p, "/raw/getsentry/sentry-api-schema/"); ok {
			ref, file, _ := strings.Cut(rest, "/")
			if _, known := sentryCommits[ref]; (known || sentryTags[ref] != "") && file == "openapi-derefed.json" {
				http.ServeFile(w, r, sentryFile(ref))
				return
			}
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	a := adapter.Sentry{APIBase: srv.URL + "/api", RawBase: srv.URL + "/raw"}
	return Plugin{Adapter: a, Diff: sentryDiff, Keep: sentryKeep}, srv.URL + "/raw/getsentry/sentry-api-schema"
}

func replaySentry(t *testing.T, p Plugin, pairs ...[2]string) []event.Event {
	t.Helper()
	s := openStore(t)
	for _, pair := range pairs {
		if rejected, err := kittest.Pair(context.Background(), vercelEnv(s), s, p, pair[0], pair[1]); err != nil || rejected {
			t.Fatalf("replay %s -> %s: rejected=%v err=%v", pair[0], pair[1], rejected, err)
		}
	}
	return kittest.Stored(t, s)
}

// The golden test: the three times Sentry deprecated operations in their
// descriptions' words, none with the flag at the time.
//
//   - 2024-07-31: POST /api/0/projects/{organization_id_or_slug}/{project_id_or_slug}/user-feedback/,
//     "*This endpoint is DEPRECATED. We document it here for older SDKs ...".
//   - 2025-07-25: GET .../issues/, "**Deprecated**: This endpoint has been
//     replaced with the [Organization Issues endpoint]".
//   - 2026-01-22: the 10 issue and metric alert-rule operations, "## Deprecated
//     🚧 Use [Fetch an Organization's Monitors](...)" and the like, never
//     flagged before they left the spec.
//
// Each event cites the description, in the commit's file.
func TestSentryGolden(t *testing.T) {
	p, base := sentryPlugin(t)
	events := replaySentry(t, p,
		[2]string{"ead7f0cf9ccb8ac817105ac11bb94fe316fcb337", "4650a66a859e015cccef2c171b115633321768e9"},
		[2]string{"d64149d65723484ca4d34c956ad017a66e1a4e07", "c2f638f601baadcb055aec4ca3c2ba9f2654c087"},
		[2]string{"47231e6180b34f222ed462b1d0cb3cdb951cc4e7", "1da1f31fac696201b1a7d20046fc199c8b1324b6"},
	)
	byVersion := map[string]int{}
	for _, e := range events {
		byVersion[e.Version]++
		if e.ChangeType != event.ChangeTypeDeprecation || !strings.HasSuffix(e.SourcePointer, "/description") {
			t.Errorf("%s: %s citing %s", e.Subject.Operation, e.ChangeType, e.SourcePointer)
		}
		if !strings.Contains(e.Subject.Operation, " /api/0/") {
			t.Errorf("%s: want the path a caller sends, with /api/0", e.Subject.Operation)
		}
		ref := strings.TrimSuffix(strings.TrimPrefix(e.SourceURL, base+"/"), "/openapi-derefed.json")
		if _, ok := sentryCommits[ref]; !ok {
			t.Fatalf("%s: cited %s", e.Subject.Operation, e.SourceURL)
		}
		if node := jsonNode(t, sentryFile(ref), e.SourcePointer); node != e.SourceExcerpt && !truncated(e.SourceExcerpt, node) {
			t.Errorf("%s: excerpt doesn't match %s#%s", e.Subject.Operation, sentryFile(ref), e.SourcePointer)
		}
		// The excerpt is the description, JSON-encoded and maybe cut at 2,000
		// characters: the words are at its start.
		var description string
		if err := json.Unmarshal([]byte(e.SourceExcerpt), &description); err != nil {
			description = strings.ReplaceAll(strings.TrimPrefix(e.SourceExcerpt, `"`), `\n`, "\n")
		}
		if !sentryDeprecatedDescription(description) {
			t.Errorf("%s: the excerpt doesn't show the words: %.80s", e.Subject.Operation, e.SourceExcerpt)
		}
	}
	if want := map[string]int{"2024-07-31": 1, "2025-07-25": 1, "2026-01-22": 10}; fmt.Sprint(byVersion) != fmt.Sprint(want) {
		t.Errorf("events by version: %v, want %v", byVersion, want)
	}
	// The golden file cites GitHub, where the local server stood in for it.
	for i := range events {
		events[i].SourceURL = strings.Replace(events[i].SourceURL, base, "https://raw.githubusercontent.com/getsentry/sentry-api-schema", 1)
	}
	kittest.Golden(t, "testdata/golden/sentry.json", events)
}

// 0.224.0 (2026-06-16) set the `deprecated` flag on user-feedback and issues,
// almost 23 and 11 months after their descriptions said so: no new event, they
// were deprecated when the words came. Read by the flag alone, the pair would
// date both deprecations 2026-06-16.
func TestSentryFlagAfterWordsIsNotNew(t *testing.T) {
	p, _ := sentryPlugin(t)
	if events := replaySentry(t, p, [2]string{"0.223.0", "0.224.0"}); len(events) != 0 {
		t.Fatalf("%d events, want none", len(events))
	}
	p.Diff.DeprecatedDescription = nil
	events := replaySentry(t, p, [2]string{"0.223.0", "0.224.0"})
	var ops []string
	for _, e := range events {
		if strings.HasSuffix(e.SourcePointer, "/description") || !strings.Contains(e.SourceExcerpt, `"deprecated":true`) {
			t.Errorf("%s: cited %s", e.Subject.Operation, e.SourcePointer)
		}
		ops = append(ops, e.Subject.Operation)
	}
	if want := "GET /api/0/projects/{organization_id_or_slug}/{project_id_or_slug}/issues/, POST /api/0/projects/{organization_id_or_slug}/{project_id_or_slug}/user-feedback/"; strings.Join(ops, ", ") != want {
		t.Errorf("by the flag alone: %v, want %s", ops, want)
	}
}

// rawSentry serves the saved copies as published, without the adapter's
// moving /api/0 into the paths.
type rawSentry struct{}

func (rawSentry) API() string                               { return "sentry/api" }
func (rawSentry) LatestTag(context.Context) (string, error) { return "", fmt.Errorf("replay only") }
func (rawSentry) Fetch(_ context.Context, tag string) (adapter.Spec, error) {
	body, err := os.ReadFile(sentryFile(tag))
	return adapter.Spec{Content: body, Version: sentryTags[tag][:10], SourceURL: "https://example.invalid/" + tag}, err
}

// The pinned drop test. sentry-api-schema 2.1.0 (2026-11-13) moved /api/0
// from every path into the server URL: the same URLs, so no event. Compared
// as published, every operation (289 in the full files) reads as removed and
// added again, including GET .../issues/, which 2.0.1 flags deprecated: a
// rule that kept removals of deprecated operations would report it shut down.
func TestSentryBasePathMoveIsNoChange(t *testing.T) {
	p, base := sentryPlugin(t)
	if events := replaySentry(t, p, [2]string{"2.0.1", "2.1.0"}); len(events) != 0 {
		t.Fatalf("%d events, want none", len(events))
	}

	// Unfiltered, what the move leaves is a few property changes, and every
	// pointer names a node in the file as published, without /api/0.
	p.Keep = nil
	for _, e := range replaySentry(t, p, [2]string{"2.0.1", "2.1.0"}) {
		if e.ChangeType == event.ChangeTypeBreaking && e.Subject.Kind == event.KindOperation {
			t.Errorf("unfiltered: %s %s", e.ChangeType, e.Subject.Operation)
		}
		ref := strings.TrimSuffix(strings.TrimPrefix(e.SourceURL, base+"/"), "/openapi-derefed.json")
		if node := jsonNode(t, sentryFile(ref), e.SourcePointer); node != e.SourceExcerpt && !truncated(e.SourceExcerpt, node) {
			t.Errorf("%s: %s doesn't name the excerpt in %s", e.Subject.Operation, e.SourcePointer, sentryFile(ref))
		}
	}

	// As published: every operation of 2.0.1 removed.
	raw := Plugin{Adapter: rawSentry{}, Diff: sentryDiff}
	var removed []string
	for _, e := range replaySentry(t, raw, [2]string{"2.0.1", "2.1.0"}) {
		if e.ChangeType == event.ChangeTypeBreaking && e.Subject.Kind == event.KindOperation {
			removed = append(removed, e.Subject.Operation)
		}
	}
	if len(removed) != 5 {
		t.Errorf("as published: removed %v, want the 5 operations of the saved 2.0.1", removed)
	}
	raw.Keep = deprecationsOnly(sentryDiff)
	events := replaySentry(t, raw, [2]string{"2.0.1", "2.1.0"})
	if len(events) != 1 || events[0].Subject.Operation != "GET /api/0/projects/{organization_id_or_slug}/{project_id_or_slug}/issues/" || !events[0].Breaking {
		t.Errorf("as published, removals of deprecated operations: %d events, want GET .../issues/ removed", len(events))
	}
}

// Three descriptions say "deprecation" without deprecating their operation:
// GET /api/0/organizations/{organization_slug}/events/ gained an "Eventsv2
// Deprecation Note" (about another endpoint) on 2022-06-16, and the same
// note moved with it to {organization_id_or_slug} on 2024-05-10; GET
// .../trace-items/attributes/ gained "examples, and deprecation details" on
// 2026-10-15. No event, where a match on "deprecat" anywhere would make one.
func TestSentryDeprecationWordsThatArentDeprecations(t *testing.T) {
	pairs := [][2]string{
		{"94bc6e593eaf8a3800341abd429d2949895926fd", "d15557ff7472748887043ab8a72687b20214a9bd"},
		{"e8ec23b4c9601dea2d51205983ec1d7b89550b85", "1ef7f21a0501fae9490d69e3ae29f0d5acbfb61c"},
	}
	p, _ := sentryPlugin(t)
	if events := replaySentry(t, p, pairs...); len(events) != 0 {
		t.Fatalf("%d events, want none: %v", len(events), events[0].Subject.Operation)
	}
	loose := regexp.MustCompile(`(?i)deprecat`)
	p.Diff.DeprecatedDescription = loose.MatchString
	if events := replaySentry(t, p, pairs...); len(events) != 2 {
		t.Errorf("matching %q: %d events, want the 2 wrong ones", loose, len(events))
	}
}

// The patterns against each way the history says it, and the three
// descriptions that only mention deprecation.
func TestSentryDeprecatedWords(t *testing.T) {
	for desc, want := range map[string]bool{
		"## Deprecated\n🚧 Use [Fetch an Organization's Monitors](/api/monitors/fetch-an-organizations-monitors/) instead.":                                                        true,
		"## Deprecated\n 🚧 Use [Delete a Monitor](/api/monitors/delete-a-monitor/) instead.":                                                                                      true,
		"**Deprecated**: This endpoint has been replaced with the [Organization Issues endpoint]":                                                                                 true,
		"\n*This endpoint is DEPRECATED. We document it here for older SDKs":                                                                                                      true,
		"This endpoint is deprecated in favor of [Organization Stats V2](/api/organizations/retrieve-event-counts-for-an-organization-v2/).":                                      true,
		"Retrieves discover (also known as events) data for a given organization.\n\n**Eventsv2 Deprecation Note**: Users who may be using the `eventsv2` endpoint should update": false,
		"Pass `expand=context` (experimental) to include each attribute's metadata:\na brief description, examples, and deprecation details.":                                     false,
		"Returns deprecated fields too.": false,
		// Sentry's own style for its notes, the colon inside the bold.
		"**Deprecated:** Use the monitors endpoints instead.": true,
		"## Deprecated 🚧\nUse monitors.":                      true,
		// A heading about part of the operation, not the operation.
		"## Deprecated fields\n- `status`":                false,
		"### Deprecated parameters\n`cursor` is ignored.": false,
	} {
		if got := sentryDeprecatedDescription(desc); got != want {
			t.Errorf("%.60q: got %v, want %v", desc, got, want)
		}
	}
	if !sentryDiff.OperationDeprecated(map[string]any{"description": "## Deprecated\nUse monitors."}) {
		t.Error("sentryDiff doesn't read the description")
	}
	if c := (diff.Change{Kind: diff.OperationRemoved, Node: map[string]any{"deprecated": true}}); sentryKeep(c) {
		t.Error("sentryKeep keeps a removal of a deprecated operation")
	}
}
