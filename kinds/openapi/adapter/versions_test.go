package adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
)

// recorded serves saved API responses (testdata/versions/, recorded
// 2026-11-20 and trimmed: testdata/README.md) by request path and query;
// any other request fails the test.
func recorded(t *testing.T, files map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path
		if r.URL.RawQuery != "" {
			key += "?" + r.URL.RawQuery
		}
		file, ok := files[key]
		if !ok {
			t.Errorf("unexpected request %s", key)
			http.NotFound(w, r)
			return
		}
		b, err := os.ReadFile("testdata/versions/" + file)
		if err != nil {
			t.Error(err)
		}
		w.Write(b)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func ids(vs []plugin.Version) string {
	var out []string
	for _, v := range vs {
		out = append(out, v.ID+"@"+v.Ref)
	}
	return strings.Join(out, " ")
}

func TestGitHubListsEverySupportedVersion(t *testing.T) {
	srv := recorded(t, map[string]string{"/versions": "github-versions.json"})
	old := githubVersionsURL
	githubVersionsURL = srv.URL + "/versions"
	t.Cleanup(func() { githubVersionsURL = old })
	vs, err := GitHub{}.Versions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(vs); got != "2022-11-28@2022-11-28 2026-03-10@2026-03-10" {
		t.Errorf("versions %s", got)
	}
}

// Every stable release from v18.10.0-ee (the first with openapi_v3.yaml), in
// version order, release candidates left out; the listing is read a second
// page, and no further once it passes v18.10.0-ee.
func TestGitLabListsEveryReleaseWithTheSpec(t *testing.T) {
	srv := recorded(t, map[string]string{
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
		t.Errorf("not in version order: %s", ids(vs))
	}
}

// Every released minor from 8.14 (the first with the OpenAPI output) to the
// newest released one, 9.5: not 9.6, the next minor's branch.
func TestElasticsearchListsEveryReleasedMinor(t *testing.T) {
	srv := recorded(t, map[string]string{"/releases": "es-releases.json", "/branches": "es-branches.json"})
	a, b := esReleasesURL, esBranchesURL
	esReleasesURL, esBranchesURL = srv.URL+"/releases", srv.URL+"/branches"
	t.Cleanup(func() { esReleasesURL, esBranchesURL = a, b })
	vs, err := Elasticsearch{}.Versions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := "8.14.0@8.14 8.15.0@8.15 8.16.0@8.16 8.17.0@8.17 8.18.0@8.18 8.19.0@8.19 9.0.0@9.0 9.1.0@9.1 9.2.0@9.2 9.3.0@9.3 9.4.0@9.4 9.5.0@9.5"
	if got := ids(vs); got != want {
		t.Errorf("versions\n %s\nwant\n %s", got, want)
	}
}

// The archived schema repo's tags, then getsentry/sentry's api@ releases,
// by date; the IDs are dates, as Fetch's Version.
func TestSentryListsBothRepos(t *testing.T) {
	srv := recorded(t, map[string]string{
		"/repos/getsentry/sentry-api-schema/releases?per_page=100&page=1": "sentry-schema-releases.json",
		"/repos/getsentry/sentry/releases?per_page=100&page=1":            "sentry-api-releases.json",
	})
	vs, err := Sentry{APIBase: srv.URL}.Versions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 21 || vs[0].Ref != "2.3.2" || vs[1].Ref != "2.4.0" || vs[2].Ref != "api@2.5.0" || vs[len(vs)-1].Ref != "api@2.16.0" {
		t.Fatalf("versions %s", ids(vs))
	}
	for i := 1; i < len(vs); i++ {
		if vs[i].Released.Before(vs[i-1].Released) || vs[i].ID != vs[i].Released.Format("2006-01-02") {
			t.Errorf("%v after %v", vs[i], vs[i-1])
		}
	}
}

// Every release tag, oldest first; the five tags of 2024-09-18 and -19 are
// ordered by their commits' dates, not their names.
func TestAlgoliaListsEveryReleaseTag(t *testing.T) {
	files := map[string]string{"/refs": "algolia-refs.json"}
	for _, tag := range []string{"2024-09-18-45edcd844", "2024-09-18-cc190fae3", "2024-09-19-6e91dbf55", "2024-09-19-a62c27d3b", "2024-09-19-e47df6533"} {
		files["/commits/released-"+tag] = "algolia-commit-released-" + tag + ".json"
	}
	srv := recorded(t, files)
	a, b := algoliaRefsURL, algoliaCommitURL
	algoliaRefsURL, algoliaCommitURL = srv.URL+"/refs", srv.URL+"/commits/%s"
	t.Cleanup(func() { algoliaRefsURL, algoliaCommitURL = a, b })
	vs, err := Algolia{Spec: "search"}.Versions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var refs []string
	for _, v := range vs {
		refs = append(refs, strings.TrimPrefix(v.Ref, "released-"))
	}
	want := "2023-08-22 2023-08-29 2024-09-18-45edcd844 2024-09-18-cc190fae3 2024-09-19-a62c27d3b 2024-09-19-6e91dbf55 2024-09-19-e47df6533 2026-11-10-8d91e3d8a3 2026-11-17-31c309fe2a"
	if got := strings.Join(refs, " "); got != want {
		t.Errorf("refs\n %s\nwant\n %s", got, want)
	}
}

// Every day specs/ changed, oldest first, at the day's last commit: the 40
// commits saved fall on 16 days, and a version ID (the date) is named once.
func TestAppwriteListsEverySpecCommit(t *testing.T) {
	srv := recorded(t, map[string]string{"/repos/appwrite/specs/commits?path=specs&per_page=100&page=1": "appwrite-commits.json"})
	vs, err := Appwrite{APIBase: srv.URL}.Versions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 16 {
		t.Fatalf("%d versions, want 16", len(vs))
	}
	for i := 1; i < len(vs); i++ {
		if !vs[i].Released.After(vs[i-1].Released) || vs[i].ID == vs[i-1].ID || len(vs[i].Ref) != 40 {
			t.Errorf("%v after %v", vs[i], vs[i-1])
		}
	}
}

// Daily keeps each day's last commit, in order.
func TestDailyKeepsEachDaysLastCommit(t *testing.T) {
	in := []plugin.Version{{ID: "2026-10-27", Ref: "a"}, {ID: "2026-10-28", Ref: "b"}, {ID: "2026-10-28", Ref: "c"}, {ID: "2026-10-30", Ref: "d"}}
	var got []string
	for _, v := range Daily(in) {
		got = append(got, v.ID+" "+v.Ref)
	}
	if want := "2026-10-27 a,2026-10-28 c,2026-10-30 d"; strings.Join(got, ",") != want {
		t.Errorf("got %v, want %s", got, want)
	}
}

// Stripe's versions are its API versions, as each release's notes name
// them: one per version, oldest first, read at the newest release carrying
// it and dated by the first; v2190, whose notes name none, is left out.
func TestStripeListsAPIVersions(t *testing.T) {
	srv := recorded(t, map[string]string{"/repos/stripe/openapi/releases?per_page=100&page=1": "stripe-openapi-releases.json"})
	old := versions.GitHubAPI
	versions.GitHubAPI = srv.URL
	t.Cleanup(func() { versions.GitHubAPI = old })
	vs, err := Stripe{}.Versions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := "2026-02-25.clover@v2203 2026-03-25.dahlia@v2248 2026-04-22.dahlia@v2275 2026-05-27.dahlia@v2323 2026-06-24.dahlia@v2348 " +
		"2026-07-29.dahlia@v2441 2026-08-26.dahlia@v2516 2026-09-30.endive@v2592 2026-10-28.endive@v2600 2026-11-18.endive@v2602"
	if got := ids(vs); got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	last := vs[len(vs)-1]
	if !last.Released.Equal(time.Date(2026, 11, 17, 22, 34, 17, 0, time.UTC)) || last.URL != "https://github.com/stripe/openapi/releases/tag/v2602" {
		t.Errorf("newest: released %s (want v2601's date), url %s", last.Released, last.URL)
	}
}
