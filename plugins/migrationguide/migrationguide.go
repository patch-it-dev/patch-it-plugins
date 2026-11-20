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
	"fmt"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/migrate"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/members"
	"github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/sdkguide"
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
			Watches:  "Members removed or renamed in the main namespace in NumPy 2.0",
			URLs:     []string{"https://github.com/numpy/numpy/blob/v2.0.0/doc/source/numpy_2_0_migration_guide.rst"},
			Packages: []string{"pypi:numpy"},
			Plugin:   Plugin{Guide: members.NumPy2},
		}},
	})
	plugin.RegisterProvider(plugin.Provider{
		ID: "posthog", Name: "PostHog", Category: plugin.CategoryAnalytics, Homepage: "https://posthog.com",
		Sources: []plugin.Source{{
			ID: "python-sdk", Layer: plugin.LayerSDK, Page: "providers/posthog.md",
			Watches:  "Members removed in the posthog Python SDK 8.0.0",
			URLs:     []string{"https://github.com/PostHog/posthog-python/blob/posthog-v8.0.0/docs/migration-7.x-to-8.0.md"},
			Packages: []string{"pypi:posthog"},
			Plugin:   Plugin{Guide: members.PostHog8},
		}},
	})
	plugin.RegisterProvider(plugin.Provider{
		ID: "newrelic", Name: "New Relic", Category: plugin.CategoryMonitoring, Homepage: "https://newrelic.com",
		Sources: []plugin.Source{{
			ID: "python-agent", Layer: plugin.LayerSDK, Page: "providers/newrelic.md",
			Watches:  "Members removed in the New Relic Python agent 11.0.0",
			URLs:     []string{"https://github.com/newrelic/docs-website/blob/f0b4f817bf9a7b76a78e26609369a8a7ce764c8e/src/content/docs/release-notes/agent-release-notes/python-release-notes/python-agent-110000.mdx"},
			Packages: []string{"pypi:newrelic"},
			Plugin:   Plugin{Guide: members.NewRelic11},
		}},
	})
}

// Guide is one source's migration guide: which releases there are, where the
// guide is at each, and how its table reads. sdkguide.Guide and
// members.Guide are the two readers.
type Guide interface {
	// Slug is the events' api and the source's old slug: "anthropic".
	Slug() string
	// Releases are the versions, oldest first, each with the ref its guide
	// would be read at.
	Releases(ctx context.Context, env plugin.Env) ([]plugin.Version, error)
	// Current is the release a plain run (Run) reads.
	Current(ctx context.Context, env plugin.Env) (plugin.Version, error)
	// GuideURL is where the guide is at v, pinned to v's tag, or "" when v
	// has no guide (a release that isn't a new major).
	GuideURL(v plugin.Version) string
	// Read turns the guide, as read at to, into events with
	// previous_version from: v1 events (converted here) or v2 ones, and the
	// rows that failed to become events. err is a guide it can't read.
	Read(page []byte, from, to plugin.Version, now time.Time) (v1 []event.ChangeEvent, v2 []event.EventV2, rejected []error, err error)
}

// Plugin reads one source's migration guide.
type Plugin struct {
	Guide Guide
}

func (p Plugin) Name() string { return p.Guide.Slug() }

// Run reads the guide at its current release and stores its events.
func (p Plugin) Run(ctx context.Context, env plugin.Env) (plugin.Result, error) {
	v, err := p.Guide.Current(ctx, env)
	if err != nil {
		return plugin.Result{}, err
	}
	to, err := p.ReadAt(ctx, env, v)
	if err != nil {
		return plugin.Result{}, err
	}
	res, err := p.Extract(ctx, env, plugin.Snapshot{}, to)
	if err != nil {
		return plugin.Result{}, err
	}
	res.Baselines = []plugin.Spec{plugin.Baseline(p.Name(), v.ID, v.Ref, to.SourceURL, to.Content, false, env.Now().UTC())}
	return res, nil
}

// ListVersions is every release of the source, oldest first.
func (p Plugin) ListVersions(ctx context.Context, env plugin.Env) ([]plugin.Version, error) {
	return p.Guide.Releases(ctx, env)
}

// ReadAt fetches the guide at v; a release with no guide reads as nothing (a
// snapshot with no content).
func (p Plugin) ReadAt(ctx context.Context, env plugin.Env, v plugin.Version) (plugin.Snapshot, error) {
	url := p.Guide.GuideURL(v)
	if url == "" {
		return plugin.Snapshot{Version: v}, nil
	}
	page, err := env.Get(ctx, url)
	if err != nil {
		return plugin.Snapshot{}, fmt.Errorf("fetching %s: %w", url, err)
	}
	return plugin.Snapshot{Version: v, Content: page, SourceURL: url}, nil
}

// Extract is an event per removal in to's guide, with previous_version
// from's release. A release with no guide has no events.
func (p Plugin) Extract(_ context.Context, env plugin.Env, from, to plugin.Snapshot) (plugin.Result, error) {
	if to.Content == nil {
		return plugin.Result{Label: to.Version.ID}, nil
	}
	v1, v2, rejected, err := p.Guide.Read(to.Content, from.Version, to.Version, env.Now().UTC())
	if err != nil {
		return plugin.Result{}, fmt.Errorf("reading %s: %w", to.SourceURL, err)
	}
	converted, invalid := migrate.Fresh(v1)
	return plugin.Result{Label: to.SourceURL, Events: append(converted, v2...), Rejected: append(rejected, invalid...)}, nil
}

var _ plugin.Versioned = Plugin{}
