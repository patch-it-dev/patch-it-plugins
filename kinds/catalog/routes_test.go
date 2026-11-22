package catalog

import "testing"

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
