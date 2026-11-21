package openapi

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/adapter"
)

// supabaseCommits are the spec commits saved under testdata/ (trimmed to the
// operations involved, values unchanged; checked 2026-11-19): short SHA ->
// full SHA and committer date. The source is Prettier-formatted (minified
// before 2025), so the excerpts are re-indented, not copied line for line.
var supabaseCommits = map[string][2]string{
	"57d668c6": {"57d668c6ce265d40cdc009ecadf8e3d80d17a310", "2025-02-10"},
	"bf04ff66": {"bf04ff6681062bde3edf9d646054b9a8b2ef286c", "2025-02-19"},
	"693bdc38": {"693bdc38d045353e5782f86febff4fddc82bb93a", "2026-06-22"},
	"eb21ae25": {"eb21ae25f7954ea1e5db34209ff69fce597b0ae3", "2026-06-29"},
	"8aef4a08": {"8aef4a08bf56e1e0e1eb8af316f7f0b2bcfd2e88", "2024-07-08"},
	"90b71e3e": {"90b71e3e42c5f933013e1a62acdc37c07e9378d7", "2024-07-15"},
	"de806266": {"de806266717abddb08e285f9dd6d17ad7d262b34", "2025-08-04"},
	"a5fc0527": {"a5fc0527045daaafd2a661146cac0b430adad20b", "2025-09-11"},
}

// fakeSupabaseAdapter serves the saved copies, cited by the real URL each was
// copied from, as adapter.Supabase.Fetch cites it.
type fakeSupabaseAdapter struct{ latest string }

func (fakeSupabaseAdapter) API() string { return "supabase/management-api" }

func (f fakeSupabaseAdapter) LatestTag(context.Context) (string, error) { return f.latest, nil }

func (fakeSupabaseAdapter) Fetch(_ context.Context, tag string) (adapter.Spec, error) {
	c, ok := supabaseCommits[tag]
	if !ok {
		return adapter.Spec{}, fmt.Errorf("no saved copy for %s", tag)
	}
	body, err := os.ReadFile(supabaseFile(tag))
	if err != nil {
		return adapter.Spec{}, err
	}
	return adapter.Spec{Content: body, Version: c[1], SourceURL: supabaseURL(tag)}, nil
}

func supabaseFile(tag string) string { return "testdata/supabase-" + tag + ".json" }

func supabaseURL(tag string) string {
	return "https://raw.githubusercontent.com/supabase/supabase/" + supabaseCommits[tag][0] + "/apps/docs/spec/api_v1_openapi.json"
}

// supabasePlugin is the plugin exactly as registered, over the saved copies.
func supabasePlugin(latest string) Plugin {
	return Plugin{Adapter: fakeSupabaseAdapter{latest: latest}, Diff: supabaseDiff, Keep: supabaseKeep}
}

// The golden test: the two operations Supabase deprecated in the
// Management API spec's history. On 2025-02-19, POST
// /v1/projects/{ref}/functions ("This endpoint is deprecated - use the deploy
// endpoint."), the pair 57d668c6 -> bf04ff66. On 2026-06-29, GET
// /v1/projects/{ref}/analytics/endpoints/logs.all, the pair fixture: the
// flag alone, no reason yet. (On 2026-09-30 its description became "This
// endpoint has been removed and always responds with `410 Gone`", with the
// operation still in the spec: a change of words, which no diff sees.)
func TestSupabaseGolden(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)
	env := vercelEnv(s)
	if rejected, err := kittest.Pair(ctx, env, s, supabasePlugin(""), "57d668c6", "bf04ff66"); err != nil || rejected {
		t.Fatalf("pair: rejected=%v err=%v", rejected, err)
	}
	if rejected, err := kittest.Pair(ctx, env, s, supabasePlugin(""), "693bdc38", "eb21ae25"); err != nil || rejected {
		t.Fatalf("pair: rejected=%v err=%v", rejected, err)
	}

	events := kittest.Stored(t, s)
	want := map[string]string{
		"POST /v1/projects/{ref}/functions 2025-02-19":                   "bf04ff66",
		"GET /v1/projects/{ref}/analytics/endpoints/logs.all 2026-06-29": "eb21ae25",
	}
	if len(events) != len(want) {
		for _, e := range events {
			t.Logf("got: %s", e.Description)
		}
		t.Fatalf("stored %d events, want %d", len(events), len(want))
	}
	for _, e := range events {
		tag, ok := want[e.Subject.Operation+" "+e.Version]
		if !ok {
			t.Errorf("unexpected event %s %s", e.Subject.Operation, e.Version)
			continue
		}
		if e.ChangeType != event.ChangeTypeDeprecation || e.Breaking {
			t.Errorf("%s: %s, want deprecation", e.Subject.Operation, e.ChangeType)
		}
		if e.SourceURL != supabaseURL(tag) {
			t.Errorf("%s: source_url %s, want %s", e.Subject.Operation, e.SourceURL, supabaseURL(tag))
		}
		// The whole operation is quoted, so the reason is in the excerpt.
		if node := jsonNode(t, supabaseFile(tag), e.SourcePointer); node != e.SourceExcerpt && !truncated(e.SourceExcerpt, node) {
			t.Errorf("%s: excerpt doesn't match %s#%s", e.Subject.Operation, tag, e.SourcePointer)
		}
		if !strings.Contains(e.SourceExcerpt, `"deprecated":true`) {
			t.Errorf("%s: excerpt lacks the flag: %.200s", e.Subject.Operation, e.SourceExcerpt)
		}
		if tag == "bf04ff66" && !strings.Contains(e.SourceExcerpt, "This endpoint is deprecated - use the deploy endpoint.") {
			t.Errorf("%s: excerpt lacks Supabase's reason: %.200s", e.Subject.Operation, e.SourceExcerpt)
		}
	}
	kittest.Golden(t, "testdata/golden/supabase.json", events)
}

// The pinned drop test. On 2024-07-15 the 14 analytics warehouse operations
// left the spec without ever being marked deprecated. Nothing says they were
// shut down, and without credentials every route answers 401, so it can't
// be checked: no event.
func TestSupabaseDropsRemovalsThatWerentDeprecated(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)
	if rejected, err := kittest.Pair(ctx, vercelEnv(s), s, supabasePlugin(""), "8aef4a08", "90b71e3e"); err != nil || rejected {
		t.Fatalf("pair: rejected=%v err=%v", rejected, err)
	}
	if events := kittest.Stored(t, s); len(events) != 0 {
		t.Fatalf("stored %d events, want none", len(events))
	}
	// Without the filter the same pair reports all 14 removals.
	s = openStore(t)
	p := supabasePlugin("")
	p.Keep = nil
	if _, err := kittest.Pair(ctx, vercelEnv(s), s, p, "8aef4a08", "90b71e3e"); err != nil {
		t.Fatal(err)
	}
	var removals int
	for _, e := range kittest.Stored(t, s) {
		if strings.Contains(e.Subject.Operation, "/analytics/warehouse/") && e.ChangeType == event.ChangeTypeBreaking {
			removals++
		}
	}
	if removals != 14 {
		t.Errorf("unfiltered: %d warehouse removals, want 14", removals)
	}
}

// On 2025-09-11 the /v1/branches/{branch_id} operations became
// /v1/branches/{branch_id_or_ref}: the same URLs. No event; by exact path,
// they'd read as removed.
func TestSupabaseBranchParameterRenameIsNoChange(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)
	p := supabasePlugin("")
	p.Keep = nil // even unfiltered: the operations are paired, not removed
	if _, err := kittest.Pair(ctx, vercelEnv(s), s, p, "de806266", "a5fc0527"); err != nil {
		t.Fatal(err)
	}
	for _, e := range kittest.Stored(t, s) {
		if e.ChangeType == event.ChangeTypeBreaking && strings.Contains(e.Description, "removed") && strings.Contains(e.Subject.Operation, "{branch_id}") {
			t.Errorf("by template: %s", e.Description)
		}
	}
	s = openStore(t)
	p.Diff.PathTemplates = false
	if _, err := kittest.Pair(ctx, vercelEnv(s), s, p, "de806266", "a5fc0527"); err != nil {
		t.Fatal(err)
	}
	var removed int
	for _, e := range kittest.Stored(t, s) {
		if strings.HasPrefix(e.Description, "Operation ") && strings.HasSuffix(e.Description, " removed.") && strings.Contains(e.Subject.Operation, "{branch_id}") {
			removed++
		}
	}
	if removed == 0 {
		t.Error("by exact path no {branch_id} operation reads as removed; the fixture no longer shows the rename")
	}
}
