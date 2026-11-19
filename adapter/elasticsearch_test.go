package adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func fakeESGitHub(t *testing.T, releases, branches string, heads, specs map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/releases":
			w.Write([]byte(releases))
		case r.URL.Path == "/branches":
			w.Write([]byte(branches))
		case strings.HasPrefix(r.URL.Path, "/commits/"):
			sha, ok := heads[strings.TrimPrefix(r.URL.Path, "/commits/")]
			if !ok {
				http.NotFound(w, r)
				return
			}
			w.Write([]byte(`{"sha":"` + sha + `"}`))
		default:
			body, ok := specs[r.URL.Path]
			if !ok {
				http.NotFound(w, r)
				return
			}
			w.Write([]byte(body))
		}
	}))
	t.Cleanup(srv.Close)
	a, b, c, d := esReleasesURL, esBranchesURL, esCommitURL, esSpecURLTemplate
	esReleasesURL, esBranchesURL = srv.URL+"/releases", srv.URL+"/branches"
	esCommitURL, esSpecURLTemplate = srv.URL+"/commits/%s", srv.URL+"/%s/spec.json"
	t.Cleanup(func() { esReleasesURL, esBranchesURL, esCommitURL, esSpecURLTemplate = a, b, c, d })
	return srv
}

const esBranches = `[{"ref":"refs/heads/main"},{"ref":"refs/heads/8.19"},{"ref":"refs/heads/9.5"},{"ref":"refs/heads/9.6"}]`

// The newest released minor, not the newest branch (9.6 isn't released), and
// not the newest release by date (8.19.25 came out after 9.5.6).
func TestElasticsearchLatestTagIsTheNewestReleasedMinor(t *testing.T) {
	fakeESGitHub(t, `[{"tag_name":"v8.19.25"},{"tag_name":"v9.5.6"},{"tag_name":"v9.6.0-rc1","prerelease":true},{"tag_name":"v9.4.9"}]`, esBranches, nil, nil)
	tag, err := Elasticsearch{}.LatestTag(context.Background())
	if err != nil || tag != "9.5" {
		t.Errorf("got %q, %v; want 9.5", tag, err)
	}
}

func TestElasticsearchLatestTagFailures(t *testing.T) {
	fakeESGitHub(t, `[{"tag_name":"v9.6.0-rc1","prerelease":true}]`, esBranches, nil, nil)
	if _, err := (Elasticsearch{}).LatestTag(context.Background()); err == nil {
		t.Error("no release: want an error")
	}
	fakeESGitHub(t, `[{"tag_name":"v9.7.0"}]`, esBranches, nil, nil)
	if _, err := (Elasticsearch{}).LatestTag(context.Background()); err == nil {
		t.Error("a release with no spec branch: want an error, not an older branch")
	}
}

// Pinned to the branch head's commit; the version is the minor's .0 release.
func TestElasticsearchFetchPinsTheHead(t *testing.T) {
	const sha = "57ecfe5fda6976b94fc8b8830ecb51f4969dccd6"
	srv := fakeESGitHub(t, "", "", map[string]string{"9.0": sha}, map[string]string{"/" + sha + "/spec.json": `{"openapi":"3.0.3","info":{"version":""},"paths":{}}`})
	spec, err := Elasticsearch{}.Fetch(context.Background(), "9.0")
	if err != nil {
		t.Fatal(err)
	}
	if spec.Version != "9.0.0" || spec.SourceURL != srv.URL+"/"+sha+"/spec.json" {
		t.Errorf("got %q %s", spec.Version, spec.SourceURL)
	}
	for _, b := range []string{"main", "v9.0", "9.0.1"} {
		if _, err := (Elasticsearch{}).Fetch(context.Background(), b); err == nil {
			t.Errorf("%s: want an error", b)
		}
	}
	fakeESGitHub(t, "", "", map[string]string{"9.0": sha}, map[string]string{"/" + sha + "/spec.json": "<html>"})
	if _, err := (Elasticsearch{}).Fetch(context.Background(), "9.0"); err == nil {
		t.Error("invalid json: want an error")
	}
}
