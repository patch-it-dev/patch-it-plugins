package kit

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
)

// Archiver takes a snapshot of a page that never changes afterwards, and hands
// back its permanent URL and the bytes it holds. A deprecations page is a live
// page; the event cites the snapshot.
type Archiver interface {
	Snapshot(ctx context.Context, pageURL string) (snapshotURL string, content []byte, err error)
}

// Wayback is the Internet Archive's Save Page Now: GET /save/<page> answers
// with a redirect to the new snapshot, /web/<14-digit timestamp>/<page>, and
// /web/<timestamp>id_/<page> serves the archived bytes as they were captured
// (checked 2026-10-13). One slow call per read that sees the page change.
//
// Both calls send userAgent: the archive answers 498 to Go's default
// ("Go-http-client/2.0") and serves the same capture to anything else (seen
// 2026-10-13).
type Wayback struct {
	Base   string       // default "https://web.archive.org"
	Client *http.Client // default: a 3-minute timeout, redirects not followed
}

const userAgent = "patch-it-plugins (+https://github.com/patch-it-dev/patch-it-plugins)"

var snapshotPath = regexp.MustCompile(`/web/(\d{14})/(.+)$`)

func (w Wayback) Snapshot(ctx context.Context, pageURL string) (string, []byte, error) {
	base := strings.TrimRight(w.Base, "/")
	if base == "" {
		base = "https://web.archive.org"
	}
	client := w.Client
	if client == nil {
		client = &http.Client{
			Timeout:       3 * time.Minute,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/save/"+pageURL, nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("asking the Wayback Machine to save %s: %w", pageURL, err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	location := resp.Header.Get("Location")
	if location == "" {
		location = resp.Header.Get("Content-Location")
	}
	m := snapshotPath.FindStringSubmatch(location)
	if m == nil {
		return "", nil, fmt.Errorf("the Wayback Machine didn't snapshot %s (status %d, location %q)", pageURL, resp.StatusCode, location)
	}
	if m[2] != pageURL {
		return "", nil, fmt.Errorf("the Wayback Machine snapshotted %s, not %s", m[2], pageURL)
	}

	body, err := fetch.Get(ctx, fmt.Sprintf("%s/web/%sid_/%s", base, m[1], pageURL), map[string]string{"User-Agent": userAgent})
	if err != nil {
		return "", nil, fmt.Errorf("reading back the snapshot of %s: %w", pageURL, err)
	}
	body, err = gunzipped(body)
	if err != nil {
		return "", nil, fmt.Errorf("reading back the snapshot of %s: %w", pageURL, err)
	}
	return fmt.Sprintf("%s/web/%s/%s", base, m[1], pageURL), body, nil
}

// gunzipped undoes the compression the archive may have stored the page with:
// an id_ capture is served as the bytes went over the wire, gzip included.
func gunzipped(b []byte) ([]byte, error) {
	if len(b) < 2 || b[0] != 0x1f || b[1] != 0x8b {
		return b, nil
	}
	r, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}
