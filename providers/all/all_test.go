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

const module = "github.com/patch-it-dev/patch-it-plugins/"

// providerDirs are the folders under providers/, one per provider.
func providerDirs(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir("..")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() && e.Name() != "all" {
			out = append(out, e.Name())
		}
	}
	if len(out) == 0 {
		t.Fatal("found no provider folder next to this one")
	}
	return out
}

// Every provider folder is imported here, so a program importing all runs
// its sources, and registers exactly its own provider, in its provider.go.
func TestEveryProviderIsImported(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "all.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	var imported []string
	for _, imp := range f.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		imported = append(imported, strings.TrimPrefix(path, module+"providers/"))
	}
	registered := map[string]bool{}
	for _, p := range plugin.Providers() {
		registered[p.ID] = true
	}
	for _, dir := range providerDirs(t) {
		if !slices.Contains(imported, dir) {
			t.Errorf("providers/%s isn't imported by all.go", dir)
		}
		if !registered[dir] {
			t.Errorf("providers/%s registers no provider %q (its provider.go calls plugin.RegisterProvider with ID %q)", dir, dir, dir)
		}
		if ids := registeredIDs(t, filepath.Join("..", dir)); !slices.Equal(ids, []string{dir}) {
			t.Errorf("providers/%s: plugin.RegisterProvider is called for %v, want once, for %q, in provider.go", dir, ids, dir)
		}
	}
	for id := range registered {
		if !slices.Contains(providerDirs(t), id) {
			t.Errorf("provider %q is registered, but there's no providers/%s/", id, id)
		}
	}
}

// registeredIDs are the provider IDs a provider's folder registers: the ID
// in each plugin.RegisterProvider call of its non-test code, which must be
// in provider.go.
func registeredIDs(t *testing.T, dir string) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || !isSelector(call.Fun, "plugin", "RegisterProvider") {
				return true
			}
			if filepath.Base(name) != "provider.go" {
				t.Errorf("%s calls plugin.RegisterProvider: only provider.go does", name)
			}
			ids = append(ids, providerID(call))
			return true
		})
	}
	return ids
}

func isSelector(e ast.Expr, pkg, name string) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == pkg
}

// providerID is the ID field of a RegisterProvider call's literal, or "?".
func providerID(call *ast.CallExpr) string {
	if len(call.Args) != 1 {
		return "?"
	}
	lit, ok := call.Args[0].(*ast.CompositeLit)
	if !ok {
		return "?"
	}
	for _, el := range lit.Elts {
		kv, ok := el.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if k, ok := kv.Key.(*ast.Ident); ok && k.Name == "ID" {
			if v, ok := kv.Value.(*ast.BasicLit); ok {
				s, _ := strconv.Unquote(v.Value)
				return s
			}
		}
	}
	return "?"
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
// fields), is found by its id, and its provider's page exists and names it.
func TestEverySourceHasMetadata(t *testing.T) {
	for _, s := range plugin.Sources() {
		page, err := os.ReadFile(filepath.Join("..", "..", s.Source.Page))
		if err != nil {
			t.Errorf("%s: its provider's page %s doesn't exist (copy docs/provider-template.md)", s.FullID(), s.Source.Page)
		} else if !strings.Contains(string(page), "`"+s.FullID()+"`") {
			t.Errorf("%s: its provider's page %s doesn't name it (`%s`)", s.FullID(), s.Source.Page, s.FullID())
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

// Every package under kinds/ is a kind (kinds/<kind>, whose plugin type
// every source of that kind runs) or a part of one (kinds/<kind>/<reader>/,
// or its test helpers); none registers a provider, and none only forwards
// to another package (type aliases and re-exported values, no code of its
// own). Nor does any provider's.
func TestKindsReadAndProvidersSayWhat(t *testing.T) {
	kinds := map[string]bool{}
	for _, s := range plugin.Sources() {
		kinds[s.Source.Kind] = true
	}
	entries, err := os.ReadDir("../../kinds")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() && !kinds[e.Name()] {
			t.Errorf("kinds/%s: no registered source is of this kind", e.Name())
		}
	}
	for _, root := range []string{"../../kinds", ".."} {
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return err
			}
			if d.Name() == "testdata" {
				return filepath.SkipDir
			}
			if root == "../../kinds" && len(registeredIDs(t, p)) > 0 {
				t.Errorf("%s registers a provider: a provider registers itself, in providers/<provider>/provider.go", p)
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
