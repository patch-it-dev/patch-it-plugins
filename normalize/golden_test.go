package normalize

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/diff"
)

// Golden-file tests for the whole pipeline: each directory under
// testdata/golden holds a spec pair (old.json, new.json) and the exact change
// events the pipeline must produce from it (events.json). The cases are the contract's
// hand-written examples: a new endpoint, a breaking removal, a deprecation, plus
// identical specs, which must produce nothing.
//
// When the pipeline's output changes on purpose, regenerate the files with
//
//	go test ./internal/normalize -run TestGolden -update
//
// and review every changed events.json in the diff before committing. That
// review is the point: a golden file nobody reads proves nothing.
var update = flag.Bool("update", false, "rewrite the golden events.json files from the current output")

// Everything that isn't part of the spec pair is fixed, so the output is exactly
// reproducible.
var goldenSource = Source{
	API:        "stripe",
	NewSpecURL: "https://raw.githubusercontent.com/stripe/openapi/v2468/latest/openapi.spec3.json",
	OldSpecURL: "https://raw.githubusercontent.com/stripe/openapi/v2460/latest/openapi.spec3.json",
	DetectedAt: time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC),
}

func TestGolden(t *testing.T) {
	dirs, err := filepath.Glob(filepath.Join("testdata", "golden", "*"))
	if err != nil || len(dirs) == 0 {
		t.Fatalf("no golden cases found: %v", err)
	}

	for _, dir := range dirs {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			oldSpec := parseFile(t, filepath.Join(dir, "old.json"))
			newSpec := parseFile(t, filepath.Join(dir, "new.json"))

			src := goldenSource
			src.Version, src.PreviousVersion = newSpec.Version, oldSpec.Version
			events, errs := Events(diff.Changes(oldSpec, newSpec), src)
			if len(errs) != 0 {
				t.Fatalf("pipeline rejected events: %v", errs)
			}

			got, err := json.MarshalIndent(events, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, '\n')

			golden := filepath.Join(dir, "events.json")
			if *update {
				if err := os.WriteFile(golden, got, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("reading %s (run with -update to create it, then review it): %v", golden, err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("pipeline output differs from %s\n--- got\n%s\n--- want\n%s", golden, got, want)
			}
		})
	}
}

func parseFile(t *testing.T, path string) *diff.Spec {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := diff.Parse(data)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	return spec
}
