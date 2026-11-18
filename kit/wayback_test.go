package kit

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const page = "https://developers.openai.com/api/docs/deprecations.md"

// fakeArchive answers like the Wayback Machine did on 2026-10-13: a 302 to
// the snapshot, and the id_ capture gzipped as it went over the wire.
// It answers 498 to Go's default User-Agent, as the real one does.
func fakeArchive(t *testing.T, location func(base string) string, body []byte) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/save/"):
			if loc := location(srv.URL); loc != "" {
				w.Header().Set("Location", loc)
			}
			w.WriteHeader(http.StatusFound)
		case strings.HasPrefix(r.UserAgent(), "Go-http-client"):
			w.WriteHeader(498)
		case r.URL.Path == "/web/20261013115538id_/"+page:
			var gz bytes.Buffer
			zw := gzip.NewWriter(&gz)
			zw.Write(body)
			zw.Close()
			w.Write(gz.Bytes())
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestASnapshotIsSavedAndReadBack(t *testing.T) {
	body := []byte("| Shutdown date | Model | Recommended replacement |\n")
	srv := fakeArchive(t, func(base string) string { return base + "/web/20261013115538/" + page }, body)
	url, got, err := Wayback{Base: srv.URL}.Snapshot(context.Background(), page)
	if err != nil {
		t.Fatal(err)
	}
	if url != srv.URL+"/web/20261013115538/"+page || !bytes.Equal(got, body) {
		t.Errorf("%s / %q", url, got)
	}
}

func TestNoSnapshotIsAnError(t *testing.T) {
	for name, loc := range map[string]func(string) string{
		"no redirect":    func(string) string { return "" },
		"another page":   func(base string) string { return base + "/web/20261013115538/https://example.com/other.md" },
		"not a snapshot": func(base string) string { return base + "/login" },
	} {
		srv := fakeArchive(t, loc, nil)
		if _, _, err := (Wayback{Base: srv.URL}).Snapshot(context.Background(), page); err == nil {
			t.Errorf("%s: want an error, never an event without a snapshot to cite", name)
		}
	}
}
