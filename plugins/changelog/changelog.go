// Package changelog is the kind for a changelog, with a parser for each
// shape: a package's changelog file in Markdown or reStructuredText (File;
// this file, parse.go, classify.go), or a vendor's machine-readable API
// changelog in JSON (Log; json.go). One Plugin reads either.
//
// A changelog file (CHANGELOG.md, CHANGES.rst, HISTORY.rst) is read at a release's tag: one
// change event per bullet that plainly removes, renames or deprecates names
// code uses. The tag never changes, so its URL is the event's provenance (no
// snapshot) and a rerun stores nothing new: the fingerprints dedupe.
//
// Which bullets count: those under a Keep a Changelog "Removed" or
// "Deprecated" heading, and the common variants ("Breaking changes",
// "Backwards incompatible changes", "Deprecations", reStructuredText titles,
// "**Deprecations**:"), plus a "Breaking change:" bullet under any heading.
// A bullet must say which names in one plain sentence ("Removed `X`",
// "`X` class removed", "Deprecated `X` in favor of `Y`"); anything else
// (dropped Python versions, behavior changes, parameters of an unnamed call,
// "signature changed") is dropped and logged: precision over recall.
package changelog

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "textual", Name: "Textual", Category: plugin.CategoryPackages, Homepage: "https://textual.textualize.io",
		Sources: []plugin.Source{{
			ID: "textual", Layer: plugin.LayerPackage, Page: "providers/textual.md",
			Watches:  "Names removed, renamed or deprecated in each release, from the release's section of CHANGELOG.md at its tag",
			URLs:     []string{"https://github.com/Textualize/textual/blob/main/CHANGELOG.md", "https://pypi.org/project/textual/"},
			Packages: []string{"pypi:textual"},
			Plugin:   Plugin{File: Textual},
		}},
	})
	plugin.RegisterProvider(plugin.Provider{
		ID: "urllib3", Name: "urllib3", Category: plugin.CategoryPackages, Homepage: "https://urllib3.readthedocs.io",
		Sources: []plugin.Source{{
			ID: "urllib3", Layer: plugin.LayerPackage, Page: "providers/urllib3.md",
			Watches:  "Names removed or deprecated in each release, from the release's section of CHANGES.rst at its tag",
			URLs:     []string{"https://github.com/urllib3/urllib3/blob/main/CHANGES.rst", "https://pypi.org/project/urllib3/"},
			Packages: []string{"pypi:urllib3"},
			Plugin:   Plugin{File: Urllib3},
		}},
	})
	plugin.RegisterProvider(plugin.Provider{
		ID: "mongodb", Name: "MongoDB", Category: plugin.CategoryDatabases, Homepage: "https://www.mongodb.com",
		Sources: []plugin.Source{{
			ID: "atlas-admin-api", Layer: plugin.LayerAPI, Page: "providers/mongodb-atlas.md",
			Watches: "Atlas Administration API versions deprecated and removed at their sunset date, per endpoint, from MongoDB's own API changelog",
			URLs:    []string{"https://github.com/mongodb/openapi/blob/main/changelog/changelog.json"},
			Plugin:  Plugin{Log: MongoDBAtlas},
		}},
	})
}

// File is one package's changelog. Unpinned (Tag and Release empty), it is
// read release by release: every version on the package registry, the file at
// that release's tag (TagPrefix + version). Pinned (At), it is one release.
type File struct {
	Package   string // the installed name and the events' api: "urllib3"
	Ecosystem string // where its releases are listed: "pypi"
	Module    string // the import name, stripped from a symbol's front: "urllib3"
	Repo      string // owner/name on GitHub
	TagPrefix string // a release's git tag is TagPrefix + its version: "v" for v0.86.0
	Path      string // the changelog's path in the repo
	Tag       string // pinned: the release's tag: "2.0.0", "v0.86.0"
	Release   string // pinned: the release as the changelog heads it: "2.0.0"
}

// At is f pinned to one release.
func (f File) At(release string) File {
	f.Release, f.Tag = release, f.TagPrefix+release
	return f
}

// Textual is Textual's Keep a Changelog CHANGELOG.md (checked 2026-11-14 at
// v0.86.0: "## [0.86.0] - 2024-11-16", "### Removed", and "Breaking change:"
// bullets under "### Changed"), tagged v<version>.
var Textual = File{
	Package: "textual", Ecosystem: "pypi", Module: "textual",
	Repo: "Textualize/textual", TagPrefix: "v", Path: "CHANGELOG.md",
}

// Urllib3 is urllib3's CHANGES.rst (checked 2026-11-14 at 2.0.0: "2.0.0
// (2023-04-26)" over "=====", "Removed" and "Deprecated" over "-----"),
// tagged <version>.
var Urllib3 = File{
	Package: "urllib3", Ecosystem: "pypi", Module: "urllib3",
	Repo: "urllib3/urllib3", Path: "CHANGES.rst",
}

// The releases the golden files pin.
var (
	Textual0860 = Textual.At("0.86.0")
	Urllib3200  = Urllib3.At("2.0.0")
)

// URL is the changelog as it was at the (pinned) tag: its own provenance.
func (f File) URL() string {
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", f.Repo, f.Tag, f.Path)
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
	heading, sections, err := ReleaseNotes(page, f.Release)
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
	return heading, out, nil
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
	return p.File.Package
}

// Run reads the pinned release, or the newest one on the registry, and
// stores its events. A release's tag never changes, so a rerun stores nothing
// new: the fingerprints dedupe.
func (p Plugin) Run(ctx context.Context, env plugin.Env) (plugin.Result, error) {
	if p.isLog() {
		return p.runLog(ctx, env)
	}
	v := plugin.Version{ID: p.File.Release, Ref: p.File.Tag}
	if p.File.Tag == "" {
		vs, err := p.ListVersions(ctx, env)
		if err != nil {
			return plugin.Result{}, err
		}
		if len(vs) == 0 {
			return plugin.Result{}, fmt.Errorf("%s: no release listed", p.File.Package)
		}
		v = vs[len(vs)-1]
	}
	to, err := p.ReadAt(ctx, env, v)
	if err != nil {
		return plugin.Result{}, err
	}
	res, err := p.Extract(ctx, env, plugin.Snapshot{}, to)
	if err != nil {
		return plugin.Result{}, err
	}
	res.Baselines = []plugin.Spec{plugin.Baseline(p.File.Package, v.ID, v.Ref, to.SourceURL, to.Content, false, env.Now().UTC())}
	return res, nil
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
	switch f.Ecosystem {
	case "pypi":
		vs, err = versions.PyPIReleases(ctx, env.Get, f.Package)
	case "npm":
		vs, err = versions.NPMReleases(ctx, env.Get, f.Package)
	default:
		return nil, fmt.Errorf("%s: no registry to list %q releases from", f.Package, f.Ecosystem)
	}
	if err != nil {
		return nil, err
	}
	for i := range vs {
		vs[i].Ref = f.TagPrefix + vs[i].ID
	}
	return vs, nil
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
	return plugin.Snapshot{Version: v, Content: page, SourceURL: f.URL()}, nil
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
		return plugin.Result{}, fmt.Errorf("reading %s: %w", f.URL(), err)
	}
	now := env.Now().UTC()
	var events []event.EventV2
	var rejected []error
	for _, fd := range findings {
		if fd.Dropped != "" {
			log.Printf("%s %s: dropped %q: %s", f.Package, f.Release, fd.Bullet.Text, fd.Dropped)
			continue
		}
		c := fd.Change
		subject := event.Subject{
			Type: event.SubjectPackageRelease, Ecosystem: f.Ecosystem, Package: f.Package,
			Change: c.Change, Symbols: c.Symbols, Replacement: c.Replacement,
		}
		e := event.EventV2{
			API:             f.Package,
			Version:         f.Release,
			PreviousVersion: from.Version.ID,
			ChangeType:      c.ChangeType,
			Breaking:        c.ChangeType == event.ChangeTypeBreaking,
			Subject:         subject,
			Description:     describe(f, c),
			MigrationHint:   cap2000(fmt.Sprintf("%s (%s's changelog for %s)", fd.Bullet.Text, f.Package, f.Release)),
			SourceURL:       f.URL(),
			SourceExcerpt:   cap2000(heading + "\n" + fd.Section + "\n" + fd.Bullet.Raw),
			Confidence:      1.0,
			DetectedAt:      now,
		}
		e.Fingerprint = event.FingerprintV2(e.API, e.Version, e.ChangeType, subject, "")
		if err := e.Validate(); err != nil {
			rejected = append(rejected, fmt.Errorf("%s %s %v: %w", f.Package, f.Release, c.Symbols, err))
			continue
		}
		events = append(events, e)
	}
	return plugin.Result{Label: f.URL(), Events: events, Rejected: rejected}, nil
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
