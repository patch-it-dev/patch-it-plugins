package apisurface

import (
	"archive/zip"
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
	"github.com/patch-it-dev/patch-it-plugins/plugins/apisurface/pysurface"
)

const (
	wheel111 = "https://files.pythonhosted.org/packages/f7/86/ea99e3f966a639a3256e2f79d78cd63d45d5eb651a5457ccbdcfd9eed24a/openai-1.1.1-py3-none-any.whl"
	wheel112 = "https://files.pythonhosted.org/packages/dd/82/b92f73453ea318c0d46f31aeb56e9d94a42606c010fb72a513f4a3cd4bac/openai-1.1.2-py3-none-any.whl"
)

// The saved releases (testdata/README.md): PyPI's answers and the two
// wheels, trimmed to the openai.types.beta package.
var saved = kittest.Pages{
	"https://pypi.org/pypi/openai/json":       "testdata/pypi-openai-excerpt.json",
	"https://pypi.org/pypi/openai/1.1.1/json": "testdata/pypi-openai-1.1.1.json",
	"https://pypi.org/pypi/openai/1.1.2/json": "testdata/pypi-openai-1.1.2.json",
	wheel111: "testdata/openai-1.1.1-py3-none-any.trimmed.whl",
	wheel112: "testdata/openai-1.1.2-py3-none-any.trimmed.whl",
}

// pair lists the releases, reads 1.1.1 and 1.1.2 and extracts the pair.
func pair(t *testing.T) []event.Event {
	t.Helper()
	ctx, env := context.Background(), kittest.Env(t, saved)
	p := Plugin{Package: OpenAIPython}
	vs, err := p.ListVersions(ctx, env)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, v := range vs {
		ids = append(ids, v.ID)
	}
	if got := strings.Join(ids, " "); got != "1.1.0 1.1.1 1.1.2" {
		t.Fatalf("versions %s", got)
	}
	last := versions.Pairs(vs)[1]
	from, err := p.ReadAt(ctx, env, last[0])
	if err != nil {
		t.Fatal(err)
	}
	to, err := p.ReadAt(ctx, env, last[1])
	if err != nil {
		t.Fatal(err)
	}
	if to.SourceURL != wheel112 {
		t.Errorf("source %s", to.SourceURL)
	}
	res, err := p.Extract(ctx, env, from, to)
	if err != nil || len(res.Rejected) > 0 {
		t.Fatalf("%v %v", err, res.Rejected)
	}
	return res.Events
}

// openai 1.1.2 renamed the misspelled AsssitantDeleted to AssistantDeleted,
// module and all: the old name is gone from both places code could import
// it. (Not read as a rename: the class has no parameters to tell it's the
// same class.)
func TestGolden(t *testing.T) {
	got := pair(t)
	kittest.Golden(t, "testdata/golden/openai-1.1.2.json", got)
	for _, e := range got {
		if e.PreviousVersion != "1.1.1" || e.Version != "1.1.2" || !strings.Contains(e.SourceExcerpt, "openai/types/beta/asssitant_deleted.py line ") {
			t.Errorf("%v: %s<-%s, excerpt %q", e.Subject.Symbols, e.Version, e.PreviousVersion, e.SourceExcerpt)
		}
	}
}

// The pinned drop test: 1.1.2 adds AssistantDeleted and its module. A name
// that appears is no breaking change: no event names it.
func TestAnAddedNameIsNoEvent(t *testing.T) {
	for _, e := range pair(t) {
		for _, s := range e.Subject.Symbols {
			if strings.Contains(s, "AssistantDeleted") || strings.Contains(s, "assistant_deleted") {
				t.Errorf("event for %s, which 1.1.2 added", s)
			}
		}
	}
}

func TestRegistersItself(t *testing.T) {
	kittest.Registered(t, Plugin{Package: OpenAIPython})
}

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
// and bases don't define is gone.
func TestAMethodMovedUpIsNotRemoved(t *testing.T) {
	self := prm("self", "either", "")
	a := Release{Surface: pysurface.Surface{
		"acme.Completions":        cls("class Completions(SyncAPIResource):"),
		"acme.Completions.create": sym("method", self),
		"acme.Completions.list":   sym("method", self),
		"acme.Completions.models": sym("method", self),
		"acme.Completions.gone":   sym("method", self),
		"acme.Files":              cls("class Files(SyncAPIResource):"),
		"acme.Files.upload":       sym("method", self),
	}}
	b := Release{
		Surface: pysurface.Surface{
			"acme.Completions": cls("class Completions(SyncAPIResource):"),
			"acme.Files":       cls("class Files(SyncAPIResource, _Uploads):"),
		},
		Classes: pysurface.Classes{
			"Completions":     {Bases: []string{"SyncAPIResource"}, Members: []string{"models"}},
			"SyncAPIResource": {Bases: []string{"_Base"}, Members: []string{"create"}},
			"_Base":           {Members: []string{"list"}},
		},
	}
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
