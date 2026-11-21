package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
)

// UpcomingChanges is a GraphQL API's file of announced removals, kept in a
// repo and read at each commit that changed it: GitHub's
// src/graphql/data/fpt/upcoming-changes.json in github/docs, which
// docs.github.com's "Breaking changes" page is built from. The file is an
// object keyed by removal date, each a list of {location, description,
// reason, date, criticality}: location is "Type.field" (or
// "Type.field.argument"), date the day it goes.
//
// A version is a commit (its date, its SHA). An entry in to's file that
// from's didn't have is a newly announced removal: a deprecation event on
// that location whose version is the removal date (GitHub's GraphQL API has
// no versions of its own, so the day it breaks is the one to gate on, as a
// model's shutdown date is), with previous_version the commit before. An
// entry a commit drops and a later one restores, with the same date, is then
// the same event again, which the store dedupes by fingerprint. An entry dated before the commit that
// adds it isn't an announcement (a revert putting back a removal already
// made: the file lost and regained 17 such entries on 2026-01-05 and 06), and
// an entry that leaves the file isn't read: it shipped, or was withdrawn,
// and the file doesn't say which.
type UpcomingChanges struct {
	API    string // the source id the events carry: "github/graphql"
	Vendor string // for sentences: "GitHub"
	Repo   string // owner/name on GitHub
	Path   string // the file's path in the repo
	// Endpoint is the GraphQL endpoint as an operation: every query and
	// mutation is a POST to it ("POST /graphql").
	Endpoint string
}

func (u UpcomingChanges) Name() string { return u.API }

// FileURL is the file at a commit: its own provenance.
func (u UpcomingChanges) FileURL(sha string) string {
	return fmt.Sprintf("%s/%s/%s/%s", github.Raw, u.Repo, sha, u.Path)
}

// ListVersions is every commit that changed the file, oldest first.
func (u UpcomingChanges) ListVersions(ctx context.Context, env plugin.Env) ([]plugin.Version, error) {
	return versions.GitHubCommits(ctx, env.Get, u.Repo, u.Path)
}

// ReadAt fetches the file at v's commit.
func (u UpcomingChanges) ReadAt(ctx context.Context, env plugin.Env, v plugin.Version) (plugin.Snapshot, error) {
	url := u.FileURL(v.Ref)
	body, err := env.Get(ctx, url)
	if err != nil {
		return plugin.Snapshot{}, fmt.Errorf("fetching %s: %w", url, err)
	}
	return plugin.Snapshot{Version: v, Content: body, SourceURL: url}, nil
}

// upcoming is one entry of the file: the fields read, in the file's order,
// which is also how an event quotes it (the entry's "owner", a GitHub team
// or handle, is left out).
type upcoming struct {
	Location    string `json:"location"`
	Description string `json:"description"`
	Reason      string `json:"reason"`
	Date        string `json:"date"`
	Criticality string `json:"criticality"`
}

func (e upcoming) key() string { return e.Location + "@" + e.Date }

// entries reads the file: every entry, in the file's order of dates.
func entries(body []byte) ([]upcoming, error) {
	var byDate map[string][]json.RawMessage
	if err := json.Unmarshal(body, &byDate); err != nil {
		return nil, err
	}
	dates := make([]string, 0, len(byDate))
	for d := range byDate {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	var out []upcoming
	for _, d := range dates {
		for _, r := range byDate[d] {
			var e upcoming
			if err := json.Unmarshal(r, &e); err != nil {
				return nil, err
			}
			out = append(out, e)
		}
	}
	return out, nil
}

var gqlTag = regexp.MustCompile(`<[^>]+>`)

// gqlText is an entry's HTML as one plain sentence.
func gqlText(s string) string {
	return strings.Join(strings.Fields(html.UnescapeString(gqlTag.ReplaceAllString(s, ""))), " ")
}

// Extract is a deprecation event per entry in to's file that from's didn't
// have, dated before to's commit or not at all excluded (see
// UpcomingChanges), with previous_version from's commit. A zero from has
// nothing to compare.
func (u UpcomingChanges) Extract(_ context.Context, env plugin.Env, from, to plugin.Snapshot) (plugin.Result, error) {
	if from.IsZero() {
		return plugin.Result{}, nil
	}
	before, err := entries(from.Content)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("reading %s: %w", from.SourceURL, err)
	}
	after, err := entries(to.Content)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("reading %s: %w", to.SourceURL, err)
	}
	had := map[string]bool{}
	for _, e := range before {
		had[e.key()] = true
	}
	var made []event.Event
	var details []string
	for _, e := range after {
		if had[e.key()] || e.Date < to.Version.ID || e.Criticality != "breaking" {
			continue
		}
		subject, ok := u.subject(e.Location)
		if !ok {
			continue
		}
		var quote strings.Builder
		enc := json.NewEncoder(&quote)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(e); err != nil {
			return plugin.Result{}, err
		}
		hint := gqlText(e.Description)
		if r := gqlText(e.Reason); r != "" && r != hint {
			hint += " " + r
		}
		made = append(made, event.Event{
			API: u.API, Version: e.Date, PreviousVersion: from.Version.ID,
			ChangeType: event.ChangeTypeDeprecation,
			Subject:    subject,
			Description: fmt.Sprintf("%s's GraphQL API removes %s on %s: code that queries or sends it fails from that date.",
				u.Vendor, e.Location, e.Date),
			MigrationHint: hint,
			SourceURL:     to.SourceURL,
			SourceExcerpt: excerpt(strings.TrimSpace(quote.String())),
			Confidence:    1.0,
			DetectedAt:    env.Now().UTC(),
		})
		details = append(details, "")
	}
	events, rejected := finish(u.API, made, details)
	return plugin.Result{Events: events, Rejected: rejected}, nil
}

// subject places a location in the contract's api_operation: an input
// object's field, or an argument of a root field (Query.field.argument), is
// a request parameter of the endpoint; any other location (a field, a root
// field itself, a field's argument, an enum's value) is a property of its
// type, as a schema.
func (u UpcomingChanges) subject(location string) (event.Subject, bool) {
	typ, rest, ok := strings.Cut(location, ".")
	if !ok || typ == "" || rest == "" {
		return event.Subject{}, false
	}
	root := typ == "Query" || typ == "Mutation" || typ == "Subscription"
	if strings.HasSuffix(typ, "Input") || (root && strings.Contains(rest, ".")) {
		return event.Subject{Type: event.SubjectAPIOperation, Kind: event.KindRequestParameter, Side: event.SideRequest,
			Operation: u.Endpoint, Property: location}, true
	}
	return event.Subject{Type: event.SubjectAPIOperation, Kind: event.KindSchemaProperty, Side: event.SideResponse,
		Schema: typ, Property: rest}, true
}

var _ plugin.Versioned = UpcomingChanges{}
