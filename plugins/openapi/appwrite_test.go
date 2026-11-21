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

// The appwrite/specs commits the tests read (checked 2026-11-19), and their
// committer dates, as GitHub gives them.
const (
	appwrite19Old = "8c00ba356b01d43fbc6c88c60fb2a4fc99e04b40" // 2026-06-19
	appwrite19New = "7d74bf9f3c0c990745e822f5935bcfaca310fd06" // 2026-06-24
	appwrite24Old = "98cc1a25215d3cead3a64f9bd4574a9c5b1ec87b" // 2026-10-08
	appwrite24New = "06e6fafa4d1ae3c815843ff0b02a56e6e5582756" // 2026-10-12
	appwrite18    = "ac705cd3e9d5e3a0693b7e40189eb1e66883e341" // 2026-02-18
)

var appwriteDates = map[string]string{
	appwrite19Old: "2026-06-19T03:20:24Z",
	appwrite19New: "2026-06-24T02:33:08Z",
	appwrite24Old: "2026-10-08T15:06:26Z",
	appwrite24New: "2026-10-12T18:11:38Z",
	appwrite18:    "2026-02-18T08:20:16Z",
}

// What GitHub lists in specs/ at each commit, as it did on 2026-11-19.
var (
	appwriteDirsTo19 = "0.10.x 0.11.x 0.12.x 0.13.x 0.14.x 0.15.x 0.6.x 0.7.x 0.8.x 0.9.x 1.0.x 1.1.x 1.2.x 1.3.x 1.4.x 1.5.x 1.6.x 1.7.x 1.8.x 1.9.x latest"
	appwriteDirs     = map[string]string{
		appwrite19Old: appwriteDirsTo19,
		appwrite19New: appwriteDirsTo19,
		appwrite24Old: strings.Replace(appwriteDirsTo19, " latest", " 2.0.x 2.1.x 2.2.x 2.3.x 2.4.x latest", 1),
		appwrite24New: strings.Replace(appwriteDirsTo19, " latest", " 2.0.x 2.1.x 2.2.x 2.3.x 2.4.x latest", 1),
		appwrite18:    strings.Replace(appwriteDirsTo19, " 1.9.x", "", 1),
	}
	// And in specs/<X.Y>.x: commit/dir -> file names -> the saved copy of
	// the one the adapter reads (trimmed: see testdata/README.md).
	appwriteFiles = map[string]map[string]string{
		appwrite19Old + "/1.9.x": {"open-api3-1.9.x-client.json": "", "open-api3-1.9.x-console.json": "", "open-api3-1.9.x-server.json": "testdata/appwrite-1.9.x-8c00ba35-server.json"},
		appwrite19New + "/1.9.x": {"open-api3-1.9.x-client.json": "", "open-api3-1.9.x-console.json": "", "open-api3-1.9.x-server.json": "testdata/appwrite-1.9.x-7d74bf9f-server.json"},
		appwrite24Old + "/2.4.x": {"open-api3-2.4.x-client.json": "", "open-api3-2.4.x-console.json": "", "open-api3-2.4.x-server.json": "", "open-api3-2.4.x.json": "testdata/appwrite-2.4.x-98cc1a25.json"},
		appwrite24New + "/2.4.x": {"open-api3-2.4.x-client.json": "", "open-api3-2.4.x-console.json": "", "open-api3-2.4.x-server.json": "", "open-api3-2.4.x.json": "testdata/appwrite-2.4.x-06e6fafa.json"},
		appwrite18 + "/1.7.x":    {"open-api3-1.7.x-client.json": "", "open-api3-1.7.x-console.json": "", "open-api3-1.7.x-server.json": "testdata/appwrite-1.7.x-ac705cd3-server.json", "swagger2-1.7.x-server.json": ""},
		appwrite18 + "/1.8.x":    {"open-api3-1.8.x-client.json": "", "open-api3-1.8.x-console.json": "", "open-api3-1.8.x-server.json": "testdata/appwrite-1.8.x-ac705cd3-server.json", "swagger2-1.8.x-server.json": ""},
	}
)

// appwritePlugin is the plugin as registered, its adapter reading GitHub's
// listings and the saved copies from a local server. It returns the base of
// the raw URLs: <base>/<sha>/specs/<dir>/<file>.
func appwritePlugin(t *testing.T) (Plugin, string) {
	t.Helper()
	const repo = "/api/repos/appwrite/specs"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ref := r.URL.Path, r.URL.Query().Get("ref")
		var names []string
		switch {
		case strings.HasPrefix(p, repo+"/commits/") && appwriteDates[strings.TrimPrefix(p, repo+"/commits/")] != "":
			sha := strings.TrimPrefix(p, repo+"/commits/")
			fmt.Fprintf(w, `{"sha":%q,"commit":{"committer":{"date":%q}}}`, sha, appwriteDates[sha])
			return
		case p == repo+"/contents/specs" && appwriteDirs[ref] != "":
			names = strings.Fields(appwriteDirs[ref])
		case strings.HasPrefix(p, repo+"/contents/specs/") && appwriteFiles[ref+"/"+strings.TrimPrefix(p, repo+"/contents/specs/")] != nil:
			for name := range appwriteFiles[ref+"/"+strings.TrimPrefix(p, repo+"/contents/specs/")] {
				names = append(names, name)
			}
		default:
			if rest, ok := strings.CutPrefix(p, "/raw/appwrite/specs/"); ok {
				sha, path, _ := strings.Cut(rest, "/specs/")
				dir, file, _ := strings.Cut(path, "/")
				if saved := appwriteFiles[sha+"/"+dir][file]; saved != "" {
					http.ServeFile(w, r, saved)
					return
				}
			}
			http.NotFound(w, r)
			return
		}
		entries := make([]map[string]string, 0, len(names))
		for _, n := range names {
			entries = append(entries, map[string]string{"name": n})
		}
		json.NewEncoder(w).Encode(entries)
	}))
	t.Cleanup(srv.Close)
	a := adapter.Appwrite{APIBase: srv.URL + "/api", RawBase: srv.URL + "/raw"}
	return Plugin{Adapter: a, Diff: appwriteDiff, Keep: appwriteKeep}, srv.URL + "/raw/appwrite/specs"
}

func pairAppwrite(t *testing.T, p Plugin, from, to string) []event.Event {
	t.Helper()
	s := openStore(t)
	if rejected, err := kittest.Pair(context.Background(), vercelEnv(s), s, p, from, to); err != nil || rejected {
		t.Fatalf("pair %s -> %s: rejected=%v err=%v", from, to, rejected, err)
	}
	return kittest.Stored(t, s)
}

func TestAppwriteRegistersItself(t *testing.T) {
	kittest.Registered(t, Plugin{Adapter: adapter.Appwrite{}, Diff: appwriteDiff, Keep: appwriteKeep})
}

// The golden test: the only deprecations in the spec's history as the
// plugin reads it. On 2026-06-24 (7d74bf9f, still info.version 1.9.5) the
// 1.9.x server spec flagged 14 more operations, 83 to 97: the six
// /databases/transactions operations and the longtext, mediumtext, text and
// varchar attribute operations of the legacy Databases API, each "since"
// 1.8.0 and replaced with a TablesDB method. None is an SDK method rename or
// has a live method ("methods": []), so the filter keeps all 14. Each event
// is named by the path a caller sends (/v1/...), dated by the commit, and
// quotes x-appwrite.deprecated in the file as published.
func TestAppwriteGolden(t *testing.T) {
	p, base := appwritePlugin(t)
	events := pairAppwrite(t, p, appwrite19Old, appwrite19New)
	if len(events) != 14 {
		t.Fatalf("%d events, want 14", len(events))
	}
	file := appwriteFiles[appwrite19New+"/1.9.x"]["open-api3-1.9.x-server.json"]
	url := base + "/" + appwrite19New + "/specs/1.9.x/open-api3-1.9.x-server.json"
	for _, e := range events {
		if e.ChangeType != event.ChangeTypeDeprecation || e.Breaking || e.Version != "2026-06-24" || e.SourceURL != url ||
			!strings.Contains(e.Subject.Operation, " /v1/databases/") || strings.HasPrefix(e.SourcePointer, "/paths/~1v1") {
			t.Errorf("%s: %s %s from %s", e.Subject.Operation, e.ChangeType, e.Version, e.SourceURL)
		}
		if !strings.HasSuffix(e.SourcePointer, "/x-appwrite/deprecated") || jsonNode(t, file, e.SourcePointer) != e.SourceExcerpt {
			t.Errorf("%s: excerpt %s doesn't match %s", e.Subject.Operation, e.SourceExcerpt, e.SourcePointer)
		}
		if !strings.Contains(e.SourceExcerpt, `"replaceWith":"tablesDB.`) || !strings.Contains(e.SourceExcerpt, `"since":"1.8.0"`) {
			t.Errorf("%s: excerpt %s", e.Subject.Operation, e.SourceExcerpt)
		}
	}
	for i := range events {
		events[i].SourceURL = strings.Replace(events[i].SourceURL, base, "https://raw.githubusercontent.com/appwrite/specs", 1)
	}
	kittest.Golden(t, "testdata/golden/appwrite.json", events)
}

// The pinned drop test. On 2026-10-12 (06e6fafa) 13 operations of 2.4.x's
// single document lost the server platform (still listed for console or
// client), none marked deprecated: it "listed server and client methods no
// released SDK ships yet". 2.5.x had all 13 on the server again the next
// day. As the adapter reads it, 13 removals; no event: removals never are.
func TestAppwriteDropsRemovals(t *testing.T) {
	p, _ := appwritePlugin(t)
	if events := pairAppwrite(t, p, appwrite24Old, appwrite24New); len(events) != 0 {
		t.Fatalf("%d events, want none: %s", len(events), events[0].Subject.Operation)
	}
	// Unfiltered, the same pair reports the 13 removals.
	p.Keep = nil
	var removed int
	for _, e := range pairAppwrite(t, p, appwrite24Old, appwrite24New) {
		if e.ChangeType == event.ChangeTypeBreaking && e.Subject.Kind == event.KindOperation {
			removed++
		}
	}
	if removed != 13 {
		t.Errorf("unfiltered: %d removals, want 13", removed)
	}
	if c := (diff.Change{Kind: diff.OperationRemoved, Node: map[string]any{"deprecated": true}}); appwriteKeep(c) {
		t.Error("appwriteKeep keeps a removal of a deprecated operation")
	}
}

// From 1.7.x to 1.8.x (both at ac705cd3), 21 server operations were flagged
// deprecated only because an SDK method name was renamed
// (createMfaChallenge to createMFAChallenge, createSms to createSMS, ...):
// replaceWith is the same operation's other name, and the route stays. The
// saved copies keep only those 21 operations' paths. No event; reading every
// flag, 21.
func TestAppwriteDropsSDKMethodRenames(t *testing.T) {
	p, _ := appwritePlugin(t)
	if events := pairAppwrite(t, p, "1.7.x@"+appwrite18, appwrite18); len(events) != 0 {
		t.Fatalf("%d events, want none: %s", len(events), events[0].Subject.Operation)
	}
	p.Diff.FlagNotDeprecation = nil
	events := pairAppwrite(t, p, "1.7.x@"+appwrite18, appwrite18)
	if len(events) != 21 {
		t.Fatalf("every flag read: %d events, want 21", len(events))
	}
	for _, e := range events {
		if e.Version != "2026-02-18" || e.PreviousVersion != "2026-02-18" {
			t.Errorf("%s: %s -> %s", e.Subject.Operation, e.PreviousVersion, e.Version)
		}
	}
}

func TestAppwriteMethodLevel(t *testing.T) {
	methods := []any{
		map[string]any{"namespace": "account", "name": "createMfaChallenge", "deprecated": map[string]any{"since": "1.8.0"}},
		map[string]any{"namespace": "account", "name": "createMFAChallenge", "deprecated": map[string]any{"since": "1.8.0"}},
	}
	for name, c := range map[string]struct {
		replaceWith string
		want        bool
	}{
		"same operation's other name": {"account.createMFAChallenge", true},
		"another operation":           {"tablesDB.createTransaction", false},
		"name in another namespace":   {"users.createMFAChallenge", false},
		"no replacement":              {"", false},
	} {
		op := map[string]any{"deprecated": true, "x-appwrite": map[string]any{
			"deprecated": map[string]any{"since": "1.8.0", "replaceWith": c.replaceWith},
			"methods":    methods,
		}}
		if got := appwriteMethodLevel(op); got != c.want {
			t.Errorf("%s: got %v, want %v", name, got, c.want)
		}
	}
	for name, c := range map[string]struct {
		methods []any
		want    bool
	}{
		// A live method remains: the route stays, whatever replaceWith says.
		"one method not deprecated": {[]any{
			map[string]any{"namespace": "databases", "name": "createX", "deprecated": map[string]any{"since": "1.8.0"}},
			map[string]any{"namespace": "databases", "name": "createY"},
		}, true},
		"deprecated: false": {[]any{map[string]any{"namespace": "databases", "name": "createX", "deprecated": false}}, true},
		"every method deprecated": {[]any{
			map[string]any{"namespace": "databases", "name": "createX", "deprecated": map[string]any{"since": "1.8.0"}},
		}, false},
		// As the 14 golden operations are: read by the flag.
		"no methods":     {[]any{}, false},
		"methods absent": {nil, false},
	} {
		xa := map[string]any{"deprecated": map[string]any{"since": "1.8.0", "replaceWith": "tablesDB.createZ"}}
		if c.methods != nil {
			xa["methods"] = c.methods
		}
		if got := appwriteMethodLevel(map[string]any{"deprecated": true, "x-appwrite": xa}); got != c.want {
			t.Errorf("%s: got %v, want %v", name, got, c.want)
		}
	}
	if appwriteMethodLevel(map[string]any{"deprecated": true}) {
		t.Error("an operation without x-appwrite reads as method-level")
	}
}
