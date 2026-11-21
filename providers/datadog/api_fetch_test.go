package datadog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeDatadogGitHub serves a latest-release endpoint and specs by path, like
// fakeGitHub but under datadog's URL shape (no legacy fallback).
func fakeDatadogGitHub(t *testing.T, releaseJSON string, specs map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/releases/latest" {
			w.Write([]byte(releaseJSON))
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

	oldRel, oldSpec := datadogReleasesURL, datadogSpecURLTemplate
	datadogReleasesURL = srv.URL + "/releases/latest"
	datadogSpecURLTemplate = srv.URL + "/%s/.generator/schemas/v2/openapi.yaml"
	t.Cleanup(func() {
		datadogReleasesURL, datadogSpecURLTemplate = oldRel, oldSpec
	})
	return srv
}

func TestDatadogLatestTagReadsTheReleaseTag(t *testing.T) {
	fakeDatadogGitHub(t, `{"tag_name":"2.62.0"}`, nil)
	tag, err := Datadog{}.LatestTag(context.Background())
	if err != nil || tag != "2.62.0" {
		t.Errorf("got %q, %v; want 2.62.0", tag, err)
	}
}

func TestDatadogLatestTagRejectsAReleaseWithoutATag(t *testing.T) {
	fakeDatadogGitHub(t, `{"name":"something"}`, nil)
	if _, err := (Datadog{}).LatestTag(context.Background()); err == nil {
		t.Error("release without tag_name: want an error")
	}
}

// Fetch's whole job beyond downloading is converting YAML to JSON, since
// diff.Parse only reads JSON. The real spec's shape (a mapping with string
// keys throughout) is what this checks, not just that some conversion ran.
func TestDatadogFetchConvertsYAMLToJSON(t *testing.T) {
	srv := fakeDatadogGitHub(t, "", map[string]string{
		"/2.61.0/.generator/schemas/v2/openapi.yaml": "" +
			"openapi: 3.0.0\n" +
			"info:\n" +
			"  version: \"1.0\"\n" +
			"components:\n" +
			"  schemas:\n" +
			"    TagIndexingRuleDynamicTags:\n" +
			"      type: object\n" +
			"      properties:\n" +
			"        exclude_not_used_in_assets:\n" +
			"          type: boolean\n",
	})
	spec, err := Datadog{}.Fetch(context.Background(), "2.61.0")
	if err != nil {
		t.Fatal(err)
	}
	if spec.SourceURL != srv.URL+"/2.61.0/.generator/schemas/v2/openapi.yaml" {
		t.Errorf("source_url = %s, want the file actually fetched", spec.SourceURL)
	}
	want := `{"components":{"schemas":{"TagIndexingRuleDynamicTags":{"properties":{"exclude_not_used_in_assets":{"type":"boolean"}},"type":"object"}}},"info":{"version":"1.0"},"openapi":"3.0.0"}`
	if string(spec.Content) != want {
		t.Errorf("content = %s, want %s", spec.Content, want)
	}
}

// Unlike Stripe, Version comes from the release tag: info.version inside
// datadog's spec is a fixed "1.0" that never moves between releases (checked
// 2026-11-13), so it can't be what tells two releases of events apart.
func TestDatadogFetchUsesTheTagAsVersionNotInfoVersion(t *testing.T) {
	fakeDatadogGitHub(t, "", map[string]string{
		"/2.61.0/.generator/schemas/v2/openapi.yaml": "info:\n  version: \"1.0\"\n",
	})
	spec, err := Datadog{}.Fetch(context.Background(), "2.61.0")
	if err != nil {
		t.Fatal(err)
	}
	if spec.Version != "2.61.0" {
		t.Errorf("version = %q, want the tag 2.61.0 (info.version is always 1.0)", spec.Version)
	}
}

func TestDatadogFetchRejectsInvalidYAML(t *testing.T) {
	fakeDatadogGitHub(t, "", map[string]string{
		"/bad/.generator/schemas/v2/openapi.yaml": "not: [valid: yaml",
	})
	_, err := Datadog{}.Fetch(context.Background(), "bad")
	if err == nil || !strings.Contains(err.Error(), "yaml") {
		t.Errorf("got %v, want a yaml parse error", err)
	}
}

func TestDatadogFetchFailsWhenSpecDoesNotExist(t *testing.T) {
	fakeDatadogGitHub(t, "", nil)
	if _, err := (Datadog{}).Fetch(context.Background(), "9.9.9"); err == nil {
		t.Error("missing tag: want an error")
	}
}
