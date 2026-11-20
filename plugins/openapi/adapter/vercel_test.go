package adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeVercelGitHub serves a latest-release endpoint and specs by path, under
// vercel/sdk's URL shape.
func fakeVercelGitHub(t *testing.T, releaseJSON string, specs map[string]string) *httptest.Server {
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

	oldRel, oldSpec := vercelReleasesURL, vercelSpecURLTemplate
	vercelReleasesURL = srv.URL + "/releases/latest"
	vercelSpecURLTemplate = srv.URL + "/%s/vercel-spec.json"
	t.Cleanup(func() { vercelReleasesURL, vercelSpecURLTemplate = oldRel, oldSpec })
	return srv
}

func TestVercelLatestTagReadsTheReleaseTag(t *testing.T) {
	fakeVercelGitHub(t, `{"tag_name":"v1.28.65"}`, nil)
	tag, err := Vercel{}.LatestTag(context.Background())
	if err != nil || tag != "v1.28.65" {
		t.Errorf("got %q, %v; want v1.28.65", tag, err)
	}
}

func TestVercelLatestTagRejectsAReleaseWithoutATag(t *testing.T) {
	fakeVercelGitHub(t, `{"name":"something"}`, nil)
	if _, err := (Vercel{}).LatestTag(context.Background()); err == nil {
		t.Error("release without tag_name: want an error")
	}
}

// The version is the tag a customer pins, not info.version, which is "0.0.1"
// in every release; the content is passed through byte for byte.
func TestVercelFetchUsesTheTagAsVersion(t *testing.T) {
	const body = `{"info":{"version":"0.0.1"},"openapi":"3.0.3","paths":{}}`
	srv := fakeVercelGitHub(t, "", map[string]string{"/v1.18.7/vercel-spec.json": body})
	spec, err := Vercel{}.Fetch(context.Background(), "v1.18.7")
	if err != nil {
		t.Fatal(err)
	}
	if spec.Version != "1.18.7" {
		t.Errorf("version = %q, want 1.18.7", spec.Version)
	}
	if spec.SourceURL != srv.URL+"/v1.18.7/vercel-spec.json" {
		t.Errorf("source_url = %s, want the file actually fetched", spec.SourceURL)
	}
	if string(spec.Content) != body {
		t.Errorf("content changed: %s", spec.Content)
	}
}

func TestVercelFetchRejectsInvalidJSON(t *testing.T) {
	fakeVercelGitHub(t, "", map[string]string{"/bad/vercel-spec.json": "<html>"})
	if _, err := (Vercel{}).Fetch(context.Background(), "bad"); err == nil {
		t.Error("invalid json: want an error")
	}
}

func TestVercelFetchFailsWhenSpecDoesNotExist(t *testing.T) {
	fakeVercelGitHub(t, "", nil)
	if _, err := (Vercel{}).Fetch(context.Background(), "v9.9.9"); err == nil {
		t.Error("missing tag: want an error")
	}
}
