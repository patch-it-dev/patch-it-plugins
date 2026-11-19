package pluginlist

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/patch-it-dev/patch-it-plugins/plugins/all"
)

const root = "../.."

// The real table agrees with the registry, and docs/plugins.json is what
// `make plugins-json` writes from them. This is how CI_go (go test) catches a
// plugin added without its row, or a row changed without regenerating.
func TestPluginsJSONIsUpToDate(t *testing.T) {
	plugins, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	want, err := Marshal(plugins)
	if err != nil {
		t.Fatal(err)
	}
	have, err := os.ReadFile(filepath.Join(root, JSON))
	if err != nil {
		t.Fatalf("%v (run `make plugins-json`)", err)
	}
	if !bytes.Equal(have, want) {
		t.Errorf("%s is out of date: run `make plugins-json` and commit it", JSON)
	}
}

func TestReadTable(t *testing.T) {
	md := "intro\n" + StartMarker + "\n" +
		"| Plugin (`--api`) | Provider | What it watches | Format plugin | Source | Status |\n" +
		"|---|---|---|---|---|---|\n" +
		"| [`foo-api`](https://github.com/patch-it-dev/patch-it-plugins/blob/main/providers/foo-api.md) | Foo | Removed endpoints | `openapi` | [foo/spec](https://github.com/foo/spec) (`openapi.json`) | paused: not released yet |\n" +
		EndMarker + "\n"
	rows, err := ReadTable([]byte(md))
	if err != nil {
		t.Fatal(err)
	}
	want := Row{ID: "foo-api", Provider: "Foo", Watches: "Removed endpoints", Format: "openapi",
		Source: "[foo/spec](https://github.com/foo/spec) (`openapi.json`)", Status: "paused: not released yet"}
	if len(rows) != 1 || rows[0] != want {
		t.Errorf("rows = %+v, want [%+v]", rows, want)
	}
}

func TestReadTableRejectsABadRow(t *testing.T) {
	for name, row := range map[string]string{
		"relative link":  "| [`foo`](../providers/foo.md) | Foo | x | `openapi` | [s](https://s.example) | live |",
		"other page":     "| [`foo`](https://github.com/patch-it-dev/patch-it-plugins/blob/main/providers/bar.md) | Foo | x | `openapi` | [s](https://s.example) | live |",
		"missing a cell": "| [`foo`](https://github.com/patch-it-dev/patch-it-plugins/blob/main/providers/foo.md) | Foo | `openapi` | [s](https://s.example) | live |",
	} {
		md := StartMarker + "\n" + row + "\n" + EndMarker
		if _, err := ReadTable([]byte(md)); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

// A registered plugin with no row fails the list, naming the rule.
func TestListWantsARowPerPlugin(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, Table), []byte(StartMarker+"\n"+EndMarker+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := List(dir)
	if err == nil || !strings.Contains(err.Error(), "stripe is registered but has no row") {
		t.Errorf("err = %v, want stripe's missing row", err)
	}
}
