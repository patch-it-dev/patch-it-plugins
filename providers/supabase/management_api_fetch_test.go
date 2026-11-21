package supabase

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
)

// fakeSupabaseGitHub serves the commits list, one commit, and specs by path.
func fakeSupabaseGitHub(t *testing.T, commitsJSON string, commits, specs map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/commits":
			w.Write([]byte(commitsJSON))
		case strings.HasPrefix(r.URL.Path, "/commits/"):
			body, ok := commits[strings.TrimPrefix(r.URL.Path, "/commits/")]
			if !ok {
				http.Error(w, `{"message":"No commit found"}`, http.StatusUnprocessableEntity)
				return
			}
			w.Write([]byte(body))
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
	oldList, oldOne, oldSpec := supabaseCommitsURL, supabaseCommitURL, supabaseSpecURLTemplate
	supabaseCommitsURL = srv.URL + "/commits"
	supabaseCommitURL = srv.URL + "/commits/%s"
	supabaseSpecURLTemplate = srv.URL + "/%s/spec.json"
	t.Cleanup(func() { supabaseCommitsURL, supabaseCommitURL, supabaseSpecURLTemplate = oldList, oldOne, oldSpec })
	return srv
}

const fullSHA = "eb21ae25f7954ea1e5db34209ff69fce597b0ae3"

func TestSupabaseLatestTagIsTheNewestSpecCommit(t *testing.T) {
	fakeSupabaseGitHub(t, `[{"sha":"`+fullSHA+`"},{"sha":"older"}]`, nil, nil)
	tag, err := Supabase{}.LatestTag(context.Background())
	if err != nil || tag != fullSHA {
		t.Errorf("got %q, %v; want %s", tag, err, fullSHA)
	}
}

func TestSupabaseLatestTagRejectsNoCommit(t *testing.T) {
	fakeSupabaseGitHub(t, `[]`, nil, nil)
	if _, err := (Supabase{}).LatestTag(context.Background()); err == nil {
		t.Error("no commit: want an error")
	}
}

// The version is the commit's date, since the Management API has none (its
// info.version is a fixed 1.0.0); the source is pinned to the full SHA even
// when a short one is asked for, and the content is passed through.
func TestSupabaseFetchPinsTheFullSHAAndDatesByCommit(t *testing.T) {
	const body = `{"openapi":"3.0.0","info":{"version":"1.0.0"},"paths":{}}`
	srv := fakeSupabaseGitHub(t, "", map[string]string{
		"eb21ae25": `{"sha":"` + fullSHA + `","commit":{"committer":{"date":"2026-06-29T08:15:50Z"}}}`,
	}, map[string]string{"/" + fullSHA + "/spec.json": body})
	spec, err := Supabase{}.Fetch(context.Background(), "eb21ae25")
	if err != nil {
		t.Fatal(err)
	}
	if spec.Version != "2026-06-29" || spec.SourceURL != srv.URL+"/"+fullSHA+"/spec.json" || string(spec.Content) != body {
		t.Errorf("got version %q, url %s, content %s", spec.Version, spec.SourceURL, spec.Content)
	}
}

func TestSupabaseFetchFailures(t *testing.T) {
	commit := `{"sha":"` + fullSHA + `","commit":{"committer":{"date":"2026-06-29T08:15:50Z"}}}`
	for name, c := range map[string]struct{ commits, specs map[string]string }{
		"unknown commit":      {nil, nil},
		"commit without date": {map[string]string{"x": `{"sha":"` + fullSHA + `"}`}, nil},
		"spec missing":        {map[string]string{"x": commit}, nil},
		"invalid json":        {map[string]string{"x": commit}, map[string]string{"/" + fullSHA + "/spec.json": "<html>"}},
	} {
		fakeSupabaseGitHub(t, "", c.commits, c.specs)
		if _, err := (Supabase{}).Fetch(context.Background(), "x"); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

// The plugin lists the commits that changed the spec, through core/versions.
func TestSupabaseListsTheSpecsCommits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/supabase/supabase/commits" && r.URL.Query().Get("path") == "apps/docs/spec/api_v1_openapi.json" {
			fmt.Fprint(w, `[{"sha":"b","commit":{"committer":{"date":"2026-02-01T00:00:00Z"}}},{"sha":"a","commit":{"committer":{"date":"2026-01-01T00:00:00Z"}}}]`)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	old := versions.GitHubAPI
	versions.GitHubAPI = srv.URL
	defer func() { versions.GitHubAPI = old }()
	vs, err := openapi.Plugin{Adapter: Supabase{}}.ListVersions(context.Background(), plugin.Env{})
	if err != nil {
		t.Fatal(err)
	}
	var refs []string
	for _, v := range vs {
		refs = append(refs, v.Ref)
	}
	if strings.Join(refs, " ") != "a b" {
		t.Errorf("refs %v, want a b", refs)
	}
}
