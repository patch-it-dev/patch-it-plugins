// Package apisurface is the kind for a published package with no changelog
// worth reading: its public API is compared between two releases, read from
// the released files themselves. A class, function or method that is gone,
// or renamed, a parameter that is gone, or one that lost its default (callers
// must now pass it), is an event.
//
// Python first: every release on PyPI (ListVersions), the release's wheel
// (or sdist) read without running any of it (ReadAt: pysurface/ reads the
// .py and .pyi files as text), and the two surfaces compared (Extract). Each
// event cites the release file and quotes the definition's file and line in
// both releases. npm (.d.ts) is a follow-up.
//
// What isn't read, on purpose (precision over recall): private names (a
// leading underscore, a module under a private one or one named internal
// or vendor, a name left out of __all__), classes nested in classes,
// overloaded functions' and properties' parameters, self and cls, a removed
// parameter when the function now takes **kwargs (passing it still works),
// a name its module still binds (an alias, an import, a star import or a
// module __getattr__ that can't be listed), and a parameter newly added
// without a default (patch-it's change-event contract has no subject for it
// yet: logged).
package apisurface

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
	"github.com/patch-it-dev/patch-it-plugins/plugins/apisurface/pysurface"
)

// Package is one published package.
type Package struct {
	ID        string // the source id the events carry: "openai/python-surface"
	Name      string // on the registry: "openai"
	Ecosystem string // "pypi"
	Module    string // the top-level import name: "openai"
}

// Plugin reads one package's releases.
type Plugin struct {
	Package Package
}

func (p Plugin) Name() string { return p.Package.ID }

// ListVersions is every final release on the registry, oldest first.
func (p Plugin) ListVersions(ctx context.Context, env plugin.Env) ([]plugin.Version, error) {
	switch p.Package.Ecosystem {
	case "pypi":
		return versions.PyPIReleases(ctx, env.Get, p.Package.Name)
	default:
		return nil, fmt.Errorf("%s: reading a %q package's API isn't supported yet", p.Package.Name, p.Package.Ecosystem)
	}
}

// Release is what ReadAt keeps of a release: its public surface, the
// classes a method could move to, and what couldn't be read.
type Release struct {
	Artifact string             `json:"artifact"`
	Surface  pysurface.Surface  `json:"surface"`
	Classes  pysurface.Classes  `json:"classes,omitempty"`
	Bindings pysurface.Bindings `json:"bindings,omitempty"` // what each public module binds
	Skipped  []string           `json:"skipped,omitempty"`  // files over the size limit
}

// ReadAt downloads the release's wheel (or sdist) and reads its public
// surface. The snapshot's content is the surface as JSON, its SourceURL the
// file read.
func (p Plugin) ReadAt(ctx context.Context, env plugin.Env, v plugin.Version) (plugin.Snapshot, error) {
	pkg := p.Package
	a, err := pickArtifact(ctx, env, pkg.Name, v.ID)
	if err != nil {
		return plugin.Snapshot{}, err
	}
	body, err := env.Get(ctx, a.URL)
	if err != nil {
		return plugin.Snapshot{}, fmt.Errorf("fetching %s: %w", a.URL, err)
	}
	files, skipped, err := sourceFiles(a, body, pkg.Module)
	if err != nil {
		return plugin.Snapshot{}, err
	}
	if len(files) == 0 {
		return plugin.Snapshot{}, fmt.Errorf("%s has no %s/ source files", a.Filename, pkg.Module)
	}
	for _, s := range skipped {
		log.Printf("%s %s: %s skipped: over %d bytes", pkg.Name, v.ID, s, MaxFileBytes)
	}
	content, err := json.Marshal(Release{Artifact: a.Filename, Surface: pysurface.Read(files), Classes: pysurface.ReadClasses(files), Bindings: pysurface.ReadBindings(files), Skipped: skipped})
	if err != nil {
		return plugin.Snapshot{}, err
	}
	return plugin.Snapshot{Version: v, Content: content, SourceURL: a.URL}, nil
}

// Extract compares from's surface with to's: an event per class, function or
// method gone or renamed, and per parameter gone or now required, with to's
// version and from's as previous_version. A zero from has nothing to compare:
// no events.
func (p Plugin) Extract(_ context.Context, env plugin.Env, from, to plugin.Snapshot) (plugin.Result, error) {
	if from.IsZero() {
		return plugin.Result{}, nil
	}
	var a, b Release
	if err := json.Unmarshal(from.Content, &a); err != nil {
		return plugin.Result{}, fmt.Errorf("reading the %s %s surface: %w", p.Package.Name, from.Version.ID, err)
	}
	if err := json.Unmarshal(to.Content, &b); err != nil {
		return plugin.Result{}, fmt.Errorf("reading the %s %s surface: %w", p.Package.Name, to.Version.ID, err)
	}
	now := env.Now().UTC()
	var events []event.Event
	var rejected []error
	for _, c := range Compare(a, b) {
		if c.Change == "" {
			log.Printf("%s %s: not an event: %s", p.Package.Name, to.Version.ID, c.Note)
			continue
		}
		e := p.event(c, from, to, a.Artifact, b.Artifact, now)
		e.Fingerprint = event.Fingerprint(e.API, e.Version, e.ChangeType, e.Subject, "")
		if err := e.Validate(); err != nil {
			rejected = append(rejected, fmt.Errorf("%s %s %s: %w", p.Package.Name, to.Version.ID, c.Symbol, err))
			continue
		}
		events = append(events, e)
	}
	return plugin.Result{Events: events, Rejected: rejected}, nil
}

var _ plugin.Versioned = Plugin{}

// event is one change as a change event.
func (p Plugin) event(c Change, from, to plugin.Snapshot, fromFile, toFile string, now time.Time) event.Event {
	pkg := p.Package
	sym := p.local(c.Symbol)
	subject := event.Subject{Type: event.SubjectPackageRelease, Ecosystem: pkg.Ecosystem, Package: pkg.Name, Change: c.Change, Symbols: []string{sym}}
	var desc, hint string
	switch c.Change {
	case event.PackageRenamedSymbol:
		subject.Replacement = p.local(c.Replacement)
		desc = fmt.Sprintf("`%s` was renamed to `%s` in %s %s: code using the old name fails.", sym, subject.Replacement, pkg.Name, to.Version.ID)
		hint = fmt.Sprintf("Use `%s`, with the same parameters.", subject.Replacement)
	case event.PackageRemovedSymbol:
		desc = fmt.Sprintf("The %s `%s` was removed in %s %s: code using it fails.", c.Kind, sym, pkg.Name, to.Version.ID)
		hint = fmt.Sprintf("%s %s has no `%s`: see the package's release notes for what replaces it.", pkg.Name, to.Version.ID, sym)
	case event.PackageRemovedParameter:
		subject.Parameter = c.Parameter
		desc = fmt.Sprintf("`%s` no longer takes the parameter `%s` in %s %s: passing it raises a TypeError.", sym, c.Parameter, pkg.Name, to.Version.ID)
		hint = fmt.Sprintf("Stop passing `%s` to `%s`; its signature in %s is quoted in the source excerpt.", c.Parameter, sym, to.Version.ID)
	case event.PackageChangedDefault:
		subject.Parameter = c.Parameter
		desc = fmt.Sprintf("`%s`'s parameter `%s` has no default in %s %s (it was `%s`): a call that leaves it out raises a TypeError.", sym, c.Parameter, pkg.Name, to.Version.ID, c.OldDefault)
		hint = fmt.Sprintf("Pass `%s=%s` to `%s` to keep the old behavior.", c.Parameter, c.OldDefault, sym)
	}
	return event.Event{
		API:             pkg.ID,
		Version:         to.Version.ID,
		PreviousVersion: from.Version.ID,
		ChangeType:      event.ChangeTypeBreaking,
		Breaking:        true,
		Subject:         subject,
		Description:     desc,
		MigrationHint:   hint,
		SourceURL:       to.SourceURL,
		SourceExcerpt:   cap2000(excerpt(pkg.Name, from.Version.ID, fromFile, c.Before) + "\n" + excerpt(pkg.Name, to.Version.ID, toFile, c.After)),
		Confidence:      1.0,
		DetectedAt:      now,
	}
}

// local is a symbol as code writes it after importing the package:
// "openai.OpenAI.with_options" is "OpenAI.with_options".
func (p Plugin) local(sym string) string { return strings.TrimPrefix(sym, p.Package.Module+".") }

// excerpt quotes one release's definition: "openai 1.0.0, openai/_client.py
// line 12: def create(...)", or that the release has none.
func excerpt(pkg, version, file string, s *pysurface.Symbol) string {
	if s == nil {
		return fmt.Sprintf("%s %s (%s): not defined", pkg, version, file)
	}
	return fmt.Sprintf("%s %s (%s), %s line %d: %s", pkg, version, file, s.File, s.Line, s.Signature)
}

func cap2000(s string) string {
	if r := []rune(s); len(r) > 2000 {
		return string(r[:1999]) + "…"
	}
	return s
}

// Change is one difference between two surfaces: an event (Change set) or
// a difference that isn't one (Note says why).
type Change struct {
	Change      event.PackageChange
	Kind        string // the symbol's kind, for sentences
	Symbol      string // its full path: "openai.OpenAI.create"
	Replacement string // renamed_symbol: the new path
	Parameter   string
	OldDefault  string
	Before      *pysurface.Symbol // its definition in the older release
	After       *pysurface.Symbol // in the newer one; nil when gone
	Note        string
}

// Compare lists what changed from a to b, sorted by symbol: removed and
// renamed symbols (a member of a removed class isn't listed again), then
// for every symbol in both, removed parameters and parameters that lost
// their default. A rename is one removed and one added symbol of the same
// kind under the same parent with the same parameters, at least one of
// them besides self or cls; anything less sure is a removal.
//
// What looks removed but may not be is a note, not an event: a symbol
// defined in a file b couldn't read (over the size limit), a name its
// module still binds in b (an alias, an import, a star import from where
// names can't be listed), and a method
// whose class is still there but whose bases changed, which one of its
// bases (or theirs) now defines, or which the class now defines as an
// attribute (create = _create, a property made a plain attribute).
func Compare(ra, rb Release) []Change {
	a, b := ra.Surface, rb.Surface
	var out []Change
	removed := map[string]bool{}
	byParent := map[string][2][]string{} // parent+kind -> removed, added
	for k, s := range a {
		if _, ok := b[k]; !ok {
			removed[k] = true
			key := parentOf(k) + " " + s.Kind
			g := byParent[key]
			g[0] = append(g[0], k)
			byParent[key] = g
		}
	}
	for k, s := range b {
		if _, ok := a[k]; !ok {
			key := parentOf(k) + " " + s.Kind
			g := byParent[key]
			g[1] = append(g[1], k)
			byParent[key] = g
		}
	}
	renamed := map[string]string{}
	for _, g := range byParent {
		if len(g[0]) == 1 && len(g[1]) == 1 {
			old, now := a[g[0][0]], b[g[1][0]]
			if old.Params != nil && !old.Overloaded && !now.Overloaded && slices.Equal(old.Params, now.Params) && len(callerParams(old.Params)) > 0 {
				renamed[g[0][0]] = g[1][0]
			}
		}
	}
	for k := range removed {
		if removed[parentOf(k)] {
			continue // its class is reported
		}
		before := a[k]
		if why := notGone(k, before, ra, rb); why != "" {
			out = append(out, Change{Symbol: k, Note: fmt.Sprintf("%s: not in the newer release, but %s", k, why)})
			continue
		}
		c := Change{Change: event.PackageRemovedSymbol, Kind: before.Kind, Symbol: k, Before: &before}
		if r, ok := renamed[k]; ok {
			after := b[r]
			c.Change, c.Replacement, c.After = event.PackageRenamedSymbol, r, &after
		}
		out = append(out, c)
	}
	for k, old := range a {
		now, ok := b[k]
		if !ok || old.Overloaded || now.Overloaded || old.Params == nil || now.Params == nil {
			continue
		}
		before, after := old, now
		kwargs := slices.ContainsFunc(now.Params, func(p pysurface.Param) bool { return p.Kind == "kwargs" })
		for i, op := range old.Params {
			if op.Kind == "args" || op.Kind == "kwargs" || (i == 0 && (op.Name == "self" || op.Name == "cls")) {
				continue // a caller never passes self or cls
			}
			i := slices.IndexFunc(now.Params, func(p pysurface.Param) bool { return p.Name == op.Name })
			switch {
			case i < 0 && kwargs:
				out = append(out, Change{Symbol: k, Note: fmt.Sprintf("%s: parameter %s gone, but **kwargs still takes it", k, op.Name)})
			case i < 0:
				out = append(out, Change{Change: event.PackageRemovedParameter, Kind: old.Kind, Symbol: k, Parameter: op.Name, Before: &before, After: &after})
			case op.HasDef && !now.Params[i].HasDef && now.Params[i].Kind != "args" && now.Params[i].Kind != "kwargs":
				out = append(out, Change{Change: event.PackageChangedDefault, Kind: old.Kind, Symbol: k, Parameter: op.Name, OldDefault: op.Default, Before: &before, After: &after})
			}
		}
		for _, np := range now.Params {
			if np.HasDef || np.Kind == "args" || np.Kind == "kwargs" {
				continue
			}
			if !slices.ContainsFunc(old.Params, func(p pysurface.Param) bool { return p.Name == np.Name }) {
				out = append(out, Change{Symbol: k, Note: fmt.Sprintf("%s: new required parameter %s (no subject for it in the contract yet)", k, np.Name)})
			}
		}
	}
	slices.SortFunc(out, func(x, y Change) int {
		return strings.Compare(x.Symbol+" "+x.Parameter+" "+x.Note, y.Symbol+" "+y.Parameter+" "+y.Note)
	})
	return out
}

// callerParams are the parameters a caller passes: a leading self or cls
// left out.
func callerParams(ps []pysurface.Param) []pysurface.Param {
	if len(ps) > 0 && (ps[0].Name == "self" || ps[0].Name == "cls") {
		return ps[1:]
	}
	return ps
}

// notGone says why symbol k, defined as before in a and missing from b's
// surface, may still be there in b; "" when it's gone.
func notGone(k string, before pysurface.Symbol, a, b Release) string {
	if slices.Contains(b.Skipped, before.File) {
		return fmt.Sprintf("%s, where it was defined, was too big to read", before.File)
	}
	if mod, ok := b.Bindings[parentOf(k)]; ok {
		name := k[strings.LastIndexByte(k, '.')+1:]
		switch {
		case slices.Contains(mod.Names, name):
			return fmt.Sprintf("%s still binds the name (an alias, an import or a value now)", parentOf(k))
		case mod.Star:
			return fmt.Sprintf("%s star-imports names that can't be listed", parentOf(k))
		}
	}
	if before.Kind != "method" {
		return ""
	}
	cls, ok := b.Surface[parentOf(k)]
	if !ok || cls.Kind != "class" {
		return ""
	}
	name := k[strings.LastIndexByte(k, '.')+1:]
	oldBases, bases := pysurface.Bases(a.Surface[parentOf(k)].Signature), pysurface.Bases(cls.Signature)
	if !slices.Equal(oldBases, bases) {
		return fmt.Sprintf("its class's bases changed (%s to %s)", strings.Join(oldBases, ", "), strings.Join(bases, ", "))
	}
	if own, ok := b.Classes[className(cls.Signature)]; ok && slices.Contains(own.Members, name) {
		return "its class still defines the name (now an attribute)"
	}
	seen := map[string]bool{}
	for len(bases) > 0 {
		base := bases[0]
		bases = bases[1:]
		if seen[base] {
			continue
		}
		seen[base] = true
		c, ok := b.Classes[base]
		if !ok {
			continue // not the package's: unchanged, as the bases are
		}
		if slices.Contains(c.Members, name) {
			return fmt.Sprintf("its base class %s defines it", base)
		}
		bases = append(bases, c.Bases...)
	}
	return ""
}

// className is "Completions" for "class Completions(SyncAPIResource): ...".
func className(signature string) string {
	n := strings.TrimPrefix(signature, "class ")
	if i := strings.IndexAny(n, "(: "); i >= 0 {
		n = n[:i]
	}
	return n
}

// parentOf is "openai.OpenAI" for "openai.OpenAI.create".
func parentOf(sym string) string {
	if i := strings.LastIndexByte(sym, '.'); i >= 0 {
		return sym[:i]
	}
	return ""
}
