package openapi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/adapter"
	"github.com/patch-it-dev/patch-it-plugins/diff"
	"github.com/patch-it-dev/patch-it-plugins/event"
	"github.com/patch-it-dev/patch-it-plugins/kit/kittest"
	"gopkg.in/yaml.v3"
)

// fakeAlgoliaAdapter serves the trimmed copies of specs/bundled/<spec>.yml
// saved under testdata/ (every line unchanged; checked 2026-11-19), converted
// from YAML to JSON like adapter.Algolia.Fetch, and cited by the real URL.
type fakeAlgoliaAdapter struct{ spec string }

func (f fakeAlgoliaAdapter) API() string { return "algolia-" + f.spec }

func (fakeAlgoliaAdapter) LatestTag(context.Context) (string, error) {
	return "", fmt.Errorf("replay only")
}

func (f fakeAlgoliaAdapter) Fetch(_ context.Context, tag string) (adapter.Spec, error) {
	body, err := os.ReadFile(algoliaFile(f.spec, tag))
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
	return adapter.Spec{Content: content, Version: strings.TrimPrefix(tag, "released-")[:10], SourceURL: algoliaURL(f.spec, tag)}, nil
}

func algoliaFile(spec, tag string) string {
	return "testdata/algolia-" + spec + "-" + strings.TrimPrefix(tag, "released-") + ".yml"
}

func algoliaURL(spec, tag string) string {
	return "https://raw.githubusercontent.com/algolia/api-clients-automation/" + tag + "/specs/bundled/" + spec + ".yml"
}

// algoliaPlugin is the plugin exactly as registered, over the saved copies.
func algoliaPlugin(spec string) Plugin {
	return Plugin{Adapter: fakeAlgoliaAdapter{spec: spec}, Diff: algoliaDiff, Keep: algoliaKeep}
}

// The golden tests, one per API with deprecations in its history, each a
// replay of the release that made them.
//
//   - Search API, released-2025-04-01-c19ee6afb: the nine multi-cluster
//     operations under /1/clusters. (The copies also keep /saveObjects, a
//     client helper; it doesn't change here, so the filter for helpers is
//     tested on its own below.)
//   - Ingestion API, released-2025-01-22-232affcac: eight /1/tasks
//     operations. PUT /1/tasks/{taskID}/disable was already deprecated
//     (2024-07-26), so it's no new event.
func TestAlgoliaGolden(t *testing.T) {
	for _, c := range []struct {
		spec, from, to string
		ops            int
	}{
		{"search", "released-2025-03-25-5c28957ec", "released-2025-04-01-c19ee6afb", 9},
		{"ingestion", "released-2025-01-07-d0417f325", "released-2025-01-22-232affcac", 8},
	} {
		t.Run(c.spec, func(t *testing.T) {
			ctx := context.Background()
			s := openStore(t)
			if rejected, err := kittest.Replay(ctx, vercelEnv(s), s, algoliaPlugin(c.spec), c.from, c.to); err != nil || rejected {
				t.Fatalf("replay: rejected=%v err=%v", rejected, err)
			}
			events := kittest.Stored(t, s)
			if len(events) != c.ops {
				for _, e := range events {
					t.Logf("got: %s", e.Description)
				}
				t.Fatalf("%d events, want %d", len(events), c.ops)
			}
			for _, e := range events {
				if e.ChangeType != event.ChangeTypeDeprecation || e.Version != c.to[9:19] || e.SourceURL != algoliaURL(c.spec, c.to) {
					t.Errorf("%s: %s %s %s", e.Subject.Operation, e.ChangeType, e.Version, e.SourceURL)
				}
				if !strings.Contains(e.SourceExcerpt, `"deprecated":true`) {
					t.Errorf("%s: excerpt lacks the flag: %.150s", e.Subject.Operation, e.SourceExcerpt)
				}
				if c.spec == "search" && !strings.HasPrefix(e.Subject.Operation, "GET /1/clusters") && !strings.HasPrefix(e.Subject.Operation, "POST /1/clusters") && !strings.HasPrefix(e.Subject.Operation, "DELETE /1/clusters") {
					t.Errorf("unexpected search event %s", e.Subject.Operation)
				}
				if node := yamlNodeJSON(t, algoliaFile(c.spec, c.to), e.SourcePointer); node != e.SourceExcerpt && !truncated(e.SourceExcerpt, node) {
					t.Errorf("%s: excerpt doesn't match %s#%s", e.Subject.Operation, c.to, e.SourcePointer)
				}
			}
			kittest.Golden(t, "testdata/golden/algolia-"+c.spec+".json", events)
		})
	}
}

// The pinned drop test. The Ingestion API's GET and POST
// /1/transformations/models left the spec in released-2024-09-19-a62c27d3b
// without being deprecated first; nothing says Algolia stopped serving them.
// No event, though the same pair unfiltered reports both removals.
func TestAlgoliaDropsRemovalsThatWerentDeprecated(t *testing.T) {
	ctx := context.Background()
	const from, to = "released-2024-09-18-cc190fae3", "released-2024-09-19-a62c27d3b"
	s := openStore(t)
	if rejected, err := kittest.Replay(ctx, vercelEnv(s), s, algoliaPlugin("ingestion"), from, to); err != nil || rejected {
		t.Fatalf("replay: rejected=%v err=%v", rejected, err)
	}
	if events := kittest.Stored(t, s); len(events) != 0 {
		t.Fatalf("%d events, want none", len(events))
	}
	s = openStore(t)
	p := algoliaPlugin("ingestion")
	p.Keep = nil
	if _, err := kittest.Replay(ctx, vercelEnv(s), s, p, from, to); err != nil {
		t.Fatal(err)
	}
	var removed int
	for _, e := range kittest.Stored(t, s) {
		if strings.HasSuffix(e.Subject.Operation, " /1/transformations/models") && e.Breaking {
			removed++
		}
	}
	if removed != 2 {
		t.Errorf("unfiltered: %d /1/transformations/models removals, want 2", removed)
	}
}

// A client helper or a custom-request method is an API client's, not a REST
// operation: deprecated or removed, it makes no event. A REST operation does.
func TestAlgoliaClientOnlyEntriesMakeNoEvent(t *testing.T) {
	for name, c := range map[string]struct {
		op   map[string]any
		kind diff.Kind
		want bool
	}{
		"helper deprecated":    {map[string]any{"x-helper": true, "operationId": "saveObjects", "deprecated": true}, diff.OperationDeprecated, false},
		"custom removed":       {map[string]any{"operationId": "customPost", "deprecated": true}, diff.OperationRemoved, false},
		"rest deprecated":      {map[string]any{"operationId": "listClusters", "deprecated": true}, diff.OperationDeprecated, true},
		"rest removed, marked": {map[string]any{"operationId": "listClusters", "deprecated": true}, diff.OperationRemoved, true},
		"rest removed, not":    {map[string]any{"operationId": "listClusters"}, diff.OperationRemoved, false},
	} {
		if got := algoliaKeep(diff.Change{Kind: c.kind, Node: c.op}); got != c.want {
			t.Errorf("%s: kept=%v, want %v", name, got, c.want)
		}
	}
}

// algolia-recommend's pinned drop test. Its history has no deprecation; the
// only removals were the custom-request methods, /1{path} becoming /{path}
// in released-2024-03-25-30207fcd0: client methods, not REST operations, and
// not deprecated first. No event, though unfiltered the pair reports the
// four removals.
func TestAlgoliaRecommendCustomRequestMoveIsNoEvent(t *testing.T) {
	ctx := context.Background()
	const from, to = "released-2024-03-19-2eac6f410", "released-2024-03-25-30207fcd0"
	s := openStore(t)
	if rejected, err := kittest.Replay(ctx, vercelEnv(s), s, algoliaPlugin("recommend"), from, to); err != nil || rejected {
		t.Fatalf("replay: rejected=%v err=%v", rejected, err)
	}
	if events := kittest.Stored(t, s); len(events) != 0 {
		t.Fatalf("%d events, want none", len(events))
	}
	s = openStore(t)
	p := algoliaPlugin("recommend")
	p.Keep = nil
	if _, err := kittest.Replay(ctx, vercelEnv(s), s, p, from, to); err != nil {
		t.Fatal(err)
	}
	var removed int
	for _, e := range kittest.Stored(t, s) {
		if strings.HasSuffix(e.Subject.Operation, " /1{path}") && e.Breaking {
			removed++
		}
	}
	if removed != 4 {
		t.Errorf("unfiltered: %d /1{path} removals, want 4", removed)
	}
}
