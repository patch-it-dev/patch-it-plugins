package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const appwriteSHA = "06e6fafa4d1ae3c815843ff0b02a56e6e5582756"

// fakeAppwriteGitHub serves the commits list, appwriteSHA's commit (dated
// 2026-10-12T18:11:38Z, as GitHub dates it), directory listings
// ("<path>?ref=<sha>" -> names) and raw files ("/<sha>/specs/<dir>/<file>").
func fakeAppwriteGitHub(t *testing.T, commits string, listings, raw map[string]string) Appwrite {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const repo = "/api/repos/appwrite/specs"
		switch {
		case r.URL.Path == repo+"/commits":
			if r.URL.Query().Get("path") != "specs" {
				http.Error(w, "want path=specs", http.StatusBadRequest)
				return
			}
			w.Write([]byte(commits))
		case r.URL.Path == repo+"/commits/"+appwriteSHA:
			w.Write([]byte(`{"sha":"` + appwriteSHA + `","commit":{"committer":{"date":"2026-10-12T18:11:38Z"}}}`))
		case strings.HasPrefix(r.URL.Path, repo+"/contents/"):
			names, ok := listings[strings.TrimPrefix(r.URL.Path, repo+"/contents/")+"?ref="+r.URL.Query().Get("ref")]
			if !ok {
				http.NotFound(w, r)
				return
			}
			var entries []map[string]string
			for _, n := range strings.Fields(names) {
				entries = append(entries, map[string]string{"name": n})
			}
			json.NewEncoder(w).Encode(entries)
		default:
			body, ok := raw[strings.TrimPrefix(r.URL.Path, "/raw/appwrite/specs")]
			if !ok {
				http.NotFound(w, r)
				return
			}
			w.Write([]byte(body))
		}
	}))
	t.Cleanup(srv.Close)
	return Appwrite{APIBase: srv.URL + "/api", RawBase: srv.URL + "/raw"}
}

func TestAppwriteLatestTagIsTheNewestSpecsCommit(t *testing.T) {
	a := fakeAppwriteGitHub(t, `[{"sha":"`+appwriteSHA+`"},{"sha":"older"}]`, nil, nil)
	if tag, err := a.LatestTag(context.Background()); err != nil || tag != appwriteSHA {
		t.Errorf("got %q, %v; want %s", tag, err, appwriteSHA)
	}
	for _, body := range []string{`[]`, `[{"sha":"short"}]`, `{}`} {
		a := fakeAppwriteGitHub(t, body, nil, nil)
		if _, err := a.LatestTag(context.Background()); err == nil {
			t.Errorf("%s: want an error", body)
		}
	}
}

// The single document wins over the server one, filtered to the server
// platform; the newest minor is the highest by number, not "latest" and not
// by string order.
func TestAppwriteFetchReadsTheNewestMinorsServerOperations(t *testing.T) {
	const doc = `{"openapi":"3.0.0","info":{"version":"2.10.1"},"servers":[{"url":"https://cloud.appwrite.io/v1"},{"url":"https://{region}.cloud.appwrite.io/v1"}],"paths":{` +
		`"/both":{"get":{"x-appwrite":{"platforms":["client","server"]},"x-n":1.50},"post":{"x-appwrite":{"platforms":["console"]}}},` +
		`"/console":{"get":{"x-appwrite":{"platforms":["console"]}},"parameters":[]}}}`
	a := fakeAppwriteGitHub(t, "", map[string]string{
		"specs?ref=" + appwriteSHA:        "1.9.x 2.9.x 2.10.x latest README.md",
		"specs/2.10.x?ref=" + appwriteSHA: "open-api3-2.10.x-server.json open-api3-2.10.x.json",
	}, map[string]string{"/" + appwriteSHA + "/specs/2.10.x/open-api3-2.10.x.json": doc})
	spec, err := a.Fetch(context.Background(), appwriteSHA)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Version != "2026-10-12" || !strings.HasSuffix(spec.SourceURL, "/raw/appwrite/specs/"+appwriteSHA+"/specs/2.10.x/open-api3-2.10.x.json") {
		t.Errorf("version %q, url %s", spec.Version, spec.SourceURL)
	}
	want := `{"info":{"version":"2.10.1"},"openapi":"3.0.0","paths":{"/v1/both":{"get":{"x-appwrite":{"platforms":["client","server"]},"x-n":1.50}}},` +
		`"servers":[{"url":"https://cloud.appwrite.io/v1"},{"url":"https://{region}.cloud.appwrite.io/v1"}],"x-patch-it-base-path":"/v1"}`
	if string(spec.Content) != want {
		t.Errorf("content %s, want %s", spec.Content, want)
	}
	// A pointer into /v1/both names the path as published.
	if got := (Appwrite{}).SourcePointer(spec.Content)("/paths/~1v1~1both/get/x-appwrite/deprecated"); got != "/paths/~1both/get/x-appwrite/deprecated" {
		t.Errorf("source pointer %s", got)
	}
}

// A single document where no operation is on the server any more (the
// platforms renamed, or gone) is an error, not a spec with nothing in it.
func TestAppwriteFetchFailsWithNoServerOperations(t *testing.T) {
	for name, doc := range map[string]string{
		"no platforms":     `{"openapi":"3.0.0","info":{"version":"2.8.0"},"paths":{"/a":{"get":{"x-appwrite":{}}}}}`,
		"renamed platform": `{"openapi":"3.0.0","info":{"version":"2.8.0"},"paths":{"/a":{"get":{"x-appwrite":{"platforms":["backend"]}}}}}`,
	} {
		a := fakeAppwriteGitHub(t, "", map[string]string{
			"specs/2.8.x?ref=" + appwriteSHA: "open-api3-2.8.x.json",
		}, map[string]string{"/" + appwriteSHA + "/specs/2.8.x/open-api3-2.8.x.json": doc})
		if _, err := a.Fetch(context.Background(), "2.8.x@"+appwriteSHA); err == nil || !strings.Contains(err.Error(), "server") {
			t.Errorf("%s: got %v, want an error about the server platform", name, err)
		}
	}
}

// A per-platform directory is read as published; a tag can name the
// directory to read an older minor.
func TestAppwriteFetchServerDocumentAndNamedDirectory(t *testing.T) {
	const doc = `{"openapi":"3.0.0","info":{"version":"1.7.4"},"paths":{"/a":{"get":{}}}}`
	a := fakeAppwriteGitHub(t, "", map[string]string{
		"specs/1.7.x?ref=" + appwriteSHA: "open-api3-1.7.x-client.json open-api3-1.7.x-server.json swagger2-1.7.x-server.json",
	}, map[string]string{"/" + appwriteSHA + "/specs/1.7.x/open-api3-1.7.x-server.json": doc})
	spec, err := a.Fetch(context.Background(), "1.7.x@"+appwriteSHA)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Version != "2026-10-12" || string(spec.Content) != doc {
		t.Errorf("version %q, content %s", spec.Version, spec.Content)
	}
}

func TestAppwriteFetchFailures(t *testing.T) {
	listings := map[string]string{
		"specs?ref=" + appwriteSHA:       "1.9.x latest",
		"specs/1.9.x?ref=" + appwriteSHA: "open-api3-1.9.x-client.json",
		"specs/1.8.x?ref=" + appwriteSHA: "open-api3-1.8.x-server.json",
		"specs/1.6.x?ref=" + appwriteSHA: "open-api3-1.6.x-server.json",
		"specs/1.5.x?ref=" + appwriteSHA: "open-api3-1.5.x-server.json",
	}
	raw := map[string]string{
		"/" + appwriteSHA + "/specs/1.5.x/open-api3-1.5.x-server.json": `{"servers":[{"url":"https://a.io/v1"},{"url":"https://b.io/v2"}],"paths":{}}`,
		"/" + appwriteSHA + "/specs/1.6.x/open-api3-1.6.x-server.json": `<html>`,
	}
	a := fakeAppwriteGitHub(t, "", listings, raw)
	for name, tag := range map[string]string{
		"short sha":          "06e6fafa",
		"bad directory":      "latest@" + appwriteSHA,
		"no server document": appwriteSHA,
		"no file at commit":  "1.8.x@" + appwriteSHA,
		"servers disagree":   "1.5.x@" + appwriteSHA,
		"invalid json":       "1.6.x@" + appwriteSHA,
		"unknown commit":     strings.Repeat("0", 40),
	} {
		if _, err := a.Fetch(context.Background(), tag); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if newestAppwriteDir([]string{"latest", "README.md"}) != "" {
		t.Error("no minor directory: want none")
	}
}
