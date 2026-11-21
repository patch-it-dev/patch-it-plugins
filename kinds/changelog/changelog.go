// Package changelog is the kind for a changelog, with a parser for each
// shape: a package's changelog file in Markdown or reStructuredText, or its
// release notes on GitHub (File; this file, parse.go, classify.go), or a
// vendor's machine-readable API changelog in JSON (Log; json.go). One Plugin
// reads either.
//
// A changelog file (CHANGELOG.md, CHANGES.rst, HISTORY.rst) is read at a
// release's tag, and a release's notes at its tag's GitHub release: one change
// event per bullet that plainly removes, renames or deprecates names code
// uses. The tag never changes, so its URL is the event's provenance (no
// snapshot) and a rerun stores nothing new: the fingerprints dedupe.
//
// Which bullets count: those under a Keep a Changelog "Removed" or
// "Deprecated" heading, and the common variants ("Breaking changes",
// "⚠ BREAKING CHANGES", "Backwards incompatible changes", "Deprecations",
// changesets' "Major Changes", reStructuredText titles, "**Deprecations**:"),
// plus a bullet marked breaking under any heading ("Breaking change:", "⚠️",
// "**[BREAKING]**", "feat!:"). An area, issue or commit the bullet starts
// with ("**client:**", "[#12](...)", "abc1234:") is read past.
// A bullet must say which names in one plain sentence ("Removed `X`",
// "`X` class removed", "Deprecated `X` in favor of `Y`"); anything else
// (dropped Python versions, behavior changes, parameters of an unnamed call,
// "signature changed") is dropped and logged: precision over recall.
package changelog

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
)

// File is one package's changelog. Unpinned (Tag and Release empty), it is
// read release by release: every version on the package registry (PyPI, npm)
// or, for an ecosystem patch-it doesn't list (Go, Maven, NuGet, RubyGems,
// Packagist, Terraform), every TagPrefix + X.Y.Z tag in the repo; the file at
// that release's tag. With Notes, the changelog is each release's notes on
// GitHub instead of a file, and the versions are the repo's releases. Pinned
// (At), it is one release.
type File struct {
	API       string // the source id the events carry: "urllib3/urllib3"
	Package   string // the installed name: "urllib3"; a Go module's path without its /vN
	Ecosystem string // where it's installed from: "pypi", "npm", "go", "maven", "nuget", "rubygems", "packagist", "terraform"
	Module    string // the import name, stripped from a symbol's front: "urllib3"
	Repo      string // owner/name on GitHub
	TagPrefix string // a release's git tag is TagPrefix + its version: "v" for v0.86.0
	Path      string // the changelog's path in the repo
	Notes     bool   // the changelog is each GitHub release's notes, not a file
	Pins      string // the source a release's section says it pins a version of ("stripe/api"; pins.go), or ""
	Tag       string // pinned: the release's tag: "2.0.0", "v0.86.0"
	Release   string // pinned: the release as the changelog heads it: "2.0.0"
}

// At is f pinned to one release.
func (f File) At(release string) File {
	f.Release, f.Tag = release, f.TagPrefix+release
	return f
}

// URL is what ReadAt fetches: the changelog as it was at the (pinned) tag,
// or the tag's release on GitHub's API.
func (f File) URL() string {
	if f.Notes {
		return fmt.Sprintf("%s/repos/%s/releases/tags/%s", versions.GitHubAPI, f.Repo, url.PathEscape(f.Tag))
	}
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", f.Repo, f.Tag, f.Path)
}

// Cite is the events' provenance: the file at the tag, or the release's
// page, where a person reads the same notes.
func (f File) Cite() string {
	if f.Notes {
		return fmt.Sprintf("https://github.com/%s/releases/tag/%s", f.Repo, url.PathEscape(f.Tag))
	}
	return f.URL()
}

// what the changelog is, in a sentence.
func (f File) what() string {
	if f.Notes {
		return "release notes"
	}
	return "changelog"
}

// installed is the package as code installs it at the release: a Go
// module's path gains /vN from major 2 on (Go's semantic import versioning).
func (f File) installed() string {
	if m := versions.Major(f.Release); f.Ecosystem == "go" && m >= 2 {
		return fmt.Sprintf("%s/v%d", f.Package, m)
	}
	return f.Package
}

// Finding is one bullet that became a change, or one that was dropped.
type Finding struct {
	Change  Change
	Section string
	Bullet  Bullet
	Dropped string
}

// Read finds the release's breaking and deprecation bullets.
func (f File) Read(page []byte) (heading string, out []Finding, err error) {
	var sections []Section
	if f.Notes {
		heading, sections, err = f.notes(page)
	} else {
		heading, sections, err = ReleaseNotes(page, f.Release)
	}
	if err != nil {
		return "", nil, err
	}
	added := f.added(sections)
	for _, s := range sections {
		k := SectionKind(s.Title)
		for _, b := range s.Bullets {
			c, ok, why := Classify(k, b.Text, f.Module)
			if !ok {
				continue
			}
			for _, sym := range c.Symbols {
				// urllib3 2.0.0 removed the urllib3.request module and added a
				// top-level urllib3.request function: code calling it works.
				if why == "" && c.ChangeType == event.ChangeTypeBreaking && added[sym] {
					why = fmt.Sprintf("the same release adds `%s` back", sym)
				}
			}
			out = append(out, Finding{Change: c, Section: s.Title, Bullet: b, Dropped: why})
		}
	}
	return heading, chained(out), nil
}

// chained follows each rename through the release's other renames and
// removals, so a replacement is the name the release ends with: "A renamed
// to B" and "B renamed to C" make A's replacement C; a chain that ends in a
// removal (or loops) leaves A removed, with no replacement to move to.
func chained(fs []Finding) []Finding {
	renamed, removed := map[string]string{}, map[string]bool{}
	for _, f := range fs {
		if f.Dropped != "" || f.Change.ChangeType != event.ChangeTypeBreaking {
			continue
		}
		for _, sym := range f.Change.Symbols {
			if f.Change.Replacement != "" {
				renamed[sym] = f.Change.Replacement
			} else {
				removed[sym] = true
			}
		}
	}
	for i, f := range fs {
		r := f.Change.Replacement
		// Only a breaking rename is followed: a deprecation in favor of a
		// name stays a deprecation, whatever happens to that name.
		if f.Dropped != "" || r == "" || f.Change.ChangeType != event.ChangeTypeBreaking {
			continue
		}
		seen := map[string]bool{}
		for next, ok := renamed[r]; ok && !seen[r]; next, ok = renamed[r] {
			seen[r] = true
			r = next
		}
		if removed[r] || seen[r] {
			fs[i].Change.Replacement, fs[i].Change.Change = "", event.PackageRemovedSymbol
			continue
		}
		fs[i].Change.Replacement = r
	}
	return fs
}

// notes are the sections of a GitHub release's body (the API's JSON for one
// release), headed by its name. A release with no notes has nothing to read.
func (f File) notes(page []byte) (string, []Section, error) {
	var r struct {
		Tag  string `json:"tag_name"`
		Name string `json:"name"`
		Body string `json:"body"`
	}
	if err := json.Unmarshal(page, &r); err != nil {
		return "", nil, fmt.Errorf("parsing the release %s: %w", f.Tag, err)
	}
	if r.Tag != f.Tag {
		return "", nil, fmt.Errorf("the release read is %q, not %q", r.Tag, f.Tag)
	}
	if strings.TrimSpace(r.Body) == "" {
		return "", nil, fmt.Errorf("release %s has no notes: %w", f.Tag, plugin.ErrNothingToRead)
	}
	heading := r.Name
	if heading == "" {
		heading = r.Tag
	}
	return heading, NotesSections(r.Body), nil
}

// added are the names the release's "Added" bullets name, as symbols.
func (f File) added(sections []Section) map[string]bool {
	out := map[string]bool{}
	for _, s := range sections {
		t := strings.ToLower(s.Title)
		if !strings.Contains(t, "added") && !strings.Contains(t, "feature") {
			continue
		}
		for _, b := range s.Bullets {
			_, codes := template(b.Text)
			for _, c := range codes {
				c = strings.TrimPrefix(strings.TrimSuffix(c, "()"), f.Module+".")
				out[c] = true
			}
		}
	}
	return out
}

// Plugin reads one changelog, in one of two shapes: set File for a
// Markdown or RST changelog (read release by release: ListVersions, ReadAt,
// Extract; or, pinned, that one release), or Log for a JSON one (read commit
// by commit: json.go).
type Plugin struct {
	File File
	Log  Log
}

// isLog is a JSON changelog.
func (p Plugin) isLog() bool { return p.Log.API != "" }

func (p Plugin) Name() string {
	if p.isLog() {
		return p.Log.API
	}
	return p.File.API
}

// ListVersions is every final release of the package on its registry, oldest
// first, each read at TagPrefix + version; a pinned File has its one release.
func (p Plugin) ListVersions(ctx context.Context, env plugin.Env) ([]plugin.Version, error) {
	if p.isLog() {
		return p.listLog(ctx, env)
	}
	f := p.File
	if f.Tag != "" {
		return []plugin.Version{{ID: f.Release, Ref: f.Tag}}, nil
	}
	var vs []plugin.Version
	var err error
	switch {
	case f.Notes:
		vs, err = versions.GitHubReleases(ctx, env.Get, f.Repo, func(tag string) bool {
			v, ok := strings.CutPrefix(tag, f.TagPrefix)
			return ok && versions.Final(v) && !strings.HasPrefix(v, "v")
		})
		for i := range vs {
			vs[i].ID = strings.TrimPrefix(vs[i].Ref, f.TagPrefix)
		}
		slices.SortStableFunc(vs, func(a, b plugin.Version) int { return versions.Compare(a.ID, b.ID) })
		return vs, err
	case f.Ecosystem == "pypi":
		vs, err = versions.PyPIReleases(ctx, env.Get, f.Package)
	case f.Ecosystem == "npm":
		vs, err = versions.NPMReleases(ctx, env.Get, f.Package)
	default:
		// No registry patch-it lists from: the repo's release tags.
		vs, err = versions.GitHubTags(ctx, env.Get, f.Repo, regexp.MustCompile(`^`+regexp.QuoteMeta(f.TagPrefix)+`(\d+(?:\.\d+)+)$`))
	}
	if err != nil {
		return nil, err
	}
	for i := range vs {
		vs[i].Ref = f.TagPrefix + vs[i].ID
	}
	return f.pinned(ctx, env, vs), nil
}

// ReadAt fetches the changelog at the release's tag.
func (p Plugin) ReadAt(ctx context.Context, env plugin.Env, v plugin.Version) (plugin.Snapshot, error) {
	if p.isLog() {
		return p.readLog(ctx, env, v)
	}
	f := p.File
	f.Release, f.Tag = v.ID, v.Ref
	page, err := env.Get(ctx, f.URL())
	if err != nil {
		return plugin.Snapshot{}, fmt.Errorf("fetching %s: %w", f.URL(), err)
	}
	return plugin.Snapshot{Version: v, Content: page, SourceURL: f.Cite()}, nil
}

// Extract is the to release's section of the changelog: an event per bullet
// that plainly removes, renames or deprecates a name, with previous_version
// from's release. Releases are extracted pair by pair, so from is the release
// right before to and to's section is everything between them.
func (p Plugin) Extract(_ context.Context, env plugin.Env, from, to plugin.Snapshot) (plugin.Result, error) {
	if p.isLog() {
		return p.extractLog(env, from, to)
	}
	f := p.File
	f.Release, f.Tag = to.Version.ID, to.Version.Ref
	heading, findings, err := f.Read(to.Content)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("reading %s: %w", f.Cite(), err)
	}
	now := env.Now().UTC()
	var events []event.Event
	var rejected []error
	seen := map[string]bool{} // a changelog that lists a bullet twice is one change
	for _, fd := range findings {
		if fd.Dropped != "" {
			log.Printf("%s %s: dropped %q: %s", f.Package, f.Release, fd.Bullet.Text, fd.Dropped)
			continue
		}
		c := fd.Change
		subject := event.Subject{
			Type: event.SubjectPackageRelease, Ecosystem: f.Ecosystem, Package: f.installed(),
			Change: c.Change, Symbols: c.Symbols, Replacement: c.Replacement,
		}
		e := event.Event{
			API:             f.API,
			Version:         f.Release,
			PreviousVersion: from.Version.ID,
			ChangeType:      c.ChangeType,
			Breaking:        c.ChangeType == event.ChangeTypeBreaking,
			Subject:         subject,
			Description:     describe(f, c),
			MigrationHint:   cap2000(fmt.Sprintf("%s (%s's %s for %s)", fd.Bullet.Text, f.Package, f.what(), f.Release)),
			SourceURL:       f.Cite(),
			SourceExcerpt:   cap2000(heading + "\n" + fd.Section + "\n" + fd.Bullet.Raw),
			Confidence:      1.0,
			DetectedAt:      now,
		}
		e.Fingerprint = event.Fingerprint(e.API, e.Version, e.ChangeType, subject, "")
		if seen[e.Fingerprint] {
			continue
		}
		seen[e.Fingerprint] = true
		if err := e.Validate(); err != nil {
			rejected = append(rejected, fmt.Errorf("%s %s %v: %w", f.Package, f.Release, c.Symbols, err))
			continue
		}
		events = append(events, e)
	}
	return plugin.Result{Events: events, Rejected: rejected}, nil
}

var _ plugin.Versioned = Plugin{}

func describe(f File, c Change) string {
	names := "`" + strings.Join(c.Symbols, "`, `") + "`"
	verb := "was"
	if len(c.Symbols) > 1 {
		verb = "were"
	}
	switch {
	case c.ChangeType == event.ChangeTypeDeprecation && c.Replacement != "":
		return fmt.Sprintf("%s %s deprecated in %s %s in favor of `%s`.", names, map[string]string{"was": "is", "were": "are"}[verb], f.Package, f.Release, c.Replacement)
	case c.ChangeType == event.ChangeTypeDeprecation:
		return fmt.Sprintf("%s %s deprecated in %s %s: a later release may remove it.", names, map[string]string{"was": "is", "were": "are"}[verb], f.Package, f.Release)
	case c.Replacement != "":
		return fmt.Sprintf("%s %s renamed to `%s` in %s %s: code using the old name fails.", names, verb, c.Replacement, f.Package, f.Release)
	}
	return fmt.Sprintf("%s %s removed in %s %s: code using it fails.", names, verb, f.Package, f.Release)
}

func cap2000(s string) string {
	if r := []rune(s); len(r) > 2000 {
		return string(r[:1999]) + "…"
	}
	return s
}
