package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

var today = time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)

// optionsFor is a source of the provider acme on template tmpl ("openapi",
// "notices/feed").
func optionsFor(root, tmpl string) Options {
	kind, reader, _ := strings.Cut(tmpl, "/")
	return Options{Root: root, ProviderID: "acme", Provider: "Acme Corp", SourceID: strings.ReplaceAll(tmpl, "/", "-"),
		Kind: kind, Reader: reader, Packages: []string{"pypi:acme-" + strings.ReplaceAll(tmpl, "/", "-")}, Today: today, Go: true}
}

// A source of every kind and reader, scaffolded into a copy of the repo:
// the copy builds, vets and passes its tests, and list-plugins lists them
// all. Then the same command again changes nothing.
func TestScaffoldEveryKind(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the go command over a copy of the repo")
	}
	goCmd, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go command on PATH")
	}
	root := copyRepo(t)
	for _, f := range formatNames() {
		o := optionsFor(root, f)
		var out bytes.Buffer
		if err := Run(o, &out); err != nil {
			t.Fatalf("%s: %v\n%s", f, err, out.String())
		}
		if !strings.Contains(out.String(), "Next steps") {
			t.Errorf("%s: no next steps in\n%s", f, out.String())
		}
	}
	for _, args := range [][]string{{"build", "./..."}, {"vet", "./..."}, {"test", "-count=1", "./plugins/...", "./internal/..."}} {
		cmd := exec.Command(goCmd, args...)
		cmd.Dir = root
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, b)
		}
	}
	list := exec.Command(goCmd, "run", "./cmd/list-plugins")
	list.Dir = root
	listed, err := list.Output()
	if err != nil {
		t.Fatalf("go run ./cmd/list-plugins: %v", err)
	}
	for _, f := range formatNames() {
		id := strings.ReplaceAll(f, "/", "-")
		if !bytes.Contains(listed, []byte(`"id": "acme/`+id+`"`)) || !bytes.Contains(listed, []byte(`"pypi:acme-`+id+`"`)) {
			t.Errorf("list-plugins doesn't list acme/%s with its package", id)
		}
	}

	before := tree(t, root)
	err = Run(optionsFor(root, "notices/feed"), io.Discard)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("second run: err = %v, want the id taken", err)
	}
	if after := tree(t, root); !slices.Equal(before, after) {
		t.Error("a refused run changed files")
	}
}

// A second source of a provider shares its page and its provider: no new
// page, and the registration names the existing ones.
func TestScaffoldSharesTheProviderPage(t *testing.T) {
	root := copyRepo(t)
	o := Options{Root: root, ProviderID: "algolia", SourceID: "analytics", Kind: "openapi", Page: "algolia", Today: today}
	if err := Run(o, io.Discard); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "providers", "algolia-analytics.md")); err == nil {
		t.Error("wrote providers/algolia-analytics.md; want the shared providers/algolia.md")
	}
	reg, _ := os.ReadFile(filepath.Join(root, "plugins", "openapi", "openapi.go"))
	for _, want := range []string{`ID: "algolia", Name: "Algolia"`, `ID: "analytics"`, `Page: "providers/algolia.md"`, `plugin.Category("search")`} {
		if !bytes.Contains(reg, []byte(want)) {
			t.Errorf("the registration has no %s", want)
		}
	}
}

func TestRefusesBadInput(t *testing.T) {
	root := copyRepo(t)
	before := tree(t, root)
	for name, o := range map[string]Options{
		"unknown kind":      {ProviderID: "acme", Provider: "Acme", SourceID: "api", Kind: "graphql"},
		"unknown reader":    {ProviderID: "acme", Provider: "Acme", SourceID: "api", Kind: "notices", Reader: "tweets"},
		"no source id":      {ProviderID: "acme", Provider: "Acme", Kind: "openapi"},
		"source registered": {ProviderID: "stripe", SourceID: "api", Kind: "openapi"},
		"other name":        {ProviderID: "stripe", Provider: "Stripe Inc", SourceID: "sdk", Kind: "apisurface"},
		"bad category":      {ProviderID: "acme", Provider: "Acme", SourceID: "api", Kind: "openapi", Category: "toys"},
		"bad package":       {ProviderID: "acme", Provider: "Acme", SourceID: "api", Kind: "openapi", Packages: []string{"pip:acme"}},
		"uppercase id":      {ProviderID: "Acme", Provider: "Acme", SourceID: "api", Kind: "notices"},
		"double hyphen":     {ProviderID: "acme", Provider: "Acme", SourceID: "a--x", Kind: "notices"},
		"no provider name":  {ProviderID: "acme", SourceID: "api", Kind: "notices"},
		"pipe in name":      {ProviderID: "acme", Provider: "Ac|me", SourceID: "api", Kind: "notices"},
		"unknown layer":     {ProviderID: "acme", Provider: "Acme", SourceID: "api", Kind: "notices", Layer: "database"},
		"http source":       {ProviderID: "acme", Provider: "Acme", SourceID: "api", Kind: "notices", URLs: []string{"http://acme.example/feed"}},
		"provider exists":   {ProviderID: "numpy", Provider: "NumPy Inc", SourceID: "changes", Kind: "changelog"},
	} {
		o.Root, o.Today = root, today
		if err := Run(o, io.Discard); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
	if after := tree(t, root); !slices.Equal(before, after) {
		t.Error("a refused run changed files")
	}
}

func TestFileBaseAvoidsBuildConstraints(t *testing.T) {
	for id, want := range map[string]string{
		"acme":         "acme",
		"acme-models":  "acme_models",
		"acme-linux":   "acme_linux_plugin",
		"acme-arm64":   "acme_arm64_plugin",
		"acme-test":    "acme_test_plugin",
		"linux":        "linux",
		"acme-js-sdk2": "acme_js_sdk2",
	} {
		if got := fileBase(id); got != want {
			t.Errorf("fileBase(%q) = %q, want %q", id, got, want)
		}
	}
	if got := ident("acme-models-2"); got != "AcmeModels2" {
		t.Errorf("ident = %q", got)
	}
}

// copyRepo copies the repo's tracked files to a temp dir.
func copyRepo(t *testing.T) string {
	t.Helper()
	src, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	cmd.Dir = src
	list, err := cmd.Output()
	if err != nil {
		t.Skipf("git ls-files: %v", err)
	}
	dst := t.TempDir()
	for _, f := range strings.Split(string(list), "\x00") {
		if f == "" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(src, f))
		if err != nil {
			continue // deleted in the worktree
		}
		if err := os.MkdirAll(filepath.Join(dst, filepath.Dir(f)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dst, f), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dst
}

// tree is every file under root with its hash, sorted.
func tree(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out = append(out, path+" "+hash(b))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func hash(b []byte) string {
	sum := sha256.Sum256(b)
	return string(sum[:])
}

// Asked at a terminal: what the flags left out, a registered provider's
// name, category and homepage not asked again.
func TestAsksWhatIsMissing(t *testing.T) {
	o := Options{ProviderID: "stripe"}
	in := bufio.NewReader(strings.NewReader("python-sdk\napisurface\nsdk\nhttps://pypi.org/project/stripe/\npypi:stripe\nRemoved names\n"))
	var out bytes.Buffer
	if err := ask(&o, in, &out); err != nil {
		t.Fatal(err)
	}
	if o.SourceID != "python-sdk" || o.Kind != "apisurface" || o.Layer != "sdk" || o.URLs[0] != "https://pypi.org/project/stripe/" || o.Packages[0] != "pypi:stripe" || o.Watches != "Removed names" || o.Provider != "" {
		t.Errorf("%+v", o)
	}
	if strings.Contains(out.String(), "Provider name") {
		t.Errorf("asked for a registered provider's name:\n%s", out.String())
	}
}
