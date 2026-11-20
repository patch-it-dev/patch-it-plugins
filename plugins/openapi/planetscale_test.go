package openapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/diff"
	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/plugins/openapi/adapter"
)

// planetscaleCommits are the planetscale/terraform-provider-planetscale
// commits whose schemas/openapi.yaml is saved under testdata/ (trimmed to the
// paths involved, every line unchanged; checked 2026-11-19), with their
// committer dates.
var planetscaleCommits = map[string]string{
	"b49c354080667638b79781801f0517ae07bc55b8": "2026-09-10T14:52:36Z",
	"f6d6ad4d3929f6c1ef021eb13c1541ba31f72107": "2026-10-03T00:53:38Z",
	"e631095aa649a6797578a1d51d861b91e0568aa1": "2026-10-23T08:19:59Z",
	"fc13e09d0c18b893aabd3e02d175d13957bba677": "2026-11-09T09:16:32Z",
}

const planetscaleRaw = "https://raw.githubusercontent.com/planetscale/terraform-provider-planetscale"

func planetscaleFile(sha string) string { return "testdata/planetscale-" + sha[:8] + ".yaml" }

// planetscalePlugin is the plugin as registered, its adapter reading GitHub's
// answers and the saved copies from a local server. It returns the base the
// spec URLs start with in place of planetscaleRaw.
func planetscalePlugin(t *testing.T) (Plugin, string) {
	t.Helper()
	const repo = "/planetscale/terraform-provider-planetscale"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if sha, ok := strings.CutPrefix(p, "/api/repos"+repo+"/commits/"); ok && planetscaleCommits[sha] != "" {
			fmt.Fprintf(w, `{"sha":%q,"commit":{"committer":{"date":%q}}}`, sha, planetscaleCommits[sha])
			return
		}
		if rest, ok := strings.CutPrefix(p, "/raw"+repo+"/"); ok {
			sha, file, _ := strings.Cut(rest, "/")
			if planetscaleCommits[sha] != "" && file == "schemas/openapi.yaml" {
				http.ServeFile(w, r, planetscaleFile(sha))
				return
			}
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	a := adapter.PlanetScale{APIBase: srv.URL + "/api", RawBase: srv.URL + "/raw"}
	return Plugin{Adapter: a, Diff: planetscaleDiff, Keep: planetscaleKeep}, srv.URL + "/raw" + repo
}

func replayPlanetScale(t *testing.T, p Plugin, from, to string) []event.Event {
	t.Helper()
	s := openStore(t)
	if rejected, err := kittest.Pair(context.Background(), vercelEnv(s), s, p, from, to); err != nil || rejected {
		t.Fatalf("replay %s -> %s: rejected=%v err=%v", from, to, rejected, err)
	}
	return kittest.Stored(t, s)
}

// The golden test: the only deprecations in the spec's history. On
// 2026-11-09 (fc13e09d) five operations' descriptions gained a first line
// "Deprecated. Use ... instead.", pointing at the change-requests
// endpoints: GET and DELETE .../bouncers/{bouncer}/resizes, GET
// .../neki-changes, GET and DELETE .../neki-changes/{id}. The flag was never
// set, so each event cites the description.
func TestPlanetScaleGolden(t *testing.T) {
	p, base := planetscalePlugin(t)
	const from, to = "e631095aa649a6797578a1d51d861b91e0568aa1", "fc13e09d0c18b893aabd3e02d175d13957bba677"
	events := replayPlanetScale(t, p, from, to)
	const branch = "/v1/organizations/{organization}/databases/{database}/branches/{branch}"
	want := map[string]bool{
		"GET " + branch + "/bouncers/{bouncer}/resizes":    true,
		"DELETE " + branch + "/bouncers/{bouncer}/resizes": true,
		"GET " + branch + "/neki-changes":                  true,
		"GET " + branch + "/neki-changes/{id}":             true,
		"DELETE " + branch + "/neki-changes/{id}":          true,
	}
	if len(events) != len(want) {
		for _, e := range events {
			t.Logf("got: %s", e.Description)
		}
		t.Fatalf("%d events, want %d", len(events), len(want))
	}
	for _, e := range events {
		if !want[e.Subject.Operation] {
			t.Errorf("unexpected event %s", e.Subject.Operation)
		}
		if e.ChangeType != event.ChangeTypeDeprecation || e.Version != "2026-11-09" || !strings.HasSuffix(e.SourcePointer, "/description") {
			t.Errorf("%s: %s %s citing %s", e.Subject.Operation, e.ChangeType, e.Version, e.SourcePointer)
		}
		if url := base + "/" + to + "/schemas/openapi.yaml"; e.SourceURL != url {
			t.Errorf("%s: source_url %s, want %s", e.Subject.Operation, e.SourceURL, url)
		}
		if node := yamlNodeJSON(t, planetscaleFile(to), e.SourcePointer); node != e.SourceExcerpt && !truncated(e.SourceExcerpt, node) {
			t.Errorf("%s: excerpt doesn't match %s#%s", e.Subject.Operation, planetscaleFile(to), e.SourcePointer)
		}
		var description string
		if err := json.Unmarshal([]byte(e.SourceExcerpt), &description); err != nil {
			description = strings.TrimPrefix(e.SourceExcerpt, `"`)
		}
		if !strings.HasPrefix(description, "Deprecated. Use ") {
			t.Errorf("%s: the excerpt doesn't start with PlanetScale's words: %.80s", e.Subject.Operation, e.SourceExcerpt)
		}
	}
	// The golden file cites GitHub, where the local server stood in for it.
	for i := range events {
		events[i].SourceURL = strings.Replace(events[i].SourceURL, base, planetscaleRaw, 1)
	}
	kittest.Golden(t, "testdata/golden/planetscale.json", events)
}

// The pinned drop test. On 2026-10-03 (f6d6ad4d) the read-only replica
// operations left the spec, without being deprecated first, as dedicated
// read replica operations arrived at new paths; PATCH
// .../configuration-profiles/{configuration_profile}/extensions/{extension}
// left too. planetscale/planetscale-go still sends .../read-only-replicas
// (main, 2026-11-19), so leaving the spec isn't a shutdown: no event. The
// same pair unfiltered reports the 7 removals.
func TestPlanetScaleDropsRemovalsThatWerentDeprecated(t *testing.T) {
	p, _ := planetscalePlugin(t)
	const from, to = "b49c354080667638b79781801f0517ae07bc55b8", "f6d6ad4d3929f6c1ef021eb13c1541ba31f72107"
	if events := replayPlanetScale(t, p, from, to); len(events) != 0 {
		t.Fatalf("%d events, want none: %s", len(events), events[0].Description)
	}
	p.Keep = nil
	var removed []string
	for _, e := range replayPlanetScale(t, p, from, to) {
		if e.Breaking && e.Subject.Kind == event.KindOperation && strings.HasSuffix(e.Description, " removed.") {
			removed = append(removed, e.Subject.Operation)
		}
	}
	var replicas int
	for _, op := range removed {
		if strings.Contains(op, "/read-only-replica") {
			replicas++
		}
	}
	if len(removed) != 7 || replicas != 6 {
		t.Errorf("unfiltered: removed %v, want the 6 read-only replica operations and PATCH .../extensions/{extension}", removed)
	}
}

// The words, against the ways the spec writes "deprecated".
func TestPlanetScaleDeprecatedWords(t *testing.T) {
	for desc, want := range map[string]bool{
		"Deprecated. Use GET /organizations/:organization/databases/:database/branches/:branch/change-requests instead.\n### Authorization": true,
		"\nDeprecated: use the change-requests endpoints.":                                                             true,
		"\n### Authorization\nA service token or OAuth token must have at least one of the following access or scopes": false,
		// How the spec describes a deprecated property, not an operation.
		"Deprecated, storage shrinking is part of storage autoscaling.":    false,
		"Returns deprecated branches too.":                                 false,
		"Lists changes.\n\nDeprecated. Use the change-requests endpoints.": false,
	} {
		if got := planetscaleDeprecatedDescription(desc); got != want {
			t.Errorf("%.60q: got %v, want %v", desc, got, want)
		}
	}
	if !planetscaleDiff.OperationDeprecated(map[string]any{"description": "Deprecated. Use x instead."}) {
		t.Error("planetscaleDiff doesn't read the description")
	}
	// A removal is kept only of an operation marked deprecated first.
	marked := diff.Change{Kind: diff.OperationRemoved, Node: map[string]any{"description": "Deprecated. Use x instead."}}
	unmarked := diff.Change{Kind: diff.OperationRemoved, Node: map[string]any{"description": "\n### Authorization"}}
	if !planetscaleKeep(marked) || planetscaleKeep(unmarked) {
		t.Error("planetscaleKeep: want the marked removal only")
	}
}

// planetscaleKept is what the plugin's options and keep rule make of two
// inline specs.
func planetscaleKept(t *testing.T, oldJSON, newJSON string) []diff.Change {
	t.Helper()
	o, err := diff.Parse([]byte(oldJSON))
	if err != nil {
		t.Fatal(err)
	}
	n, err := diff.Parse([]byte(newJSON))
	if err != nil {
		t.Fatal(err)
	}
	var kept []diff.Change
	for _, c := range diff.ChangesWith(o, n, planetscaleDiff) {
		if planetscaleKeep(c) {
			kept = append(kept, c)
		}
	}
	return kept
}

// PathTemplates: a deprecated operation whose path parameter is renamed is
// the same URL, not a removal. Without it, the keep rule would report the
// deprecated operation shut down.
func TestPlanetScaleParameterRenameIsNotARemoval(t *testing.T) {
	const op = `"get":{"description":"Deprecated. Use GET /b instead.","parameters":[{"name":"%s","in":"path","required":true,"schema":{"type":"string"}}],"responses":{"200":{"description":"OK"}}}`
	oldSpec := `{"openapi":"3.0.1","paths":{"/a/{id}":{` + strings.Replace(op, "%s", "id", 1) + `}}}`
	newSpec := `{"openapi":"3.0.1","paths":{"/a/{name}":{` + strings.Replace(op, "%s", "name", 1) + `}}}`
	if kept := planetscaleKept(t, oldSpec, newSpec); len(kept) != 0 {
		t.Errorf("kept %v, want nothing", kept)
	}
}

// QuoteDeprecatedOperation: PlanetScale has never set the `deprecated` flag,
// but if it starts to, the event quotes the operation, with its "Use X
// instead", not a bare `true`.
func TestPlanetScaleFlagQuotesTheOperation(t *testing.T) {
	oldSpec := `{"openapi":"3.0.1","paths":{"/a":{"get":{"description":"Lists a.","responses":{"200":{"description":"OK"}}}}}}`
	newSpec := `{"openapi":"3.0.1","paths":{"/a":{"get":{"deprecated":true,"description":"Lists a. Use GET /b instead.","responses":{"200":{"description":"OK"}}}}}}`
	kept := planetscaleKept(t, oldSpec, newSpec)
	if len(kept) != 1 || kept[0].Kind != diff.OperationDeprecated || kept[0].Pointer != "/paths/~1a/get" {
		t.Errorf("kept %+v, want one deprecation citing /paths/~1a/get", kept)
	}
}
