package sdkguide

import (
	"context"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
)

// What the migration-guide kind (plugins/migrationguide) asks of a guide.

// Slug is the source id the events carry: "anthropic/python-sdk".
func (g Guide) Slug() string { return g.API }

// Releases is every final release of the SDK on PyPI, oldest first, each
// read at its v<version> tag; a pinned Guide has its one major.
func (g Guide) Releases(ctx context.Context, env plugin.Env) ([]plugin.Version, error) {
	if g.Major != 0 {
		return []plugin.Version{{ID: g.Version(), Ref: "v" + g.Version()}}, nil
	}
	vs, err := versions.PyPIReleases(ctx, env.Get, g.Package)
	if err != nil {
		return nil, err
	}
	for i := range vs {
		vs[i].Ref = "v" + vs[i].ID
	}
	return vs, nil
}

// GuideURL is the guide at v's tag when v is the first release of its major
// (X.0.0): only a new major has a guide to read. Any other release has none
// ("").
func (g Guide) GuideURL(_ context.Context, _ plugin.Env, v plugin.Version) (string, error) {
	if !versions.FirstOfMajor(v.ID) {
		return "", nil
	}
	return g.At(versions.Major(v.ID)).URL(), nil
}

// Read is an event per parameter the to major removed, read from its
// guide (at the major's tag: sourceURL), with previous_version from's
// release.
func (g Guide) Read(page []byte, _ string, from, to plugin.Version, now time.Time) (_ []event.Event, rejected []error, err error) {
	g = g.At(versions.Major(to.ID))
	removals, err := Parse(page, g)
	if err != nil {
		return nil, nil, err
	}
	events, errs := Events(g, removals, from.ID, now)
	return events, errs, nil
}
