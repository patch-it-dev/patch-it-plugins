package versions

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// serve answers saved (here: written out) responses by URL; anything else
// fails the test.
func serve(t *testing.T, pages map[string]string) Getter {
	return func(_ context.Context, u string) ([]byte, error) {
		b, ok := pages[u]
		if !ok {
			t.Errorf("fetched %s, which the test didn't save", u)
			return nil, fmt.Errorf("no page %s", u)
		}
		return []byte(b), nil
	}
}

func ids(vs []plugin.Version) string {
	var out []string
	for _, v := range vs {
		out = append(out, v.ID+"@"+v.Ref)
	}
	return strings.Join(out, " ")
}

func TestGitHubReleasesSkipsDraftsAndPrereleasesOldestFirst(t *testing.T) {
	get := serve(t, map[string]string{
		"https://api.github.com/repos/acme/spec/releases?per_page=100&page=1": `[
			{"tag_name":"v3","published_at":"2026-03-01T00:00:00Z","html_url":"https://github.com/acme/spec/releases/tag/v3"},
			{"tag_name":"v3-rc","prerelease":true,"published_at":"2026-02-20T00:00:00Z"},
			{"tag_name":"v4","draft":true},
			{"tag_name":"v2","published_at":"2026-02-01T00:00:00Z"},
			{"tag_name":"canary-1","published_at":"2026-02-15T00:00:00Z"}
		]`,
	})
	vs, err := GitHubReleases(context.Background(), get, "acme/spec", func(tag string) bool { return !strings.HasPrefix(tag, "canary") })
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(vs); got != "v2@v2 v3@v3" {
		t.Errorf("got %s", got)
	}
}

func TestGitHubReleasesReadsEveryPage(t *testing.T) {
	var full []string
	for i := 0; i < 100; i++ {
		full = append(full, fmt.Sprintf(`{"tag_name":"v%d","published_at":"2025-01-01T00:00:%02dZ"}`, i, i%60))
	}
	get := serve(t, map[string]string{
		"https://api.github.com/repos/acme/spec/releases?per_page=100&page=1": "[" + strings.Join(full, ",") + "]",
		"https://api.github.com/repos/acme/spec/releases?per_page=100&page=2": `[{"tag_name":"v100","published_at":"2026-01-01T00:00:00Z"}]`,
	})
	vs, err := GitHubReleases(context.Background(), get, "acme/spec", nil)
	if err != nil || len(vs) != 101 || vs[100].ID != "v100" {
		t.Fatalf("got %d versions, last %v, err %v", len(vs), vs[len(vs)-1], err)
	}
}

func TestGitHubTagsMatchesAndSortsByVersion(t *testing.T) {
	get := serve(t, map[string]string{
		"https://api.github.com/repos/acme/sdk/tags?per_page=100&page=1": `[{"name":"v1.10.0"},{"name":"v1.9.0"},{"name":"v2.0.0rc1"},{"name":"nightly"},{"name":"v1.2.0"}]`,
	})
	vs, err := GitHubTags(context.Background(), get, "acme/sdk", regexp.MustCompile(`^v(\d+\.\d+\.\d+)$`))
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(vs); got != "1.2.0@v1.2.0 1.9.0@v1.9.0 1.10.0@v1.10.0" {
		t.Errorf("got %s", got)
	}
}

func TestGitHubCommitsAreDatedVersions(t *testing.T) {
	get := serve(t, map[string]string{
		"https://api.github.com/repos/acme/spec/commits?path=spec%2Fopenapi.json&per_page=100&page=1": `[
			{"sha":"bbb","commit":{"committer":{"date":"2026-11-19T10:00:00Z"}}},
			{"sha":"aaa","commit":{"committer":{"date":"2026-10-01T09:00:00Z"}}}
		]`,
	})
	vs, err := GitHubCommits(context.Background(), get, "acme/spec", "spec/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(vs); got != "2026-10-01@aaa 2026-11-19@bbb" {
		t.Errorf("got %s", got)
	}
}

func TestPyPIReleasesAreFinalAndNotYanked(t *testing.T) {
	get := serve(t, map[string]string{
		"https://pypi.org/pypi/acme/json": `{"releases":{
			"1.0.0":[{"upload_time_iso_8601":"2024-01-01T00:00:00Z"}],
			"1.10.0":[{"upload_time_iso_8601":"2025-01-01T00:00:00Z"}],
			"1.9.0":[{"upload_time_iso_8601":"2024-06-01T00:00:00Z"}],
			"2.0.0rc1":[{"upload_time_iso_8601":"2025-02-01T00:00:00Z"}],
			"1.9.1":[{"upload_time_iso_8601":"2024-07-01T00:00:00Z","yanked":true}],
			"0.9":[]
		}}`,
	})
	vs, err := PyPIReleases(context.Background(), get, "acme")
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(vs); got != "1.0.0@1.0.0 1.9.0@1.9.0 1.10.0@1.10.0" {
		t.Errorf("got %s", got)
	}
	if !vs[0].Released.Equal(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("released %v", vs[0].Released)
	}
	all, err := PyPIReleasesWithYanked(context.Background(), get, "acme")
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(all); got != "1.0.0@1.0.0 1.9.0@1.9.0 1.9.1@1.9.1 1.10.0@1.10.0" {
		t.Errorf("with yanked: got %s", got)
	}
}

func TestFirstOfMajor(t *testing.T) {
	for v, want := range map[string]bool{"2.0.0": true, "11.0.0": true, "2.0.1": false, "2.1.0": false, "0.0.0": false, "2.0": false, "2.0.0rc1": false} {
		if got := FirstOfMajor(v); got != want {
			t.Errorf("FirstOfMajor(%q) = %v, want %v", v, got, want)
		}
	}
}

func TestNPMReleases(t *testing.T) {
	get := serve(t, map[string]string{
		"https://registry.npmjs.org/@acme%2Fsdk": `{"versions":{"1.0.0":{},"1.1.0":{"deprecated":"use 1.1.1"},"1.1.1":{},"2.0.0-beta.1":{}},
			"time":{"1.0.0":"2024-01-01T00:00:00Z","1.1.1":"2024-02-01T00:00:00Z"}}`,
	})
	vs, err := NPMReleases(context.Background(), get, "@acme/sdk")
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(vs); got != "1.0.0@1.0.0 1.1.1@1.1.1" {
		t.Errorf("got %s", got)
	}
}

func TestCompareFinalMajorPairs(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{{"1.9", "1.10", -1}, {"2.0.0", "2.0.0", 0}, {"2026-01-02", "2025-12-31", 1}, {"v2", "v10", -1}, {"1.0", "1.0.1", -1}} {
		if got := Compare(c.a, c.b); got != c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
	for v, want := range map[string]bool{"2.0.0": true, "v0.86.0": true, "2.0.0rc1": false, "1.0.0-beta.1": false, "1.0.post1": false} {
		if Final(v) != want {
			t.Errorf("Final(%q) = %v", v, !want)
		}
	}
	if Major("posthog-v8.1.0") != 8 || Major("none") != -1 {
		t.Error("Major")
	}
	ps := Pairs([]plugin.Version{{ID: "1"}, {ID: "2"}, {ID: "3"}})
	if len(ps) != 2 || ps[0][0].ID != "1" || ps[1][1].ID != "3" {
		t.Errorf("Pairs = %v", ps)
	}
}
