package apisurface

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/pysurface"
)

func sym(kind string, params ...pysurface.Param) pysurface.Symbol {
	if params == nil {
		params = []pysurface.Param{}
	}
	return pysurface.Symbol{Kind: kind, File: "acme/x.py", Line: 1, Signature: "def f()", Params: params}
}

func prm(name, kind, def string) pysurface.Param {
	return pysurface.Param{Name: name, Kind: kind, Default: def, HasDef: def != ""}
}

// Each kind of change, and what isn't one.
func TestCompare(t *testing.T) {
	a := pysurface.Surface{
		"acme.Gone":             sym("class"),
		"acme.Gone.method":      sym("method"),
		"acme.old_name":         sym("function", prm("x", "either", "")),
		"acme.f":                sym("function", prm("a", "either", ""), prm("b", "keyword", "1"), prm("c", "keyword", "2")),
		"acme.g":                sym("function", prm("a", "either", "")),
		"acme.h":                sym("function", prm("a", "either", "")),
		"acme.overloaded":       {Kind: "function", Overloaded: true},
		"acme.Kept":             sym("class", prm("self", "either", ""), prm("timeout", "either", "600")),
		"acme.Kept.with_kwargs": sym("method", prm("self", "either", ""), prm("opt", "keyword", "None")),
	}
	b := pysurface.Surface{
		"acme.new_name":         sym("function", prm("x", "either", "")),
		"acme.f":                sym("function", prm("a", "either", ""), prm("b", "keyword", "")),
		"acme.g":                sym("function", prm("a", "either", ""), prm("z", "keyword", "")),
		"acme.h":                sym("function", prm("a", "positional", "")),
		"acme.overloaded":       {Kind: "function", Overloaded: true},
		"acme.Kept":             sym("class", prm("self", "either", ""), prm("timeout", "either", "600")),
		"acme.Kept.with_kwargs": sym("method", prm("self", "either", ""), prm("kw", "kwargs", "")),
	}
	var got []string
	for _, c := range Compare(Release{Surface: a}, Release{Surface: b}) {
		if c.Change == "" {
			got = append(got, "note "+c.Symbol)
			continue
		}
		got = append(got, string(c.Change)+" "+c.Symbol+" "+c.Parameter+c.Replacement)
	}
	want := []string{
		"removed_symbol acme.Gone ",
		"note acme.Kept.with_kwargs",
		"changed_default acme.f b",
		"removed_parameter acme.f c",
		"note acme.g",
		"renamed_symbol acme.old_name acme.new_name",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// changes is Compare's output as lines, as TestCompare writes them.
func changes(a, b Release) []string {
	var got []string
	for _, c := range Compare(a, b) {
		if c.Change == "" {
			got = append(got, "note "+c.Symbol)
			continue
		}
		got = append(got, string(c.Change)+" "+c.Symbol+" "+c.Parameter+c.Replacement)
	}
	return got
}

func cls(sig string, params ...pysurface.Param) pysurface.Symbol {
	s := sym("class", params...)
	s.Signature = sig
	return s
}

// A method that keeps only self in common with the one added beside it
// isn't a rename: close(self) gone and aclose(self) added is a removal.
func TestSelfAloneIsNoRename(t *testing.T) {
	self := prm("self", "either", "")
	a := Release{Surface: pysurface.Surface{
		"acme.C":       cls("class C:"),
		"acme.C.close": sym("method", self),
		"acme.C.send":  sym("method", self, prm("body", "either", "")),
	}}
	b := Release{Surface: pysurface.Surface{
		"acme.C":         cls("class C:"),
		"acme.C.aclose":  sym("method", self),
		"acme.C.send_v2": sym("method", self, prm("body", "either", "")),
	}}
	// Two removed, two added: neither pairs one to one, so both are
	// removals; then one of each, with a real parameter in common, is a
	// rename.
	want := "removed_symbol acme.C.close \nremoved_symbol acme.C.send "
	if got := strings.Join(changes(a, b), "\n"); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	delete(a.Surface, "acme.C.close")
	delete(b.Surface, "acme.C.aclose")
	if got := strings.Join(changes(a, b), "\n"); got != "renamed_symbol acme.C.send acme.C.send_v2" {
		t.Errorf("with a real parameter: %s", got)
	}
	a.Surface = pysurface.Surface{"acme.C": cls("class C:"), "acme.C.close": sym("method", self)}
	b.Surface = pysurface.Surface{"acme.C": cls("class C:"), "acme.C.aclose": sym("method", self)}
	if got := strings.Join(changes(a, b), "\n"); got != "removed_symbol acme.C.close " {
		t.Errorf("self only: %s", got)
	}
}

// A file the newer release couldn't read (over the size limit) is a gap,
// not a removal of everything in it.
func TestASkippedFileRemovesNothing(t *testing.T) {
	big := sym("class")
	big.File = "acme/big.py"
	gone := sym("function")
	a := Release{Surface: pysurface.Surface{"acme.Big": big, "acme.big.Big": big, "acme.gone": gone}}
	b := Release{Surface: pysurface.Surface{}, Skipped: []string{"acme/big.py"}}
	want := "note acme.Big\nnote acme.big.Big\nremoved_symbol acme.gone "
	if got := strings.Join(changes(a, b), "\n"); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

// A method that left its class for a base class, or became an attribute,
// or whose class's bases changed, is still there: a note. One its class
// and bases don't define is gone. The two releases are read by the real
// reader, the leaf class included.
func TestAMethodMovedUpIsNotRemoved(t *testing.T) {
	read := func(files map[string]string) Release {
		var fs []pysurface.File
		for p, text := range files {
			fs = append(fs, pysurface.File{Path: p, Text: text})
		}
		return Release{Surface: pysurface.Read(fs), Classes: pysurface.ReadClasses(fs)}
	}
	a := read(map[string]string{
		"acme/_base.py":    "class _Base:\n    pass\n\nclass SyncAPIResource(_Base):\n    pass\n",
		"acme/__init__.py": "from ._base import SyncAPIResource\n\nclass Completions(SyncAPIResource):\n    def create(self, a):\n        pass\n    def list(self):\n        pass\n    def models(self):\n        pass\n    def gone(self):\n        pass\n\nclass Files(SyncAPIResource):\n    def upload(self, f):\n        pass\n",
	})
	b := read(map[string]string{
		"acme/_base.py":    "class _Base:\n    def list(self):\n        pass\n\nclass SyncAPIResource(_Base):\n    def create(self, a):\n        pass\n",
		"acme/__init__.py": "from ._base import SyncAPIResource\n\nclass Completions(SyncAPIResource):\n    models = _models\n\nclass Files(SyncAPIResource, _Uploads):\n    pass\n",
	})
	want := "note acme.Completions.create\nremoved_symbol acme.Completions.gone \nnote acme.Completions.list\nnote acme.Completions.models\nnote acme.Files.upload"
	if got := strings.Join(changes(a, b), "\n"); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

// An archive is read, never unpacked or run: paths outside the package, a
// path climbing out, and an oversized file are left out.
func TestSourceFilesStayInThePackage(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"acme/__init__.py":          "def ok(): pass\n",
		"acme/big.py":               strings.Repeat("#", 64) + "\n",
		"../acme/evil.py":           "def evil(): pass\n",
		"tests/test_acme.py":        "def test(): pass\n",
		"acme-1.0.dist-info/RECORD": "",
	} {
		w, _ := zw.Create(name)
		w.Write([]byte(body))
	}
	zw.Close()
	old := MaxFileBytes
	MaxFileBytes = 32
	t.Cleanup(func() { MaxFileBytes = old })
	files, skipped, err := sourceFiles(artifact{Filename: "acme-1.0-py3-none-any.whl", Kind: "bdist_wheel"}, buf.Bytes(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Path != "acme/__init__.py" || len(skipped) != 1 || skipped[0] != "acme/big.py" {
		t.Errorf("files %v, skipped %v", files, skipped)
	}
}

var _ plugin.Versioned = Plugin{}

// A class or function gone from a module's surface while the module still
// binds the name is still importable: the class made an alias of another
// (anthropic 2.5.0: DeadlineExceededError = GatewayTimeoutError), names
// re-exported from a private module (google-genai 3.4.0's interactions),
// or a module that star-imports from where names can't be listed. A note,
// not an event. A name the module no longer binds at all is gone.
func TestANameTheModuleStillBindsIsNotRemoved(t *testing.T) {
	read := func(files map[string]string) Release {
		var fs []pysurface.File
		for p, text := range files {
			fs = append(fs, pysurface.File{Path: p, Text: text})
		}
		return Release{Surface: pysurface.Read(fs), Classes: pysurface.ReadClasses(fs), Bindings: pysurface.ReadBindings(fs)}
	}
	a := read(map[string]string{
		"acme/__init__.py":     "from ._exceptions import DeadlineExceededError, Gone\n",
		"acme/_exceptions.py":  "class DeadlineExceededError(Exception): pass\nclass Gone(Exception): pass\n",
		"acme/interactions.py": "class Webhook:\n    def __init__(self, url): pass\n",
		"acme/starred.py":      "class Agent:\n    def __init__(self, name): pass\n",
	})
	b := read(map[string]string{
		"acme/__init__.py":     "from ._exceptions import DeadlineExceededError\n",
		"acme/_exceptions.py":  "class GatewayTimeoutError(Exception): pass\nDeadlineExceededError = GatewayTimeoutError\n",
		"acme/interactions.py": "from ._gen.webhooks import Webhook\n",
		"acme/starred.py":      "from ._gen.agents import *\n",
	})
	want := "note acme.DeadlineExceededError\nremoved_symbol acme.Gone \nnote acme.interactions.Webhook\nnote acme.starred.Agent"
	if got := strings.Join(changes(a, b), "\n"); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

// A method made a staticmethod loses self, which no caller passes:
// supabase 1.0.0's SupabaseRealtimeClient.get_payload_records. Not a removed
// parameter. (cls likewise, for a classmethod made static.)
func TestSelfGoneIsNoRemovedParameter(t *testing.T) {
	self := prm("self", "either", "")
	a := Release{Surface: pysurface.Surface{"acme.C": cls("class C:"), "acme.C.records": sym("method", self, prm("payload", "either", ""))}}
	b := Release{Surface: pysurface.Surface{"acme.C": cls("class C:"), "acme.C.records": sym("method", prm("payload", "either", ""))}}
	if got := changes(a, b); len(got) != 0 {
		t.Errorf("got %v, want nothing", got)
	}
}

// TypeScript passes arguments by position: a parameter renamed in place is
// no change; one gone off the end (with no rest parameter now) is removed;
// one optional before and required now, under the same name, lost its
// "default"; a new required one at the end is a note.
func TestPositionalParameters(t *testing.T) {
	pos := func(name string, optional bool) pysurface.Param {
		return pysurface.Param{Name: name, Kind: "positional", HasDef: optional}
	}
	rest := pysurface.Param{Name: "args", Kind: "args", HasDef: true}
	a := Release{Positional: true, Surface: pysurface.Surface{
		"acme.renamed":  sym("function", pos("a", false), pos("b", true)),
		"acme.shorter":  sym("function", pos("a", false), pos("b", true), pos("c", true)),
		"acme.rest":     sym("function", pos("a", false), pos("b", true)),
		"acme.req":      sym("function", pos("a", false), pos("opts", true)),
		"acme.grew":     sym("function", pos("a", false)),
		"acme.inserted": sym("function", pos("a", false), pos("b", true)),
	}}
	b := Release{Positional: true, Surface: pysurface.Surface{
		"acme.renamed":  sym("function", pos("x", false), pos("y", true)),
		"acme.shorter":  sym("function", pos("a", false)),
		"acme.rest":     sym("function", pos("a", false), rest),
		"acme.req":      sym("function", pos("a", false), pos("opts", false)),
		"acme.grew":     sym("function", pos("a", false), pos("b", false)),
		"acme.inserted": sym("function", pos("a", false), pos("x", false), pos("b", true)),
	}}
	// A required parameter inserted before an optional one (inserted) is a
	// note: the optional one wasn't made required, it moved.
	want := "note acme.grew\nnote acme.inserted\nchanged_default acme.req opts\nremoved_parameter acme.shorter b\nremoved_parameter acme.shorter c"
	if got := strings.Join(changes(a, b), "\n"); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

// One declaration reachable at several import paths is one change, its
// paths in Also, the shortest first: openai's AgentSessionBudgetReachedEvent
// is exported by openai/resources/beta, .../beta/agents and
// .../beta/agents/agents.
func TestOneDeclarationIsOneChange(t *testing.T) {
	d := sym("class")
	d.File, d.Line = "acme/agents.d.ts", 7
	other := sym("class")
	other.File, other.Line = "acme/other.d.ts", 3
	a := Release{Surface: pysurface.Surface{"acme.beta.Event": d, "acme.beta.agents.Event": d, "acme/x.Event": d, "acme.Other": other}}
	got := Compare(a, Release{Surface: pysurface.Surface{}})
	if len(got) != 2 || got[0].Symbol != "acme.Other" || got[1].Symbol != "acme/x.Event" || strings.Join(got[1].Also, ",") != "acme.beta.Event,acme.beta.agents.Event" {
		for _, c := range got {
			t.Logf("%s %v", c.Symbol, c.Also)
		}
		t.Errorf("got %d changes", len(got))
	}
}

// npm: a member of a name now re-exported from another package, or now a
// value or a type (whose members aren't read) may still be there
// (@sentry/node 11.13.0's logger, a namespace made a const object); a method gone from a class that extends a class of
// another package may be that class's (@neondatabase/serverless 1.0.0's
// Pool.connect, pg's). Notes.
func TestNPMNotesForWhatAnotherPackageMayHold(t *testing.T) {
	a := Release{Positional: true, Surface: pysurface.Surface{
		"acme.logger": sym("namespace"), "acme.logger.info": sym("function"),
		"acme.log2": sym("namespace"), "acme.log2.info": sym("function"),
		"acme.Pool": cls("class Pool extends Pool_2"), "acme.Pool.connect": sym("method"),
		"acme.Own": cls("class Own extends Base"), "acme.Own.gone": sym("method"),
	}, Classes: pysurface.Classes{"Pool": {Bases: []string{"Pool_2"}}, "Own": {Bases: []string{"Base"}}, "Base": {}}}
	b := Release{Positional: true, Surface: pysurface.Surface{
		"acme.logger": {Kind: "export", File: "x.d.ts", Line: 1, Signature: "export { logger } from \"@acme/core\""},
		"acme.log2":   sym("variable"),
		"acme.Pool":   cls("class Pool extends Pool_2"),
		"acme.Own":    cls("class Own extends Base"),
	}, Classes: pysurface.Classes{"Pool": {Bases: []string{"Pool_2"}}, "Own": {Bases: []string{"Base"}}, "Base": {}}}
	want := "removed_symbol acme.Own.gone \nnote acme.Pool.connect\nnote acme.log2.info\nnote acme.logger.info"
	if got := strings.Join(changes(a, b), "\n"); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}
