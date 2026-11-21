package stripe

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/openapitest"
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

	oldRel, oldSpec := stripeReleasesURL, stripeSpecURLTemplate
	stripeReleasesURL = srv.URL + "/releases/latest"
	stripeSpecURLTemplate = srv.URL + "/%s/latest/openapi.spec3.json"
	t.Cleanup(func() { stripeReleasesURL, stripeSpecURLTemplate = oldRel, oldSpec })
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
	})
	spec, err := Stripe{}.Fetch(context.Background(), "v2468")
	if err != nil {
		t.Fatal(err)
	}
	if spec.Version != "2026-08-26.dahlia" || spec.SourceURL != srv.URL+"/v2468/latest/openapi.spec3.json" {
		t.Errorf("got version %q from %s; want the latest/ file", spec.Version, spec.SourceURL)
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

// Stripe's versions are its API versions, as each release's notes name
// them: one per version, oldest first, read at the newest release carrying
// it and dated by the first; v2190, whose notes name none, is left out.
func TestStripeListsAPIVersions(t *testing.T) {
	srv := openapitest.Recorded(t, map[string]string{"/repos/stripe/openapi/releases?per_page=100&page=1": "stripe-openapi-releases.json"})
	old := versions.GitHubAPI
	versions.GitHubAPI = srv.URL
	t.Cleanup(func() { versions.GitHubAPI = old })
	vs, err := Stripe{}.Versions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := "2026-02-25.clover@v2203 2026-03-25.dahlia@v2248 2026-04-22.dahlia@v2275 2026-05-27.dahlia@v2323 2026-06-24.dahlia@v2348 " +
		"2026-07-29.dahlia@v2441 2026-08-26.dahlia@v2516 2026-09-30.endive@v2592 2026-10-28.endive@v2600 2026-11-18.endive@v2602"
	if got := openapitest.IDs(vs); got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	last := vs[len(vs)-1]
	if !last.Released.Equal(time.Date(2026, 11, 17, 22, 34, 17, 0, time.UTC)) || last.URL != "https://github.com/stripe/openapi/releases/tag/v2602" {
		t.Errorf("newest: released %s (want v2601's date), url %s", last.Released, last.URL)
	}
}

// The plugin lists the API versions the releases' notes name, through
// core/versions.
func TestThePluginListsStripesAPIVersions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/stripe/openapi/releases" {
			fmt.Fprint(w, `[{"tag_name":"v2","published_at":"2026-02-01T00:00:00Z","body":"| openapi/spec3.sdk.{json,yaml} | 2026-02-01.b |"},`+
				`{"tag_name":"v1","published_at":"2026-01-01T00:00:00Z","body":"| openapi/spec3.sdk.{json,yaml} | 2026-01-01.a |"}]`)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	old := versions.GitHubAPI
	versions.GitHubAPI = srv.URL
	defer func() { versions.GitHubAPI = old }()
	vs, err := openapi.Plugin{Adapter: Stripe{}}.ListVersions(context.Background(), plugin.Env{})
	if err != nil {
		t.Fatal(err)
	}
	var refs []string
	for _, v := range vs {
		refs = append(refs, v.Ref)
	}
	if strings.Join(refs, " ") != "v1 v2" {
		t.Errorf("refs %v, want v1 v2", refs)
	}
}
