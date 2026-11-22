package catalog

import (
	"context"
	"fmt"
	"regexp"
	"sort"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
)

// Routes is a provider's own file of named operations, kept as plain text
// in its repo and read at each commit that changed it: Dropbox's Stone IDL
// namespace files (dropbox/dropbox-api-spec), each line `route name(Args,
// Result, Error)`, optionally followed by `deprecated` or `deprecated by
// other_name`. Unlike Commands (one file per command, at a release tag) or
// UpcomingChanges (dated entries in one JSON file), a version here is a
// commit and an entry is just a name on one line of free-form text: the
// name, not the line, is what's compared, so a release that only reorders
// or reformats the file (Dropbox did, in the commit the golden test is
// pinned on either side of) reads as no change, never as every route
// removed and added back. A removed name and an added name that compute
// the same Operation in the same pair are a renamed spelling, not a
// removal (real, 2018-08-14: Dropbox's own copy_v2 became copy:2, the same
// call either way) — the one case a plain name diff can't tell apart from
// a real removal on its own.
type Routes struct {
	API    string // the source id the events carry: "dropbox/files-api"
	Vendor string // for sentences: "Dropbox"
	Repo   string // owner/name on GitHub
	Path   string // the file's path in the repo: "files.stone"
	// Operation turns a route's own name, as the file declares it
	// ("search:2", "properties/template/list"), into the HTTP call it is.
	// Also what tells a renamed spelling from a real removal: see above.
	Operation func(name string) string

	shape *plugin.Shape // the source's, set by RegisterProvider (WithShape); nil: no guard
}

// RoutesMinRatio is the shape guard's ratio for a route file: like a
// command catalog, a provider deprecates far more often than it deletes, so
// only a file left with under 80% of the routes the read before found
// changed shape.
const RoutesMinRatio = 0.8

func (r Routes) Name() string { return r.API }

// WithShape is r guarding its pairs with s.
func (r Routes) WithShape(s plugin.Shape) plugin.Plugin {
	r.shape = &s
	return r
}

// FileURL is the file at one commit: its own provenance, as a commit SHA
// never changes.
func (r Routes) FileURL(sha string) string {
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", r.Repo, sha, r.Path)
}

// ListVersions is every commit that changed the file, oldest first.
func (r Routes) ListVersions(ctx context.Context, env plugin.Env) ([]plugin.Version, error) {
	return versions.GitHubCommits(ctx, env.Get, r.Repo, r.Path)
}

// ReadAt fetches the file at v's commit.
func (r Routes) ReadAt(ctx context.Context, env plugin.Env, v plugin.Version) (plugin.Snapshot, error) {
	url := r.FileURL(v.Ref)
	body, err := env.Get(ctx, url)
	if err != nil {
		return plugin.Snapshot{}, fmt.Errorf("fetching %s: %w", url, err)
	}
	return plugin.Snapshot{Version: v, Content: body, SourceURL: url}, nil
}

// routeDecl matches one route declaration line: `route name(...)` or
// `route name (...)`, its first submatch the bare name, up to the first
// space or `(`, so a trailing ` deprecated` or ` deprecated by x` is never
// part of it. Declared at the start of a line (unindented), as every route
// in a Stone namespace file is; a struct or union field never starts a line
// this way.
var routeDecl = regexp.MustCompile(`(?m)^route ([^\s(]+)\s*\(.*$`)

// routesIn is every route declaration in content, by name, to its whole
// line (the excerpt an event quotes).
func routesIn(content []byte) map[string]string {
	out := map[string]string{}
	for _, m := range routeDecl.FindAllStringSubmatch(string(content), -1) {
		out[m[1]] = m[0]
	}
	return out
}

// Extract is a breaking event per route declared in from's file and gone
// from to's, and an added one per route declared in to's and not in from's,
// each citing the file at the version whose content it quotes (from's for a
// removal, to's for an addition), with previous_version from's commit. A
// zero from has nothing to compare.
func (r Routes) Extract(_ context.Context, env plugin.Env, from, to plugin.Snapshot) (plugin.Result, error) {
	if from.IsZero() {
		return plugin.Result{}, nil
	}
	before := routesIn(from.Content)
	after := routesIn(to.Content)
	if err := r.shape.CheckCount("routes", RoutesMinRatio, len(before), len(after)); err != nil {
		return plugin.Result{}, fmt.Errorf("reading %s: %w", to.SourceURL, err)
	}
	// renamedTo/renamedFrom catch Dropbox renaming a route's own Stone
	// spelling while the real HTTP call stays identical (confirmed real,
	// 2018-08-14: copy_v2 -> copy:2, create_folder_v2 -> create_folder:2,
	// delete_v2 -> delete:2, move_v2 -> move:2,
	// upload_session/append_v2 -> upload_session/append:2 — every one
	// still the same /2/files/<x>_v2 call). A removed name and an added
	// name that compute the same Operation are that rename: neither event,
	// the same rule Commands applies to a command file that moves.
	renamedTo := map[string]bool{}   // by the operation a removed name used to answer to
	renamedFrom := map[string]bool{} // by the operation an added name now answers to
	for n := range before {
		if _, ok := after[n]; !ok {
			renamedTo[r.Operation(n)] = true
		}
	}
	for n := range after {
		if _, ok := before[n]; !ok {
			renamedFrom[r.Operation(n)] = true
		}
	}
	names := map[string]bool{}
	for n := range before {
		names[n] = true
	}
	for n := range after {
		names[n] = true
	}
	sorted := make([]string, 0, len(names))
	for n := range names {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)
	var made []event.Event
	now := env.Now().UTC()
	for _, n := range sorted {
		beforeLine, wasBefore := before[n]
		afterLine, isAfter := after[n]
		op := r.Operation(n)
		switch {
		case wasBefore && !isAfter && renamedFrom[op]:
			// This exact operation is also newly answered to by a
			// differently-spelled route in the same pair: a rename, not a
			// removal.
		case !wasBefore && isAfter && renamedTo[op]:
			// The other half of the same rename.
		case wasBefore && !isAfter:
			made = append(made, event.Event{
				API: r.API, Version: to.Version.ID, PreviousVersion: from.Version.ID,
				Change: event.ChangeBreaking,
				Subject: event.Subject{Type: event.SubjectAPIOperation, Kind: event.KindOperation, Side: event.SideRequest,
					Operation: r.Operation(n)},
				Description:   fmt.Sprintf("%s's %s route is gone from its API spec (it was there as of %s): calling it fails.", r.Vendor, n, from.Version.ID),
				SourceURL:     from.SourceURL,
				SourceExcerpt: beforeLine,
				Confidence:    1.0,
				DetectedAt:    now,
			})
		case !wasBefore && isAfter:
			made = append(made, event.Event{
				API: r.API, Version: to.Version.ID, PreviousVersion: from.Version.ID,
				Change: event.ChangeAdded,
				Subject: event.Subject{Type: event.SubjectAPIOperation, Kind: event.KindOperation, Side: event.SideRequest,
					Operation: r.Operation(n)},
				Description:   fmt.Sprintf("%s's %s route is new in its API spec (it wasn't there as of %s).", r.Vendor, n, from.Version.ID),
				SourceURL:     to.SourceURL,
				SourceExcerpt: afterLine,
				Confidence:    1.0,
				DetectedAt:    now,
			})
		}
	}
	events, rejected := finish(r.API, made, nil)
	return plugin.Result{Events: events, Rejected: rejected}, nil
}

var (
	_ plugin.Versioned = Routes{}
	_ plugin.Guarded   = Routes{}
)
