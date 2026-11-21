package planetscale

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const planetscaleSHA = "fc13e09d0c18b893aabd3e02d175d13957bba677"

// fakePlanetScaleGitHub serves the commits list, one commit by any prefix of
// planetscaleSHA, and the spec at planetscaleSHA (404 when spec is "").
func fakePlanetScaleGitHub(t *testing.T, commitsJSON, spec string) PlanetScale {
	t.Helper()
	const repo = "/api/repos/planetscale/terraform-provider-planetscale"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch p := r.URL.Path; {
		case p == repo+"/commits" && r.URL.Query().Get("path") == "schemas/openapi.yaml":
			w.Write([]byte(commitsJSON))
		case strings.HasPrefix(p, repo+"/commits/") && strings.HasPrefix(planetscaleSHA, strings.TrimPrefix(p, repo+"/commits/")):
			w.Write([]byte(`{"sha":"` + planetscaleSHA + `","commit":{"committer":{"date":"2026-11-09T09:16:32Z"}}}`))
		case p == "/raw/planetscale/terraform-provider-planetscale/"+planetscaleSHA+"/schemas/openapi.yaml" && spec != "":
			w.Write([]byte(spec))
		default:
			http.Error(w, "404: Not Found", http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return PlanetScale{APIBase: srv.URL + "/api", RawBase: srv.URL + "/raw"}
}

func TestPlanetScaleLatestTagIsTheNewestSpecCommit(t *testing.T) {
	p := fakePlanetScaleGitHub(t, `[{"sha":"`+planetscaleSHA+`"},{"sha":"older"}]`, "")
	tag, err := p.LatestTag(context.Background())
	if err != nil || tag != planetscaleSHA {
		t.Errorf("got %q, %v; want %s", tag, err, planetscaleSHA)
	}
	p = fakePlanetScaleGitHub(t, `[]`, "")
	if _, err := p.LatestTag(context.Background()); err == nil {
		t.Error("no commit: want an error")
	}
}

// The version is the commit's date, since the API has none (info.version is
// a fixed v1); the source is pinned to the full SHA even when a short one is
// asked for; the YAML becomes JSON with its values unchanged.
func TestPlanetScaleFetchPinsTheFullSHAAndDatesByCommit(t *testing.T) {
	const spec = "openapi: 3.0.1\ninfo:\n  version: v1\npaths:\n  /organizations:\n    get:\n      description: |-\n        Deprecated. Use x instead.\n      responses:\n        \"200\":\n          description: OK\n"
	p := fakePlanetScaleGitHub(t, "", spec)
	got, err := p.Fetch(context.Background(), "fc13e09d")
	if err != nil {
		t.Fatal(err)
	}
	if want := p.RawBase + "/planetscale/terraform-provider-planetscale/" + planetscaleSHA + "/schemas/openapi.yaml"; got.Version != "2026-11-09" || got.SourceURL != want {
		t.Errorf("got version %q, url %s", got.Version, got.SourceURL)
	}
	var doc struct {
		Paths map[string]map[string]struct {
			Description string `json:"description"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(got.Content, &doc); err != nil || doc.Paths["/organizations"]["get"].Description != "Deprecated. Use x instead." {
		t.Errorf("content %s (%v)", got.Content, err)
	}
}

// A file that isn't a spec fails the fetch: diffed, it would read as every
// operation removed. The commit that deleted the file (f33a0c3b35) answers
// 404; a stub served 200 has no paths.
func TestPlanetScaleFetchRefusesWhatIsntASpec(t *testing.T) {
	for name, body := range map[string]string{
		"deleted (404)":  "",
		"404 stub":       "404: Not Found",
		"no paths":       "openapi: 3.0.1\ninfo:\n  version: v1\n",
		"empty paths":    "openapi: 3.0.1\npaths: {}\n",
		"not yaml":       "openapi: [3.0.1\n",
		"a yaml scalar":  "Not Found",
		"html error":     "<html><body>Not Found</body></html>",
		"paths a string": "openapi: 3.0.1\npaths: none\n",
	} {
		p := fakePlanetScaleGitHub(t, "", body)
		if _, err := p.Fetch(context.Background(), planetscaleSHA); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	p := fakePlanetScaleGitHub(t, "", "openapi: 3.0.1\npaths:\n  /x:\n    get: {}\n")
	if _, err := p.Fetch(context.Background(), "0000000"); err == nil {
		t.Error("unknown commit: want an error")
	}
}
