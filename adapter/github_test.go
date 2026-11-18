package adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeGitHubVersions serves /versions and spec files by path, like
// fakeDatadogGitHub but under github's own URL shape: no release tag, just a
// list of supported dated API versions.
func fakeGitHubVersions(t *testing.T, versionsJSON string, specs map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/versions" {
			w.Write([]byte(versionsJSON))
			return
		}
		body, ok := specs[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	oldVersions, oldSpec := githubVersionsURL, githubSpecURLTemplate
	githubVersionsURL = srv.URL + "/versions"
	githubSpecURLTemplate = srv.URL + "/%s.json"
	t.Cleanup(func() {
		githubVersionsURL, githubSpecURLTemplate = oldVersions, oldSpec
	})
	return srv
}

func TestGitHubLatestTagPicksTheNewestVersion(t *testing.T) {
	fakeGitHubVersions(t, `["2026-03-10","2022-11-28"]`, nil)
	tag, err := GitHub{}.LatestTag(context.Background())
	if err != nil || tag != "2026-03-10" {
		t.Errorf("got %q, %v; want 2026-03-10", tag, err)
	}
}

// GitHub's own docs don't promise an order, so this must not depend on one.
func TestGitHubLatestTagDoesNotRelyOnResponseOrder(t *testing.T) {
	fakeGitHubVersions(t, `["2022-11-28","2026-03-10"]`, nil)
	tag, err := GitHub{}.LatestTag(context.Background())
	if err != nil || tag != "2026-03-10" {
		t.Errorf("got %q, %v; want 2026-03-10 regardless of response order", tag, err)
	}
}

func TestGitHubLatestTagRejectsAnEmptyList(t *testing.T) {
	fakeGitHubVersions(t, `[]`, nil)
	if _, err := (GitHub{}).LatestTag(context.Background()); err == nil {
		t.Error("empty versions list: want an error")
	}
}

// Fetch's whole job is downloading: unlike Datadog's bundled YAML, GitHub's
// spec is already JSON, so Content is the response body verbatim.
func TestGitHubFetchReturnsTheSpecVerbatim(t *testing.T) {
	spec := `{"openapi":"3.0.3","info":{"version":"1.1.4"},"components":{"schemas":{}}}`
	srv := fakeGitHubVersions(t, "", map[string]string{
		"/2026-03-10.json": spec,
	})
	got, err := GitHub{}.Fetch(context.Background(), "2026-03-10")
	if err != nil {
		t.Fatal(err)
	}
	if got.SourceURL != srv.URL+"/2026-03-10.json" {
		t.Errorf("source_url = %s, want the file actually fetched", got.SourceURL)
	}
	if string(got.Content) != spec {
		t.Errorf("content = %s, want %s", got.Content, spec)
	}
}

// Unlike Stripe, Version comes from the requested dated version, not
// info.version: info.version inside github's spec is a fixed package version
// ("1.1.4" at every tag sampled, checked 2026-11-14), so it can't tell two API
// versions apart.
func TestGitHubFetchUsesTheRequestedTagAsVersionNotInfoVersion(t *testing.T) {
	fakeGitHubVersions(t, "", map[string]string{
		"/2026-03-10.json": `{"info":{"version":"1.1.4"}}`,
	})
	spec, err := GitHub{}.Fetch(context.Background(), "2026-03-10")
	if err != nil {
		t.Fatal(err)
	}
	if spec.Version != "2026-03-10" {
		t.Errorf("version = %q, want the requested tag 2026-03-10 (info.version is always 1.1.4)", spec.Version)
	}
}

func TestGitHubFetchRejectsInvalidJSON(t *testing.T) {
	fakeGitHubVersions(t, "", map[string]string{
		"/bad.json": "not json",
	})
	_, err := GitHub{}.Fetch(context.Background(), "bad")
	if err == nil || !strings.Contains(err.Error(), "json") {
		t.Errorf("got %v, want a json error", err)
	}
}

func TestGitHubFetchFailsWhenSpecDoesNotExist(t *testing.T) {
	fakeGitHubVersions(t, "", nil)
	if _, err := (GitHub{}).Fetch(context.Background(), "1999-01-01"); err == nil {
		t.Error("missing version: want an error")
	}
}
