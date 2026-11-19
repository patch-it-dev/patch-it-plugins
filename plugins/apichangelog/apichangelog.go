// Package apichangelog is the source plugin for a vendor's machine-readable
// API changelog: a JSON file, kept in a repo, that lists per day each
// endpoint's changes per API version, each with a change code (oasdiff's) and
// whether it's backward compatible. MongoDB publishes one for the Atlas
// Administration API.
//
// It reads only an endpoint's life cycle, in the vendor's own words: an API
// version deprecated (`deprecation`), unless the vendor reactivates it later,
// and an API version removed at its sunset (`breaking`). The rest of the changelog is property-level, and the vendor's
// "not backward compatible" there is broad (a response enum gaining a value
// counts), so it isn't read: precision over recall.
//
// The file is read at the commit that last changed it, whose URL never
// changes, so that URL is the events' provenance (no snapshot). Every run
// reads the whole file; the fingerprints dedupe.
package apichangelog

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"

	"github.com/patch-it-dev/patch-it-plugins/event"
	"github.com/patch-it-dev/patch-it-plugins/kit"
	"github.com/patch-it-dev/patch-it-plugins/plugin"
)

func init() {
	plugin.Register(Plugin{Log: MongoDBAtlas})
}

// Log is one vendor's changelog file.
type Log struct {
	API    string // the events' api: "mongodb-atlas"
	Vendor string // for sentences: "MongoDB"
	Name   string // the API, for sentences: "Atlas Administration API"
	Repo   string // owner/name on GitHub
	Path   string // the file's path in the repo
}

// MongoDBAtlas is the Atlas Administration API's changelog,
// changelog/changelog.json in mongodb/openapi (checked 2026-11-19: 188 days
// from 2023-02-01 to 2026-11-18, released by MongoDB's CI, "Release API
// Changelog", a few times a month; the docs page
// https://www.mongodb.com/docs/atlas/reference/api-resources-spec/changelog/
// is built from it). Atlas versions each resource by date (`2023-01-01`),
// chosen per request in the Accept header.
var MongoDBAtlas = Log{
	API: "mongodb-atlas", Vendor: "MongoDB", Name: "Atlas Administration API",
	Repo: "mongodb/openapi", Path: "changelog/changelog.json",
}

// CommitsURL lists the newest commit that changed the file.
func (l Log) CommitsURL() string {
	return fmt.Sprintf("https://api.github.com/repos/%s/commits?path=%s&per_page=1", l.Repo, l.Path)
}

// URL is the file at one commit: its own provenance.
func (l Log) URL(sha string) string {
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", l.Repo, sha, l.Path)
}

// The file's shape: newest day first.
type day struct {
	Date  string `json:"date"`
	Paths []struct {
		Path     string `json:"path"`
		Method   string `json:"httpMethod"`
		Versions []struct {
			Version string   `json:"version"`
			Changes []change `json:"changes"`
		} `json:"versions"`
	} `json:"paths"`
}

type change struct {
	Change             string `json:"change"`
	ChangeCode         string `json:"changeCode"`
	BackwardCompatible bool   `json:"backwardCompatible"`
}

// lifecycle is what each change code this plugin reads means, each about a
// whole endpoint at one API version: "endpoint deprecated"; "new resource
// added 2023-02-01. Resource version 2023-01-01 deprecated and marked for
// removal on 2025-06-01"; "endpoint with API Version '2023-01-01' was removed
// as it has reached its sunset date '2026-01-22'".
//
// Not read: "api path removed without deprecation". Its one entry
// (streams/activeVpcPeeringConnections, 2025-01-10) comes with
// `endpoint-added` the same day, and the endpoint changes again later: it
// wasn't removed.
var lifecycle = map[string]event.ChangeType{
	"endpoint-deprecated":         event.ChangeTypeDeprecation,
	"resource-version-deprecated": event.ChangeTypeDeprecation,
	"endpoint-removed":            event.ChangeTypeBreaking,
}

// reactivated withdraws a deprecation: "endpoint reactivated". MongoDB does
// it (13 times by 2026-11-18): GET /api/atlas/v2/groups/{groupId}/clusters
// at 2024-05-30 was deprecated on 2024-06-25 and reactivated the next day.
const reactivated = "endpoint-reactivated"

// Finding is one life-cycle change.
type Finding struct {
	ChangeType event.ChangeType
	Date       string // the changelog's day
	Operation  string // "GET /api/atlas/v2/groups/{groupId}/clusters"
	Version    string // the API version it's about: "2023-01-01"
	Code       string
	Pointer    string // RFC 6901, into the file
	Excerpt    string // the change, as JSON
}

// Read finds every life-cycle change in the file that still stands, by
// replaying the days oldest first: a reactivation of an endpoint's version
// withdraws its deprecation, and a deprecation after that is a new one. A
// deprecation counts from the first time it's announced since the last
// reactivation; the changelog repeats some.
func (l Log) Read(b []byte) ([]Finding, error) {
	var days []day
	if err := json.Unmarshal(b, &days); err != nil {
		return nil, fmt.Errorf("not a changelog: %w", err)
	}
	if len(days) == 0 {
		return nil, fmt.Errorf("the changelog has no days: it may have moved or changed shape")
	}
	type item struct {
		seq int
		f   Finding
	}
	deprecated := map[string]item{} // by operation and version, until reactivated
	removed := map[string]item{}
	seq := 0
	for di := len(days) - 1; di >= 0; di-- {
		d := days[di]
		for pi, p := range d.Paths {
			for vi, v := range p.Versions {
				for ci, c := range v.Changes {
					op := p.Method + " " + p.Path
					key := op + " " + v.Version
					if c.ChangeCode == reactivated {
						delete(deprecated, key)
						continue
					}
					ct, ok := lifecycle[c.ChangeCode]
					if !ok {
						continue
					}
					ex, err := json.Marshal(c)
					if err != nil {
						return nil, err
					}
					f := Finding{
						ChangeType: ct, Date: d.Date, Operation: op, Version: v.Version, Code: c.ChangeCode,
						Pointer: fmt.Sprintf("/%d/paths/%d/versions/%d/changes/%d", di, pi, vi, ci),
						Excerpt: string(ex),
					}
					seq++
					target := deprecated
					if ct == event.ChangeTypeBreaking {
						target = removed
					}
					if _, dup := target[key]; !dup {
						target[key] = item{seq, f}
					}
				}
			}
		}
	}
	all := make([]item, 0, len(deprecated)+len(removed))
	for _, it := range deprecated {
		all = append(all, it)
	}
	for _, it := range removed {
		all = append(all, it)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].seq < all[j].seq })
	out := make([]Finding, len(all))
	for i, it := range all {
		out[i] = it.f
	}
	return out, nil
}

// Plugin reads one vendor's changelog.
type Plugin struct {
	Log Log
}

func (p Plugin) Name() string { return p.Log.API }

// Run reads the file at the newest commit that changed it.
func (p Plugin) Run(ctx context.Context, env plugin.Env) (plugin.Result, error) {
	l := p.Log
	body, err := env.Get(ctx, l.CommitsURL())
	if err != nil {
		return plugin.Result{}, fmt.Errorf("looking up %s's newest changelog commit: %w", l.API, err)
	}
	var commits []struct {
		SHA string `json:"sha"`
	}
	if err := json.Unmarshal(body, &commits); err != nil {
		return plugin.Result{}, fmt.Errorf("parsing github's commits response: %w", err)
	}
	if len(commits) == 0 || commits[0].SHA == "" {
		return plugin.Result{}, fmt.Errorf("github lists no commit that changed %s in %s", l.Path, l.Repo)
	}
	sha := commits[0].SHA
	url := l.URL(sha)
	page, err := env.Get(ctx, url)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("fetching %s: %w", url, err)
	}
	last, same, err := kit.Unchanged(env, l.API, page)
	if err != nil {
		return plugin.Result{}, err
	}
	if same {
		log.Printf("%s: no change since %s", l.API, last.SourceURL)
		return plugin.Result{}, nil
	}
	findings, err := l.Read(page)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("reading %s: %w", url, err)
	}

	now := env.Now().UTC()
	var events []event.EventV2
	var rejected []error
	for _, f := range findings {
		e := event.EventV2{
			API:           l.API,
			Version:       f.Version,
			ChangeType:    f.ChangeType,
			Breaking:      f.ChangeType == event.ChangeTypeBreaking,
			Subject:       event.Subject{Type: event.SubjectAPIOperation, Kind: event.AffectedKindOperation, Side: event.SideRequest, Operation: f.Operation},
			Description:   describe(l, f),
			SourceURL:     url,
			SourcePointer: f.Pointer,
			SourceExcerpt: f.Excerpt,
			Confidence:    1.0,
			DetectedAt:    now,
		}
		e.Fingerprint = event.FingerprintV2(e.API, e.Version, e.ChangeType, e.Subject, "")
		if err := e.Validate(); err != nil {
			rejected = append(rejected, fmt.Errorf("%s %s %s: %w", l.API, f.Operation, f.Version, err))
			continue
		}
		events = append(events, e)
	}
	return plugin.Result{
		Label:     url,
		Events:    events,
		Rejected:  rejected,
		Baselines: []plugin.Spec{plugin.Baseline(l.API, sha, sha, url, page, false, now)},
	}, nil
}

func describe(l Log, f Finding) string {
	switch f.ChangeType {
	case event.ChangeTypeBreaking:
		return fmt.Sprintf("%s %s, API version %s, removed on %s (%s's changelog, %s).", l.Name, f.Operation, f.Version, f.Date, l.Vendor, f.Code)
	default:
		return fmt.Sprintf("%s %s, API version %s, deprecated on %s (%s's changelog, %s).", l.Name, f.Operation, f.Version, f.Date, l.Vendor, f.Code)
	}
}
