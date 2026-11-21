package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeNeonGitHub serves the commits list, one commit, and specs by path.
func fakeNeonGitHub(t *testing.T, commitsJSON string, commits, specs map[string]string) *httptest.Server {
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
	oldList, oldOne, oldSpec := neonCommitsURL, neonCommitURL, neonSpecURLTemplate
	neonCommitsURL = srv.URL + "/commits"
	neonCommitURL = srv.URL + "/commits/%s"
	neonSpecURLTemplate = srv.URL + "/%s/spec.json"
	t.Cleanup(func() { neonCommitsURL, neonCommitURL, neonSpecURLTemplate = oldList, oldOne, oldSpec })
	return srv
}

const neonSHA = "93b93dc9e4d1b4ab48dac6c8ccba76b35541b476"

func TestNeonLatestTagIsTheNewestSpecCommit(t *testing.T) {
	fakeNeonGitHub(t, `[{"sha":"`+neonSHA+`"},{"sha":"older"}]`, nil, nil)
	tag, err := Neon{}.LatestTag(context.Background())
	if err != nil || tag != neonSHA {
		t.Errorf("got %q, %v; want %s", tag, err, neonSHA)
	}
}

func TestNeonLatestTagRejectsNoCommit(t *testing.T) {
	fakeNeonGitHub(t, `[]`, nil, nil)
	if _, err := (Neon{}).LatestTag(context.Background()); err == nil {
		t.Error("no commit: want an error")
	}
}

// The version is the commit's date, since the API has none (its
// info.version is a fixed v2); the source is pinned to the full SHA even
// when a short one is asked for, and a spec whose server has no path is
// passed through byte for byte.
func TestNeonFetchPinsTheFullSHAAndDatesByCommit(t *testing.T) {
	const body = `{"openapi":"3.0.3","info":{"version":"v2"},"paths":{}}`
	srv := fakeNeonGitHub(t, "", map[string]string{
		"93b93dc9": `{"sha":"` + neonSHA + `","commit":{"committer":{"date":"2026-08-23T11:50:46Z"}}}`,
	}, map[string]string{"/" + neonSHA + "/spec.json": body})
	spec, err := Neon{}.Fetch(context.Background(), "93b93dc9")
	if err != nil {
		t.Fatal(err)
	}
	if spec.Version != "2026-08-23" || spec.SourceURL != srv.URL+"/"+neonSHA+"/spec.json" || string(spec.Content) != body {
		t.Errorf("got version %q, url %s, content %s", spec.Version, spec.SourceURL, spec.Content)
	}
}

func TestNeonFetchFailures(t *testing.T) {
	commit := `{"sha":"` + neonSHA + `","commit":{"committer":{"date":"2026-08-23T11:50:46Z"}}}`
	for name, c := range map[string]struct{ commits, specs map[string]string }{
		"unknown commit":      {nil, nil},
		"commit without date": {map[string]string{"x": `{"sha":"` + neonSHA + `"}`}, nil},
		"spec missing":        {map[string]string{"x": commit}, nil},
		"invalid json":        {map[string]string{"x": commit}, map[string]string{"/" + neonSHA + "/spec.json": "<html>"}},
	} {
		fakeNeonGitHub(t, "", c.commits, c.specs)
		if _, err := (Neon{}).Fetch(context.Background(), "x"); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

// Neon's server is "https://console.neon.tech/api/v2": Fetch moves /api/v2
// into the paths, so an operation is named by the path a caller sends, and
// SourcePointer points back into the path as published.
func TestNeonFetchMovesTheServerPathIntoThePaths(t *testing.T) {
	const body = `{"openapi":"3.0.3","servers":[{"url":"https://console.neon.tech/api/v2"}],"paths":{"/projects":{"get":{}}}}`
	fakeNeonGitHub(t, "", map[string]string{
		"93b93dc9": `{"sha":"` + neonSHA + `","commit":{"committer":{"date":"2026-08-23T11:50:46Z"}}}`,
	}, map[string]string{"/" + neonSHA + "/spec.json": body})
	spec, err := Neon{}.Fetch(context.Background(), "93b93dc9")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Paths map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(spec.Content, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc.Paths["/api/v2/projects"]; !ok || len(doc.Paths) != 1 {
		t.Errorf("paths %v, want only /api/v2/projects", doc.Paths)
	}
	if got := (Neon{}).SourcePointer(spec.Content)("/paths/~1api~1v2~1projects/get"); got != "/paths/~1projects/get" {
		t.Errorf("SourcePointer = %s, want /paths/~1projects/get", got)
	}
}
