package adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func fakeAlgoliaGitHub(t *testing.T, refs string, commitDates, specs map[string]string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/refs":
			w.Write([]byte(refs))
		case strings.HasPrefix(r.URL.Path, "/commits/"):
			d, ok := commitDates[strings.TrimPrefix(r.URL.Path, "/commits/")]
			if !ok {
				http.NotFound(w, r)
				return
			}
			w.Write([]byte(`{"commit":{"committer":{"date":"` + d + `"}}}`))
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
	oldRefs, oldCommit, oldSpec := algoliaRefsURL, algoliaCommitURL, algoliaSpecURLTemplate
	algoliaRefsURL = srv.URL + "/refs"
	algoliaCommitURL = srv.URL + "/commits/%s"
	algoliaSpecURLTemplate = srv.URL + "/%s/%s.yml"
	t.Cleanup(func() { algoliaRefsURL, algoliaCommitURL, algoliaSpecURLTemplate = oldRefs, oldCommit, oldSpec })
}

func refs(tags ...string) string {
	var parts []string
	for _, t := range tags {
		parts = append(parts, `{"ref":"refs/tags/`+t+`","object":{"type":"commit"}}`)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func TestAlgoliaLatestTagIsTheNewestDay(t *testing.T) {
	fakeAlgoliaGitHub(t, refs("released-2023-08-22", "released-2026-11-17-31c309fe2a", "released-2026-11-10-8d91e3d8a3", "v4.0.0"), nil, nil)
	tag, err := Algolia{Spec: "search"}.LatestTag(context.Background())
	if err != nil || tag != "released-2026-11-17-31c309fe2a" {
		t.Errorf("got %q, %v", tag, err)
	}
}

// Three releases on 2024-09-19: the suffix is a commit, so its order means
// nothing; the commit dates decide (a62c27d3b 10:17, 6e91dbf55 13:16,
// e47df6533 19:28, checked 2026-11-19).
func TestAlgoliaLatestTagBreaksASameDayTieByCommitDate(t *testing.T) {
	fakeAlgoliaGitHub(t, refs("released-2024-09-19-6e91dbf55", "released-2024-09-19-e47df6533", "released-2024-09-19-a62c27d3b"), map[string]string{
		"released-2024-09-19-6e91dbf55": "2024-09-19T13:16:52Z",
		"released-2024-09-19-a62c27d3b": "2024-09-19T10:17:21Z",
		"released-2024-09-19-e47df6533": "2024-09-19T19:28:05Z",
	}, nil)
	tag, err := Algolia{Spec: "search"}.LatestTag(context.Background())
	if err != nil || tag != "released-2024-09-19-e47df6533" {
		t.Errorf("got %q, %v; want the 19:28 release", tag, err)
	}
}

func TestAlgoliaLatestTagFailures(t *testing.T) {
	fakeAlgoliaGitHub(t, refs("v4.0.0"), nil, nil)
	if _, err := (Algolia{Spec: "search"}).LatestTag(context.Background()); err == nil {
		t.Error("no release tag: want an error")
	}
	fakeAlgoliaGitHub(t, refs("released-2024-09-19-6e91dbf55", "released-2024-09-19-a62c27d3b"), nil, nil)
	if _, err := (Algolia{Spec: "search"}).LatestTag(context.Background()); err == nil {
		t.Error("a same-day tie without commit dates: want an error, not a guess")
	}
}

// The version is the release's date; the YAML is converted to JSON.
func TestAlgoliaFetch(t *testing.T) {
	fakeAlgoliaGitHub(t, "", nil, map[string]string{
		"/released-2025-04-01-c19ee6afb/search.yml": "openapi: 3.0.2\ninfo:\n  title: Search API\npaths:\n  /1/clusters:\n    get:\n      deprecated: true\n",
	})
	spec, err := Algolia{Spec: "search"}.Fetch(context.Background(), "released-2025-04-01-c19ee6afb")
	if err != nil {
		t.Fatal(err)
	}
	want := `{"info":{"title":"Search API"},"openapi":"3.0.2","paths":{"/1/clusters":{"get":{"deprecated":true}}}}`
	if spec.Version != "2025-04-01" || string(spec.Content) != want || !strings.HasSuffix(spec.SourceURL, "/released-2025-04-01-c19ee6afb/search.yml") {
		t.Errorf("got %q %s %s", spec.Version, spec.Content, spec.SourceURL)
	}
	if (Algolia{Spec: "search"}).API() != "algolia-search" {
		t.Error("api slug")
	}
	for _, tag := range []string{"v4.0.0", "released-2025-04-01-c19ee6afb-x", "released-xx"} {
		if _, err := (Algolia{Spec: "search"}).Fetch(context.Background(), tag); err == nil {
			t.Errorf("%s: want an error", tag)
		}
	}
	fakeAlgoliaGitHub(t, "", nil, map[string]string{"/released-2025-04-01/search.yml": "not: [valid: yaml"})
	if _, err := (Algolia{Spec: "search"}).Fetch(context.Background(), "released-2025-04-01"); err == nil {
		t.Error("invalid yaml: want an error")
	}
}
