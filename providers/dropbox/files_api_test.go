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
// c8364404) are gone at 6c68656c; upload_session/append_batch is new.
// tags/add, tags/get and tags/remove are new to files.stone but not to the
// namespace: at c8364404 they're in file_tagging.stone, also namespace
// files, which 6c68656c deletes. Moved: no event. Every other route is unchanged, even
// though the file reorders many of them between the two commits: a line
// diff would read most of the file as removed and re-added, which is why
// Routes compares names, never lines.
func TestFilesAPIGolden(t *testing.T) {
	env := kittest.Env(t, kittest.Pages{
		FilesAPI.FileURL("c8364404953d875801d496a81f786c5545f78223"):                                                             "testdata/dropbox-files-c8364404.stone",
		FilesAPI.FileURL("6c68656caafdff65a30943fe186a1800b9f35bbc"):                                                             "testdata/dropbox-files-6c68656c.stone",
		FilesAPI.ListingURL("c8364404953d875801d496a81f786c5545f78223"):                                                          "testdata/dropbox-spec-listing-c8364404.json",
		FilesAPI.ListingURL("6c68656caafdff65a30943fe186a1800b9f35bbc"):                                                          "testdata/dropbox-spec-listing-6c68656c.json",
		"https://raw.githubusercontent.com/dropbox/dropbox-api-spec/c8364404953d875801d496a81f786c5545f78223/file_tagging.stone": "testdata/dropbox-file_tagging-c8364404.stone",
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

// The pinned rename test: a real commit (2018-08-14) where Dropbox
// changed its own Stone spelling for five routes (copy_v2 -> copy:2,
// create_folder_v2 -> create_folder:2, delete_v2 -> delete:2,
// move_v2 -> move:2, upload_session/append_v2 -> upload_session/append:2)
// with no change to the real HTTP call either spelling answers to. Found by
// walking the full real history outside this test suite: a naive name
// diff reports all five as a route removed and a route added, which is
// wrong, so this is pinned here to keep it that way. get_temporary_upload_link,
// genuinely new in the same commit, must still come through as added.
func TestFilesAPIDropsARenamedSpelling(t *testing.T) {
	env := kittest.Env(t, kittest.Pages{
		FilesAPI.FileURL("222ba8c06cbf005e38f63c253b44b60d82200f17"):    "testdata/dropbox-files-222ba8c0.stone",
		FilesAPI.FileURL("ff540e4a194313d394ca506f6a6aefa2f34bf968"):    "testdata/dropbox-files-ff540e4a.stone",
		FilesAPI.ListingURL("222ba8c06cbf005e38f63c253b44b60d82200f17"): "testdata/dropbox-spec-listing-222ba8c0.json",
	})
	got := kittest.Versions(t, env, FilesAPI,
		plugin.Version{ID: "2018-05-17", Ref: "222ba8c06cbf005e38f63c253b44b60d82200f17"},
		plugin.Version{ID: "2018-08-14", Ref: "ff540e4a194313d394ca506f6a6aefa2f34bf968"})
	if len(got) != 1 {
		t.Fatalf("%d events, want 1 (only get_temporary_upload_link added): %+v", len(got), got)
	}
	if got[0].Change != "added" || got[0].Subject.Operation != "POST /2/files/get_temporary_upload_link" {
		t.Errorf("got %+v", got[0])
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
