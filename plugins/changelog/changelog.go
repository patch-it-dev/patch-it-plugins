// Package changelog is the source plugin for a package's changelog file
// (CHANGELOG.md, CHANGES.rst, HISTORY.rst), read at a release's tag: one
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
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "textual", Name: "Textual", Category: plugin.CategoryPackages, Homepage: "https://textual.textualize.io",
		Sources: []plugin.Source{{
			ID: "textual", Layer: plugin.LayerPackage, Page: "providers/textual.md",
			Watches: "Removals and breaking changes in release 0.86.0",
			URLs:    []string{"https://github.com/Textualize/textual/blob/v0.86.0/CHANGELOG.md"},
			Plugin:  Plugin{File: Textual0860},
		}},
	})
	plugin.RegisterProvider(plugin.Provider{
		ID: "urllib3", Name: "urllib3", Category: plugin.CategoryPackages, Homepage: "https://urllib3.readthedocs.io",
		Sources: []plugin.Source{{
			ID: "urllib3", Layer: plugin.LayerPackage, Page: "providers/urllib3.md",
			Watches: "Removals and deprecations in release 2.0.0",
			URLs:    []string{"https://github.com/urllib3/urllib3/blob/2.0.0/CHANGES.rst"},
			Plugin:  Plugin{File: Urllib3200},
		}},
	})
}

// File is one package's changelog at one release.
type File struct {
	Package   string // the installed name and the events' api: "urllib3"
	Ecosystem string // "pypi"
	Module    string // the import name, stripped from a symbol's front: "urllib3"
	Repo      string // owner/name on GitHub
	Tag       string // the release's tag: "2.0.0", "v0.86.0"
	Release   string // the release as the changelog heads it: "2.0.0"
	Path      string // the changelog's path in the repo
}

// Textual0860 is Textual's Keep a Changelog CHANGELOG.md at v0.86.0 (checked
// 2026-11-14: "## [0.86.0] - 2024-11-16", "### Removed", and "Breaking
// change:" bullets under "### Changed").
var Textual0860 = File{
	Package: "textual", Ecosystem: "pypi", Module: "textual",
	Repo: "Textualize/textual", Tag: "v0.86.0", Release: "0.86.0", Path: "CHANGELOG.md",
}

// Urllib3200 is urllib3's CHANGES.rst at 2.0.0 (checked 2026-11-14: "2.0.0
// (2023-04-26)" over "=====", "Removed" and "Deprecated" over "-----").
var Urllib3200 = File{
	Package: "urllib3", Ecosystem: "pypi", Module: "urllib3",
	Repo: "urllib3/urllib3", Tag: "2.0.0", Release: "2.0.0", Path: "CHANGES.rst",
}

// URL is the changelog as it was at the tag: its own provenance.
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

// Plugin reads one changelog at one release.
type Plugin struct {
	File File
}

func (p Plugin) Name() string { return p.File.Package }

func (p Plugin) Run(ctx context.Context, env plugin.Env) (plugin.Result, error) {
	f := p.File
	page, err := env.Get(ctx, f.URL())
	if err != nil {
		return plugin.Result{}, fmt.Errorf("fetching %s: %w", f.URL(), err)
	}
	heading, findings, err := f.Read(page)
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
			API:           f.Package,
			Version:       f.Release,
			ChangeType:    c.ChangeType,
			Breaking:      c.ChangeType == event.ChangeTypeBreaking,
			Subject:       subject,
			Description:   describe(f, c),
			MigrationHint: cap2000(fmt.Sprintf("%s (%s's changelog for %s)", fd.Bullet.Text, f.Package, f.Release)),
			SourceURL:     f.URL(),
			SourceExcerpt: cap2000(heading + "\n" + fd.Section + "\n" + fd.Bullet.Raw),
			Confidence:    1.0,
			DetectedAt:    now,
		}
		e.Fingerprint = event.FingerprintV2(e.API, e.Version, e.ChangeType, subject, "")
		if err := e.Validate(); err != nil {
			rejected = append(rejected, fmt.Errorf("%s %s %v: %w", f.Package, f.Release, c.Symbols, err))
			continue
		}
		events = append(events, e)
	}
	return plugin.Result{
		Label:     f.URL(),
		Events:    events,
		Rejected:  rejected,
		Baselines: []plugin.Spec{plugin.Baseline(f.Package, f.Release, f.Tag, f.URL(), page, false, now)},
	}, nil
}

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
