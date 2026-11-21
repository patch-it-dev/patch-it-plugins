package all

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// release is a specific release in a string: a version number with three
// parts ("2.0.0", "v0.86.0", "posthog-v8.0.0", "python-agent-110000" aside)
// or a full commit SHA. A source lists its versions (PyPI, npm, tags,
// commits); only a test pins one, through its saved fixtures.
var release = regexp.MustCompile(`(^|[^0-9A-Za-z.])v?\d+\.\d+\.\d+([^0-9]|$)|(^|[^0-9a-f])[0-9a-f]{40}([^0-9a-f]|$)`)

// No plugin's code names a specific release or commit: every version comes
// from where the source publishes them. Comments may name the release a
// reader was checked at; a string the code uses may not.
func TestNoPinnedReleaseInPluginCode(t *testing.T) {
	fset := token.NewFileSet()
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "testdata" {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			s, err := strconv.Unquote(lit.Value)
			if err == nil && release.MatchString(s) {
				t.Errorf("%s: %q names a specific release; list versions from the source and pin one only in a test", fset.Position(lit.Pos()), s)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
