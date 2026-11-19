// Command list-plugins writes docs/plugins.json: every registered plugin
// joined with its row in docs/plugins.md (id, provider, what it watches,
// format plugin, source URL, status, provider page), and sets the plugin
// count on the README's badge. It fails when the
// registry and the table disagree. Run it from the repo root:
//
//	go run ./cmd/list-plugins           # rewrite docs/plugins.json (make plugins-json)
//	go run ./cmd/list-plugins -check    # fail if docs/plugins.json is out of date (make check)
//	go run ./cmd/list-plugins -o -      # print it
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/patch-it-dev/patch-it-plugins/internal/pluginlist"
	_ "github.com/patch-it-dev/patch-it-plugins/plugins/all"
)

func main() {
	root := flag.String("root", ".", "the repo root")
	out := flag.String("o", pluginlist.JSON, "where to write the list, relative to -root; - for stdout")
	check := flag.Bool("check", false, "don't write: fail if the file differs from what would be written")
	flag.Parse()
	if err := run(*root, *out, *check); err != nil {
		fmt.Fprintln(os.Stderr, "list-plugins:", err)
		os.Exit(1)
	}
}

func run(root, out string, check bool) error {
	plugins, err := pluginlist.List(root)
	if err != nil {
		return err
	}
	b, err := pluginlist.Marshal(plugins)
	if err != nil {
		return err
	}
	if out == "-" {
		_, err := os.Stdout.Write(b)
		return err
	}
	readmePath := filepath.Join(root, pluginlist.Readme)
	readme, err := os.ReadFile(readmePath)
	if err != nil {
		return err
	}
	badged, _ := pluginlist.Badge(readme, len(plugins))
	path := filepath.Join(root, out)
	if check {
		have, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(have, b) || !bytes.Equal(readme, badged) {
			return fmt.Errorf("%s or the README's plugins badge is out of date: run `make plugins-json` and commit them", out)
		}
		return nil
	}
	if !bytes.Equal(readme, badged) {
		if err := os.WriteFile(readmePath, badged, 0o644); err != nil {
			return err
		}
	}
	return os.WriteFile(path, b, 0o644)
}
