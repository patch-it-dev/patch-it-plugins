// Package apisurfacetest is what the tests of providers' apisurface sources
// share: a package's saved releases (testdata/pypi/<name>/ or
// testdata/npm/<package>/, testdata/README.md) served where the registry
// has them, and a source's golden pair, drop pair and registration in one
// call.
package apisurfacetest

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface"
)

// Case is one source's saved releases: a golden pair, the breaking events it
// must store, and a drop pair, a real release that must store none. Both
// pairs are consecutive releases on the registry. What a pair adds (added
// events) is in the golden file, not in Want, and a drop pair may add.
type Case struct {
	Package apisurface.Package
	Golden  [2]string
	Want    []string // each breaking event as "<change> <symbol> <parameter>"
	Drop    [2]string
}

// PyPI checks a PyPI source: its golden pair stores exactly the breaking
// events listed (and what it adds, in the golden file), each citing the newer wheel and quoting both releases, against
// testdata/golden/<provider>-<source>.json; its drop pair stores nothing
// that breaks or deprecates; and
// it is registered.
func PyPI(t *testing.T, c Case) {
	t.Helper()
	kittest.Registered(t, apisurface.Plugin{Package: c.Package})
	t.Run("golden", func(t *testing.T) {
		got := pypiPair(t, c, c.Golden)
		var lines []string
		for _, e := range got {
			if e.Change != event.ChangeAdded {
				lines = append(lines, string(e.Subject.Change)+" "+strings.Join(e.Subject.Symbols, ",")+" "+e.Subject.Parameter)
			}
			if e.Version != c.Golden[1] || e.PreviousVersion != c.Golden[0] || !strings.Contains(e.SourceURL, "files.pythonhosted.org") || !strings.Contains(e.SourceExcerpt, c.Package.Name+" "+c.Golden[0]+" (") {
				t.Errorf("%s: %s<-%s cites %s, excerpt %q", e.Subject.Symbols, e.Version, e.PreviousVersion, e.SourceURL, e.SourceExcerpt)
			}
		}
		slices.Sort(lines)
		if !slices.Equal(lines, c.Want) {
			t.Errorf("got\n%s\nwant\n%s", strings.Join(lines, "\n"), strings.Join(c.Want, "\n"))
		}
		kittest.Golden(t, goldenFile(c), got)
	})
	t.Run("drop", func(t *testing.T) {
		for _, e := range pypiPair(t, c, c.Drop) {
			if e.Change != event.ChangeAdded {
				t.Errorf("%s %v %s", e.Subject.Change, e.Subject.Symbols, e.Subject.Parameter)
			}
		}
	})
}

// NPM checks an npm source as PyPI does a PyPI one, the events citing the
// registry.
func NPM(t *testing.T, c Case) {
	t.Helper()
	kittest.Registered(t, apisurface.Plugin{Package: c.Package})
	t.Run("golden", func(t *testing.T) {
		got := npmPair(t, c, c.Golden)
		if len(got) == 0 {
			t.Fatal("no events")
		}
		var lines []string
		for _, e := range got {
			if e.Change != event.ChangeAdded {
				lines = append(lines, string(e.Subject.Change)+" "+strings.Join(e.Subject.Symbols, ",")+" "+e.Subject.Parameter)
			}
			if e.Version != c.Golden[1] || e.PreviousVersion != c.Golden[0] || !strings.HasPrefix(e.SourceURL, "https://registry.npmjs.org/") || !strings.Contains(e.SourceExcerpt, c.Package.Name+" "+c.Golden[0]+" (") || e.Subject.Ecosystem != "npm" {
				t.Errorf("%s: %s<-%s cites %s, excerpt %q", e.Subject.Symbols, e.Version, e.PreviousVersion, e.SourceURL, e.SourceExcerpt)
			}
		}
		slices.Sort(lines)
		if !slices.Equal(lines, c.Want) {
			t.Errorf("got\n%s\nwant\n%s", strings.Join(lines, "\n"), strings.Join(c.Want, "\n"))
		}
		kittest.Golden(t, goldenFile(c), got)
	})
	t.Run("drop", func(t *testing.T) {
		for _, e := range npmPair(t, c, c.Drop) {
			if e.Change != event.ChangeAdded {
				t.Errorf("%s %v %s", e.Subject.Change, e.Subject.Symbols, e.Subject.Parameter)
			}
		}
	})
}

// goldenFile is testdata/golden/<provider>-<source>.json.
func goldenFile(c Case) string {
	provider, source, _ := strings.Cut(c.Package.ID, "/")
	return "testdata/golden/" + provider + "-" + source + ".json"
}

// pypiPages serves a case's saved files: PyPI's answers, and each picked
// wheel, trimmed, under its real URL.
func pypiPages(t *testing.T, name string) kittest.Pages {
	t.Helper()
	dir := filepath.Join("testdata", "pypi", name)
	pages := kittest.Pages{"https://pypi.org/pypi/" + name + "/json": filepath.Join(dir, "pypi-"+name+"-excerpt.json")}
	files, err := filepath.Glob(filepath.Join(dir, "pypi-"+name+"-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		v := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(f), "pypi-"+name+"-"), ".json")
		if v == "excerpt" {
			continue
		}
		pages["https://pypi.org/pypi/"+name+"/"+v+"/json"] = f
		var doc struct {
			URLs []struct {
				URL      string `json:"url"`
				Filename string `json:"filename"`
			} `json:"urls"`
		}
		if err := json.Unmarshal(kittest.Read(t, f), &doc); err != nil {
			t.Fatal(err)
		}
		for _, u := range doc.URLs {
			trimmed := filepath.Join(dir, strings.TrimSuffix(u.Filename, ".whl")+".trimmed.whl")
			if _, err := os.Stat(trimmed); err == nil {
				pages[u.URL] = trimmed
			}
		}
	}
	return pages
}

// pypiPair stores one pair of a case's releases, listed as PyPI lists them.
func pypiPair(t *testing.T, c Case, pair [2]string) []event.Event {
	t.Helper()
	ctx, env := context.Background(), kittest.Env(t, pypiPages(t, c.Package.Name))
	p := apisurface.Plugin{Package: c.Package}
	vs, err := p.ListVersions(ctx, env)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]plugin.Version{}
	for _, v := range vs {
		byID[v.ID] = v
	}
	from, ok1 := byID[pair[0]]
	to, ok2 := byID[pair[1]]
	if !ok1 || !ok2 {
		t.Fatalf("%s: %v not both listed in %v", c.Package.Name, pair, vs)
	}
	return kittest.Versions(t, env, p, from, to)
}

// npmDir is a package's directory under testdata/npm.
func npmDir(name string) string {
	return filepath.Join("testdata", "npm", strings.ReplaceAll(strings.TrimPrefix(name, "@"), "/", "-"))
}

// npmPages serves a case's saved files under the registry's URLs: the
// package document, each version's, and each trimmed tarball.
func npmPages(t *testing.T, name string) kittest.Pages {
	t.Helper()
	dir, safe := npmDir(name), strings.ReplaceAll(strings.TrimPrefix(name, "@"), "/", "-")
	reg := "https://registry.npmjs.org/" + strings.ReplaceAll(name, "/", "%2F")
	pages := kittest.Pages{reg: filepath.Join(dir, "npm-"+safe+"-excerpt.json")}
	files, err := filepath.Glob(filepath.Join(dir, "npm-"+safe+"-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		v := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(f), "npm-"+safe+"-"), ".json")
		if v == "excerpt" {
			continue
		}
		pages[reg+"/"+v] = f
		var doc struct {
			Dist struct {
				Tarball string `json:"tarball"`
			} `json:"dist"`
		}
		if err := json.Unmarshal(kittest.Read(t, f), &doc); err != nil {
			t.Fatal(err)
		}
		pages[doc.Dist.Tarball] = filepath.Join(dir, strings.TrimSuffix(filepath.Base(doc.Dist.Tarball), ".tgz")+".trimmed.tgz")
	}
	return pages
}

func npmPair(t *testing.T, c Case, pair [2]string) []event.Event {
	t.Helper()
	ctx, env := context.Background(), kittest.Env(t, npmPages(t, c.Package.Name))
	p := apisurface.Plugin{Package: c.Package}
	vs, err := p.ListVersions(ctx, env)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]plugin.Version{}
	for _, v := range vs {
		byID[v.ID] = v
	}
	from, ok1 := byID[pair[0]]
	to, ok2 := byID[pair[1]]
	if !ok1 || !ok2 {
		t.Fatalf("%s: %v not both listed in %v", c.Package.Name, pair, vs)
	}
	return kittest.Versions(t, env, p, from, to)
}
