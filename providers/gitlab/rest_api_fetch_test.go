package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/openapitest"
)

// fakeGitLab serves a tags listing and specs by path, 404 for anything else.
func fakeGitLab(t *testing.T, tagsJSON string, specs map[string]string) GitLab {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tags" {
			w.Write([]byte(tagsJSON))
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
	return GitLab{TagsURL: srv.URL + "/tags", SpecURLTemplate: srv.URL + "/%s/openapi_v3.yaml"}
}

// The listing interleaves release candidates, the newest of which sorts first
// in GitLab's own order (seen 2026-11-15), and an older line's later patch.
func TestGitLabLatestTagIsTheHighestStableRelease(t *testing.T) {
	g := fakeGitLab(t, `[
		{"name":"v19.6.0-rc12-ee"},
		{"name":"v19.4.7-ee"},
		{"name":"v19.5.10-ee"},
		{"name":"v19.5.9-ee"},
		{"name":"v19.5.0-ee"},
		{"name":"v19.10.0-rc1-ee"},
		{"name":"v18.11.20-ee"},
		{"name":"v19.5.11"}
	]`, nil)
	tag, err := g.LatestTag(context.Background())
	if err != nil || tag != "v19.5.10-ee" {
		t.Errorf("got %q, %v; want v19.5.10-ee (numeric, not string, order; no rc; -ee only)", tag, err)
	}
}

func TestGitLabVersionIsTheBareReleaseVersion(t *testing.T) {
	for tag, want := range map[string]string{
		"v19.0.0-ee":  "19.0.0",
		"v18.11.2-ee": "18.11.2",
		"master":      "master", // not a release: its own name
	} {
		if got := gitlabVersion(tag); got != want {
			t.Errorf("gitlabVersion(%q) = %q, want %q", tag, got, want)
		}
	}
}

func TestGitLabLatestTagNeedsAStableRelease(t *testing.T) {
	g := fakeGitLab(t, `[{"name":"v19.6.0-rc1-ee"}]`, nil)
	if _, err := g.LatestTag(context.Background()); err == nil {
		t.Error("only release candidates: want an error")
	}
}

const gitlabSpecYAML = `openapi: 3.0.0
info:
  title: GitLab REST API
  version: v4
paths:
  "/api/v4/import/bitbucket":
    post:
      responses:
        201:
          description: Created
      requestBody:
        content:
          application/json:
            schema:
              "$ref": "#/components/schemas/RequestBody_f0e6af1da635"
  "/api/v4/a/b":
    put:
      requestBody:
        content:
          application/json:
            schema:
              "$ref": "#/components/schemas/RequestBody_f0e6af1da635"
components:
  schemas:
    APIEntitiesBadge:
      type: object
    RequestBody_f0e6af1da635:
      type: object
      properties:
        bitbucket_email:
          type: string
        owner:
          "$ref": "#/components/schemas/APIEntitiesBadge"
`

func TestGitLabFetchInlinesRequestBodiesAndUsesTheTag(t *testing.T) {
	g := fakeGitLab(t, "", map[string]string{"/v19.0.0-ee/openapi_v3.yaml": gitlabSpecYAML})
	spec, err := g.Fetch(context.Background(), "v19.0.0-ee")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if spec.Version != "19.0.0" {
		t.Errorf("version %q, want 19.0.0 from the tag, not info.version v4", spec.Version)
	}
	if !strings.HasSuffix(spec.SourceURL, "/v19.0.0-ee/openapi_v3.yaml") {
		t.Errorf("source url %q", spec.SourceURL)
	}

	var doc map[string]any
	if err := json.Unmarshal(spec.Content, &doc); err != nil {
		t.Fatalf("content isn't JSON: %v", err)
	}
	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	if _, ok := schemas["RequestBody_f0e6af1da635"]; ok {
		t.Error("the hashed body schema is still in components")
	}
	if _, ok := schemas["APIEntitiesBadge"]; !ok {
		t.Error("a named schema was dropped")
	}
	for _, path := range []string{"/api/v4/import/bitbucket", "/api/v4/a/b"} {
		var op map[string]any
		for _, o := range doc["paths"].(map[string]any)[path].(map[string]any) {
			op = o.(map[string]any)
		}
		body := op["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
		props, _ := body["properties"].(map[string]any)
		if props["bitbucket_email"] == nil {
			t.Errorf("%s: body not inlined: %v", path, body)
			continue
		}
		// One level only: a ref inside the body stays a ref.
		if ref := props["owner"].(map[string]any)["$ref"]; ref != "#/components/schemas/APIEntitiesBadge" {
			t.Errorf("%s: nested ref became %v", path, ref)
		}
	}
	// An unquoted response code (201:) still converts.
	if !strings.Contains(string(spec.Content), `"201":`) {
		t.Error("the integer response code key was lost")
	}
}

// Two operations sharing a body get copies, so nothing that edits one
// operation's node can change the other's.
func TestGitLabFetchCopiesASharedBody(t *testing.T) {
	g := fakeGitLab(t, "", map[string]string{"/v1/openapi_v3.yaml": gitlabSpecYAML})
	spec, err := g.Fetch(context.Background(), "v1")
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(spec.Content), `"bitbucket_email"`); n != 2 {
		t.Errorf("bitbucket_email appears %d times, want once per operation", n)
	}
}

func TestGitLabSourcePointerPointsBackIntoThePublishedFile(t *testing.T) {
	g := fakeGitLab(t, "", map[string]string{"/v1/openapi_v3.yaml": gitlabSpecYAML})
	spec, err := g.Fetch(context.Background(), "v1")
	if err != nil {
		t.Fatal(err)
	}
	ptr := g.SourcePointer(spec.Content)
	body := "/paths/~1api~1v4~1import~1bitbucket/post/requestBody/content/application~1json/schema"
	cases := map[string]string{
		body + "/properties/bitbucket_email": "/components/schemas/RequestBody_f0e6af1da635/properties/bitbucket_email",
		body:                                 "/components/schemas/RequestBody_f0e6af1da635",
		// Not inside an inlined body: unchanged.
		body + "X": body + "X",
		"/paths/~1api~1v4~1import~1bitbucket/post": "/paths/~1api~1v4~1import~1bitbucket/post",
		"/components/schemas/APIEntitiesBadge":     "/components/schemas/APIEntitiesBadge",
	}
	for in, want := range cases {
		if got := ptr(in); got != want {
			t.Errorf("SourcePointer(%s) = %s, want %s", in, got, want)
		}
	}
	// A spec stored before this mapping existed, or unreadable, maps nothing.
	if got := g.SourcePointer([]byte("not json"))(body); got != body {
		t.Errorf("unreadable content changed the pointer to %s", got)
	}
}

func TestGitLabFetchRejectsBadSpecs(t *testing.T) {
	g := fakeGitLab(t, "", map[string]string{
		"/bad/openapi_v3.yaml":    "paths: [unclosed",
		"/scalar/openapi_v3.yaml": "just a string",
	})
	for _, tag := range []string{"bad", "scalar", "missing"} {
		if _, err := g.Fetch(context.Background(), tag); err == nil {
			t.Errorf("%s: want an error", tag)
		}
	}
}

// Every stable release from v18.10.0-ee (the first with openapi_v3.yaml), in
// version order, release candidates left out; the listing is read a second
// page, and no further once it passes v18.10.0-ee.
func TestGitLabListsEveryReleaseWithTheSpec(t *testing.T) {
	srv := openapitest.Recorded(t, map[string]string{
		"/tags?order_by=version&sort=desc&per_page=100&page=1": "gitlab-tags-1.json",
		"/tags?order_by=version&sort=desc&per_page=100&page=2": "gitlab-tags-2.json",
	})
	vs, err := GitLab{TagsURL: srv.URL + "/tags?order_by=version&sort=desc&per_page=100"}.Versions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if vs[0].ID != "18.10.0" || vs[0].Ref != "v18.10.0-ee" || vs[len(vs)-1].ID != "19.6.1" {
		t.Errorf("first %v, last %v", vs[0], vs[len(vs)-1])
	}
	seen := map[string]bool{}
	for _, v := range vs {
		if strings.Contains(v.Ref, "rc") || v.Released.IsZero() || seen[v.ID] {
			t.Errorf("version %v", v)
		}
		seen[v.ID] = true
	}
	at := func(id string) int { return slices.IndexFunc(vs, func(v plugin.Version) bool { return v.ID == id }) }
	if !(at("18.11.14") < at("19.0.0") && at("19.0.0") < at("19.6.0") && at("18.10.1") > at("18.10.0")) {
		t.Errorf("not in version order: %s", openapitest.IDs(vs))
	}
}
