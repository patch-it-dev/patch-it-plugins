// Command list-plugins prints what patch-it watches, as JSON: every registered
// provider with its sources (id, kind, layer, what it watches, the
// URLs it reads, its page), built from the plugin registry at run time. The
// Release workflow attaches its output to each release as plugins.json; it is
// never committed.
//
//	go run ./cmd/list-plugins                       # print it (make plugins)
//	go run ./cmd/list-plugins -ref v0.3.0 -o f.json # link the pages at a tag, write a file
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/patch-it-dev/patch-it-plugins/internal/pluginlist"
	_ "github.com/patch-it-dev/patch-it-plugins/plugins/all"
)

func main() {
	ref := flag.String("ref", "main", "the git ref the provider pages are linked at: a release tag, or main")
	out := flag.String("o", "-", "where to write the list; - for stdout")
	flag.Parse()
	if err := run(*ref, *out); err != nil {
		fmt.Fprintln(os.Stderr, "list-plugins:", err)
		os.Exit(1)
	}
}

func run(ref, out string) error {
	b, err := pluginlist.Marshal(pluginlist.Build(ref))
	if err != nil {
		return err
	}
	if out == "-" {
		_, err := os.Stdout.Write(b)
		return err
	}
	return os.WriteFile(out, b, 0o644)
}
