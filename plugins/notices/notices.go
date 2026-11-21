// Package notices is the kind for a live page of dated notices: something
// stops, or is deprecated, on a date. A model provider's deprecations table
// (retire/), a vendor's RSS or Atom changelog feed (feed/), a vendor's API
// end-of-life table (eol/). Each Page reader knows how its page reads; the
// steps around it are written once, here: fetch the live page, stop when it
// hasn't changed since the last run, read it, snapshot it in the Wayback
// Machine and check the snapshot says the same (kit.Cite), make the events
// from the snapshot so each cites what it quotes, and save the baseline.
//
// Two options a reader turns on with a method: ShapeGuard (a read with fewer
// than 90% of the last one's notices is an error: the page changed shape),
// and ArchiveWhenBlocked (a live page that answers 401 or 403 is read from a
// fresh Wayback Machine snapshot instead).
package notices

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/kit"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
	"github.com/patch-it-dev/patch-it-plugins/plugins/notices/eol"
	"github.com/patch-it-dev/patch-it-plugins/plugins/notices/feed"
	"github.com/patch-it-dev/patch-it-plugins/plugins/notices/retire"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "openai", Name: "OpenAI", Category: plugin.CategoryAI, Homepage: "https://openai.com",
		Sources: []plugin.Source{{
			ID: "models", Layer: plugin.LayerAIModels, Page: "providers/openai-models.md",
			Watches:  "Model deprecations and shutdown dates (Chat Completions and Responses text models)",
			URLs:     []string{"https://developers.openai.com/api/docs/deprecations"},
			Packages: []string{"pypi:openai", "npm:openai"},
			Plugin:   Plugin{Page: retire.OpenAI, Archiver: kit.Wayback{}},
		}},
	})
	plugin.RegisterProvider(plugin.Provider{
		ID: "anthropic", Name: "Anthropic", Category: plugin.CategoryAI, Homepage: "https://www.anthropic.com",
		Sources: []plugin.Source{{
			ID: "models", Layer: plugin.LayerAIModels, Page: "providers/anthropic-models.md",
			Watches:  "Model deprecations and retirement dates (Messages API models)",
			URLs:     []string{"https://platform.claude.com/docs/en/about-claude/model-deprecations"},
			Packages: []string{"pypi:anthropic", "npm:@anthropic-ai/sdk"},
			Plugin:   Plugin{Page: retire.Anthropic, Archiver: kit.Wayback{}},
		}},
	})
	plugin.RegisterProvider(plugin.Provider{
		ID: "github", Name: "GitHub", Category: plugin.CategoryDevTools, Homepage: "https://github.com",
		Sources: []plugin.Source{{
			ID: "actions", Layer: plugin.LayerCI, Page: "providers/github-actions.md",
			Watches: "GitHub Actions retirements, such as runner images and the `runs-on` labels they remove",
			URLs:    []string{"https://github.blog/changelog/label/actions/feed/"},
			Plugin:  Plugin{Page: feed.GitHubActions, Archiver: kit.Wayback{}},
		}},
	})
	plugin.RegisterProvider(plugin.Provider{
		ID: "google", Name: "Google", Category: plugin.CategoryAI, Homepage: "https://cloud.google.com",
		Sources: []plugin.Source{{
			ID: "vertex-ai-models", Layer: plugin.LayerAIModels, Page: "providers/vertex-ai-models.md",
			Watches: "Generative AI model deprecations on Vertex AI",
			URLs:    []string{"https://docs.cloud.google.com/feeds/generative-ai-on-vertex-ai-release-notes.xml"},
			Plugin:  Plugin{Page: feed.VertexAIModels, Archiver: kit.Wayback{}},
		}},
	})
	plugin.RegisterProvider(plugin.Provider{
		ID: "cloudflare", Name: "Cloudflare", Category: plugin.CategoryInfrastructure, Homepage: "https://www.cloudflare.com",
		Sources: []plugin.Source{{
			ID: "api", Layer: plugin.LayerAPI, Page: "providers/cloudflare.md",
			Watches:  "REST API operations Cloudflare deprecates, breaking on their end-of-life date when it gives one",
			URLs:     []string{"https://developers.cloudflare.com/fundamentals/api/reference/deprecations/index.xml"},
			Packages: []string{"pypi:cloudflare", "npm:cloudflare"},
			Plugin:   Plugin{Page: feed.CloudflareAPI, Archiver: kit.Wayback{}},
		}},
	})
	plugin.RegisterProvider(plugin.Provider{
		ID: "salesforce", Name: "Salesforce", Category: plugin.CategoryCRM, Homepage: "https://www.salesforce.com",
		Sources: []plugin.Source{{
			ID: "rest-api", Layer: plugin.LayerAPI, Page: "providers/salesforce.md",
			Watches: "REST API version ranges deprecated and retired, from Salesforce's own API end-of-life policy page",
			URLs:    []string{"https://developer.salesforce.com/docs/platform/api-rest/guide/api-rest-eol.html"},
			Plugin:  Plugin{Page: eol.Salesforce, Archiver: kit.Wayback{}},
		}},
	})
}

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
	// notices that failed to become events.
	Events(found kit.Notices, citeURL string, now time.Time) ([]event.Event, []error)
}

// shapeGuarded is a Page whose read with far fewer notices than the last
// is an error.
type shapeGuarded interface{ ShapeGuard() bool }

// archivedWhenBlocked is a Page read from a fresh Wayback Machine snapshot
// when the live page refuses a plain fetch.
type archivedWhenBlocked interface{ ArchiveWhenBlocked() bool }

// Plugin reads one page.
type Plugin struct {
	Page     Page
	Archiver kit.Archiver
}

func (p Plugin) Name() string { return p.Page.Slug() }

// Run stores an event per notice on the page. The events are read from the
// snapshot itself, so every excerpt is on the page it cites; a snapshot that
// doesn't hold what the live page says (the page changed in between) stores
// nothing this run. An unchanged page since the last run costs one GET (a
// 304 with the HTTP cache) and no snapshot.
func (p Plugin) Run(ctx context.Context, env plugin.Env) (plugin.Result, error) {
	pg := p.Page
	api, url := pg.Slug(), pg.PageURL()
	live, archived, err := p.fetch(ctx, env, url)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("fetching %s: %w", url, err)
	}
	hash := plugin.SHA256(live)
	last, same, err := kit.Unchanged(env, api, live)
	if err != nil {
		return plugin.Result{}, err
	}
	if same {
		log.Printf("%s: no change since %s", api, last.SourceURL)
		return plugin.Result{}, nil
	}
	want, err := pg.Find(live)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("reading %s: %w", url, err)
	}
	// A provider renaming a column makes its table unreadable, and the run
	// would quietly read fewer notices. Past notices stay on the page, so a
	// real drop is rare: fail loudly instead of storing a partial list.
	if g, ok := pg.(shapeGuarded); ok && g.ShapeGuard() && last != nil {
		if before, err := pg.Find(last.Content); err == nil && want.Count()*10 < before.Count()*9 {
			return plugin.Result{}, fmt.Errorf("%s now lists %d notices, %d at %s: the page's shape may have changed", url, want.Count(), before.Count(), last.SourceURL)
		}
	}

	snapURL, snap, got := archived, live, want
	if snapURL == "" {
		if snapURL, snap, got, err = kit.Cite(ctx, p.Archiver, url, pg.Find, want); err != nil {
			return plugin.Result{}, err
		}
	}

	now := env.Now().UTC()
	events, rejected := pg.Events(got, snapURL, now)
	// The hash of the live page, so the next run compares like with like.
	baseline := plugin.Baseline(api, now.Format("2006-01-02"), snapURL, snapURL, snap, now)
	baseline.SHA256 = hash
	return plugin.Result{Label: snapURL, Events: events, Rejected: rejected, Baselines: []plugin.Spec{baseline}}, nil
}

// fetch reads the live page. When the page's reader asks for it
// (ArchiveWhenBlocked) and the provider's own site blocks a plain fetch with
// 401 or 403 (Salesforce's docs site, every time, checked 2026-11-20;
// confirmed the same day that a real Wayback Machine save-page-now capture
// of that page parses to the same findings as the live page would), it
// falls back to a fresh Wayback Machine snapshot, taken right now by the
// archive's own crawler rather than patch-it's, and returns that content
// with its own permanent URL in archivedURL: it needs no further citation
// step, having come from the archive already.
//
// Only 401 and 403 fall back. A 404 or 410 means the page moved, not that
// it's blocked: asking the archive to snapshot a page that's gone would
// waste the call and hide the real cause behind the archive's own error.
// A 429 or a 5xx is returned as is: retrying it against the archive would
// hide a transient failure as a real page, not fix one.
func (p Plugin) fetch(ctx context.Context, env plugin.Env, pageURL string) (body []byte, archivedURL string, err error) {
	body, err = env.Get(ctx, pageURL)
	if err == nil {
		return body, "", nil
	}
	b, ok := p.Page.(archivedWhenBlocked)
	var se *fetch.StatusError
	if !ok || !b.ArchiveWhenBlocked() || !errors.As(err, &se) || (se.StatusCode != 401 && se.StatusCode != 403) {
		return nil, "", err
	}
	archivedURL, body, archErr := p.Archiver.Snapshot(ctx, pageURL)
	if archErr != nil {
		return nil, "", fmt.Errorf("fetching it directly failed (%w), and so did asking the Wayback Machine to: %w", err, archErr)
	}
	return body, archivedURL, nil
}

// A living page has no versions of its own, so its history is the Web
// Archive's: every distinct capture is a version, dated (I19.11). A backfill
// reads the page at each capture and extracts each consecutive pair, the
// same events a run on that day would have stored, each citing its capture.

// ListVersions is every distinct Web Archive capture of the page, oldest
// first (versions.WaybackCaptures); the caller keeps the ones in its window.
func (p Plugin) ListVersions(ctx context.Context, env plugin.Env) ([]plugin.Version, error) {
	return versions.WaybackCaptures(ctx, env.Get, p.Page.PageURL(), time.Time{}, time.Time{})
}

// ReadAt reads the page as the capture v holds it: the capture's original
// bytes (v.Ref), cited by its address (v.URL).
func (p Plugin) ReadAt(ctx context.Context, env plugin.Env, v plugin.Version) (plugin.Snapshot, error) {
	body, err := env.Get(ctx, v.Ref)
	if err != nil {
		return plugin.Snapshot{}, fmt.Errorf("fetching %s as captured on %s: %w", p.Page.PageURL(), v.ID, err)
	}
	return plugin.Snapshot{Version: v, Content: body, SourceURL: cmp.Or(v.URL, v.Ref)}, nil
}

// Extract is an event per notice on the page at to that wasn't at from
// (every one when from is zero), citing to's capture. The events keep the
// notice's own version (its date or API version), so previous_version stays
// empty: from is an earlier day of the page, not an earlier release.
func (p Plugin) Extract(_ context.Context, env plugin.Env, from, to plugin.Snapshot) (plugin.Result, error) {
	now := env.Now().UTC()
	events, rejected, err := p.events(to, now)
	if err != nil {
		return plugin.Result{}, err
	}
	if !from.IsZero() {
		before, _, err := p.events(from, now)
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
	return plugin.Result{Label: to.SourceURL, Events: events, Rejected: rejected}, nil
}

// events reads one snapshot's notices into events citing it.
func (p Plugin) events(s plugin.Snapshot, now time.Time) ([]event.Event, []error, error) {
	found, err := p.Page.Find(s.Content)
	if err != nil {
		return nil, nil, fmt.Errorf("reading %s: %w", s.SourceURL, err)
	}
	events, rejected := p.Page.Events(found, s.SourceURL, now)
	return events, rejected, nil
}

var _ plugin.Versioned = Plugin{}
