// Package catalog is the source plugin for a provider that publishes its
// whole command or endpoint surface as one structured JSON file, one entry
// keyed by name: fetch the newest snapshot, diff it against the last one
// seen (or two given snapshots, to replay a historical change), and report
// every name that disappeared entirely between them. Unlike openapi's typed
// spec, an entry here is opaque: no request or response shape to walk, just
// whether the name is still there. Redis first: its command surface has no
// OpenAPI spec, but redis-doc's commands.json is the real, machine-readable
// file redis.io's own command reference is generated from.
//
// Precision over recall: a name that moves to a different key (a genuine
// rename) reads as one removal and one addition, same as any other. This
// never guesses a rename from two disappearances and appearances landing in
// the same diff; it only ever reports what's gone.
package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/event"
	"github.com/patch-it-dev/patch-it-plugins/plugin"
)

func init() {
	plugin.Register(Plugin{Catalog: Redis})
}

// Catalog is one provider's command or endpoint catalog: a single JSON
// object, one key per entry, fetched at a git ref.
type Catalog struct {
	API       string // the events' api: "redis"
	Ecosystem string // the package_release subject's ecosystem: "redis"
	Package   string // the package_release subject's package: "redis"
	Repo      string // owner/name on GitHub the catalog lives in
	Path      string // the catalog's path in the repo
	RefsURL   string // GitHub's commits API, for the newest ref: .../commits/<branch>
}

// Redis is redis-doc's commands.json, regenerated from each real Redis
// release and read by redis.io's own command reference (checked 2026-11-18).
// The repo has no tags, so a ref here is a commit SHA, not a release.
var Redis = Catalog{
	API: "redis", Ecosystem: "redis", Package: "redis",
	Repo: "redis/redis-doc", Path: "commands.json",
	RefsURL: "https://api.github.com/repos/redis/redis-doc/commits/master",
}

// URL is the catalog as it was at ref: its own provenance.
func (c Catalog) URL(ref string) string {
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", c.Repo, ref, c.Path)
}

// parse reads the catalog as a flat map of name to its raw entry: every
// field of an entry (arity, complexity, ACL categories...) is unread, except
// where an entry's own bytes become an event's excerpt.
func parse(data []byte) (map[string]json.RawMessage, error) {
	var out map[string]json.RawMessage
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parsing catalog: %w", err)
	}
	return out, nil
}

// removed is every name of old that new doesn't have, sorted so the result
// (and the fingerprints built from it) don't depend on map order.
func removed(old, new map[string]json.RawMessage) []string {
	var out []string
	for name := range old {
		if _, ok := new[name]; !ok {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// Plugin reads one catalog.
type Plugin struct {
	Catalog Catalog
}

func (p Plugin) Name() string { return p.Catalog.API }

// Run diffs the newest snapshot against whatever was last stored.
func (p Plugin) Run(ctx context.Context, env plugin.Env) (plugin.Result, error) {
	c := p.Catalog
	ref, err := p.latestRef(ctx, env)
	if err != nil {
		return plugin.Result{}, err
	}
	newContent, err := env.Get(ctx, c.URL(ref))
	if err != nil {
		return plugin.Result{}, fmt.Errorf("fetching %s: %w", c.URL(ref), err)
	}
	last, err := env.Baselines.LastSpec(c.API)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("loading last catalog: %w", err)
	}
	if last != nil && last.SHA256 == plugin.SHA256(newContent) {
		log.Printf("%s: no change (%s)", c.API, ref)
		return plugin.Result{}, nil
	}

	var oldContent []byte
	var from string
	if last != nil {
		oldContent, from = last.Content, last.Tag
	}
	now := env.Now()
	events, rejected, err := p.events(oldContent, newContent, from, ref, now)
	if err != nil {
		return plugin.Result{}, err
	}
	return plugin.Result{
		Label:     ref,
		Events:    events,
		Rejected:  rejected,
		Baselines: []plugin.Spec{plugin.Baseline(c.API, ref, ref, c.URL(ref), newContent, false, now)},
	}, nil
}

// Replay diffs two given snapshots. Both are saved marked as replays, so the
// normal-run baseline never moves; the events are stored like any others.
func (p Plugin) Replay(ctx context.Context, env plugin.Env, from, to string) (plugin.Result, error) {
	c := p.Catalog
	oldContent, err := env.Get(ctx, c.URL(from))
	if err != nil {
		return plugin.Result{}, fmt.Errorf("fetching %s: %w", c.URL(from), err)
	}
	newContent, err := env.Get(ctx, c.URL(to))
	if err != nil {
		return plugin.Result{}, fmt.Errorf("fetching %s: %w", c.URL(to), err)
	}
	now := env.Now()
	events, rejected, err := p.events(oldContent, newContent, from, to, now)
	if err != nil {
		return plugin.Result{}, err
	}
	return plugin.Result{
		Label:    to,
		Events:   events,
		Rejected: rejected,
		Baselines: []plugin.Spec{
			plugin.Baseline(c.API, from, from, c.URL(from), oldContent, true, now),
			plugin.Baseline(c.API, to, to, c.URL(to), newContent, true, now),
		},
	}, nil
}

// latestRef asks GitHub for the catalog repo's newest commit.
func (p Plugin) latestRef(ctx context.Context, env plugin.Env) (string, error) {
	body, err := env.Get(ctx, p.Catalog.RefsURL)
	if err != nil {
		return "", fmt.Errorf("looking up the newest commit: %w", err)
	}
	var commit struct {
		SHA string `json:"sha"`
	}
	if err := json.Unmarshal(body, &commit); err != nil {
		return "", fmt.Errorf("parsing the commit lookup: %w", err)
	}
	if commit.SHA == "" {
		return "", fmt.Errorf("the commit lookup named no sha")
	}
	return commit.SHA, nil
}

// events diffs two parsed snapshots into events. oldContent is empty on a
// plugin's first run (no baseline yet): diffing against no snapshot reports
// nothing, the same as an unchanged one.
func (p Plugin) events(oldContent, newContent []byte, from, to string, now time.Time) ([]event.EventV2, []error, error) {
	c := p.Catalog
	newEntries, err := parse(newContent)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing %s: %w", c.URL(to), err)
	}
	var oldEntries map[string]json.RawMessage
	if len(oldContent) > 0 {
		oldEntries, err = parse(oldContent)
		if err != nil {
			return nil, nil, fmt.Errorf("parsing %s: %w", c.URL(from), err)
		}
	}

	var events []event.EventV2
	var rejected []error
	for _, name := range removed(oldEntries, newEntries) {
		subject := event.Subject{
			Type: event.SubjectPackageRelease, Ecosystem: c.Ecosystem, Package: c.Package,
			Change: event.PackageRemovedSymbol, Symbols: []string{name},
		}
		e := event.EventV2{
			API: c.API, Version: to, PreviousVersion: from,
			ChangeType:    event.ChangeTypeBreaking,
			Breaking:      true,
			Subject:       subject,
			Description:   fmt.Sprintf("%s no longer exists: calling it now fails.", name),
			SourceURL:     c.URL(from),
			SourceExcerpt: excerpt(name, oldEntries[name]),
			Confidence:    1.0,
			DetectedAt:    now,
		}
		e.Fingerprint = event.FingerprintV2(e.API, e.Version, e.ChangeType, subject, "")
		if err := e.Validate(); err != nil {
			rejected = append(rejected, fmt.Errorf("%s: %w", name, err))
			continue
		}
		events = append(events, e)
	}
	return events, rejected, nil
}

// excerpt quotes the entry exactly as the old snapshot had it: the evidence
// that name existed, in the catalog's own shape. Capped at the contract's
// 2000-character limit, the entry's closing braces never cut off: nested
// command-line arguments can run long.
func excerpt(name string, raw json.RawMessage) string {
	s := fmt.Sprintf("%q: %s", name, string(raw))
	if r := []rune(s); len(r) > 2000 {
		s = string(r[:1999]) + "…"
	}
	return s
}
