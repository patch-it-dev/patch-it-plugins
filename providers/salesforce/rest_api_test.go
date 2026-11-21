package salesforce

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/kit"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/eol"
)

func TestEOLRegistersItself(t *testing.T) {
	kittest.Registered(t, notices.Plugin{Page: RESTAPI})
}

// The golden test: RESTAPI's own end-of-life policy page (checked
// 2026-11-20, saved 2026-10-31), real and unedited but for dropping the
// surrounding navigation chrome. It already holds two real, past retirements
// (versions 7.0 through 20.0, retired as of Summer '22; versions 21.0
// through 30.0, retired as of Summer '25) and one real, scheduled one
// (versions 31.0 through 40.0, deprecated from Summer '27, retired from
// Summer '28): 34 events, and the still-"Supported." 41.0 through 67.0 band
// names none, the pinned drop test.
func TestEOLGolden(t *testing.T) {
	saved := kittest.Pages{RESTAPI.URL: "testdata/salesforce-api-rest-eol-2026-10-31.html"}
	archive := &kittest.Archive{T: t, Pages: saved}
	src, _ := plugin.LookupSource(RESTAPI.API)
	p := notices.Plugin{Page: RESTAPI, Archiver: archive}.WithShape(src.Source.Shape).(plugin.Living) // its anchors on the saved page too
	got := kittest.Page(t, kittest.Env(t, saved), p)
	if archive.Calls != 1 {
		t.Errorf("%d snapshots, want 1: the second read's page is unchanged", archive.Calls)
	}
	kittest.Golden(t, "testdata/golden/salesforce.json", got)
}

// The pinned drop test: "Versions 41.0 through 67.0: Supported." is a real
// row on the page, and it must name no version.
func TestDropsAStillSupportedBand(t *testing.T) {
	page := kittest.Read(t, "testdata/salesforce-api-rest-eol-2026-10-31.html")
	found, err := eol.Findings(page)
	if err != nil {
		t.Fatal(err)
	}
	supported := map[string]bool{}
	for v := 41; v <= 67; v++ {
		supported[fmt.Sprintf("%d.0", v)] = true
	}
	var dropped bool
	for _, f := range found {
		if f.Dropped == "" && supported[f.Version] {
			t.Errorf("version %s got an event, want none: it's still Supported", f.Version)
		}
		if f.Dropped != "" && f.Row == "Versions 41.0 through 67.0" {
			dropped = true
		}
	}
	if !dropped {
		t.Error("the still-supported band wasn't found and dropped")
	}
}

// RESTAPI's docs site answers 403 to a plain GET, or 503 from its CDN;
// the live page is never reachable through env.Get. Read must fall back to
// the Wayback Machine's own fetch and use its snapshot directly, not fail
// outright. 401 falls back the same way.
func TestFallsBackToTheArchiveWhenTheLiveFetchIsBlocked(t *testing.T) {
	for _, code := range []int{401, 403, 503} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			saved := kittest.Pages{RESTAPI.URL: "testdata/salesforce-api-rest-eol-2026-10-31.html"}
			archive := &kittest.Archive{T: t, Pages: saved}
			env := plugin.Env{
				Get: func(context.Context, string) ([]byte, error) {
					return nil, &fetch.StatusError{URL: RESTAPI.URL, StatusCode: code}
				},
				Now: func() time.Time { return kittest.Clock },
			}
			got := kittest.Page(t, env, notices.Plugin{Page: RESTAPI, Archiver: archive})
			if len(got) == 0 {
				t.Fatal("no events: the archive fallback didn't run")
			}
			// Unlike the normal path, a permanently blocked live page has no
			// cheap way to check "did it change": fetch needs the archive's
			// own capture just to get bytes to hash, on every read, not only
			// a changed one. The second read still stores nothing new: the
			// hash matches.
			if archive.Calls != 2 {
				t.Errorf("%d snapshots, want 2: the live page is blocked every read", archive.Calls)
			}
			for _, e := range got {
				if e.SourceURL == RESTAPI.URL {
					t.Errorf("%s: source_url is the blocked live page, want the archive's snapshot", e.Fingerprint)
				}
			}
		})
	}
}

// A real failure (not the provider blocking a plain fetch) must stay a
// failure: falling back to the archive for a timeout, a moved page or a 5xx
// the reader doesn't name would hide it as a real page.
func TestDoesNotFallBackOnATransientError(t *testing.T) {
	for _, code := range []int{404, 429, 500, 502} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			env := plugin.Env{
				Get: func(context.Context, string) ([]byte, error) {
					return nil, &fetch.StatusError{URL: RESTAPI.URL, StatusCode: code}
				},
				Now: func() time.Time { return kittest.Clock },
			}
			// No Pages: a fallback here would fail the test by asking the
			// archive for a snapshot it was never given.
			_, err := notices.Plugin{Page: RESTAPI, Archiver: &kittest.Archive{T: t}}.Read(context.Background(), env, plugin.Snapshot{})
			if err == nil {
				t.Fatalf("want an error for a %d, got none", code)
			}
		})
	}
}

// A blocked page that the archive can't capture, or whose capture holds no
// table (a block page saved in its place), fails the read: the run records
// an error, never "nothing changed" and never a skip.
func TestABlockedPageTheArchiveCantReadIsAFailedRun(t *testing.T) {
	blocked := plugin.Env{
		Get: func(context.Context, string) ([]byte, error) {
			return nil, &fetch.StatusError{URL: RESTAPI.URL, StatusCode: 403}
		},
		Now: func() time.Time { return kittest.Clock },
	}
	last := plugin.Snapshot{Version: plugin.Version{ID: "2026-11-01T00:00:00Z"}, Content: []byte("<table></table>"), CompareHash: "x"}
	for name, archive := range map[string]kit.Archiver{
		"no snapshot":  failingArchive{},
		"a block page": blockPage{},
	} {
		_, err := notices.Plugin{Page: RESTAPI, Archiver: archive}.Read(context.Background(), blocked, last)
		if err == nil || errors.Is(err, plugin.ErrNothingToRead) {
			t.Errorf("%s: %v, want a failure that isn't plugin.ErrNothingToRead", name, err)
		}
	}
}

type failingArchive struct{}

func (failingArchive) Snapshot(context.Context, string) (string, []byte, error) {
	return "", nil, errors.New("the Wayback Machine didn't snapshot it")
}

type blockPage struct{}

func (blockPage) Snapshot(_ context.Context, u string) (string, []byte, error) {
	return "https://web.archive.org/web/20261121000000/" + u, []byte("<html><body>Access Denied</body></html>"), nil
}
