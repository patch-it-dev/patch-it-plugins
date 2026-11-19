package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeSentryGitHub serves GitHub's API, release downloads and raw files from
// routes: a request path (decoded) to its body, under /api, /dl and /raw.
// Anything else is a 404. It returns the adapter pointed at it.
func fakeSentryGitHub(t *testing.T, routes map[string]string) (Sentry, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return Sentry{APIBase: srv.URL + "/api", DownloadBase: srv.URL + "/dl", RawBase: srv.URL + "/raw"}, srv.URL
}

const (
	sentryReleases = "/api/repos/getsentry/sentry/releases"
	withSpec       = `"assets":[{"name":"sentry-api.tgz"},{"name":"openapi-derefed.json"}]`
)

// The highest api@ version, not the newest by date (api@2.9.0 was published
// after api@2.12.0), and never the server's releases, a draft, a prerelease
// or a release without the spec.
func TestSentryLatestTagIsTheHighestAPIRelease(t *testing.T) {
	s, _ := fakeSentryGitHub(t, map[string]string{sentryReleases: `[
		{"tag_name":"api@3.0.0","prerelease":true,` + withSpec + `},
		{"tag_name":"api@2.17.0","draft":true,` + withSpec + `},
		{"tag_name":"api@2.16.1","assets":[{"name":"sentry-api.tgz"}]},
		{"tag_name":"api@2.9.0",` + withSpec + `},
		{"tag_name":"26.11.0","assets":[]},
		{"tag_name":"scraps@9.0.0",` + withSpec + `},
		{"tag_name":"api@2.12.0",` + withSpec + `},
		{"tag_name":"api@2.10.1",` + withSpec + `}]`})
	tag, err := s.LatestTag(context.Background())
	if err != nil || tag != "api@2.12.0" {
		t.Errorf("got %q, %v; want api@2.12.0", tag, err)
	}
	s, _ = fakeSentryGitHub(t, map[string]string{sentryReleases: `[{"tag_name":"26.11.0","assets":[]}]`})
	if _, err := s.LatestTag(context.Background()); err == nil {
		t.Error("no api@ release: want an error")
	}
}

// A spec as published from sentry-api-schema 2.1.0 on: the server carries
// /api/0, the paths don't.
const sentryNewSpec = `{"openapi":"3.0.3","info":{"version":"v0"},
	"servers":[{"url":"https://{region}.sentry.io/api/0","variables":{"region":{"default":"us"}}}],
	"paths":{"/":{"get":{"operationId":"index"}},"/organizations/{organization_id_or_slug}/":{"get":{"x-big":12345678901234567890}}}}`

// A spec as published before 2.1.0: /api/0 in every path.
const sentryOldSpec = `{"openapi":"3.0.3","info":{"version":"v0"},
	"servers":[{"url":"https://{region}.sentry.io"}],
	"paths":{"/api/0/":{"get":{"operationId":"index"}}}}`

// An api@ release: the asset, cited by its download URL (the tag's @ spelled
// %40, as GitHub's own links do), dated by the release's publication, with
// /api/0 moved from the server into the paths.
func TestSentryFetchRelease(t *testing.T) {
	s, base := fakeSentryGitHub(t, map[string]string{
		sentryReleases + "/tags/api@2.16.0":                                `{"tag_name":"api@2.16.0","published_at":"2026-11-18T20:55:52Z",` + withSpec + `}`,
		"/dl/getsentry/sentry/releases/download/api@2.16.0/" + sentryAsset: sentryNewSpec,
	})
	spec, err := s.Fetch(context.Background(), "api@2.16.0")
	if err != nil {
		t.Fatal(err)
	}
	if spec.Version != "2026-11-18" || spec.SourceURL != base+"/dl/getsentry/sentry/releases/download/api%402.16.0/openapi-derefed.json" {
		t.Errorf("got %q %s", spec.Version, spec.SourceURL)
	}
	var doc struct {
		Paths map[string]map[string]map[string]json.RawMessage `json:"paths"`
		Base  string                                           `json:"x-patch-it-base-path"`
	}
	if err := json.Unmarshal(spec.Content, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc.Paths["/api/0/organizations/{organization_id_or_slug}/"]; !ok || len(doc.Paths) != 2 || doc.Base != "/api/0" {
		t.Errorf("paths %v, base %q: want /api/0 moved into every path", doc.Paths, doc.Base)
	}
	if _, ok := doc.Paths["/api/0/"]; !ok {
		t.Error(`"/" should become "/api/0/"`)
	}
	// Values are unchanged: a number too big for a float64 keeps its digits.
	if got := string(doc.Paths["/api/0/organizations/{organization_id_or_slug}/"]["get"]["x-big"]); got != "12345678901234567890" {
		t.Errorf("x-big is %s", got)
	}

	ptr := s.SourcePointer(spec.Content)
	for in, want := range map[string]string{
		"/paths/~1api~10~1organizations~1{organization_id_or_slug}~1/get": "/paths/~1organizations~1{organization_id_or_slug}~1/get",
		"/paths/~1api~10~1/get/description":                               "/paths/~1/get/description",
		"/components/schemas/X":                                           "/components/schemas/X",
		"/paths/~1api~10x/get":                                            "/paths/~1api~10x/get",
	} {
		if got := ptr(in); got != want {
			t.Errorf("SourcePointer(%s) = %s, want %s", in, got, want)
		}
	}

	s, _ = fakeSentryGitHub(t, map[string]string{sentryReleases + "/tags/api@2.16.0": `{"tag_name":"api@2.16.0","published_at":"2026-11-18T20:55:52Z","assets":[]}`})
	if _, err := s.Fetch(context.Background(), "api@2.16.0"); err == nil {
		t.Error("a release without the spec: want an error")
	}
}

// A sentry-api-schema tag: the file at the tag, dated by the tag's release.
// Its server has no path, so the spec is returned byte for byte and every
// pointer stays as it is. A commit, for the history before the first tag, is
// dated by the commit.
func TestSentryFetchSchemaTagOrCommit(t *testing.T) {
	sha := strings.Repeat("a", 40)
	const schema = "/api/repos/getsentry/sentry-api-schema"
	s, base := fakeSentryGitHub(t, map[string]string{
		schema + "/releases/tags/0.224.0":                             `{"tag_name":"0.224.0","published_at":"2026-06-16T23:16:45Z"}`,
		"/raw/getsentry/sentry-api-schema/0.224.0/" + sentryAsset:     sentryOldSpec,
		schema + "/releases/tags/0.225.0":                             `{"tag_name":"0.225.0","published_at":"2026-06-17T01:00:00Z"}`,
		"/raw/getsentry/sentry-api-schema/0.225.0/" + sentryAsset:     `<<<<<<< HEAD`,
		schema + "/commits/" + sha:                                    `{"sha":"` + sha + `","commit":{"committer":{"date":"2026-01-22T18:02:11Z"}}}`,
		"/raw/getsentry/sentry-api-schema/" + sha + "/" + sentryAsset: sentryOldSpec,
	})
	spec, err := s.Fetch(context.Background(), "0.224.0")
	if err != nil {
		t.Fatal(err)
	}
	if spec.Version != "2026-06-16" || spec.SourceURL != base+"/raw/getsentry/sentry-api-schema/0.224.0/openapi-derefed.json" || string(spec.Content) != sentryOldSpec {
		t.Errorf("got %q %s %.40s", spec.Version, spec.SourceURL, spec.Content)
	}
	if got := s.SourcePointer(spec.Content)("/paths/~1api~10~1/get"); got != "/paths/~1api~10~1/get" {
		t.Errorf("SourcePointer changed a pointer into a spec Fetch didn't change: %s", got)
	}

	spec, err = s.Fetch(context.Background(), sha)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Version != "2026-01-22" || spec.SourceURL != base+"/raw/getsentry/sentry-api-schema/"+sha+"/openapi-derefed.json" {
		t.Errorf("got %q %s", spec.Version, spec.SourceURL)
	}

	// One commit in the history has merge-conflict markers in the spec.
	if _, err := s.Fetch(context.Background(), "0.225.0"); err == nil {
		t.Error("invalid json: want an error")
	}
	for _, tag := range []string{"main", "v2.16.0", "api@2.16", "aaaaaaa", "2.16.0-rc1", "0.226.0"} {
		if _, err := s.Fetch(context.Background(), tag); err == nil {
			t.Errorf("%s: want an error", tag)
		}
	}
}

func TestSentryBasePathRefusesWhatItCantMove(t *testing.T) {
	if _, err := sentryBasePath([]byte(`{"servers":[{"url":"https://x/api"}],"paths":{"/a":{},"a":{}}}`)); err == nil {
		t.Error("a path without a leading /: want an error")
	}
	if _, err := sentryBasePath([]byte(`{"servers":[{"url":"https://x/api/0"},{"url":"https://y"}],"paths":{}}`)); err == nil {
		t.Error("servers with different paths: want an error")
	}
}

func TestServerPath(t *testing.T) {
	for in, want := range map[string]string{
		"https://{region}.sentry.io/api/0":  "/api/0",
		"https://{region}.sentry.io/api/0/": "/api/0",
		"https://{region}.sentry.io":        "",
		"https://sentry.io/":                "",
		"/api/0":                            "",
	} {
		if got := serverPath(in); got != want {
			t.Errorf("serverPath(%q) = %q, want %q", in, got, want)
		}
	}
}
