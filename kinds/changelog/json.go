// The JSON parser: a vendor's machine-readable API changelog, a JSON file
// kept in a repo that lists per day each endpoint's changes per API
// version, each with a change code (oasdiff's) and whether it's backward
// compatible. MongoDB publishes one for the Atlas Administration API.
//
// It reads only an endpoint's life cycle, in the vendor's own words: an API
// version deprecated (`deprecation`), unless the vendor reactivates it later,
// and an API version removed at its sunset (`breaking`). The rest of the
// changelog is property-level, and the vendor's "not backward compatible"
// there is broad (a response enum gaining a value counts), so it isn't read:
// precision over recall.
//
// Its versions are the commits that changed the file, whose URLs never
// change, so a commit's URL is the events' provenance (no snapshot). A pair
// of commits gives the findings the newer one has and the older one didn't,
// with the older commit's date as previous_version.

package changelog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
)

// Log is one vendor's changelog file.
type Log struct {
	API    string // the source id the events carry: "mongodb/atlas-admin-api"
	Vendor string // for sentences: "MongoDB"
	Name   string // the API, for sentences: "Atlas Administration API"
	Repo   string // owner/name on GitHub
	Path   string // the file's path in the repo
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
			Version string            `json:"version"`
			Changes []json.RawMessage `json:"changes"`
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

// LogFinding is one life-cycle change.
type LogFinding struct {
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
// reactivation; the changelog repeats some. A removal counts from its first
// announcement too, and is withdrawn, with any deprecation before it, if the
// same endpoint and version shows up again on a later day with anything but
// another removal (none of the 38 in the file on 2026-11-18 does). The excerpt is the change as the file writes
// it, whitespace aside.
func (l Log) Read(b []byte) ([]LogFinding, error) {
	var days []day
	if err := json.Unmarshal(b, &days); err != nil {
		return nil, fmt.Errorf("not a changelog: %w", err)
	}
	if len(days) == 0 {
		return nil, fmt.Errorf("the changelog has no days: it may have moved or changed shape")
	}
	type item struct {
		seq int
		f   LogFinding
	}
	deprecated := map[string]item{} // by operation and version, until reactivated
	removed := map[string]item{}
	seq := 0
	for di := len(days) - 1; di >= 0; di-- {
		d := days[di]
		for pi, p := range d.Paths {
			for vi, v := range p.Versions {
				for ci, raw := range v.Changes {
					var c change
					if err := json.Unmarshal(raw, &c); err != nil {
						return nil, fmt.Errorf("day %s, %s %s: %w", d.Date, p.Method, p.Path, err)
					}
					op := p.Method + " " + p.Path
					key := op + " " + v.Version
					if r, ok := removed[key]; ok && r.f.Date < d.Date && lifecycle[c.ChangeCode] != event.ChangeTypeBreaking {
						// It came back. Whether its old deprecation still
						// applies is unclear, so that goes too.
						delete(removed, key)
						delete(deprecated, key)
					}
					if c.ChangeCode == reactivated {
						delete(deprecated, key)
						continue
					}
					ct, ok := lifecycle[c.ChangeCode]
					if !ok {
						continue
					}
					var ex bytes.Buffer
					if err := json.Compact(&ex, raw); err != nil {
						return nil, err
					}
					f := LogFinding{
						ChangeType: ct, Date: d.Date, Operation: op, Version: v.Version, Code: c.ChangeCode,
						Pointer: fmt.Sprintf("/%d/paths/%d/versions/%d/changes/%d", di, pi, vi, ci),
						Excerpt: ex.String(),
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
	out := make([]LogFinding, len(all))
	for i, it := range all {
		out[i] = it.f
	}
	return out, nil
}

// listLog is every commit that changed the file, oldest first: the ID is
// the commit's date, the Ref its SHA.
func (p Plugin) listLog(ctx context.Context, env plugin.Env) ([]plugin.Version, error) {
	return versions.GitHubCommits(ctx, env.Get, p.Log.Repo, p.Log.Path)
}

// readLog fetches the file at the commit v.Ref.
func (p Plugin) readLog(ctx context.Context, env plugin.Env, v plugin.Version) (plugin.Snapshot, error) {
	url := p.Log.URL(v.Ref)
	page, err := env.Get(ctx, url)
	if err != nil {
		return plugin.Snapshot{}, fmt.Errorf("fetching %s: %w", url, err)
	}
	return plugin.Snapshot{Version: v, Content: page, SourceURL: url}, nil
}

// extractLog is an event per life-cycle change that stands at to and didn't
// at from (every one, when nothing was read at from). The events are about
// the endpoint's API version, so they carry that version, not the commit's;
// previous_version is from's commit, the version of the file they were
// compared with.
func (p Plugin) extractLog(env plugin.Env, from, to plugin.Snapshot) (plugin.Result, error) {
	l := p.Log
	if from.Content != nil {
		if err := p.shape.CheckCount("days", DefaultMinRatio, days(from.Content), days(to.Content)); err != nil {
			return plugin.Result{}, fmt.Errorf("reading %s: %w", to.SourceURL, err)
		}
	}
	findings, err := l.Read(to.Content)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("reading %s: %w", to.SourceURL, err)
	}
	if from.Content != nil {
		before, err := l.Read(from.Content)
		if err != nil {
			return plugin.Result{}, fmt.Errorf("reading %s: %w", from.SourceURL, err)
		}
		had := map[string]bool{}
		for _, f := range before {
			had[f.key()] = true
		}
		kept := findings[:0:0]
		for _, f := range findings {
			if !had[f.key()] {
				kept = append(kept, f)
			}
		}
		findings = kept
	}
	now := env.Now().UTC()
	var events []event.Event
	var rejected []error
	for _, f := range findings {
		e := event.Event{
			API:             l.API,
			Version:         f.Version,
			PreviousVersion: from.Version.ID,
			ChangeType:      f.ChangeType,
			Breaking:        f.ChangeType == event.ChangeTypeBreaking,
			Subject:         event.Subject{Type: event.SubjectAPIOperation, Kind: event.KindOperation, Side: event.SideRequest, Operation: f.Operation},
			Description:     describeLog(l, f),
			SourceURL:       to.SourceURL,
			SourcePointer:   f.Pointer,
			SourceExcerpt:   f.Excerpt,
			Confidence:      1.0,
			DetectedAt:      now,
		}
		e.Fingerprint = event.Fingerprint(e.API, e.Version, e.ChangeType, e.Subject, "")
		if err := e.Validate(); err != nil {
			rejected = append(rejected, fmt.Errorf("%s %s %s: %w", l.API, f.Operation, f.Version, err))
			continue
		}
		events = append(events, e)
	}
	return plugin.Result{Events: events, Rejected: rejected}, nil
}

// days is how many days the file lists, 0 when it isn't a list.
func days(b []byte) int {
	var ds []json.RawMessage
	if json.Unmarshal(b, &ds) != nil {
		return 0
	}
	return len(ds)
}

// key is what makes two findings the same change: an endpoint's API version
// deprecated, or removed.
func (f LogFinding) key() string { return string(f.ChangeType) + " " + f.Operation + " " + f.Version }

func describeLog(l Log, f LogFinding) string {
	switch f.ChangeType {
	case event.ChangeTypeBreaking:
		return fmt.Sprintf("%s %s, API version %s, removed on %s (%s's changelog, %s).", l.Name, f.Operation, f.Version, f.Date, l.Vendor, f.Code)
	default:
		return fmt.Sprintf("%s %s, API version %s, deprecated on %s (%s's changelog, %s).", l.Name, f.Operation, f.Version, f.Date, l.Vendor, f.Code)
	}
}
