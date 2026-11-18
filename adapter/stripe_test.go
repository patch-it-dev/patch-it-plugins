package adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeGitHub serves a latest-release endpoint and specs by path. Paths missing
// from specs return 404, like raw.githubusercontent.com does.
func fakeGitHub(t *testing.T, releaseJSON string, specs map[string]string) *httptest.Server {
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

	oldRel, oldSpec, oldLegacy := stripeReleasesURL, stripeSpecURLTemplate, stripeLegacySpecURLTemplate
	stripeReleasesURL = srv.URL + "/releases/latest"
	stripeSpecURLTemplate = srv.URL + "/%s/latest/openapi.spec3.json"
	stripeLegacySpecURLTemplate = srv.URL + "/%s/openapi/spec3.json"
	t.Cleanup(func() {
		stripeReleasesURL, stripeSpecURLTemplate, stripeLegacySpecURLTemplate = oldRel, oldSpec, oldLegacy
	})
	return srv
}

func TestLatestTagReadsTheReleaseTag(t *testing.T) {
	fakeGitHub(t, `{"tag_name":"v2484"}`, nil)
	tag, err := Stripe{}.LatestTag(context.Background())
	if err != nil || tag != "v2484" {
		t.Errorf("got %q, %v; want v2484", tag, err)
	}
}

func TestLatestTagRejectsAReleaseWithoutATag(t *testing.T) {
	fakeGitHub(t, `{"name":"something"}`, nil)
	if _, err := (Stripe{}).LatestTag(context.Background()); err == nil {
		t.Error("release without tag_name: want an error")
	}
}

func TestFetchUsesTheRecommendedSpec(t *testing.T) {
	srv := fakeGitHub(t, "", map[string]string{
		"/v2468/latest/openapi.spec3.json": `{"info":{"version":"2026-08-26.dahlia"}}`,
		"/v2468/openapi/spec3.json":        `{"info":{"version":"legacy-should-not-be-used"}}`,
	})
	spec, err := Stripe{}.Fetch(context.Background(), "v2468")
	if err != nil {
		t.Fatal(err)
	}
	if spec.Version != "2026-08-26.dahlia" || spec.SourceURL != srv.URL+"/v2468/latest/openapi.spec3.json" {
		t.Errorf("got version %q from %s; want the latest/ file", spec.Version, spec.SourceURL)
	}
}

// The demo change lives on tag v207, which predates Stripe's latest/ directory.
// Without this fallback the demo couldn't be fetched at all, and the event's
// provenance must name the file that was actually used.
func TestFetchFallsBackToTheLegacySpecForOldTags(t *testing.T) {
	srv := fakeGitHub(t, "", map[string]string{
		"/v207/openapi/spec3.json": `{"info":{"version":"2022-08-01"}}`,
	})
	spec, err := Stripe{}.Fetch(context.Background(), "v207")
	if err != nil {
		t.Fatalf("old tag: %v", err)
	}
	if spec.Version != "2022-08-01" {
		t.Errorf("version %q, want 2022-08-01", spec.Version)
	}
	if spec.SourceURL != srv.URL+"/v207/openapi/spec3.json" {
		t.Errorf("source_url %s, want the legacy file that was actually fetched", spec.SourceURL)
	}
}

func TestFetchRejectsASpecWithoutAVersion(t *testing.T) {
	fakeGitHub(t, "", map[string]string{"/v1/latest/openapi.spec3.json": `{"info":{}}`})
	_, err := Stripe{}.Fetch(context.Background(), "v1")
	if err == nil || !strings.Contains(err.Error(), "info.version") {
		t.Errorf("got %v, want an info.version error", err)
	}
}

func TestFetchFailsWhenNeitherSpecExists(t *testing.T) {
	fakeGitHub(t, "", nil)
	if _, err := (Stripe{}).Fetch(context.Background(), "v999999"); err == nil {
		t.Error("missing tag: want an error")
	}
}
