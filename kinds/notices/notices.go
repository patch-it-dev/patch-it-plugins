// Package notices is the kind for a live page of dated notices: something
// stops, or is deprecated, on a date. A model provider's deprecations table
// (retire/), a vendor's RSS or Atom changelog feed (feed/), a vendor's API
// end-of-life table (eol/). Such a page has no versions of its own: the kind
// is plugin.Living. Each Page reader knows how its page reads; the steps
// around it are written once, here. Read fetches the live page, stops when it
// hasn't changed since the snapshot kept from the read before, reads it,
// snapshots it in the Wayback Machine and checks the snapshot says the same
// (kit.Cite); Extract makes the events from that snapshot, so each cites what
// it quotes, keeping only the notices the read before didn't have.
//
// Every read runs the shape guard (plugin.Shape) before anything is kept:
// a read that finds fewer than DefaultMinRatio of the notices the read
// before found (a reader can set its own ratio with a MinRatio method: a
// feed's is lower), or that misses one of the source's anchors, fails with
// plugin.ErrShapeChanged. One option a reader turns on with a method:
// ArchiveWhenBlocked (a live page that answers one of the statuses its
// provider blocks plain fetches with is read from a fresh Wayback Machine
// snapshot instead).
//
// A living page that can't be read is a failed run, never "nothing
// changed": a page with no notices to read (plugin.ErrNothingToRead from its
// reader) is the page changing shape or a block page served in its place,
// so Read returns it as plugin.ErrShapeChanged, which errors.Is doesn't take
// for nothing to read.
package notices

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/kit"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// Page is one live page of notices and how it reads: retire.Page,
// feed.Feed and eol.Page.
type Page interface {
	// Slug is the source id the events carry in api: "openai/models".
	Slug() string
	// PageURL is the live page.
	PageURL() string
	// Find reads the page's notices.
	Find(page []byte) (kit.Notices, error)
	// Events turn what Find read into events citing citeURL, and the
	// notices that failed to become events. An event of a version-pair
	// type (an api_operation) carries previous as previous_version.
	Events(found kit.Notices, citeURL, previous string, now time.Time) ([]event.Event, []error)
}

// DefaultMinRatio is the shape guard's ratio for a notices page: past
// notices stay on the page, so a real drop is rare, and a read with fewer
// than 90% of the last read's notices means a table or a column changed.
const DefaultMinRatio = 0.9

// ratioed is a Page with a shape-guard ratio of its own.
type ratioed interface{ MinRatio() float64 }

// archivedWhenBlocked is a Page read from a fresh Wayback Machine snapshot
// when the live page refuses a plain fetch: ArchiveWhenBlocked is the HTTP
// statuses its provider blocks one with.
type archivedWhenBlocked interface{ ArchiveWhenBlocked() []int }

// Plugin reads one page.
type Plugin struct {
	Page     Page
	Archiver kit.Archiver

	shape *plugin.Shape // the source's, set by RegisterProvider (WithShape); nil: no guard
}

func (p Plugin) Name() string { return p.Page.Slug() }

// WithShape is p guarding its reads with s.
func (p Plugin) WithShape(s plugin.Shape) plugin.Plugin {
	p.shape = &s
	return p
}

// Read reads the page as it is now. A page that hashes as it did at last
// (CompareHash) returns last: one GET (a 304 with the HTTP cache), no
// snapshot. Otherwise the page is snapshotted in the Wayback Machine and the
// snapshot must hold what the live page says (kit.Cite): the new snapshot,
// dated now, is that capture, and a capture that says something else (the
// page changed in between) is an error, so nothing is kept this time.
func (p Plugin) Read(ctx context.Context, env plugin.Env, last plugin.Snapshot) (plugin.Snapshot, error) {
	pg := p.Page
	url := pg.PageURL()
	live, archived, err := p.fetch(ctx, env, url)
	if err != nil {
		return plugin.Snapshot{}, fmt.Errorf("fetching %s: %w", url, err)
	}
	hash := plugin.SHA256(live)
	if !last.IsZero() && hash == last.CompareHash {
		log.Printf("%s: no change since %s", pg.Slug(), last.SourceURL)
		return last, nil
	}
	want, err := pg.Find(live)
	if errors.Is(err, plugin.ErrNothingToRead) {
		// Not a skip: the page this source reads holds no notices, so it
		// changed shape or something else was served in its place.
		return plugin.Snapshot{}, fmt.Errorf("reading %s: %w: the page has no notices to read, it may be blocked: %v", url, plugin.ErrShapeChanged, err)
	}
	if err != nil {
		return plugin.Snapshot{}, fmt.Errorf("reading %s: %w", url, err)
	}
	if err := p.guard(want, last); err != nil {
		return plugin.Snapshot{}, fmt.Errorf("reading %s: %w", url, err)
	}
	snapURL, snap := archived, live
	if snapURL == "" {
		if snapURL, snap, _, err = kit.Cite(ctx, p.Archiver, url, pg.Find, want); err != nil {
			return plugin.Snapshot{}, err
		}
	}
	now := env.Now().UTC()
	v := plugin.Version{ID: now.Format(time.RFC3339), Released: now, Ref: snapURL, URL: snapURL}
	// The live page's hash, so the next read compares like with like.
	return plugin.Snapshot{Version: v, Content: snap, SourceURL: snapURL, CompareHash: hash}, nil
}

// guard is the shape guard on a read that found want, after the read that
// kept last: every anchor is there, and want isn't far fewer notices than
// last's. A provider renaming a column makes its table unreadable, and the
// read would quietly find fewer notices: fail loudly instead of keeping a
// partial list.
func (p Plugin) guard(want kit.Notices, last plugin.Snapshot) error {
	if err := p.shape.CheckAnchors(want.Has); err != nil {
		return err
	}
	before := -1
	if last.Content != nil {
		if b, err := p.Page.Find(last.Content); err == nil {
			before = b.Count()
		}
	}
	ratio := DefaultMinRatio
	if r, ok := p.Page.(ratioed); ok {
		ratio = r.MinRatio()
	}
	if err := p.shape.CheckCount("notices", ratio, before, want.Count()); err != nil {
		return fmt.Errorf("%w (the read before: %s)", err, last.SourceURL)
	}
	return nil
}

// fetch reads the live page. When the page's reader asks for it
// (ArchiveWhenBlocked) and the provider's own site blocks a plain fetch with
// one of the statuses the reader names (Salesforce's docs site: 403 on
// 2026-11-20, 503 from its CDN on 2026-11-21, after fetch's own retries;
// confirmed on 2026-11-20 that a real Wayback Machine capture of that page
// parses to the same findings as the live page would), it falls back to a
// fresh Wayback Machine snapshot, taken right now by the archive's own
// crawler rather than patch-it's, and returns that content with its own
// permanent URL in archivedURL: it needs no further citation step, having
// come from the archive already.
//
// Any other status is returned as is. A 404 or 410 means the page moved,
// not that it's blocked: asking the archive to snapshot a page that's gone
// would waste the call and hide the real cause behind the archive's own
// error. A 429, or a 5xx the reader doesn't name, is a transient failure:
// retrying it against the archive would hide it as a real page. Either way
// the run fails, and so does a fallback whose snapshot can't be taken.
func (p Plugin) fetch(ctx context.Context, env plugin.Env, pageURL string) (body []byte, archivedURL string, err error) {
	body, err = env.Get(ctx, pageURL)
	if err == nil {
		return body, "", nil
	}
	b, ok := p.Page.(archivedWhenBlocked)
	var se *fetch.StatusError
	if !ok || !errors.As(err, &se) || !slices.Contains(b.ArchiveWhenBlocked(), se.StatusCode) {
		return nil, "", err
	}
	archivedURL, body, archErr := p.Archiver.Snapshot(ctx, pageURL)
	if archErr != nil {
		return nil, "", fmt.Errorf("fetching it directly failed (%w), and so did asking the Wayback Machine to: %w", err, archErr)
	}
	return body, archivedURL, nil
}

// Extract is an event per notice on the page at to that wasn't at from
// (every one when from is zero), citing to's capture. The events keep the
// notice's own version (its date or API version). One of a version-pair type
// (an api_operation: an operation's end of life) names the read it was
// compared with as previous_version: from's dated version, or on a first
// read, with nothing before it, to's own.
func (p Plugin) Extract(_ context.Context, env plugin.Env, from, to plugin.Snapshot) (plugin.Result, error) {
	now := env.Now().UTC()
	previous := from.Version.ID
	if from.IsZero() {
		previous = to.Version.ID
	}
	events, rejected, err := p.events(to, previous, now)
	if err != nil {
		return plugin.Result{}, err
	}
	if !from.IsZero() {
		before, _, err := p.events(from, previous, now)
		if err != nil {
			return plugin.Result{}, err
		}
		had := map[string]bool{}
		for _, e := range before {
			had[e.Fingerprint] = true
		}
		kept := events[:0:0]
		for _, e := range events {
			if !had[e.Fingerprint] {
				kept = append(kept, e)
			}
		}
		events = kept
	}
	return plugin.Result{Events: events, Rejected: rejected}, nil
}

// events reads one snapshot's notices into events citing it.
func (p Plugin) events(s plugin.Snapshot, previous string, now time.Time) ([]event.Event, []error, error) {
	found, err := p.Page.Find(s.Content)
	if err != nil {
		return nil, nil, fmt.Errorf("reading %s: %w", s.SourceURL, err)
	}
	events, rejected := p.Page.Events(found, s.SourceURL, previous, now)
	return events, rejected, nil
}

var (
	_ plugin.Living  = Plugin{}
	_ plugin.Guarded = Plugin{}
)
