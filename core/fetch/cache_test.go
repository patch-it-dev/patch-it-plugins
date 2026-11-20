package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// validating answers like a server with an ETag and a Last-Modified: 304 when
// the request names the current ones, the body otherwise. body and etag are
// what it serves now; it records the conditional headers it saw.
type validating struct {
	body, etag, modified string
	sent304              atomic.Int32
	seen                 []string
}

func (v *validating) server(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v.seen = append(v.seen, r.Header.Get("If-None-Match")+"|"+r.Header.Get("If-Modified-Since"))
		if v.etag != "" {
			w.Header().Set("ETag", v.etag)
		}
		if v.modified != "" {
			w.Header().Set("Last-Modified", v.modified)
		}
		if (v.etag != "" && r.Header.Get("If-None-Match") == v.etag) ||
			(v.etag == "" && v.modified != "" && r.Header.Get("If-Modified-Since") == v.modified) {
			v.sent304.Add(1)
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Write([]byte(v.body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestTheCacheRevalidatesAndServesTheKeptBodyOn304(t *testing.T) {
	for name, c := range map[string]*Cache{"memory": {}, "disk": {Dir: t.TempDir()}} {
		v := &validating{body: "v1", etag: `"a"`, modified: "Sat, 14 Nov 2026 08:00:00 GMT"}
		srv := v.server(t)
		ctx := context.Background()
		for i := range 2 {
			got, err := c.Get(ctx, srv.URL, nil)
			if err != nil || string(got) != "v1" {
				t.Fatalf("%s, GET %d: %q, %v", name, i+1, got, err)
			}
		}
		if v.sent304.Load() != 1 || v.seen[1] != `"a"|Sat, 14 Nov 2026 08:00:00 GMT` {
			t.Errorf("%s: %d 304s, conditional headers %q", name, v.sent304.Load(), v.seen)
		}
		// The page changes: the new body is served and kept.
		v.body, v.etag = "v2", `"b"`
		for range 2 {
			if got, _ := c.Get(ctx, srv.URL, nil); string(got) != "v2" {
				t.Errorf("%s: after a change got %q, want v2", name, got)
			}
		}
		if v.sent304.Load() != 2 {
			t.Errorf("%s: %d 304s, want 2", name, v.sent304.Load())
		}
	}
}

// Kept on disk, an entry outlives the process: a new Cache over the same Dir
// revalidates.
func TestTheDiskCacheSurvivesARestart(t *testing.T) {
	dir := t.TempDir()
	v := &validating{body: "page", modified: "Sat, 14 Nov 2026 08:00:00 GMT"}
	srv := v.server(t)
	if _, err := (&Cache{Dir: dir}).Get(context.Background(), srv.URL, nil); err != nil {
		t.Fatal(err)
	}
	got, err := (&Cache{Dir: dir}).Get(context.Background(), srv.URL, nil)
	if err != nil || string(got) != "page" || v.sent304.Load() != 1 {
		t.Errorf("%q, %v, %d 304s", got, err, v.sent304.Load())
	}
}

// Without validators there's nothing to ask with: every GET is a full one.
func TestAResponseWithoutValidatorsIsNotKept(t *testing.T) {
	v := &validating{body: "page"}
	srv := v.server(t)
	c := &Cache{}
	for range 2 {
		if got, err := c.Get(context.Background(), srv.URL, nil); err != nil || string(got) != "page" {
			t.Fatalf("%q, %v", got, err)
		}
	}
	if v.seen[1] != "|" {
		t.Errorf("sent validators it never got: %q", v.seen[1])
	}
}

// A 304 nobody asked for is the server's error, not an empty page.
func TestAnUnasked304IsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotModified)
	}))
	t.Cleanup(srv.Close)
	if _, err := (&Cache{}).Get(context.Background(), srv.URL, nil); err == nil {
		t.Error("want an error")
	}
	if _, err := Get(context.Background(), srv.URL, nil); err == nil {
		t.Error("plain Get: want an error")
	}
}
