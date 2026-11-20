// Package feed is the source plugin for a vendor's RSS or Atom changelog
// feed: one event per breaking or deprecation entry (or per name such an
// entry lists), quoting the line that says so. Rung 3 of the source ladder.
//
// A feed is prose, so each feed comes with a Reader that knows what its
// vendor's entries look like and which subject they describe. The rules are
// the same for every reader (precision over recall):
//
//   - An entry the vendor doesn't mark as breaking or deprecation (a category,
//     a heading) is skipped.
//   - A marked entry becomes an event only when the reader finds a typed
//     subject in it (a runner label, a model id in code) and a dated
//     sentence. Otherwise it is dropped, logged with why, never guessed.
//   - `breaking` needs the vendor to say the thing stops on a date ("will be
//     retired on", "will be shut down on"). Anything softer ("we recommend
//     updating before") is downgraded to `deprecation`.
//
// A feed is a live document, so the events cite a Wayback Machine snapshot
// of it, read back and checked against the live feed (kit.Cite), like the
// retirements plugin.
package feed

import (
	"context"
	"fmt"
	"log"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

func init() {
	plugin.Register(Plugin{Feed: GitHubActions, Archiver: kit.Wayback{}})
	plugin.Register(Plugin{Feed: VertexAIModels, Archiver: kit.Wayback{}})
	plugin.Register(Plugin{Feed: CloudflareAPI, Archiver: kit.Wayback{}})
}

// Feed is one vendor feed and how to read its entries.
type Feed struct {
	API    string // the events' api: "github-actions"
	Vendor string // for sentences: "GitHub"
	URL    string // the feed itself
	Read   Reader
}

// Reader turns one entry into findings: what it can name for sure, and what
// it drops. It returns nothing for an entry that isn't breaking or a
// deprecation.
type Reader func(e Entry) []Finding

// Finding is one change a reader found in an entry, or one it dropped.
type Finding struct {
	ChangeType  event.ChangeType // breaking or deprecation
	Version     string           // per the subject's type: the removal or shutdown date
	Subject     event.Subject
	Description string
	Hint        string
	Excerpt     string // the entry's own lines that say so
	Entry       string // the entry's link, for the log
	Dropped     string // set: the entry was marked but this is why no event
}

// Findings reads every entry of a feed.
func (f Feed) Findings(b []byte) ([]Finding, error) {
	entries, err := Parse(b)
	if err != nil {
		return nil, err
	}
	var out []Finding
	for _, e := range entries {
		out = append(out, f.Read(e)...)
	}
	return out, nil
}

// Plugin reads one feed.
type Plugin struct {
	Feed     Feed
	Archiver kit.Archiver
}

func (p Plugin) Name() string { return p.Feed.API }

// Run reads the feed as it is now. An unchanged feed since the last run costs
// one GET (a 304 with the HTTP cache) and no snapshot.
func (p Plugin) Run(ctx context.Context, env plugin.Env) (plugin.Result, error) {
	f := p.Feed
	live, err := env.Get(ctx, f.URL)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("fetching %s: %w", f.URL, err)
	}
	last, same, err := kit.Unchanged(env, f.API, live)
	if err != nil {
		return plugin.Result{}, err
	}
	if same {
		log.Printf("%s: no change since %s", f.API, last.SourceURL)
		return plugin.Result{}, nil
	}
	want, err := f.Findings(live)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("reading %s: %w", f.URL, err)
	}
	snapURL, snap, got, err := kit.Cite(ctx, p.Archiver, f.URL, f.Findings, want)
	if err != nil {
		return plugin.Result{}, err
	}

	now := env.Now().UTC()
	var events []event.EventV2
	var rejected []error
	for _, fd := range got {
		if fd.Dropped != "" {
			log.Printf("%s: dropped %s: %s", f.API, fd.Entry, fd.Dropped)
			continue
		}
		e := event.EventV2{
			API:           f.API,
			Version:       fd.Version,
			ChangeType:    fd.ChangeType,
			Breaking:      fd.ChangeType == event.ChangeTypeBreaking,
			Subject:       fd.Subject,
			Description:   fd.Description,
			MigrationHint: fd.Hint,
			SourceURL:     snapURL,
			SourceExcerpt: fd.Excerpt,
			Confidence:    1.0,
			DetectedAt:    now,
		}
		e.Fingerprint = event.FingerprintV2(e.API, e.Version, e.ChangeType, e.Subject, "")
		if err := e.Validate(); err != nil {
			rejected = append(rejected, fmt.Errorf("%s %s: %w", f.API, fd.Entry, err))
			continue
		}
		events = append(events, e)
	}
	baseline := plugin.Baseline(f.API, now.Format("2006-01-02"), snapURL, snapURL, snap, false, now)
	baseline.SHA256 = plugin.SHA256(live) // so the next run compares like with like
	return plugin.Result{Label: snapURL, Events: events, Rejected: rejected, Baselines: []plugin.Spec{baseline}}, nil
}
