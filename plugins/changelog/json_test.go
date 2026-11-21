package changelog

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

const (
	atlasSHA     = "c0f99ca9145e5aa74d08ae4451020857e93b5f28" // 2026-11-18 20:15 UTC, "Release API Changelog"
	atlasFixture = "testdata/mongodb-atlas-changelog-c0f99ca9.json"
)

// atlasBefore is the commit that changed the changelog before atlasSHA
// (2026-11-09 20:16 UTC, "Release API Changelog").
var atlasBefore = plugin.Version{ID: "2026-11-09", Ref: "f49749a61f67254d63931bbbefc8880dabd18054"}

// logPair is the pair atlasBefore -> sha with the changelog at sha saved as
// file. The commit before is named, not read: every change in file is
// extracted, as if the file before had none.
func logPair(t *testing.T, sha, file string) []event.Event {
	t.Helper()
	env := kittest.Env(t, kittest.Pages{MongoDBAtlas.URL(sha): file})
	p := Plugin{Log: MongoDBAtlas}
	to, err := p.ReadAt(context.Background(), env, plugin.Version{ID: "2026-11-18", Ref: sha})
	if err != nil {
		t.Fatal(err)
	}
	return kittest.Extracted(t, env, p, plugin.Snapshot{Version: atlasBefore}, to)
}

// The golden test: MongoDB's changelog as released at c0f99ca9 (2026-11-18),
// with every change this plugin doesn't read replaced by {} so that every
// pointer is the one in the real file (testdata/README.md).
func TestJSONGolden(t *testing.T) {
	got := logPair(t, atlasSHA, atlasFixture)
	kittest.Golden(t, "testdata/golden/mongodb-atlas.json", got)

	// Every event cites the pinned file, and its excerpt is the change its
	// pointer names there.
	for _, e := range got {
		if e.SourceURL != MongoDBAtlas.URL(atlasSHA) {
			t.Errorf("%s: source_url %s", e.Subject.Operation, e.SourceURL)
		}
		if node := nodeAt(t, atlasFixture, e.SourcePointer); !sameJSON(node, e.SourceExcerpt) {
			t.Errorf("%s: excerpt %s, but %s holds %s", e.Subject.Operation, e.SourceExcerpt, e.SourcePointer, node)
		}
	}
}

// The historical breaking change: on 2026-01-22 Atlas removed API version
// 2023-01-01 of the shared-tier ("tenant") backup endpoints, "as it has
// reached its sunset date '2026-01-22'".
func TestTenantBackupSunset(t *testing.T) {
	fs, err := MongoDBAtlas.Read(kittest.Read(t, atlasFixture))
	if err != nil {
		t.Fatal(err)
	}
	var n int
	for _, f := range fs {
		if !strings.Contains(f.Operation, "/backup/tenant/") || f.Version != "2023-01-01" {
			continue
		}
		n++
		if f.ChangeType != event.ChangeTypeBreaking || f.Date != "2026-01-22" || f.Code != "endpoint-removed" {
			t.Errorf("%s: %s %s %s", f.Operation, f.ChangeType, f.Date, f.Code)
		}
		if !strings.Contains(f.Excerpt, "has reached its sunset date '2026-01-22'") {
			t.Errorf("%s: excerpt %s", f.Operation, f.Excerpt)
		}
	}
	if n == 0 {
		t.Error("no tenant backup removal read")
	}
}

// The pinned drop test: the changelog's newest day (2026-11-18), whole and
// unchanged: 104 changes, 23 of them marked not backward compatible
// (`request-property-one-of-removed`, …), none an endpoint's life cycle.
// No event.
func TestPropertyChangesMakeNoEvent(t *testing.T) {
	const sha = "0000000000000000000000000000000000000001"
	file := "testdata/mongodb-atlas-changelog-2026-11-18-day.json"
	var days []day
	if err := json.Unmarshal(kittest.Read(t, file), &days); err != nil {
		t.Fatal(err)
	}
	var notBC int
	for _, p := range days[0].Paths {
		for _, v := range p.Versions {
			for _, raw := range v.Changes {
				var c change
				if err := json.Unmarshal(raw, &c); err != nil {
					t.Fatal(err)
				}
				if !c.BackwardCompatible {
					notBC++
				}
			}
		}
	}
	if notBC != 23 {
		t.Fatalf("the atlasFixture has %d changes not backward compatible, want 23", notBC)
	}
	if got := logPair(t, sha, file); len(got) != 0 {
		for _, e := range got {
			t.Logf("got: %s", e.Description)
		}
		t.Fatalf("%d events, want none", len(got))
	}
}

// MongoDB deprecated GET and POST /api/atlas/v2/groups/{groupId}/clusters at
// API version 2024-05-30 on 2024-06-25 and reactivated them on 2024-06-26 (and
// 2023-11-15 on 2024-05-09, reactivated 2024-05-16): versions still in use.
// No deprecation of either stands.
func TestReactivationWithdrawsTheDeprecation(t *testing.T) {
	// The saved file does hold those deprecations and reactivations.
	var days []day
	if err := json.Unmarshal(kittest.Read(t, atlasFixture), &days); err != nil {
		t.Fatal(err)
	}
	codes := map[string]int{}
	for _, d := range days {
		for _, p := range d.Paths {
			for _, v := range p.Versions {
				for _, raw := range v.Changes {
					var c change
					if err := json.Unmarshal(raw, &c); err != nil {
						t.Fatal(err)
					}
					if p.Path == "/api/atlas/v2/groups/{groupId}/clusters" && v.Version == "2024-05-30" {
						codes[c.ChangeCode]++
					}
				}
			}
		}
	}
	if codes["endpoint-deprecated"] == 0 || codes["endpoint-reactivated"] == 0 {
		t.Fatalf("the atlasFixture's clusters 2024-05-30 entries are %v: they no longer show a reactivation", codes)
	}
	fs, err := MongoDBAtlas.Read(kittest.Read(t, atlasFixture))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if strings.HasSuffix(f.Operation, " /api/atlas/v2/groups/{groupId}/clusters") && (f.Version == "2024-05-30" || f.Version == "2023-11-15") {
			t.Errorf("%s %s: %s on %s, but MongoDB reactivated it", f.Operation, f.Version, f.ChangeType, f.Date)
		}
	}
}

// Deprecated, reactivated, deprecated again: one deprecation, the second.
// Deprecated, then removed: both. And "api path removed without deprecation"
// isn't read (its one real entry was an endpoint that stayed).
func TestLifeCycleInOrder(t *testing.T) {
	ch := func(code string) string {
		return `{"change":"x","changeCode":"` + code + `","backwardCompatible":true}`
	}
	day := func(date, path, code string) string {
		return `{"date":"` + date + `","paths":[{"path":"` + path + `","httpMethod":"GET","versions":[{"version":"2023-01-01","changes":[` + ch(code) + `]}]}]}`
	}
	b := []byte("[" + strings.Join([]string{ // newest day first, as the file is
		day("2026-05-01", "/b", "endpoint-removed"),
		day("2026-04-01", "/c", "api-path-removed-without-deprecation"),
		day("2026-03-01", "/a", "endpoint-deprecated"),
		day("2026-02-01", "/a", "endpoint-reactivated"),
		day("2026-01-15", "/b", "resource-version-deprecated"),
		day("2026-01-01", "/a", "endpoint-deprecated"),
	}, ",") + "]")
	fs, err := MongoDBAtlas.Read(b)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range fs {
		got = append(got, f.Date+" "+f.Operation+" "+string(f.ChangeType))
	}
	want := []string{"2026-01-15 GET /b deprecation", "2026-03-01 GET /a deprecation", "2026-05-01 GET /b breaking"}
	if strings.Join(got, "; ") != strings.Join(want, "; ") {
		t.Errorf("got %q, want %q", got, want)
	}
}

// A removal followed by the same endpoint and version on a later day (an
// `endpoint-added`, any change) didn't happen: no event. And the excerpt
// keeps every field the vendor wrote, in its order.
func TestRemovalUndoneAndRawExcerpt(t *testing.T) {
	b := []byte(`[
	 {"date":"2026-03-01","paths":[{"path":"/a","httpMethod":"GET","versions":[{"version":"2023-01-01","changes":[{"change":"endpoint added","changeCode":"endpoint-added","backwardCompatible":true}]}]}]},
	 {"date":"2026-02-01","paths":[{"path":"/a","httpMethod":"GET","versions":[{"version":"2023-01-01","changes":[{"change":"removed","changeCode":"endpoint-removed","backwardCompatible":false}]}]},
	                            {"path":"/b","httpMethod":"GET","versions":[{"version":"2023-01-01","changes":[{"change":"removed","changeCode":"endpoint-removed","backwardCompatible":false,"sunset":"2026-02-01"}]}]}]}
	]`)
	fs, err := MongoDBAtlas.Read(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 || fs[0].Operation != "GET /b" {
		t.Fatalf("got %+v, want only GET /b's removal", fs)
	}
	if want := `{"change":"removed","changeCode":"endpoint-removed","backwardCompatible":false,"sunset":"2026-02-01"}`; fs[0].Excerpt != want {
		t.Errorf("excerpt %s, want %s", fs[0].Excerpt, want)
	}
}

// A removal announced twice keeps its first announcement. A version that's
// deprecated, removed and then back has neither event.
func TestRepeatedRemovalAndReturn(t *testing.T) {
	day := func(date, path, code string) string {
		return `{"date":"` + date + `","paths":[{"path":"` + path + `","httpMethod":"GET","versions":[{"version":"2023-01-01","changes":[{"change":"x","changeCode":"` + code + `","backwardCompatible":true}]}]}]}`
	}
	b := []byte("[" + strings.Join([]string{
		day("2026-04-01", "/a", "endpoint-added"),
		day("2026-03-01", "/b", "endpoint-removed"),
		day("2026-02-01", "/b", "endpoint-removed"),
		day("2026-02-01", "/a", "endpoint-removed"),
		day("2026-01-01", "/a", "endpoint-deprecated"),
	}, ",") + "]")
	fs, err := MongoDBAtlas.Read(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 || fs[0].Operation != "GET /b" || fs[0].Date != "2026-02-01" || fs[0].Pointer != "/2/paths/0/versions/0/changes/0" {
		t.Errorf("got %+v, want GET /b's removal once, as announced on 2026-02-01", fs)
	}
}

// A change announced again on a later day is one event, as first announced.
func TestRepeatsAreReadOnce(t *testing.T) {
	b := []byte(`[
	 {"date":"2026-02-01","paths":[{"path":"/a","httpMethod":"GET","versions":[{"version":"2023-01-01","changes":[{"change":"endpoint deprecated","changeCode":"endpoint-deprecated","backwardCompatible":true}]}]}]},
	 {"date":"2026-01-01","paths":[{"path":"/a","httpMethod":"GET","versions":[{"version":"2023-01-01","changes":[{"change":"endpoint deprecated","changeCode":"endpoint-deprecated","backwardCompatible":true},{"change":"added the new optional request property","changeCode":"new-optional-request-property","backwardCompatible":true}]}]}]}
	]`)
	fs, err := MongoDBAtlas.Read(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 || fs[0].Date != "2026-01-01" || fs[0].Pointer != "/1/paths/0/versions/0/changes/0" {
		t.Errorf("got %+v, want the 2026-01-01 announcement once", fs)
	}
}

func TestReadRefusesWhatIsntAChangelog(t *testing.T) {
	for name, b := range map[string]string{"html": "<html>moved</html>", "empty": "[]", "object": `{"date":"x"}`} {
		if _, err := MongoDBAtlas.Read([]byte(b)); err == nil {
			t.Errorf("%s: want an error, never zero events read as a quiet week", name)
		}
	}
}

func sameJSON(a, b string) bool {
	var x, y any
	if json.Unmarshal([]byte(a), &x) != nil || json.Unmarshal([]byte(b), &y) != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}

// nodeAt resolves an RFC 6901 pointer in a JSON file and serializes the node.
func nodeAt(t *testing.T, file, pointer string) string {
	t.Helper()
	var node any
	if err := json.Unmarshal(kittest.Read(t, file), &node); err != nil {
		t.Fatal(err)
	}
	for _, tok := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		switch n := node.(type) {
		case map[string]any:
			node = n[tok]
		case []any:
			i, err := strconv.Atoi(tok)
			if err != nil || i >= len(n) {
				return "<no node>"
			}
			node = n[i]
		default:
			return "<no node>"
		}
	}
	b, _ := json.Marshal(node)
	return string(b)
}

// Three commits of a JSON changelog are three versions, oldest first, and
// two consecutive pairs; each pair gives only what the newer commit added.
func TestJSONChangelogPairByPair(t *testing.T) {
	sha := func(n int) string { return fmt.Sprintf("%040d", n) }
	pages := kittest.Pages{
		"https://api.github.com/repos/mongodb/openapi/commits?path=changelog%2Fchangelog.json&per_page=100&page=1": "testdata/log-history/commits.json",
	}
	for n := 1; n <= 3; n++ {
		pages[MongoDBAtlas.URL(sha(n))] = fmt.Sprintf("testdata/log-history/changelog-%d.json", n)
	}
	env := kittest.Env(t, pages)
	ctx := context.Background()
	p := Plugin{Log: MongoDBAtlas}
	vs, err := p.ListVersions(ctx, env)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, v := range vs {
		ids = append(ids, v.ID)
	}
	if got := strings.Join(ids, " "); got != "2026-01-01 2026-02-01 2026-03-01" {
		t.Fatalf("versions %s", got)
	}
	var got []string
	snaps := make([]plugin.Snapshot, len(vs))
	for i, v := range vs {
		if snaps[i], err = p.ReadAt(ctx, env, v); err != nil {
			t.Fatal(err)
		}
	}
	for _, pair := range [][2]int{{0, 1}, {1, 2}} {
		res, err := p.Extract(ctx, env, snaps[pair[0]], snaps[pair[1]])
		if err != nil || len(res.Rejected) > 0 {
			t.Fatalf("%v %v", err, res.Rejected)
		}
		for _, e := range res.Events {
			got = append(got, vs[pair[1]].ID+": "+e.Subject.Operation+" "+string(e.ChangeType))
		}
	}
	want := "2026-02-01: GET /b deprecation; 2026-03-01: GET /a breaking"
	if strings.Join(got, "; ") != want {
		t.Errorf("pairs gave %q, want %q", strings.Join(got, "; "), want)
	}
}
