package all

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// Every package next to this one that registers a plugin is imported here, so
// a program importing all runs it.
func TestEveryPluginIsImported(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "all.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	var imported []string
	for _, imp := range f.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		imported = append(imported, filepath.Base(path))
	}
	dirs, err := os.ReadDir("..")
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, d := range dirs {
		if !d.IsDir() || !registers(t, filepath.Join("..", d.Name())) {
			continue
		}
		found++
		if !slices.Contains(imported, d.Name()) {
			t.Errorf("plugin package %s registers a plugin but all.go doesn't import it", d.Name())
		}
	}
	if found == 0 {
		t.Fatal("found no plugin package next to this one")
	}
}

// registers reports whether a package's non-test code calls
// plugin.RegisterProvider (or plugin.Register).
func registers(t *testing.T, dir string) bool {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "plugin.Register(") || strings.Contains(string(b), "plugin.RegisterProvider(") {
			return true
		}
	}
	return false
}

// Importing all registers plugins, under unique names, sorted.
func TestImportingAllRegistersThePlugins(t *testing.T) {
	names := plugin.Names()
	if len(names) == 0 {
		t.Fatal("no plugin registered")
	}
	if !slices.IsSorted(names) || len(slices.Compact(slices.Clone(names))) != len(names) {
		t.Errorf("names %v: want sorted and unique", names)
	}
}

// Every registered source has complete metadata (RegisterProvider checks the
// fields), is found by its id, and its page exists.
func TestEverySourceHasMetadata(t *testing.T) {
	for _, s := range plugin.Sources() {
		if _, err := os.Stat(filepath.Join("..", "..", s.Source.Page)); err != nil {
			t.Errorf("%s: its page %s doesn't exist (copy providers/_template.md)", s.FullID(), s.Source.Page)
		}
		if p, ok := plugin.Lookup(s.FullID()); !ok || p.Name() != s.FullID() {
			t.Errorf("Lookup(%q) didn't find it", s.FullID())
		}
	}
	if len(plugin.Names()) != len(plugin.Sources()) {
		t.Errorf("%d plugins, %d sources", len(plugin.Names()), len(plugin.Sources()))
	}
}

// Every registered source's kind is an answer of docs/which-kind.md, so the
// guide sends every source somewhere.
func TestWhichKindNamesEveryKind(t *testing.T) {
	guide, err := os.ReadFile("../../docs/which-kind.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range plugin.Sources() {
		if !strings.Contains(string(guide), "| `"+s.Source.Kind+"` |") {
			t.Errorf("%s's kind, %s, isn't an answer in docs/which-kind.md", s.FullID(), s.Source.Kind)
		}
	}
}

// Every package under plugins/ is a kind that registers its sources, or a
// reader package inside one (plugins/<kind>/<reader>/); and none only
// forwards to another package (type aliases and re-exported values, no code
// of its own).
func TestEveryPackageIsAKindOrPartOfOne(t *testing.T) {
	dirs, err := os.ReadDir("..")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range dirs {
		if d.IsDir() && d.Name() != "all" && !registers(t, filepath.Join("..", d.Name())) {
			t.Errorf("plugins/%s registers no source: a kind registers its sources, a reader lives inside its kind", d.Name())
		}
	}
	err = filepath.WalkDir("..", func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() || d.Name() == "testdata" {
			if d != nil && d.IsDir() && d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return err
		}
		if forwardsOnly(t, p) {
			t.Errorf("%s only forwards to another package: import that package instead", p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// forwardsOnly reports whether a package's non-test code is nothing but type
// aliases and values copied from other packages.
func forwardsOnly(t *testing.T, dir string) bool {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	decls := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			g, ok := decl.(*ast.GenDecl)
			if !ok {
				return false // a func: code of its own
			}
			for _, spec := range g.Specs {
				switch s := spec.(type) {
				case *ast.ImportSpec:
					continue
				case *ast.TypeSpec:
					if !s.Assign.IsValid() {
						return false
					}
				case *ast.ValueSpec:
					for _, v := range s.Values {
						if _, sel := v.(*ast.SelectorExpr); !sel {
							return false
						}
					}
				}
				decls++
			}
		}
	}
	return decls > 0
}
