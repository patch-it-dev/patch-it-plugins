package openapi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/adapter"
	"gopkg.in/yaml.v3"
)

// flagsmithTags are the saved copies under testdata/ (checked 2026-11-19):
// tag -> file and the full commit SHA each was copied from.
//
// v2.286.0 / v2.287.0 are release tags: "feat(API): Remove the experimental
// flag update endpoints" (#8736, commit 20cd1c2da) removed three whole
// operations (POST .../delete-segment-override/, .../update-flag-v1/,
// .../update-flag-v2/) between them, each marked "EXPERIMENTAL ENDPOINT -
// Subject to change without notice" in its own description, and listed under
// "Features" in the v2.287.0 release notes.
//
// desc-c3ef832 / desc-f504f16 are two commits, not tags (git log -S on
// openapi.yaml): "feat: rename admin api to management" (#8464, commit
// f504f160d, parent c3ef83244) only reworded two description strings ("the
// Admin API" to "the Management API"); nothing it names changed. The pinned
// drop test.
var flagsmithTags = map[string]string{
	"v2.286.0": "testdata/flagsmith-v2.286.0.yaml",
	"v2.287.0": "testdata/flagsmith-v2.287.0.yaml",
	"c3ef832":  "testdata/flagsmith-desc-c3ef832.yaml",
	"f504f16":  "testdata/flagsmith-desc-f504f16.yaml",
}

func TestFlagsmithRegistersItself(t *testing.T) {
	kittest.Registered(t, Plugin{Adapter: adapter.Flagsmith{}})
}

// fakeFlagsmithAdapter serves the saved copies, cited by the real URL each
// was copied from, converted from YAML to JSON and the tag's leading "v"
// trimmed from Version the same way adapter.Flagsmith.Fetch does.
type fakeFlagsmithAdapter struct{}

func (fakeFlagsmithAdapter) API() string { return "flagsmith/api" }

func (fakeFlagsmithAdapter) LatestTag(context.Context) (string, error) { return "v2.287.0", nil }

func (fakeFlagsmithAdapter) Fetch(_ context.Context, tag string) (adapter.Spec, error) {
	file, ok := flagsmithTags[tag]
	if !ok {
		return adapter.Spec{}, fmt.Errorf("no saved copy for %s", tag)
	}
	body, err := os.ReadFile(file)
	if err != nil {
		return adapter.Spec{}, err
	}
	var doc any
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return adapter.Spec{}, err
	}
	content, err := json.Marshal(doc)
	if err != nil {
		return adapter.Spec{}, err
	}
	return adapter.Spec{
		Content:   content,
		Version:   strings.TrimPrefix(tag, "v"),
		SourceURL: "https://raw.githubusercontent.com/Flagsmith/flagsmith/" + tag + "/openapi.yaml",
	}, nil
}

// The golden test and the pair fixture: the real removal of the three
// experimental flag-update endpoints between v2.286.0 and v2.287.0.
func TestFlagsmithGolden(t *testing.T) {
	ctx := context.Background()
	env := kittest.Env(t, nil)
	sink := kittest.NewMem()
	p := Plugin{Adapter: fakeFlagsmithAdapter{}, Keep: requestSide}
	if rejected, err := kittest.Pair(ctx, env, sink, p, "v2.286.0", "v2.287.0"); err != nil || rejected {
		t.Fatalf("pair: rejected=%v err=%v", rejected, err)
	}
	kittest.Golden(t, "testdata/golden/flagsmith.json", kittest.Stored(t, sink))
}

// The pinned drop test: a commit that only reworded two description strings
// ("the Admin API" to "the Management API") must produce no event.
func TestFlagsmithDropsADescriptionOnlyChange(t *testing.T) {
	ctx := context.Background()
	env := kittest.Env(t, nil)
	sink := kittest.NewMem()
	p := Plugin{Adapter: fakeFlagsmithAdapter{}, Keep: requestSide}
	if rejected, err := kittest.Pair(ctx, env, sink, p, "c3ef832", "f504f16"); err != nil || rejected {
		t.Fatalf("pair: rejected=%v err=%v", rejected, err)
	}
	if got := kittest.Stored(t, sink); len(got) != 0 {
		t.Errorf("%d events from a description-only change, want none", len(got))
	}
}
