// Package retirements is the source plugin for a model provider's deprecations
// page (OpenAI, Anthropic): every retired model becomes an event. There is no
// spec to diff: the page is read as it is now, and the events cite a Wayback
// Machine snapshot of it, never the live page (internal/plugin/kit).
package retirements

import (
	"context"
	"fmt"
	"log"

	"github.com/patch-it-dev/patch-it-plugins/kit"
	"github.com/patch-it-dev/patch-it-plugins/migrate"
	"github.com/patch-it-dev/patch-it-plugins/plugin"
	"github.com/patch-it-dev/patch-it-plugins/retire"
)

func init() {
	plugin.Register(Plugin{Page: retire.OpenAI, Archiver: kit.Wayback{}})
	plugin.Register(Plugin{Page: retire.Anthropic, Archiver: kit.Wayback{}})
}

// Plugin reads one provider's page.
type Plugin struct {
	Page     retire.Page
	Archiver kit.Archiver
}

func (p Plugin) Name() string { return p.Page.API }

// Run stores an event for every retired model on the page. The events are read
// from the snapshot itself, so every excerpt is on the page it cites; a
// snapshot that doesn't hold what the live page says (the page changed in
// between) stores nothing this run. An unchanged page since the last run costs
// one GET and no snapshot.
func (p Plugin) Run(ctx context.Context, env plugin.Env) (plugin.Result, error) {
	live, err := env.Get(ctx, p.Page.URL)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("fetching %s: %w", p.Page.URL, err)
	}
	hash := plugin.SHA256(live)
	last, same, err := kit.Unchanged(env, p.Page.API, live)
	if err != nil {
		return plugin.Result{}, err
	}
	if same {
		log.Printf("%s: no change since %s", p.Page.API, last.SourceURL)
		return plugin.Result{}, nil
	}
	want, err := retire.Parse(live, p.Page.Keep)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("reading %s: %w", p.Page.URL, err)
	}
	// A provider renaming a column makes its table unreadable, and the run would
	// quietly read fewer retirements. Past retirements stay on both pages, so a
	// real drop is rare: fail loudly instead of storing a partial list.
	if last != nil {
		if before, err := retire.Parse(last.Content, p.Page.Keep); err == nil && len(want)*10 < len(before)*9 {
			return plugin.Result{}, fmt.Errorf("%s now lists %d retirements, %d at %s: the page's shape may have changed", p.Page.URL, len(want), len(before), last.SourceURL)
		}
	}

	read := func(b []byte) ([]retire.Retirement, error) { return retire.Parse(b, p.Page.Keep) }
	snapURL, snap, got, err := kit.Cite(ctx, p.Archiver, p.Page.URL, read, want)
	if err != nil {
		return plugin.Result{}, err
	}

	now := env.Now().UTC()
	events, errs := retire.Events(p.Page, got, snapURL, now)
	// The hash of the live page, so the next run compares like with like.
	baseline := plugin.Baseline(p.Page.API, now.Format("2006-01-02"), snapURL, snapURL, snap, false, now)
	baseline.SHA256 = hash
	converted, invalid := migrate.Fresh(events)
	return plugin.Result{Label: snapURL, Events: converted, Rejected: append(errs, invalid...), Baselines: []plugin.Spec{baseline}}, nil
}
