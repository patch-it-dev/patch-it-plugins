package adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeConvexGitHub serves the commits list, one commit, and specs by path.
func fakeConvexGitHub(t *testing.T, commitsJSON string, commits, specs map[string]string) *httptest.Server {
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
	oldList, oldOne, oldSpec := convexCommitsURL, convexCommitURL, convexSpecURLTemplate
	convexCommitsURL = srv.URL + "/commits"
	convexCommitURL = srv.URL + "/commits/%s"
	convexSpecURLTemplate = srv.URL + "/%s/spec.json"
	t.Cleanup(func() { convexCommitsURL, convexCommitURL, convexSpecURLTemplate = oldList, oldOne, oldSpec })
	return srv
}

const convexSHA = "5e5159cbea5188e9da5400febda1b031a233eea0"

func TestConvexLatestTagIsTheNewestSpecCommit(t *testing.T) {
	fakeConvexGitHub(t, `[{"sha":"`+convexSHA+`"},{"sha":"older"}]`, nil, nil)
	tag, err := Convex{}.LatestTag(context.Background())
	if err != nil || tag != convexSHA {
		t.Errorf("got %q, %v; want %s", tag, err, convexSHA)
	}
}

func TestConvexLatestTagRejectsNoCommit(t *testing.T) {
	fakeConvexGitHub(t, `[]`, nil, nil)
	if _, err := (Convex{}).LatestTag(context.Background()); err == nil {
		t.Error("no commit: want an error")
	}
}

// The version is the commit's committer date, in UTC, since the Deployment
// API has none (its info.version is a fixed 1.0.0); the source is pinned to
// the full SHA even when a short one is asked for, and the content is passed
// through.
func TestConvexFetchPinsTheFullSHAAndDatesByCommit(t *testing.T) {
	const body = `{"openapi":"3.1.0","info":{"version":"1.0.0"},"paths":{}}`
	srv := fakeConvexGitHub(t, "", map[string]string{
		"5e5159cbea": `{"sha":"` + convexSHA + `","commit":{"committer":{"date":"2026-11-05T19:36:44-08:00"}}}`,
	}, map[string]string{"/" + convexSHA + "/spec.json": body})
	spec, err := Convex{}.Fetch(context.Background(), "5e5159cbea")
	if err != nil {
		t.Fatal(err)
	}
	if spec.Version != "2026-11-06" || spec.SourceURL != srv.URL+"/"+convexSHA+"/spec.json" || string(spec.Content) != body {
		t.Errorf("got version %q, url %s, content %s", spec.Version, spec.SourceURL, spec.Content)
	}
}

func TestConvexFetchFailures(t *testing.T) {
	commit := `{"sha":"` + convexSHA + `","commit":{"committer":{"date":"2026-11-05T19:36:44Z"}}}`
	for name, c := range map[string]struct{ commits, specs map[string]string }{
		"unknown commit":      {nil, nil},
		"commit without date": {map[string]string{"x": `{"sha":"` + convexSHA + `"}`}, nil},
		"spec missing":        {map[string]string{"x": commit}, nil},
		"invalid json":        {map[string]string{"x": commit}, map[string]string{"/" + convexSHA + "/spec.json": "<html>"}},
	} {
		fakeConvexGitHub(t, "", c.commits, c.specs)
		if _, err := (Convex{}).Fetch(context.Background(), "x"); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}
