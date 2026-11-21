// Package migrationguide is the kind for a guide that lists what a release
// removed, read at the tag of the release it describes: an SDK's
// MIGRATION.md and the request parameters a major removed (sdkguide/), or a
// package's migration guide and the members of its module a release removed
// (members/, RST or Markdown tables). The tag never changes, so the guide's
// URL is the events' provenance and a rerun stores nothing new: the
// fingerprints dedupe.
//
// Every source is one Plugin with a Guide, the reader that knows which
// releases there are, where the guide is at each, and how its table reads.
// Listing, reading and extracting are written once, here.
package migrationguide

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// Guide is one source's migration guide: which releases there are, where the
// guide is at each, and how its table reads. sdkguide.Guide and
// members.Guide are the two readers.
type Guide interface {
	// Slug is the source id the events carry in api: "anthropic/python-sdk".
	Slug() string
	// Releases are the versions, oldest first, each with the ref its guide
	// would be read at.
	Releases(ctx context.Context, env plugin.Env) ([]plugin.Version, error)
	// GuideURL is where the guide is at v, pinned to v's tag (or a commit),
	// or "" when v has no guide (a release that isn't a new major).
	GuideURL(ctx context.Context, env plugin.Env, v plugin.Version) (string, error)
	// Read turns the guide, as read at to from sourceURL, into events with
	// previous_version from, and the rows that failed to become events. err
	// is a guide it can't read.
	Read(page []byte, sourceURL string, from, to plugin.Version, now time.Time) (events []event.Event, rejected []error, err error)
}

// Plugin reads one source's migration guide.
//
// Each major's guide is its own document, so a guide is never compared with
// the one before: the shape guard here is the runner's
// Shape.NothingToRead over the releases (a guide that stops being found).
type Plugin struct {
	Guide Guide
}

func (p Plugin) Name() string { return p.Guide.Slug() }

// WithShape is p as it is: a guide has no count to compare, and whoever
// runs it reads the source's NothingToReadAfter from the registry
// (plugin.LookupSource) for Shape.NothingToRead.
func (p Plugin) WithShape(plugin.Shape) plugin.Plugin { return p }

// ListVersions is every release of the source, oldest first.
func (p Plugin) ListVersions(ctx context.Context, env plugin.Env) ([]plugin.Version, error) {
	return p.Guide.Releases(ctx, env)
}

// ReadAt fetches the guide at v; a release with no guide reads as nothing (a
// snapshot with no content). A new major whose guide isn't there (a 404 at
// its tag) has nothing to read: plugin.ErrNothingToRead, so the pair is
// kept as skipped, not as a failure, nor as a release with no changes.
func (p Plugin) ReadAt(ctx context.Context, env plugin.Env, v plugin.Version) (plugin.Snapshot, error) {
	url, err := p.Guide.GuideURL(ctx, env, v)
	if err != nil {
		return plugin.Snapshot{}, err
	}
	if url == "" {
		return plugin.Snapshot{Version: v}, nil
	}
	page, err := env.Get(ctx, url)
	var se *fetch.StatusError
	if errors.As(err, &se) && se.StatusCode == 404 {
		return plugin.Snapshot{}, fmt.Errorf("no guide at %s: %w", url, plugin.ErrNothingToRead)
	}
	if err != nil {
		return plugin.Snapshot{}, fmt.Errorf("fetching %s: %w", url, err)
	}
	return plugin.Snapshot{Version: v, Content: page, SourceURL: url}, nil
}

// Extract is an event per removal in to's guide, with previous_version
// from's release. A release with no guide has no events.
func (p Plugin) Extract(_ context.Context, env plugin.Env, from, to plugin.Snapshot) (plugin.Result, error) {
	if to.Content == nil {
		return plugin.Result{}, nil
	}
	events, rejected, err := p.Guide.Read(to.Content, to.SourceURL, from.Version, to.Version, env.Now().UTC())
	if err != nil {
		return plugin.Result{}, fmt.Errorf("reading %s: %w", to.SourceURL, err)
	}
	return plugin.Result{Events: events, Rejected: rejected}, nil
}

var (
	_ plugin.Versioned = Plugin{}
	_ plugin.Guarded   = Plugin{}
)
