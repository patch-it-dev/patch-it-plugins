package neon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/adapter"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/openapitest"
)

// neonCommits are the spec commits saved under testdata/ (trimmed to the
// operations involved, every kept line unchanged; checked 2026-11-19): short
// SHA -> full SHA and committer date. The spec's own license is
// "Proprietary", so only these excerpts are saved, never the whole file.
var neonCommits = map[string][2]string{
	"5c57d009": {"5c57d0095a9a4594a3c8b09d499b132902e7c42e", "2026-08-17"},
	"93b93dc9": {"93b93dc9e4d1b4ab48dac6c8ccba76b35541b476", "2026-08-23"},
	"4c6b3be4": {"4c6b3be46b1b333e7f910db51218ad144de27179", "2026-07-25"},
	"4497de89": {"4497de89a5e84b38956fde686d271f99317b8fdb", "2026-08-09"},
}

// fakeNeonAdapter serves the saved copies, cited by the real URL each was
// copied from and reshaped, as Neon.Fetch cites and reshapes it.
type fakeNeonAdapter struct{ latest string }

func (fakeNeonAdapter) API() string { return "neon/api" }

func (f fakeNeonAdapter) LatestTag(context.Context) (string, error) { return f.latest, nil }

func (fakeNeonAdapter) Fetch(_ context.Context, tag string) (adapter.Spec, error) {
	c, ok := neonCommits[tag]
	if !ok {
		return adapter.Spec{}, fmt.Errorf("no saved copy for %s", tag)
	}
	body, err := os.ReadFile(neonFile(tag))
	if err != nil {
		return adapter.Spec{}, err
	}
	// As Neon.Fetch does: /api/v2 moved from the server URL into the
	// paths.
	content, err := adapter.ServerBasePath(body)
	if err != nil {
		return adapter.Spec{}, err
	}
	return adapter.Spec{Content: content, Version: c[1], SourceURL: neonURL(tag)}, nil
}

func (fakeNeonAdapter) SourcePointer(content []byte) func(string) string {
	return adapter.BasePathPointer(content)
}

func neonFile(tag string) string { return "testdata/neon-" + tag + ".json" }

func neonURL(tag string) string {
	return "https://raw.githubusercontent.com/neondatabase/neon-pkgs/" + neonCommits[tag][0] + "/packages/sdk/spec/neon-openapi.json"
}

// neonPlugin is the plugin exactly as registered, over the given adapter.
func neonPlugin(a adapter.Adapter) openapi.Plugin {
	return openapi.Plugin{Adapter: a, Diff: neonDiff, Keep: neonKeep}
}

// The golden test, the pair fixture: the one operation Neon deprecated in
// neon-pkgs's copy of the spec. On 2026-08-23 POST
// /projects/{project_id}/branches/{branch_id}/auth/send_test_email (beta)
// gained the flag, an x-sunset of 2027-02-28 and "Deprecated: to test a
// branch's already-saved configuration, use `sendNeonAuthEmailProviderTest`".
// The same commit added that operation (dropped: new), and GET
// .../auth/email_server, deprecated in both, is no new event.
func TestNeonGolden(t *testing.T) {
	ctx := context.Background()
	s := kittest.NewMem()
	if rejected, err := kittest.Pair(ctx, openapitest.Env(s), s, neonPlugin(fakeNeonAdapter{}), "5c57d009", "93b93dc9"); err != nil || rejected {
		t.Fatalf("pair: rejected=%v err=%v", rejected, err)
	}
	events := kittest.Stored(t, s)
	if len(events) != 1 {
		for _, e := range events {
			t.Logf("got: %s", e.Description)
		}
		t.Fatalf("stored %d events, want 1", len(events))
	}
	e := events[0]
	if e.Subject.Operation != "POST /api/v2/projects/{project_id}/branches/{branch_id}/auth/send_test_email" || e.Version != "2026-08-23" || e.PreviousVersion != "2026-08-17" {
		t.Errorf("got %s %s (after %s)", e.Subject.Operation, e.Version, e.PreviousVersion)
	}
	if e.ChangeType != event.ChangeTypeDeprecation || e.Breaking {
		t.Errorf("%s, want a deprecation", e.ChangeType)
	}
	if e.SourceURL != neonURL("93b93dc9") {
		t.Errorf("source_url %s", e.SourceURL)
	}
	// The whole operation is quoted, so the reason and the sunset are in it.
	if node := openapitest.JSONNode(t, neonFile("93b93dc9"), e.SourcePointer); node != e.SourceExcerpt && !openapitest.Truncated(e.SourceExcerpt, node) {
		t.Errorf("excerpt doesn't match 93b93dc9#%s", e.SourcePointer)
	}
	for _, want := range []string{`"deprecated":true`, "use `sendNeonAuthEmailProviderTest`", `"x-sunset":"2027-02-28"`} {
		if !strings.Contains(e.SourceExcerpt, want) {
			t.Errorf("excerpt lacks %s: %.300s", want, e.SourceExcerpt)
		}
	}
	kittest.Golden(t, "testdata/golden/neon.json", events)
}

// The pinned drop test. On 2026-08-09 POST
// /projects/{project_id}/branches/{branch_id}/recover left the spec without
// ever being marked deprecated. neon-pkgs's PR #408 checked: "The endpoint
// was not removed from the API. It was removed from the published spec." No
// event.
func TestNeonDropsRemovalsThatWerentDeprecated(t *testing.T) {
	ctx := context.Background()
	s := kittest.NewMem()
	if rejected, err := kittest.Pair(ctx, openapitest.Env(s), s, neonPlugin(fakeNeonAdapter{}), "4c6b3be4", "4497de89"); err != nil || rejected {
		t.Fatalf("pair: rejected=%v err=%v", rejected, err)
	}
	if events := kittest.Stored(t, s); len(events) != 0 {
		t.Fatalf("stored %d events, want none", len(events))
	}
	// Without the filter the same pair reports the removal.
	s = kittest.NewMem()
	p := neonPlugin(fakeNeonAdapter{})
	p.Keep = nil
	if _, err := kittest.Pair(ctx, openapitest.Env(s), s, p, "4c6b3be4", "4497de89"); err != nil {
		t.Fatal(err)
	}
	var removals []string
	for _, e := range kittest.Stored(t, s) {
		if e.ChangeType == event.ChangeTypeBreaking && strings.HasPrefix(e.Description, "Operation ") && strings.HasSuffix(e.Description, " removed.") {
			removals = append(removals, e.Subject.Operation)
		}
	}
	if len(removals) != 1 || removals[0] != "POST /api/v2/projects/{project_id}/branches/{branch_id}/recover" {
		t.Errorf("unfiltered removals %v, want only POST .../branches/{branch_id}/recover", removals)
	}
}

// neonAPIClient is a different source from the plugin's: the same spec as
// shipped (YAML) in the npm package @neondatabase/api-client, which @neon/sdk
// succeeded (no release since 2.7.3, 2026-07-01). neon-pkgs's history starts on
// 2026-06-24, after the one removal Neon announced, so the rule for removals
// is checked here, on real Neon data: GET /consumption_history/account,
// deprecated (with x-sunset 2026-06-01) in 2.7.2, gone in 2.7.3. Neon's
// changelog of 2026-04-03 called it "deprecated, with a planned sunset of June
// 1, 2026", and that of 2026-06-19 "The deprecated `GET
// /consumption_history/account` endpoint has been removed."
type neonAPIClient struct{}

func (neonAPIClient) API() string { return "neon/api" }

func (neonAPIClient) LatestTag(context.Context) (string, error) { return "2.7.3", nil }

func (neonAPIClient) Fetch(_ context.Context, version string) (adapter.Spec, error) {
	body, err := os.ReadFile("testdata/neon-api-client-" + version + ".yaml")
	if err != nil {
		return adapter.Spec{}, err
	}
	var doc any
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return adapter.Spec{}, err
	}
	content, err := json.Marshal(doc)
	if err != nil {
		return adapter.Spec{}, err
	}
	if content, err = adapter.ServerBasePath(content); err != nil { // as Neon.Fetch does
		return adapter.Spec{}, err
	}
	return adapter.Spec{Content: content, Version: version, SourceURL: "https://registry.npmjs.org/@neondatabase/api-client/-/api-client-" + version + ".tgz"}, nil
}

func TestNeonKeepsTheRemovalOfADeprecatedOperation(t *testing.T) {
	ctx := context.Background()
	s := kittest.NewMem()
	if rejected, err := kittest.Pair(ctx, openapitest.Env(s), s, neonPlugin(neonAPIClient{}), "2.7.2", "2.7.3"); err != nil || rejected {
		t.Fatalf("pair: rejected=%v err=%v", rejected, err)
	}
	events := kittest.Stored(t, s)
	if len(events) != 1 {
		for _, e := range events {
			t.Logf("got: %s", e.Description)
		}
		t.Fatalf("stored %d events, want 1", len(events))
	}
	e := events[0]
	if e.Subject.Operation != "GET /api/v2/consumption_history/account" || e.ChangeType != event.ChangeTypeBreaking || !e.Breaking {
		t.Errorf("got %s %s breaking=%v, want GET /consumption_history/account removed", e.Subject.Operation, e.ChangeType, e.Breaking)
	}
	if !strings.Contains(e.SourceExcerpt, `"deprecated":true`) {
		t.Errorf("excerpt lacks the flag the removal was kept for: %.300s", e.SourceExcerpt)
	}
}
