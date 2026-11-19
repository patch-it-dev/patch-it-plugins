package diff

import (
	"strings"
	"testing"
)

func pathsSpec(paths string) string {
	return `{"openapi":"3.0.0","info":{"version":"v"},"paths":{` + paths + `}}`
}

func kinds(changes []Change) []string {
	var got []string
	for _, c := range changes {
		got = append(got, string(c.Kind)+" "+c.Operation+" "+c.Property)
	}
	return got
}

var endsDeprecated = func(s string) bool { return strings.HasSuffix(s, "(deprecated)") }

// A deprecation said only in the summary is one when DeprecatedSummary says
// so, and points at the summary; without it, only the flag counts.
func TestDeprecatedSummary(t *testing.T) {
	oldSpec := mustParse(t, pathsSpec(`"/v4/price":{"get":{"summary":"Check the price"}}`))
	newSpec := mustParse(t, pathsSpec(`"/v4/price":{"get":{"summary":"Check the price (deprecated)"}}`))

	if got := Changes(oldSpec, newSpec); len(got) != 0 {
		t.Errorf("without DeprecatedSummary: %v, want nothing", kinds(got))
	}
	got := ChangesWith(oldSpec, newSpec, Options{DeprecatedSummary: endsDeprecated})
	if len(got) != 1 || got[0].Kind != OperationDeprecated || got[0].Pointer != "/paths/~1v4~1price/get/summary" ||
		got[0].Node != "Check the price (deprecated)" {
		t.Fatalf("got %+v, want one operation_deprecated citing the summary", got)
	}
}

// Already deprecated in words, then flagged: not deprecated again.
func TestDeprecatedOnceWhicheverWaySaid(t *testing.T) {
	oldSpec := mustParse(t, pathsSpec(`"/a":{"get":{"summary":"A (deprecated)"}}`))
	newSpec := mustParse(t, pathsSpec(`"/a":{"get":{"summary":"A (deprecated)","deprecated":true}}`))
	if got := ChangesWith(oldSpec, newSpec, Options{DeprecatedSummary: endsDeprecated}); len(got) != 0 {
		t.Errorf("got %v, want nothing", kinds(got))
	}
}

func TestOperationDeprecated(t *testing.T) {
	o := Options{DeprecatedSummary: endsDeprecated}
	for _, c := range []struct {
		op   map[string]any
		want bool
	}{
		{map[string]any{"deprecated": true}, true},
		{map[string]any{"summary": "A (deprecated)"}, true},
		{map[string]any{"summary": "A"}, false},
		{nil, false},
	} {
		if got := o.OperationDeprecated(c.op); got != c.want {
			t.Errorf("%v: got %v, want %v", c.op, got, c.want)
		}
	}
	if (Options{}).OperationDeprecated(map[string]any{"summary": "A (deprecated)"}) {
		t.Error("zero Options reads the summary")
	}
}

// Renaming a path parameter changes no URL a caller sends: with PathTemplates
// it's no change at all, and the parameter is still compared, under its new
// name.
func TestPathTemplates(t *testing.T) {
	oldSpec := mustParse(t, pathsSpec(`"/drives/{name}":{"delete":{"parameters":[
		{"name":"name","in":"path","required":true,"schema":{"type":"string"}},
		{"name":"teamId","in":"query","schema":{"type":"string"}}]}}`))
	same := mustParse(t, pathsSpec(`"/drives/{nameOrId}":{"delete":{"parameters":[
		{"name":"nameOrId","in":"path","required":true,"schema":{"type":"string"}},
		{"name":"teamId","in":"query","schema":{"type":"string"}}]}}`))
	retyped := mustParse(t, pathsSpec(`"/drives/{nameOrId}":{"delete":{"parameters":[
		{"name":"nameOrId","in":"path","required":true,"schema":{"type":"integer"}},
		{"name":"teamId","in":"query","schema":{"type":"string"}}]}}`))

	if got := kinds(Changes(oldSpec, same)); len(got) != 2 ||
		got[0] != "operation_added DELETE /drives/{nameOrId} " || got[1] != "operation_removed DELETE /drives/{name} " {
		t.Errorf("by exact path: %v, want the removal and the addition", got)
	}
	if got := ChangesWith(oldSpec, same, Options{PathTemplates: true}); len(got) != 0 {
		t.Errorf("by template: %v, want nothing", kinds(got))
	}
	got := ChangesWith(oldSpec, retyped, Options{PathTemplates: true})
	if len(got) != 1 || got[0].Kind != TypeChanged || got[0].Operation != "DELETE /drives/{nameOrId}" || got[0].Property != "nameOrId" ||
		got[0].Pointer != "/paths/~1drives~1{nameOrId}/delete/parameters/0/schema" {
		t.Errorf("got %+v, want the path parameter's type change under its new name", got)
	}
}

// A query parameter that happens to share an old path parameter's name isn't
// renamed with it.
func TestPathTemplatesRenameOnlyPathParameters(t *testing.T) {
	oldSpec := mustParse(t, pathsSpec(`"/x/{id}":{"get":{"parameters":[
		{"name":"id","in":"path","required":true,"schema":{"type":"string"}},
		{"name":"id","in":"query","schema":{"type":"string"}}]}}`))
	newSpec := mustParse(t, pathsSpec(`"/x/{xId}":{"get":{"parameters":[
		{"name":"xId","in":"path","required":true,"schema":{"type":"string"}}]}}`))
	got := kinds(ChangesWith(oldSpec, newSpec, Options{PathTemplates: true}))
	if len(got) != 1 || got[0] != "property_removed GET /x/{xId} id" {
		t.Errorf("got %v, want the query parameter's removal only", got)
	}
}

// Two paths in one spec with the same template can't be told apart by it, so
// they're paired by exact path.
func TestPathTemplatesAmbiguousFallsBackToExactPaths(t *testing.T) {
	both := pathsSpec(`"/d/{name}":{"get":{}},"/d/{id}":{"get":{}}`)
	oldSpec, newSpec := mustParse(t, both), mustParse(t, pathsSpec(`"/d/{id}":{"get":{}}`))
	got := kinds(ChangesWith(oldSpec, newSpec, Options{PathTemplates: true}))
	if len(got) != 1 || got[0] != "operation_removed GET /d/{name} " {
		t.Errorf("got %v, want GET /d/{name} removed", got)
	}
}
