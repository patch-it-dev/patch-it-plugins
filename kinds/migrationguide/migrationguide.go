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
	"github.com/patch-it-dev/patch-it-plugins/kinds/migrationguide/members"
	"github.com/patch-it-dev/patch-it-plugins/kinds/migrationguide/sdkguide"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "anthropic", Name: "Anthropic", Category: plugin.CategoryAI, Homepage: "https://www.anthropic.com",
		Sources: []plugin.Source{{
			ID: "python-sdk", Layer: plugin.LayerSDK, Page: "providers/anthropic.md",
			Watches:  "Request parameters each new major of the Python SDK removes from `messages.create()` and friends, from MIGRATION.md at the major's tag",
			URLs:     []string{"https://github.com/anthropics/anthropic-sdk-python/blob/main/MIGRATION.md", "https://pypi.org/project/anthropic/"},
			Packages: []string{"pypi:anthropic"},
			Plugin:   Plugin{Guide: sdkguide.Anthropic},
		}},
	})
	plugin.RegisterProvider(plugin.Provider{
		ID: "numpy", Name: "NumPy", Category: plugin.CategoryPackages, Homepage: "https://numpy.org",
		Sources: []plugin.Source{{
			ID: "numpy", Layer: plugin.LayerPackage, Page: "providers/numpy.md",
			Watches:  "Members a new major removes or renames in the main namespace, from that major's migration guide at its first tag",
			URLs:     []string{"https://numpy.org/devdocs/numpy_2_0_migration_guide.html", "https://pypi.org/project/numpy/"},
			Packages: []string{"pypi:numpy"},
			Plugin:   Plugin{Guide: members.NumPy},
		}},
	})
	plugin.RegisterProvider(plugin.Provider{
		ID: "posthog", Name: "PostHog", Category: plugin.CategoryAnalytics, Homepage: "https://posthog.com",
		Sources: []plugin.Source{{
			ID: "python-sdk", Layer: plugin.LayerSDK, Page: "providers/posthog.md",
			Watches:  "Members a new major of the posthog Python SDK removes, from its migration guide in docs/ at the major's first tag",
			URLs:     []string{"https://github.com/PostHog/posthog-python/tree/main/docs", "https://pypi.org/project/posthog/"},
			Packages: []string{"pypi:posthog"},
			Plugin:   Plugin{Guide: members.PostHog},
		}},
	})
	plugin.RegisterProvider(plugin.Provider{
		ID: "newrelic", Name: "New Relic", Category: plugin.CategoryMonitoring, Homepage: "https://newrelic.com",
		Sources: []plugin.Source{{
			ID: "python-agent", Layer: plugin.LayerSDK, Page: "providers/newrelic.md",
			Watches:  "Members a new major of the New Relic Python agent removes, from the major's release notes (Removed features)",
			URLs:     []string{"https://docs.newrelic.com/docs/release-notes/agent-release-notes/python-release-notes/", "https://pypi.org/project/newrelic/"},
			Packages: []string{"pypi:newrelic"},
			Plugin:   Plugin{Guide: members.NewRelic},
		}},
	})
}

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
type Plugin struct {
	Guide Guide
}

func (p Plugin) Name() string { return p.Guide.Slug() }

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

var _ plugin.Versioned = Plugin{}
