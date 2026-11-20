// Package openapi is the source plugin for an API that publishes an OpenAPI
// spec per release (Stripe, Datadog, GitLab, GitHub, Vercel, Supabase,
// Algolia, Elasticsearch, Sentry, Neon, Convex, PlanetScale, Appwrite): fetch
// the newest spec, diff it against the last one seen, normalize the changes
// into events, and attach migration hints from the API's changelog when it
// has one. It can also replay two given releases.
package openapi

import (
	"context"
	"fmt"
	"log"

	"github.com/patch-it-dev/patch-it-plugins/adapter"
	"github.com/patch-it-dev/patch-it-plugins/diff"
	"github.com/patch-it-dev/patch-it-plugins/event"
	"github.com/patch-it-dev/patch-it-plugins/hint"
	"github.com/patch-it-dev/patch-it-plugins/migrate"
	"github.com/patch-it-dev/patch-it-plugins/normalize"
	"github.com/patch-it-dev/patch-it-plugins/plugin"
)

func init() {
	plugin.Register(Plugin{Adapter: adapter.Stripe{}, Hints: hint.Stripe{}})
	// No Hints yet: Datadog's changelog isn't Stripe's per-parameter table
	// shape, so there's no hint.Source for it to read.
	plugin.Register(Plugin{Adapter: adapter.Datadog{}})
	// No Hints: GitLab's deprecations are prose per feature, not a changelog
	// keyed by API version, so there's nothing for hint.Source to look up.
	plugin.Register(Plugin{Adapter: adapter.GitLab{}, Keep: requestSide})
	// No Hints yet: GitHub's per-version CHANGELOG.md is prose with no table
	// at all, so there's nothing for hint.Source to key on.
	plugin.Register(Plugin{Adapter: adapter.GitHub{}})
	// No Hints: Vercel's changelog is product news, not keyed by API change.
	// Only operation deprecations and removals are kept: see vercelKeep.
	plugin.Register(Plugin{Adapter: adapter.Vercel{}, Diff: vercelDiff, Keep: vercelKeep})
	// No Hints: Supabase's changelog is platform news, not keyed by
	// operation. Only operation deprecations and removals: see supabase.go.
	plugin.Register(Plugin{Adapter: adapter.Supabase{}, Diff: supabaseDiff, Keep: supabaseKeep})
	// No Hints: Algolia's changelog is a hosted product feed, not keyed by
	// operation. One plugin per API spec: see algolia.go.
	for _, spec := range []string{"search", "ingestion", "recommend"} {
		plugin.Register(Plugin{Adapter: adapter.Algolia{Spec: spec}, Diff: algoliaDiff, Keep: algoliaKeep})
	}
	// No Hints: Elastic's breaking-changes page is prose per area, not keyed
	// by operation. See elasticsearch.go.
	plugin.Register(Plugin{Adapter: adapter.Elasticsearch{}, Diff: elasticsearchDiff, Keep: elasticsearchKeep, RemovalsNeedANewVersion: true})
	// No Hints: Sentry's changelog is product news, not keyed by operation.
	// Operation deprecations only, most said in the description: see
	// sentry.go.
	plugin.Register(Plugin{Adapter: adapter.Sentry{}, Diff: sentryDiff, Keep: sentryKeep})
	// No Hints: Neon's changelog is a weekly digest, not keyed by operation.
	// Only operation deprecations and removals: see neon.go.
	plugin.Register(Plugin{Adapter: adapter.Neon{}, Diff: neonDiff, Keep: neonKeep})
	// No Hints: Convex has no API changelog; its news feed is product news.
	// Only operation deprecations and removals: see convex.go.
	plugin.Register(Plugin{Adapter: adapter.Convex{}, Diff: convexDiff, Keep: convexKeep})
	// No Hints: PlanetScale's changelog is product news, not keyed by
	// operation. Operation deprecations only, said in the description: see
	// planetscale.go.
	plugin.Register(Plugin{Adapter: adapter.PlanetScale{}, Diff: planetscaleDiff, Keep: planetscaleKeep})
	// No Hints: Appwrite's changelog is product news, not keyed by operation.
	// Operation deprecations only, SDK method renames left out: see
	// appwrite.go.
	plugin.Register(Plugin{Adapter: adapter.Appwrite{}, Diff: appwriteDiff, Keep: appwriteKeep})
}

// requestSide keeps the changes to what callers send. GitLab's response
// schemas are generated from entity docs it is still backfilling: between
// v18.11.0-ee and v19.0.0-ee, 280 response properties went from the generator's
// default "string" to their real type and 74 schemas were folded into others,
// while the API returned the same JSON. That reads as ~360 breaking changes
// nobody would hit, so response-side changes are dropped, not guessed at.
func requestSide(c diff.Change) bool { return c.Side == event.SideRequest }

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

// Run diffs the newest release against whatever was last stored.
func (p Plugin) Run(ctx context.Context, env plugin.Env) (plugin.Result, error) {
	a := p.Adapter
	tag, err := a.LatestTag(ctx)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("finding latest tag: %w", err)
	}
	newSpec, err := a.Fetch(ctx, tag)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("fetching %s %s: %w", a.API(), tag, err)
	}
	last, err := env.Baselines.LastSpec(a.API())
	if err != nil {
		return plugin.Result{}, fmt.Errorf("loading last spec: %w", err)
	}
	if last != nil && last.SHA256 == plugin.SHA256(newSpec.Content) {
		log.Printf("%s: no change (%s)", a.API(), tag)
		return plugin.Result{}, nil
	}

	var oldParsed *diff.Spec
	var previousVersion, oldSourceURL string
	if last != nil {
		oldParsed, err = diff.Parse(last.Content)
		if err != nil {
			return plugin.Result{}, fmt.Errorf("parsing stored spec: %w", err)
		}
		previousVersion = last.Version
		oldSourceURL = last.SourceURL
	}
	newParsed, err := diff.Parse(newSpec.Content)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("parsing fetched spec: %w", err)
	}

	var oldContent []byte
	if last != nil {
		oldContent = last.Content
	}
	events, errs := p.events(oldParsed, newParsed, oldContent, newSpec.Content, normalize.Source{
		API:             a.API(),
		Version:         newSpec.Version,
		PreviousVersion: previousVersion,
		NewSpecURL:      newSpec.SourceURL,
		OldSpecURL:      oldSourceURL,
		DetectedAt:      env.Now().UTC(),
	})
	p.attachHints(ctx, newSpec.Version, events)
	converted, invalid := migrate.Fresh(events)
	return plugin.Result{
		Label:     tag,
		Events:    converted,
		Rejected:  append(errs, invalid...),
		Baselines: []plugin.Spec{plugin.Baseline(a.API(), newSpec.Version, tag, newSpec.SourceURL, newSpec.Content, false, env.Now())},
	}, nil
}

// Replay diffs two given releases. Both specs are saved marked as replays, so
// the normal-run baseline never moves; the events are stored like any others:
// that's how a historical change becomes something the feed can serve.
func (p Plugin) Replay(ctx context.Context, env plugin.Env, from, to string) (plugin.Result, error) {
	a := p.Adapter
	oldSpec, err := a.Fetch(ctx, from)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("fetching %s %s: %w", a.API(), from, err)
	}
	newSpec, err := a.Fetch(ctx, to)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("fetching %s %s: %w", a.API(), to, err)
	}
	oldParsed, err := diff.Parse(oldSpec.Content)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("parsing %s: %w", from, err)
	}
	newParsed, err := diff.Parse(newSpec.Content)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("parsing %s: %w", to, err)
	}
	events, errs := p.events(oldParsed, newParsed, oldSpec.Content, newSpec.Content, normalize.Source{
		API:             a.API(),
		Version:         newSpec.Version,
		PreviousVersion: oldSpec.Version,
		NewSpecURL:      newSpec.SourceURL,
		OldSpecURL:      oldSpec.SourceURL,
		DetectedAt:      env.Now().UTC(),
	})
	p.attachHints(ctx, newSpec.Version, events)
	now := env.Now()
	converted, invalid := migrate.Fresh(events)
	return plugin.Result{
		Label:    to,
		Events:   converted,
		Rejected: append(errs, invalid...),
		Baselines: []plugin.Spec{
			plugin.Baseline(a.API(), oldSpec.Version, from, oldSpec.SourceURL, oldSpec.Content, true, now),
			plugin.Baseline(a.API(), newSpec.Version, to, newSpec.SourceURL, newSpec.Content, true, now),
		},
	}, nil
}

// events diffs two parsed specs into events, applying Keep, and pointing each
// event's source_pointer into the published file when the adapter reshaped it.
func (p Plugin) events(oldParsed, newParsed *diff.Spec, oldContent, newContent []byte, src normalize.Source) ([]event.ChangeEvent, []error) {
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
func (p Plugin) attachHints(ctx context.Context, version string, events []event.ChangeEvent) {
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
