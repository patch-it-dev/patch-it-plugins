package all

import (
	"go/parser"
	"go/token"
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

// registers reports whether a package's non-test code calls plugin.Register.
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
		if strings.Contains(string(b), "plugin.Register(") {
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
