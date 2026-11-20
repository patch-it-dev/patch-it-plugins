// Package migrationguide is the source plugin for an SDK's migration guide,
// read at the release tag of each major it describes: an event per request
// parameter the release removed (sdkguide/). A tag never changes, so a rerun
// stores nothing new: the fingerprints dedupe.
package migrationguide

import (
	"context"
	"fmt"

	"github.com/patch-it-dev/patch-it-plugins/core/migrate"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
	"github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/sdkguide"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "anthropic", Name: "Anthropic", Category: plugin.CategoryAI, Homepage: "https://www.anthropic.com",
		Sources: []plugin.Source{{
			ID: "python-sdk", Layer: plugin.LayerSDK, Page: "providers/anthropic.md",
			Watches: "Request parameters each new major of the Python SDK removes from `messages.create()` and friends, from MIGRATION.md at the major's tag",
			URLs:    []string{"https://github.com/anthropics/anthropic-sdk-python/blob/main/MIGRATION.md", "https://pypi.org/project/anthropic/"},
			Plugin:  Plugin{Guide: sdkguide.Anthropic},
		}},
	})
}

// Plugin reads an SDK's migration guide: at the first release of each major
// (ListVersions, ReadAt, Extract), or, with a pinned Guide, that one major.
type Plugin struct {
	Guide sdkguide.Guide
}

func (p Plugin) Name() string { return p.Guide.API }

// Run reads the pinned major's guide, or the newest major's.
func (p Plugin) Run(ctx context.Context, env plugin.Env) (plugin.Result, error) {
	g := p.Guide
	v := plugin.Version{ID: g.Version(), Ref: "v" + g.Version()}
	if g.Major == 0 {
		vs, err := p.ListVersions(ctx, env)
		if err != nil {
			return plugin.Result{}, err
		}
		v = plugin.Version{}
		for _, c := range vs {
			if firstOfMajor(c.ID) {
				v = c
			}
		}
		if v.ID == "" {
			return plugin.Result{}, fmt.Errorf("%s: no major release listed", g.API)
		}
	}
	to, err := p.ReadAt(ctx, env, v)
	if err != nil {
		return plugin.Result{}, err
	}
	res, err := p.Extract(ctx, env, plugin.Snapshot{}, to)
	if err != nil {
		return plugin.Result{}, err
	}
	res.Baselines = []plugin.Spec{plugin.Baseline(g.API, v.ID, v.Ref, to.SourceURL, to.Content, false, env.Now().UTC())}
	return res, nil
}

// ListVersions is every final release of the SDK on PyPI, oldest first; a
// pinned Guide has its one major.
func (p Plugin) ListVersions(ctx context.Context, env plugin.Env) ([]plugin.Version, error) {
	g := p.Guide
	if g.Major != 0 {
		return []plugin.Version{{ID: g.Version(), Ref: "v" + g.Version()}}, nil
	}
	vs, err := versions.PyPIReleases(ctx, env.Get, g.API)
	if err != nil {
		return nil, err
	}
	for i := range vs {
		vs[i].Ref = "v" + vs[i].ID
	}
	return vs, nil
}

// ReadAt fetches the guide at the release's tag, when the release is the
// first of its major (X.0.0): only a new major has a guide to read. Any other
// release reads as nothing (a snapshot with no content).
func (p Plugin) ReadAt(ctx context.Context, env plugin.Env, v plugin.Version) (plugin.Snapshot, error) {
	if !firstOfMajor(v.ID) {
		return plugin.Snapshot{Version: v}, nil
	}
	g := p.Guide.At(versions.Major(v.ID))
	page, err := env.Get(ctx, g.URL())
	if err != nil {
		return plugin.Snapshot{}, fmt.Errorf("fetching %s: %w", g.URL(), err)
	}
	return plugin.Snapshot{Version: v, Content: page, SourceURL: g.URL()}, nil
}

// Extract is an event per request parameter the to major removed, read from
// its guide, with previous_version from's release. A release that isn't a new
// major has no guide and no events.
func (p Plugin) Extract(_ context.Context, env plugin.Env, from, to plugin.Snapshot) (plugin.Result, error) {
	if to.Content == nil {
		return plugin.Result{Label: to.Version.ID}, nil
	}
	g := p.Guide.At(versions.Major(to.Version.ID))
	removals, err := sdkguide.Parse(to.Content, g)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("reading %s: %w", g.URL(), err)
	}
	events, errs := sdkguide.Events(g, removals, env.Now().UTC())
	for i := range events {
		events[i].PreviousVersion = from.Version.ID
	}
	converted, invalid := migrate.Fresh(events)
	return plugin.Result{Label: g.URL(), Events: converted, Rejected: append(errs, invalid...)}, nil
}

// firstOfMajor is true for X.0.0, the release a major's guide is tagged at.
func firstOfMajor(v string) bool {
	m := versions.Major(v)
	return m > 0 && v == fmt.Sprintf("%d.0.0", m)
}

var _ plugin.Versioned = Plugin{}
