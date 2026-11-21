package all

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// aliasOK marks a declaration this test lets through. Put it in the
// declaration's doc comment with the reason, e.g.
//
//	// alias-ok: <why this alias is the API itself, not an old path kept working>
const aliasOK = "alias-ok:"

// No forwarders: when code moves, the old path goes in the same change and
// every caller imports the new one. So no file in this module, tests
// included, declares at package level
//
//   - a type alias (type X = Y),
//   - a var or const that only re-exports another package's symbol
//     (var X = otherpkg.Y),
//   - a func whose body only calls another package's func with its own
//     parameters (func X(a, b T) R { return otherpkg.Y(a, b) }),
//
// unless its doc comment says why with "alias-ok: <reason>".
func TestNoAliasesOrForwarders(t *testing.T) {
	checked := 0
	err := filepath.WalkDir("../..", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && p != "../.." && (d.Name() == "testdata" || strings.HasPrefix(d.Name(), ".")) {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(p, ".go") {
			return nil
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, p, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		checked++
		for _, what := range forwarders(f) {
			t.Errorf("%s: %s. Delete it and point its callers at the real one, or say why it must stay with %q in its doc comment.",
				fset.Position(what.pos), what.desc, aliasOK)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked < 100 {
		t.Fatalf("checked only %d files; is the walk rooted at the module?", checked)
	}
}

type forwarder struct {
	pos  token.Pos
	name string
	desc string
}

// forwarders lists the package-level aliases and re-exports in f.
func forwarders(f *ast.File) []forwarder {
	imported := map[string]bool{}
	for _, imp := range f.Imports {
		name := importName(strings.Trim(imp.Path.Value, `"`))
		if imp.Name != nil {
			name = imp.Name.Name
		}
		imported[name] = true
	}
	// pkgSym says whether e is a bare otherpkg.Name.
	pkgSym := func(e ast.Expr) bool {
		sel, ok := e.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		id, ok := sel.X.(*ast.Ident)
		return ok && imported[id.Name]
	}
	allowed := func(docs ...*ast.CommentGroup) bool {
		for _, d := range docs {
			if d != nil && strings.Contains(d.Text(), aliasOK) {
				return true
			}
		}
		return false
	}

	var out []forwarder
	for _, decl := range f.Decls {
		switch decl := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					if spec.Assign.IsValid() && !allowed(decl.Doc, spec.Doc) {
						out = append(out, forwarder{spec.Pos(), spec.Name.Name, "type alias " + spec.Name.Name})
					}
				case *ast.ValueSpec:
					for i, v := range spec.Values {
						if pkgSym(v) && i < len(spec.Names) && !allowed(decl.Doc, spec.Doc) {
							out = append(out, forwarder{spec.Pos(), spec.Names[i].Name, decl.Tok.String() + " " + spec.Names[i].Name + " re-exports another package's symbol"})
						}
					}
				}
			}
		case *ast.FuncDecl:
			if decl.Recv == nil && decl.Body != nil && len(decl.Body.List) == 1 && !allowed(decl.Doc) {
				if call := onlyCall(decl.Body.List[0]); call != nil && pkgSym(call.Fun) && passesParams(decl.Type.Params, call.Args) {
					out = append(out, forwarder{decl.Pos(), decl.Name.Name, "func " + decl.Name.Name + " only forwards to another package's func"})
				}
			}
		}
	}
	return out
}

// importName is the name an unnamed import is used by: the last element,
// without a major-version suffix (gopkg.in/yaml.v3 is yaml, example.com/x/v2
// is x).
func importName(path string) string {
	elems := strings.Split(path, "/")
	name := elems[len(elems)-1]
	if len(elems) > 1 && isMajor(name) {
		name = elems[len(elems)-2]
	}
	if i := strings.Index(name, ".v"); i > 0 && isMajor(name[i+1:]) {
		name = name[:i]
	}
	return name
}

// isMajor says whether s is a major version element: v2, v3, ...
func isMajor(s string) bool {
	if len(s) < 2 || s[0] != 'v' {
		return false
	}
	for _, r := range s[1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// onlyCall is the call in `return f(...)` or `f(...)`, else nil.
func onlyCall(s ast.Stmt) *ast.CallExpr {
	switch s := s.(type) {
	case *ast.ReturnStmt:
		if len(s.Results) == 1 {
			c, _ := s.Results[0].(*ast.CallExpr)
			return c
		}
	case *ast.ExprStmt:
		c, _ := s.X.(*ast.CallExpr)
		return c
	}
	return nil
}

// passesParams says whether args are exactly the function's parameters, in order.
func passesParams(params *ast.FieldList, args []ast.Expr) bool {
	var names []string
	for _, fld := range params.List {
		for _, n := range fld.Names {
			names = append(names, n.Name)
		}
	}
	if len(names) != len(args) {
		return false
	}
	for i, a := range args {
		id, ok := a.(*ast.Ident)
		if !ok || id.Name != names[i] {
			return false
		}
	}
	return true
}

// The check itself: each shape it refuses is caught, and the marker lets one through.
func TestForwardersCatchesEachShape(t *testing.T) {
	src := `package p

import (
	"strings"
	x "example.com/extract"
	"gopkg.in/yaml.v3"
	"example.com/sdk/v2"
)

type A = x.Entry
type B = int

var C = x.ParseFeed
const D = strings.Title

func E(s string) string { return strings.ToUpper(s) }

var J = yaml.Marshal
var K = sdk.Client

// alias-ok: kept on purpose, for the test
type F = x.Entry

var G = x.ParseFeed("")
func H(s string) string { return strings.ToUpper(s + "!") }
func I(s string) string { return local(s) }
`
	f, err := parser.ParseFile(token.NewFileSet(), "p.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, w := range forwarders(f) {
		got = append(got, w.name)
	}
	if want := "A B C D E J K"; strings.Join(got, " ") != want {
		t.Errorf("caught %v, want %s", got, want)
	}
}
