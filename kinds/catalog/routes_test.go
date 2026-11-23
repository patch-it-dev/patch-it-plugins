package catalog

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/github"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// A route's name stops at the first space or `(`: a trailing ` deprecated`
// or ` deprecated by other_name` is never part of it, and the two spacing
// styles Dropbox's file has used (`route x(...)` and `route x (...)`) both
// read the same name.
func TestRoutesInReadsTheNameOnly(t *testing.T) {
	content := []byte(`namespace files

route copy (RelocationArg, Metadata, RelocationError) deprecated
route copy:2(RelocationArg, RelocationResult, RelocationError)
route search/continue:2 (SearchV2ContinueArg, SearchV2Result, SearchError) deprecated by search/continue:3
    route indented (Arg, Result, Error)
`)
	got := routesIn(content)
	want := map[string]bool{"copy": true, "copy:2": true, "search/continue:2": true}
	if len(got) != len(want) {
		t.Fatalf("routesIn = %v, want keys %v", got, want)
	}
	for name := range want {
		if _, ok := got[name]; !ok {
			t.Errorf("missing %q", name)
		}
	}
	if _, ok := got["indented"]; ok {
		t.Error("an indented line isn't a top-level route declaration")
	}
}

// A route gone from the file but declared at the newer commit in another
// file of the same namespace moved: no event. The same name in a file of
// another namespace is a different route, so that one is a removal. A
// listing that can't be read fails the pair rather than reporting.
func TestARouteMovedToAnotherFileOfItsNamespaceIsNoRemoval(t *testing.T) {
	r := Routes{API: "x/files-api", Vendor: "X", Repo: "o/r", Path: "files.stone",
		Operation: func(n string) string { return "POST /2/files/" + n }}
	from := plugin.Snapshot{Version: plugin.Version{ID: "2026-01-01", Ref: "aaa"}, SourceURL: r.FileURL("aaa"),
		Content: []byte("namespace files\nroute keep(A, B, C)\nroute tags/add(A, B, C)\n")}
	to := plugin.Snapshot{Version: plugin.Version{ID: "2026-02-01", Ref: "bbb"}, SourceURL: r.FileURL("bbb"),
		Content: []byte("namespace files\nroute keep(A, B, C)\n")}
	listing := `[{"path":"files.stone","type":"file"},{"path":"tagging.stone","type":"file"},{"path":"README.md","type":"file"}]`
	for _, c := range []struct {
		name, tagging string
		events        int
		fails         bool
	}{
		{"same namespace", "namespace files\nroute tags/add (A, B, C)\n", 0, false},
		{"other namespace", "namespace sharing\nroute tags/add (A, B, C)\n", 1, false},
		{"unreadable listing", "", 0, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			pages := map[string]string{github.Raw + "/o/r/bbb/tagging.stone": c.tagging}
			if !c.fails {
				pages[r.ListingURL("bbb")] = listing
			}
			env := plugin.Env{Now: func() time.Time { return time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC) },
				Get: func(_ context.Context, u string) ([]byte, error) {
					if b, ok := pages[u]; ok {
						return []byte(b), nil
					}
					return nil, fmt.Errorf("no page %s", u)
				}}
			res, err := r.Extract(context.Background(), env, from, to)
			if c.fails {
				if err == nil {
					t.Fatalf("got %d events and no error, want an error", len(res.Events))
				}
				return
			}
			if err != nil || len(res.Events) != c.events {
				t.Fatalf("got %d events, err %v; want %d", len(res.Events), err, c.events)
			}
		})
	}
}
