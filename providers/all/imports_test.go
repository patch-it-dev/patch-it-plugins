package all

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The import graph, tests included: core/ knows no kind and no provider; a
// kind (how to read) knows no provider; a provider (what to read) imports
// only core/ and kinds/, never another provider. Only this package, and the
// commands through it, imports every provider.
func TestTheImportGraph(t *testing.T) {
	top, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, root := range []string{"../../core", "../../kinds", ".."} {
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && d.Name() == "testdata" {
				return filepath.SkipDir
			}
			if d.IsDir() || !strings.HasSuffix(p, ".go") {
				return nil
			}
			abs, err := filepath.Abs(p)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(top, abs)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if strings.HasPrefix(rel, "providers/all/") {
				return nil
			}
			f, err := parser.ParseFile(token.NewFileSet(), p, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			checked++
			for _, imp := range f.Imports {
				path, _ := strconv.Unquote(imp.Path.Value)
				if why := forbidden(rel, path); why != "" {
					t.Errorf("%s imports %s: %s", rel, path, why)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if checked < 100 {
		t.Fatalf("checked %d files: the walk found too little to guard anything", checked)
	}
}

// forbidden is why file (a path from the repo's root) may not import path,
// or "".
func forbidden(file, path string) string {
	pkg, ok := strings.CutPrefix(path, module)
	if !ok {
		return ""
	}
	switch top, _, _ := strings.Cut(file, "/"); top {
	case "core":
		if strings.HasPrefix(pkg, "kinds/") || strings.HasPrefix(pkg, "providers/") {
			return "core/ knows no kind and no provider"
		}
	case "kinds":
		if strings.HasPrefix(pkg, "providers/") {
			return "a kind says how to read, never what: it imports no provider"
		}
	case "providers":
		own := strings.SplitN(file, "/", 3)[1]
		if !strings.HasPrefix(pkg, "core/") && !strings.HasPrefix(pkg, "kinds/") && pkg != "providers/"+own {
			return "a provider imports only core/ and kinds/, never another provider"
		}
	}
	return ""
}
