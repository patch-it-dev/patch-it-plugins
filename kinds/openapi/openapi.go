// Package openapi is the kind for an API that publishes an OpenAPI spec per
// release (a tag, a release, or the commits that changed the spec): every
// release is listed, each consecutive pair diffed (ListVersions, ReadAt,
// Extract), the changes normalized into events, and migration hints
// attached from the API's changelog when it has one.
package openapi

import (
	"context"
	"fmt"
	"log"

	"github.com/patch-it-dev/patch-it-plugins/core/diff"
	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/hint"
	"github.com/patch-it-dev/patch-it-plugins/core/normalize"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/adapter"
)

// RequestSide keeps the changes to what callers send. GitLab's response
// schemas are generated from entity docs it is still backfilling: between
// v18.11.0-ee and v19.0.0-ee, 280 response properties went from the generator's
// default "string" to their real type and 74 schemas were folded into others,
// while the API returned the same JSON. That reads as ~360 breaking changes
// nobody would hit, so response-side changes are dropped, not guessed at.
func RequestSide(c diff.Change) bool { return c.Side == event.SideRequest }

// Plugin runs one API's spec adapter.
type Plugin struct {
	Adapter adapter.Adapter
	// Hints fills migration_hint from the API's own changelog. Optional: nil,
	// or a changelog that can't be reached, leaves the normalizer's hints.
	Hints hint.Source
	// Keep, when set, drops the changes it returns false for, before they
	// become events: for a spec whose structure changes more often than its
	// API does. Nil keeps every change.
	Keep func(diff.Change) bool
	// Diff adapts the differ to the spec's conventions. The zero value is
	// right for Stripe's.
	Diff diff.Options
	// RemovalsNeedANewVersion drops operation removals between two specs of
	// the same version: for an API whose spec is fixed up on a release
	// branch after the release, where a removal there is a correction to the
	// spec, not a release removing an API (Elasticsearch removes APIs in a
	// new minor or major, never in a patch).
	RemovalsNeedANewVersion bool
}

func (p Plugin) Name() string { return p.Adapter.API() }

// ListVersions is every release of the spec, oldest first, when the adapter
// can list them (adapter.Lister); otherwise only the newest (LatestTag).
func (p Plugin) ListVersions(ctx context.Context, _ plugin.Env) ([]plugin.Version, error) {
	if l, ok := p.Adapter.(adapter.Lister); ok {
		return l.Versions(ctx)
	}
	tag, err := p.Adapter.LatestTag(ctx)
	if err != nil {
		return nil, fmt.Errorf("finding latest tag: %w", err)
	}
	return []plugin.Version{{ID: tag, Ref: tag}}, nil
}

// ReadAt fetches the spec at v.Ref. The snapshot's version ID is the version
// the spec itself declares (what events carry), which may differ from the tag;
// for an adapter that lists the specs' own versions (SpecVersioned) it must
// be v's.
func (p Plugin) ReadAt(ctx context.Context, _ plugin.Env, v plugin.Version) (plugin.Snapshot, error) {
	s, err := p.Adapter.Fetch(ctx, v.Ref)
	if err != nil {
		return plugin.Snapshot{}, fmt.Errorf("fetching %s %s: %w", p.Adapter.API(), v.Ref, err)
	}
	if _, ok := p.Adapter.(adapter.SpecVersioned); ok && s.Version != v.ID {
		return plugin.Snapshot{}, fmt.Errorf("%s %s: the spec at %s declares %s: not published there yet", p.Adapter.API(), v.ID, v.Ref, s.Version)
	}
	v.ID = s.Version
	return plugin.Snapshot{Version: v, Content: s.Content, SourceURL: s.SourceURL}, nil
}

// Extract diffs two snapshots of the spec into events, with to's version and
// from's as previous_version. A zero from is an empty spec before the first.
func (p Plugin) Extract(ctx context.Context, env plugin.Env, from, to plugin.Snapshot) (plugin.Result, error) {
	var oldParsed *diff.Spec
	if !from.IsZero() {
		var err error
		if oldParsed, err = diff.Parse(from.Content); err != nil {
			return plugin.Result{}, fmt.Errorf("parsing the %s spec at %s: %w", p.Adapter.API(), from.Version.ID, err)
		}
	}
	newParsed, err := diff.Parse(to.Content)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("parsing the %s spec at %s: %w", p.Adapter.API(), to.Version.ID, err)
	}
	events, errs := p.events(oldParsed, newParsed, from.Content, to.Content, normalize.Source{
		API:             p.Adapter.API(),
		Version:         to.Version.ID,
		PreviousVersion: from.Version.ID,
		NewSpecURL:      to.SourceURL,
		OldSpecURL:      from.SourceURL,
		DetectedAt:      env.Now().UTC(),
	})
	p.attachHints(ctx, to.Version.ID, events)
	return plugin.Result{Events: events, Rejected: errs}, nil
}

var _ plugin.Versioned = Plugin{}

// events diffs two parsed specs into events, applying Keep, and pointing each
// event's source_pointer into the published file when the adapter reshaped it.
func (p Plugin) events(oldParsed, newParsed *diff.Spec, oldContent, newContent []byte, src normalize.Source) ([]event.Event, []error) {
	changes := diff.ChangesWith(oldParsed, newParsed, p.Diff)
	if p.Keep != nil {
		kept := changes[:0:0]
		for _, c := range changes {
			if p.Keep(c) {
				kept = append(kept, c)
			}
		}
		if dropped := len(changes) - len(kept); dropped > 0 {
			log.Printf("%s %s: %d of %d changes dropped by the source's filter", src.API, src.Version, dropped, len(changes))
		}
		changes = kept
	}
	if p.RemovalsNeedANewVersion && src.Version == src.PreviousVersion {
		kept := changes[:0:0]
		for _, c := range changes {
			if c.Kind != diff.OperationRemoved {
				kept = append(kept, c)
			}
		}
		if dropped := len(changes) - len(kept); dropped > 0 {
			log.Printf("%s %s: %d removals dropped: same version as the last spec", src.API, src.Version, dropped)
		}
		changes = kept
	}
	events, errs := normalize.Events(changes, src)
	rp, ok := p.Adapter.(adapter.Reshaper)
	if !ok || len(events) == 0 {
		return events, errs
	}
	oldPtr, newPtr := rp.SourcePointer(oldContent), rp.SourcePointer(newContent)
	for i := range events {
		e := &events[i]
		// normalize cites the old spec only for evidence that lives there.
		if e.SourceURL == src.OldSpecURL && e.SourceURL != src.NewSpecURL {
			e.SourcePointer = oldPtr(e.SourcePointer)
		} else {
			e.SourcePointer = newPtr(e.SourcePointer)
		}
	}
	return events, errs
}

// attachHints looks the events up in the API's changelog for their version.
// Never fails the run: a hint is help for the fix, not provenance for the event.
func (p Plugin) attachHints(ctx context.Context, version string, events []event.Event) {
	if p.Hints == nil || len(events) == 0 {
		return
	}
	api := p.Adapter.API()
	ix, err := p.Hints.Index(ctx, version)
	if err != nil {
		log.Printf("%s %s: migration hints unavailable: %v", api, version, err)
		return
	}
	n := ix.Attach(events)
	log.Printf("%s %s: migration hints on %d of %d events, from %d changelog entries", api, version, n, len(events), len(ix.Entries))
}

// DeprecationsOnly keeps operation deprecations, and removals of operations
// the old spec had marked deprecated (by opts' reading of "marked"):
// everything else is dropped. For a spec whose removals and property changes
// were mostly not changes to the API (vercel, supabase).
func DeprecationsOnly(opts diff.Options) func(diff.Change) bool {
	return func(c diff.Change) bool {
		switch c.Kind {
		case diff.OperationDeprecated:
			return true
		case diff.OperationRemoved:
			op, _ := c.Node.(map[string]any)
			return opts.OperationDeprecated(op)
		default:
			return false
		}
	}
}
