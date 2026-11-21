package changelog

import (
	"context"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
)

// An ecosystem patch-it lists no registry for (Go here) lists the repo's
// TagPrefix + X.Y.Z tags; a source read from release notes lists the repo's
// published releases. Pre-releases, drafts and other tags are left out.
func TestReleasesFromTagsAndFromGitHubReleases(t *testing.T) {
	env := kittest.Env(t, kittest.Pages{
		"https://api.github.com/repos/stripe/stripe-go/tags?per_page=100&page=1":    "testdata/versions/stripe-go-tags.json",
		"https://api.github.com/repos/octokit/rest.js/releases?per_page=100&page=1": "testdata/versions/octokit-rest-releases.json",
	})
	for _, c := range []struct {
		file File
		want string
	}{
		{File{API: "stripe/go", Ecosystem: "go", Repo: "stripe/stripe-go", TagPrefix: "v", Path: "CHANGELOG.md"}, "72.121.0@v72.121.0 72.122.0@v72.122.0 73.0.0@v73.0.0"},
		{File{API: "github/octokit-rest", Ecosystem: "npm", Repo: "octokit/rest.js", TagPrefix: "v", Notes: true}, "17.11.2@v17.11.2 18.0.0@v18.0.0"},
	} {
		vs, err := Plugin{File: c.file}.ListVersions(context.Background(), env)
		if err != nil {
			t.Fatal(err)
		}
		var ids []string
		for _, v := range vs {
			ids = append(ids, v.ID+"@"+v.Ref)
		}
		if got := strings.Join(ids, " "); got != c.want {
			t.Errorf("%s: %s, want %s", c.file.API, got, c.want)
		}
	}
}
