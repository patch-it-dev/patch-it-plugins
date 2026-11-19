package main

import (
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

// A plugin of every format, scaffolded into a copy of the repo: the copy
// builds, vets and passes its tests, and docs/plugins.json lists them all.
// Then the same command again changes nothing.
func TestScaffoldEveryFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the go command over a copy of the repo")
	}
	goCmd, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go command on PATH")
	}
	root := copyRepo(t)
	for _, f := range formatNames() {
		o := Options{Root: root, API: "acme-" + f, Provider: "Acme Corp", Format: f, Today: today, Go: true}
		var out bytes.Buffer
		if err := Run(o, &out); err != nil {
			t.Fatalf("%s: %v\n%s", f, err, out.String())
		}
		if !strings.Contains(out.String(), "Next steps") {
			t.Errorf("%s: no next steps in\n%s", f, out.String())
		}
	}
	for _, args := range [][]string{{"build", "./..."}, {"vet", "./..."}, {"test", "-count=1", "./plugins/...", "./internal/...", "./migrate/..."}} {
		cmd := exec.Command(goCmd, args...)
		cmd.Dir = root
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, b)
		}
	}
	listed, err := os.ReadFile(filepath.Join(root, "docs", "plugins.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range formatNames() {
		if !bytes.Contains(listed, []byte(`"id": "acme-`+f+`"`)) {
			t.Errorf("docs/plugins.json doesn't list acme-%s", f)
		}
	}

	before := tree(t, root)
	err = Run(Options{Root: root, API: "acme-feed", Provider: "Acme Corp", Format: "feed", Today: today, Go: true}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "already has a row") {
		t.Errorf("second run: err = %v, want the id taken", err)
	}
	if after := tree(t, root); !slices.Equal(before, after) {
		t.Error("a refused run changed files")
	}
}

// A second plugin of a provider shares its page: no new page, and the row
// links the existing one.
func TestScaffoldSharesTheProviderPage(t *testing.T) {
	root := copyRepo(t)
	o := Options{Root: root, API: "algolia-analytics", Provider: "Algolia", Format: "openapi", Page: "algolia", Today: today}
	if err := Run(o, io.Discard); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "providers", "algolia-analytics.md")); err == nil {
		t.Error("wrote providers/algolia-analytics.md; want the shared providers/algolia.md")
	}
	table, _ := os.ReadFile(filepath.Join(root, "docs", "plugins.md"))
	if !bytes.Contains(table, []byte("[`algolia-analytics`](https://github.com/patch-it-dev/patch-it-plugins/blob/main/providers/algolia.md)")) {
		t.Error("the row doesn't link providers/algolia.md")
	}
}

func TestRefusesBadInput(t *testing.T) {
	root := copyRepo(t)
	before := tree(t, root)
	for name, o := range map[string]Options{
		"unknown format":  {API: "acme", Provider: "Acme", Format: "graphql"},
		"uppercase id":    {API: "Acme", Provider: "Acme", Format: "feed"},
		"double hyphen":   {API: "acme--x", Provider: "Acme", Format: "feed"},
		"id all":          {API: "all", Provider: "Acme", Format: "feed"},
		"registered id":   {API: "stripe", Provider: "Stripe", Format: "openapi"},
		"no provider":     {API: "acme", Format: "feed"},
		"pipe in name":    {API: "acme", Provider: "Ac|me", Format: "feed"},
		"unknown layer":   {API: "acme", Provider: "Acme", Format: "feed", Layer: "database"},
		"http source":     {API: "acme", Provider: "Acme", Format: "feed", Source: "http://acme.example/feed"},
		"provider exists": {API: "numpy", Provider: "NumPy", Format: "changelog"},
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
