package openapi

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/plugins/openapi/adapter"
)

// esHeads are the elasticsearch-specification branch heads saved under
// testdata/ (trimmed to the endpoints involved, every line unchanged; checked
// 2026-11-19): branch -> full SHA.
var esHeads = map[string]string{
	"8.19": "fd5d063c4824de5d33618b7fca4df0e6c1b8489f",
	"9.0":  "57ecfe5fda6976b94fc8b8830ecb51f4969dccd6",
	"9.1":  "12639d0e5ec07f064c909f15160ff68d608dbc2c",
	"9.2":  "87a82640d8aa1139163dc7071bf00f971ce5e453",
}

type fakeESAdapter struct{}

func (fakeESAdapter) API() string { return "elastic/elasticsearch-api" }

func (fakeESAdapter) LatestTag(context.Context) (string, error) { return "", fmt.Errorf("replay only") }

func (fakeESAdapter) Fetch(_ context.Context, branch string) (adapter.Spec, error) {
	if _, ok := esHeads[branch]; !ok {
		return adapter.Spec{}, fmt.Errorf("no saved copy for %s", branch)
	}
	body, err := os.ReadFile(esFile(branch))
	if err != nil {
		return adapter.Spec{}, err
	}
	return adapter.Spec{Content: body, Version: branch + ".0", SourceURL: esURL(branch)}, nil
}

func esFile(branch string) string {
	return "testdata/elasticsearch-" + branch + "-" + esHeads[branch][:8] + ".json"
}

func esURL(branch string) string {
	return "https://raw.githubusercontent.com/elastic/elasticsearch-specification/" + esHeads[branch] + "/output/openapi/elasticsearch-openapi.json"
}

func esPlugin() Plugin {
	return Plugin{Adapter: fakeESAdapter{}, Diff: elasticsearchDiff, Keep: elasticsearchKeep, RemovalsNeedANewVersion: true}
}

// sameMinor serves two saved specs as two commits of one branch: both 9.0.0.
type sameMinor struct{ fakeESAdapter }

func (s sameMinor) Fetch(ctx context.Context, branch string) (adapter.Spec, error) {
	spec, err := s.fakeESAdapter.Fetch(ctx, branch)
	spec.Version = "9.0.0"
	return spec, err
}

// A release branch's spec gets fixes after its .0 release (the 9.2 branch
// dropped GET /_project/tags after 9.2.0), and Elasticsearch removes APIs in
// a new minor or major, never a patch: a removal between two commits of one
// minor is a correction to the spec, even of a deprecated operation. The
// same specs as the golden test, read as one branch, keep the five
// deprecations and none of the three removals.
func TestElasticsearchNoRemovalWithinAMinor(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)
	p := esPlugin()
	p.Adapter = sameMinor{}
	if rejected, err := kittest.Pair(ctx, vercelEnv(s), s, p, "8.19", "9.0"); err != nil || rejected {
		t.Fatalf("replay: rejected=%v err=%v", rejected, err)
	}
	n := map[event.ChangeType]int{}
	for _, e := range kittest.Stored(t, s) {
		n[e.ChangeType]++
	}
	if n[event.ChangeTypeBreaking] != 0 || n[event.ChangeTypeDeprecation] != 5 {
		t.Errorf("got %v, want 5 deprecations and no removal", n)
	}
}

// The golden test and the historical breaking change: Elasticsearch 9.0.
// GET and POST /{index}/_knn_search and POST /{index}/_unfreeze, deprecated
// in 8.x, are gone ("Remove unfreeze REST endpoint", Elastic's 9.0.0
// breaking changes), and the five /_application/analytics operations are
// deprecated. /_data_stream/{name}/_options, which 8.19 has and the 9.0
// branch (cut earlier) doesn't, was never deprecated: no event.
func TestElasticsearchGolden(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)
	if rejected, err := kittest.Pair(ctx, vercelEnv(s), s, esPlugin(), "8.19", "9.0"); err != nil || rejected {
		t.Fatalf("replay: rejected=%v err=%v", rejected, err)
	}
	events := kittest.Stored(t, s)
	byType := map[event.ChangeType][]string{}
	for _, e := range events {
		byType[e.ChangeType] = append(byType[e.ChangeType], e.Subject.Operation)
		if e.Version != "9.0.0" || e.PreviousVersion != "8.19.0" {
			t.Errorf("%s: version %s from %s", e.Subject.Operation, e.Version, e.PreviousVersion)
		}
		want := esURL("9.0")
		if e.Breaking {
			want = esURL("8.19") // a removal is evidenced by the old spec
		}
		if e.SourceURL != want || !strings.Contains(e.SourceExcerpt, `"deprecated":true`) {
			t.Errorf("%s: cited %s, excerpt %.120s", e.Subject.Operation, e.SourceURL, e.SourceExcerpt)
		}
		file := esFile("9.0")
		if e.Breaking {
			file = esFile("8.19")
		}
		if node := jsonNode(t, file, e.SourcePointer); node != e.SourceExcerpt && !truncated(e.SourceExcerpt, node) {
			t.Errorf("%s: excerpt doesn't match %s#%s", e.Subject.Operation, file, e.SourcePointer)
		}
	}
	breaking := strings.Join(byType[event.ChangeTypeBreaking], ", ")
	for _, op := range []string{"GET /{index}/_knn_search", "POST /{index}/_knn_search", "POST /{index}/_unfreeze"} {
		if !strings.Contains(breaking, op) {
			t.Errorf("no removal of %s in %s", op, breaking)
		}
	}
	if len(byType[event.ChangeTypeBreaking]) != 3 || len(byType[event.ChangeTypeDeprecation]) != 5 {
		t.Errorf("got %v, want 3 removals and 5 /_application/analytics deprecations", byType)
	}
	for _, op := range byType[event.ChangeTypeDeprecation] {
		if !strings.Contains(op, " /_application/analytics") {
			t.Errorf("unexpected deprecation %s", op)
		}
	}
	kittest.Golden(t, "testdata/golden/elasticsearch.json", events)
}

// The pinned drop test. From 9.2 the spec leaves out /_autoscaling and
// /_nodes/{node_id}/shutdown, which Elastic documents as "designed for
// indirect use by Elasticsearch Service, Elastic Cloud Enterprise, and
// Elastic Cloud on Kubernetes. Direct use is not supported." Never
// deprecated: no event, though unfiltered the pair reports them removed.
func TestElasticsearchDropsUndeprecatedRemovals(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)
	if rejected, err := kittest.Pair(ctx, vercelEnv(s), s, esPlugin(), "9.1", "9.2"); err != nil || rejected {
		t.Fatalf("replay: rejected=%v err=%v", rejected, err)
	}
	if events := kittest.Stored(t, s); len(events) != 0 {
		t.Fatalf("%d events, want none", len(events))
	}
	s = openStore(t)
	p := esPlugin()
	p.Keep = nil
	if _, err := kittest.Pair(ctx, vercelEnv(s), s, p, "9.1", "9.2"); err != nil {
		t.Fatal(err)
	}
	var removed int
	for _, e := range kittest.Stored(t, s) {
		if e.Breaking && (strings.Contains(e.Subject.Operation, "/_autoscaling") || strings.Contains(e.Subject.Operation, "/shutdown")) {
			removed++
		}
	}
	if removed != 8 {
		t.Errorf("unfiltered: %d removals, want the 8 autoscaling and shutdown operations", removed)
	}
}
