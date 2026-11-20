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

	"github.com/patch-it-dev/patch-it-plugins/core/diff"
	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/hint"
	"github.com/patch-it-dev/patch-it-plugins/core/migrate"
	"github.com/patch-it-dev/patch-it-plugins/core/normalize"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/plugins/openapi/adapter"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "stripe", Name: "Stripe", Category: plugin.CategoryPayments, Homepage: "https://stripe.com",
		Sources: []plugin.Source{{
			ID: "api", Layer: plugin.LayerAPI, Page: "providers/stripe.md",
			Watches:  "REST API changes between OpenAPI spec releases (removed and renamed endpoints, parameters and fields), with migration hints from Stripe's API changelog",
			URLs:     []string{"https://github.com/stripe/openapi"},
			Packages: []string{"pypi:stripe", "npm:stripe"},
			Plugin:   Plugin{Adapter: adapter.Stripe{}, Hints: hint.Stripe{}},
		}},
	})
	// No Hints yet: Datadog's changelog isn't Stripe's per-parameter table
	// shape, so there's no hint.Source for it to read.
	plugin.RegisterProvider(plugin.Provider{
		ID: "datadog", Name: "Datadog", Category: plugin.CategoryMonitoring, Homepage: "https://www.datadoghq.com",
		Sources: []plugin.Source{{
			ID: "api", Layer: plugin.LayerAPI, Page: "providers/datadog.md",
			Watches:  "API v2 changes between OpenAPI spec releases",
			URLs:     []string{"https://github.com/DataDog/datadog-api-client-python"},
			Packages: []string{"pypi:datadog-api-client", "npm:@datadog/datadog-api-client"},
			Plugin:   Plugin{Adapter: adapter.Datadog{}},
		}},
	})
	// No Hints: GitLab's deprecations are prose per feature, not a changelog
	// keyed by API version, so there's nothing for hint.Source to look up.
	plugin.RegisterProvider(plugin.Provider{
		ID: "gitlab", Name: "GitLab", Category: plugin.CategoryDevTools, Homepage: "https://about.gitlab.com",
		Sources: []plugin.Source{{
			ID: "rest-api", Layer: plugin.LayerAPI, Page: "providers/gitlab.md",
			Watches: "REST API changes between releases, request side only (what callers send)",
			URLs:    []string{"https://gitlab.com/gitlab-org/gitlab/-/tree/master/doc/api/openapi"},
			Plugin:  Plugin{Adapter: adapter.GitLab{}, Keep: requestSide},
		}},
	})
	// No Hints yet: GitHub's per-version CHANGELOG.md is prose with no table
	// at all, so there's nothing for hint.Source to key on. (GitHub's Actions
	// source is registered by plugins/notices.)
	plugin.RegisterProvider(plugin.Provider{
		ID: "github", Name: "GitHub", Category: plugin.CategoryDevTools, Homepage: "https://github.com",
		Sources: []plugin.Source{{
			ID: "rest-api", Layer: plugin.LayerAPI, Page: "providers/github.md",
			Watches: "REST API changes between dated API versions",
			URLs:    []string{"https://github.com/github/rest-api-description"},
			Plugin:  Plugin{Adapter: adapter.GitHub{}},
		}},
	})
	// No Hints: Vercel's changelog is product news, not keyed by API change.
	// Only operation deprecations and removals are kept: see vercelKeep.
	plugin.RegisterProvider(plugin.Provider{
		ID: "vercel", Name: "Vercel", Category: plugin.CategoryHosting, Homepage: "https://vercel.com",
		Sources: []plugin.Source{{
			ID: "api", Layer: plugin.LayerAPI, Page: "providers/vercel.md",
			Watches:  "REST API operations deprecated, and deprecated operations removed, between SDK releases of the spec",
			URLs:     []string{"https://github.com/vercel/sdk"},
			Packages: []string{"npm:@vercel/sdk"},
			Plugin:   Plugin{Adapter: adapter.Vercel{}, Diff: vercelDiff, Keep: vercelKeep},
		}},
	})
	// No Hints: Supabase's changelog is platform news, not keyed by
	// operation. Only operation deprecations and removals: see supabase.go.
	plugin.RegisterProvider(plugin.Provider{
		ID: "supabase", Name: "Supabase", Category: plugin.CategoryDatabases, Homepage: "https://supabase.com",
		Sources: []plugin.Source{{
			ID: "management-api", Layer: plugin.LayerAPI, Page: "providers/supabase.md",
			Watches: "Management API operations deprecated, and deprecated operations removed, between commits of the spec",
			URLs:    []string{"https://github.com/supabase/supabase"},
			Plugin:  Plugin{Adapter: adapter.Supabase{}, Diff: supabaseDiff, Keep: supabaseKeep},
		}},
	})
	// No Hints: Algolia's changelog is a hosted product feed, not keyed by
	// operation. One source per API spec: see algolia.go.
	algolia := plugin.Provider{ID: "algolia", Name: "Algolia", Category: plugin.CategorySearch, Homepage: "https://www.algolia.com"}
	for _, spec := range []struct{ name, title string }{{"search", "Search"}, {"ingestion", "Ingestion"}, {"recommend", "Recommend"}} {
		algolia.Sources = append(algolia.Sources, plugin.Source{
			ID: spec.name + "-api", Layer: plugin.LayerAPI, Page: "providers/algolia.md",
			Watches:  spec.title + " API operations deprecated, and deprecated operations removed, between releases of the spec",
			URLs:     []string{"https://github.com/algolia/api-clients-automation", "https://github.com/algolia/api-clients-automation/blob/main/specs/bundled/" + spec.name + ".yml"},
			Packages: []string{"pypi:algoliasearch", "npm:algoliasearch"},
			Plugin:   Plugin{Adapter: adapter.Algolia{Spec: spec.name}, Diff: algoliaDiff, Keep: algoliaKeep},
		})
	}
	plugin.RegisterProvider(algolia)
	// No Hints: Elastic's breaking-changes page is prose per area, not keyed
	// by operation. See elasticsearch.go.
	plugin.RegisterProvider(plugin.Provider{
		ID: "elastic", Name: "Elastic", Category: plugin.CategorySearch, Homepage: "https://www.elastic.co",
		Sources: []plugin.Source{{
			ID: "elasticsearch-api", Layer: plugin.LayerAPI, Page: "providers/elasticsearch.md",
			Watches:  "Elasticsearch REST API operations deprecated, and deprecated operations removed, between released minor versions of the spec",
			URLs:     []string{"https://github.com/elastic/elasticsearch-specification"},
			Packages: []string{"pypi:elasticsearch", "npm:@elastic/elasticsearch"},
			Plugin:   Plugin{Adapter: adapter.Elasticsearch{}, Diff: elasticsearchDiff, Keep: elasticsearchKeep, RemovalsNeedANewVersion: true},
		}},
	})
	// No Hints: Sentry's changelog is product news, not keyed by operation.
	// Operation deprecations only, most said in the description: see
	// sentry.go.
	plugin.RegisterProvider(plugin.Provider{
		ID: "sentry", Name: "Sentry", Category: plugin.CategoryMonitoring, Homepage: "https://sentry.io",
		Sources: []plugin.Source{{
			ID: "api", Layer: plugin.LayerAPI, Page: "providers/sentry.md",
			Watches: "REST API operations deprecated between releases of the spec, most of them in the operation's description",
			URLs:    []string{"https://github.com/getsentry/sentry/releases"},
			Plugin:  Plugin{Adapter: adapter.Sentry{}, Diff: sentryDiff, Keep: sentryKeep},
		}},
	})
	// No Hints: Neon's changelog is a weekly digest, not keyed by operation.
	// Only operation deprecations and removals: see neon.go.
	plugin.RegisterProvider(plugin.Provider{
		ID: "neon", Name: "Neon", Category: plugin.CategoryDatabases, Homepage: "https://neon.com",
		Sources: []plugin.Source{{
			ID: "api", Layer: plugin.LayerAPI, Page: "providers/neon.md",
			Watches:  "Neon API operations deprecated, and deprecated operations removed, between commits of the spec",
			URLs:     []string{"https://github.com/neondatabase/neon-pkgs"},
			Packages: []string{"npm:@neondatabase/api-client"},
			Plugin:   Plugin{Adapter: adapter.Neon{}, Diff: neonDiff, Keep: neonKeep},
		}},
	})
	// No Hints: Convex has no API changelog; its news feed is product news.
	// Only operation deprecations and removals: see convex.go.
	plugin.RegisterProvider(plugin.Provider{
		ID: "convex", Name: "Convex", Category: plugin.CategoryDatabases, Homepage: "https://www.convex.dev",
		Sources: []plugin.Source{{
			ID: "deployment-api", Layer: plugin.LayerAPI, Page: "providers/convex.md",
			Watches: "Deployment API operations deprecated, and deprecated operations removed, between commits of the spec",
			URLs:    []string{"https://github.com/get-convex/convex-backend"},
			Plugin:  Plugin{Adapter: adapter.Convex{}, Diff: convexDiff, Keep: convexKeep},
		}},
	})
	// No Hints: PlanetScale's changelog is product news, not keyed by
	// operation. Operation deprecations only, said in the description: see
	// planetscale.go.
	plugin.RegisterProvider(plugin.Provider{
		ID: "planetscale", Name: "PlanetScale", Category: plugin.CategoryDatabases, Homepage: "https://planetscale.com",
		Sources: []plugin.Source{{
			ID: "api", Layer: plugin.LayerAPI, Page: "providers/planetscale.md",
			Watches: "API operations deprecated, and deprecated operations removed, between commits of the spec, said in the operation's description",
			URLs:    []string{"https://github.com/planetscale/terraform-provider-planetscale"},
			Plugin:  Plugin{Adapter: adapter.PlanetScale{}, Diff: planetscaleDiff, Keep: planetscaleKeep},
		}},
	})
	// No Hints: Appwrite's changelog is product news, not keyed by operation.
	// Operation deprecations only, SDK method renames left out: see
	// appwrite.go.
	plugin.RegisterProvider(plugin.Provider{
		ID: "appwrite", Name: "Appwrite", Category: plugin.CategoryHosting, Homepage: "https://appwrite.io",
		Sources: []plugin.Source{{
			ID: "api", Layer: plugin.LayerAPI, Page: "providers/appwrite.md",
			Watches:  "REST API operations a server SDK calls that are deprecated between commits of the spec, SDK method renames left out",
			URLs:     []string{"https://github.com/appwrite/specs"},
			Packages: []string{"pypi:appwrite", "npm:node-appwrite"},
			Plugin:   Plugin{Adapter: adapter.Appwrite{}, Diff: appwriteDiff, Keep: appwriteKeep},
		}},
	})
	// No Hints: Flagsmith's release notes are product news, not keyed by
	// operation. requestSide, same reason as GitLab's: a named request-body
	// schema shared by several operations (FeatureValue, SegmentOverride)
	// reads as its own removal, side "response" by the differ's default,
	// whether or not any operation that used it is actually gone.
	plugin.RegisterProvider(plugin.Provider{
		ID: "flagsmith", Name: "Flagsmith", Category: plugin.CategoryDevTools, Homepage: "https://www.flagsmith.com",
		Sources: []plugin.Source{{
			ID: "api", Layer: plugin.LayerAPI, Page: "providers/flagsmith.md",
			Watches: "REST API changes between releases of Flagsmith's own OpenAPI spec (removed endpoints and request parameters)",
			URLs:    []string{"https://github.com/Flagsmith/flagsmith/blob/main/openapi.yaml"},
			Plugin:  Plugin{Adapter: adapter.Flagsmith{}, Keep: requestSide},
		}},
	})
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

// Run diffs the newest release against whatever was last stored: the newest
// version (LatestTag), ReadAt it, and Extract against the last baseline.
func (p Plugin) Run(ctx context.Context, env plugin.Env) (plugin.Result, error) {
	a := p.Adapter
	tag, err := a.LatestTag(ctx)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("finding latest tag: %w", err)
	}
	to, err := p.ReadAt(ctx, env, plugin.Version{ID: tag, Ref: tag})
	if err != nil {
		return plugin.Result{}, err
	}
	last, err := env.Baselines.LastSpec(a.API())
	if err != nil {
		return plugin.Result{}, fmt.Errorf("loading last spec: %w", err)
	}
	if last != nil && last.SHA256 == plugin.SHA256(to.Content) {
		log.Printf("%s: no change (%s)", a.API(), tag)
		return plugin.Result{}, nil
	}
	var from plugin.Snapshot
	if last != nil {
		from = plugin.Snapshot{Version: plugin.Version{ID: last.Version, Ref: last.Tag}, Content: last.Content, SourceURL: last.SourceURL}
	}
	res, err := p.Extract(ctx, env, from, to)
	if err != nil {
		return plugin.Result{}, err
	}
	res.Label = tag
	res.Baselines = []plugin.Spec{plugin.Baseline(a.API(), to.Version.ID, tag, to.SourceURL, to.Content, false, env.Now())}
	return res, nil
}

// Replay diffs two given releases. Both specs are saved marked as replays, so
// the normal-run baseline never moves; the events are stored like any others:
// that's how a historical change becomes something the feed can serve.
func (p Plugin) Replay(ctx context.Context, env plugin.Env, from, to string) (plugin.Result, error) {
	old, err := p.ReadAt(ctx, env, plugin.Version{ID: from, Ref: from})
	if err != nil {
		return plugin.Result{}, err
	}
	next, err := p.ReadAt(ctx, env, plugin.Version{ID: to, Ref: to})
	if err != nil {
		return plugin.Result{}, err
	}
	res, err := p.Extract(ctx, env, old, next)
	if err != nil {
		return plugin.Result{}, err
	}
	now := env.Now()
	res.Label = to
	res.Baselines = []plugin.Spec{
		plugin.Baseline(p.Adapter.API(), old.Version.ID, from, old.SourceURL, old.Content, true, now),
		plugin.Baseline(p.Adapter.API(), next.Version.ID, to, next.SourceURL, next.Content, true, now),
	}
	return res, nil
}

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
// the spec itself declares (what events carry), which may differ from the tag.
func (p Plugin) ReadAt(ctx context.Context, _ plugin.Env, v plugin.Version) (plugin.Snapshot, error) {
	s, err := p.Adapter.Fetch(ctx, v.Ref)
	if err != nil {
		return plugin.Snapshot{}, fmt.Errorf("fetching %s %s: %w", p.Adapter.API(), v.Ref, err)
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
	converted, invalid := migrate.Fresh(events)
	return plugin.Result{Label: to.Version.Ref, Events: converted, Rejected: append(errs, invalid...)}, nil
}

var _ plugin.Versioned = Plugin{}

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
