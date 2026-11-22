package dropbox

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

func TestFilesAPIRegistersItself(t *testing.T) {
	kittest.Registered(t, FilesAPI)
}

// The golden test: two real, consecutive commits of files.stone (the file
// didn't change in between: checked 2026-11-21, no commit touches it from
// 2023-04-26 to 2026-05-12). properties/remove, properties/template/get and
// properties/template/list (all three already marked `deprecated` at
// c8364404) are gone at 6c68656c; tags/add, tags/get, tags/remove and
// upload_session/append_batch are new. Every other route is unchanged, even
// though the file reorders many of them between the two commits: a line
// diff would read most of the file as removed and re-added, which is why
// Routes compares names, never lines.
func TestFilesAPIGolden(t *testing.T) {
	env := kittest.Env(t, kittest.Pages{
		FilesAPI.FileURL("c8364404953d875801d496a81f786c5545f78223"): "testdata/dropbox-files-c8364404.stone",
		FilesAPI.FileURL("6c68656caafdff65a30943fe186a1800b9f35bbc"): "testdata/dropbox-files-6c68656c.stone",
	})
	got := kittest.Versions(t, env, FilesAPI,
		plugin.Version{ID: "2023-04-26", Ref: "c8364404953d875801d496a81f786c5545f78223"},
		plugin.Version{ID: "2026-05-12", Ref: "6c68656caafdff65a30943fe186a1800b9f35bbc"})
	kittest.Golden(t, "testdata/golden/dropbox-files-api.json", got)
}

// The pinned drop test: the very next real commit to touch files.stone
// (2026-06-07) only changes field-level comments and annotations. Not one
// route's name appears or disappears, so nothing is stored.
func TestFilesAPIDropsACommentOnlyChange(t *testing.T) {
	env := kittest.Env(t, kittest.Pages{
		FilesAPI.FileURL("6c68656caafdff65a30943fe186a1800b9f35bbc"): "testdata/dropbox-files-6c68656c.stone",
		FilesAPI.FileURL("a4fc0169eda7dc981facd4208ebc7b66a4db00ee"): "testdata/dropbox-files-a4fc0169.stone",
	})
	got := kittest.Versions(t, env, FilesAPI,
		plugin.Version{ID: "2026-05-12", Ref: "6c68656caafdff65a30943fe186a1800b9f35bbc"},
		plugin.Version{ID: "2026-06-07", Ref: "a4fc0169eda7dc981facd4208ebc7b66a4db00ee"})
	if len(got) != 0 {
		t.Errorf("%d events, want 0: this commit only touches comments", len(got))
	}
}

// A route's trailing :N (search:2, copy_batch/check:2) maps to the real
// HTTP path's _vN (search_v2, copy_batch/check_v2), never the Stone
// spelling: checked against the published openapi/openapi.yaml on
// 2026-11-21.
func TestFilesAPIOperationMapsTheVSuffix(t *testing.T) {
	for name, want := range map[string]string{
		"search":             "POST /2/files/search",
		"search:2":           "POST /2/files/search_v2",
		"copy_batch/check:2": "POST /2/files/copy_batch/check_v2",
		"properties/remove":  "POST /2/files/properties/remove",
	} {
		if got := FilesAPI.Operation(name); got != want {
			t.Errorf("Operation(%q) = %q, want %q", name, got, want)
		}
	}
}
