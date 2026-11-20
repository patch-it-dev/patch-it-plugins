// Package migrationguide is the source plugin for an SDK's migration guide,
// read at the release tag of the major it describes: an event per request
// parameter the release removed (sdkguide/). The tag never changes, so
// a rerun stores nothing new: the fingerprints dedupe.
package migrationguide

import (
	"context"
	"fmt"

	"github.com/patch-it-dev/patch-it-plugins/core/migrate"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/sdkguide"
)

func init() {
	plugin.Register(Plugin{Guide: sdkguide.AnthropicV1})
}

// Plugin reads one SDK's guide to one major release.
type Plugin struct {
	Guide sdkguide.Guide
}

func (p Plugin) Name() string { return p.Guide.API }

func (p Plugin) Run(ctx context.Context, env plugin.Env) (plugin.Result, error) {
	g := p.Guide
	page, err := env.Get(ctx, g.URL())
	if err != nil {
		return plugin.Result{}, fmt.Errorf("fetching %s: %w", g.URL(), err)
	}
	removals, err := sdkguide.Parse(page, g)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("reading %s: %w", g.URL(), err)
	}
	now := env.Now().UTC()
	events, errs := sdkguide.Events(g, removals, now)
	converted, invalid := migrate.Fresh(events)
	return plugin.Result{
		Label:     g.URL(),
		Events:    converted,
		Rejected:  append(errs, invalid...),
		Baselines: []plugin.Spec{plugin.Baseline(g.API, g.Version(), "v"+g.Version(), g.URL(), page, false, now)},
	}, nil
}
