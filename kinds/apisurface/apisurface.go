// Package apisurface is the kind for a published package with no changelog
// worth reading: its public API is compared between two releases, read from
// the released files themselves. A class, function or method that is gone,
// or renamed, a parameter that is gone, or one that lost its default (callers
// must now pass it), is a breaking event; a module, class, function or method
// that is new, or a new parameter with a default, is an added one.
//
// Every release on PyPI or npm (ListVersions), the release's files read
// without running any of them (ReadAt: pysurface/ reads a wheel's or sdist's
// .py and .pyi files as text; dtsurface/ reads an npm tarball's TypeScript
// declarations, .d.ts, .d.mts and .d.cts, as text), and the two surfaces
// compared (Extract): Python's parameters by name, TypeScript's by
// position. Each event cites the release file and quotes the definition's
// file and line in both releases; one declaration reachable at several
// import paths is one event.
//
// What isn't read, on purpose (precision over recall): private names (a
// leading underscore, a module under a private one or one named internal
// or vendor, a name left out of __all__), classes nested in classes,
// overloaded functions' and properties' parameters, self and cls, a removed
// parameter when the function now takes **kwargs (passing it still works),
// a name its module still binds (an alias, an import, a star import or a
// module __getattr__ that can't be listed), and a parameter newly added
// without a default (patch-it's change-event contract has no subject for it
// yet: logged). The same rules, the other way round, keep a name that only
// looks new (the older release bound it, or couldn't be read) from being an
// addition; the members of a new class or module are part of its addition.
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
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/pysurface"
)

// Package is one published package.
type Package struct {
	ID        string // the source id the events carry: "openai/python-surface"
	Name      string // on the registry: "openai"
	Ecosystem string // "pypi" or "npm"
	Module    string // the top-level import name: "openai"; an npm package's own name
}

// Plugin reads one package's releases.
type Plugin struct {
	Package Package

	shape *plugin.Shape // the source's, set by RegisterProvider (WithShape); nil: no guard
}

// DefaultMinRatio is the shape guard's ratio for a package's surface: a
// release whose public API reads as less than half the one before is a
// package that moved its code (a new top-level module, declarations
// shipped elsewhere) or a reader that no longer finds it, not half an API
// removed. A source that does shrink that much on purpose sets its own.
const DefaultMinRatio = 0.5

func (p Plugin) Name() string { return p.Package.ID }

// WithShape is p guarding its pairs with s.
func (p Plugin) WithShape(s plugin.Shape) plugin.Plugin {
	p.shape = &s
	return p
}

// ListVersions is every final release on the registry, oldest first.
func (p Plugin) ListVersions(ctx context.Context, env plugin.Env) ([]plugin.Version, error) {
	switch p.Package.Ecosystem {
	case "pypi":
		return versions.PyPIReleases(ctx, env.Get, p.Package.Name)
	case "npm":
		return versions.NPMReleases(ctx, env.Get, p.Package.Name)
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
	// Positional is a release whose calls pass arguments by position
	// (npm, read from TypeScript declarations): parameters compare by
	// place, not by name.
	Positional bool `json:"positional,omitempty"`
}

// ReadAt downloads the release's wheel (or sdist), or an npm release's
// tarball (readNPM), and reads its public surface. The snapshot's content is the surface as JSON, its SourceURL the
// file read.
func (p Plugin) ReadAt(ctx context.Context, env plugin.Env, v plugin.Version) (plugin.Snapshot, error) {
	pkg := p.Package
	if pkg.Ecosystem == "npm" {
		return p.readNPM(ctx, env, v)
	}
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
// method gone, renamed or new, and per parameter gone, now required or new
// (with a default), with to's version and from's as previous_version. A zero from has nothing to compare:
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
	if err := p.shape.CheckCount("public names", DefaultMinRatio, len(a.Surface), len(b.Surface)); err != nil {
		return plugin.Result{}, fmt.Errorf("the %s %s surface: %w", p.Package.Name, to.Version.ID, err)
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
		e.Fingerprint = event.Fingerprint(e.API, e.Version, e.Change, e.Subject, "")
		if err := e.Validate(); err != nil {
			rejected = append(rejected, fmt.Errorf("%s %s %s: %w", p.Package.Name, to.Version.ID, c.Symbol, err))
			continue
		}
		events = append(events, e)
	}
	return plugin.Result{Events: events, Rejected: rejected}, nil
}

var (
	_ plugin.Versioned = Plugin{}
	_ plugin.Guarded   = Plugin{}
)

// event is one change as a change event.
func (p Plugin) event(c Change, from, to plugin.Snapshot, fromFile, toFile string, now time.Time) event.Event {
	pkg := p.Package
	sym := p.local(c.Symbol)
	subject := event.Subject{Type: event.SubjectPackageRelease, Ecosystem: pkg.Ecosystem, Package: pkg.Name, Change: c.Change, Symbols: []string{sym}}
	for _, a := range c.Also {
		subject.Symbols = append(subject.Symbols, p.local(a))
	}
	var desc, hint string
	change := event.ChangeBreaking
	switch c.Change {
	case event.PackageAddedSymbol:
		change = event.ChangeAdded
		desc = fmt.Sprintf("New %s `%s` in %s %s.", c.Kind, sym, pkg.Name, to.Version.ID)
	case event.PackageAddedParameter:
		change = event.ChangeAdded
		subject.Parameter = c.Parameter
		desc = fmt.Sprintf("`%s` takes a new optional parameter `%s` in %s %s.", sym, c.Parameter, pkg.Name, to.Version.ID)
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
		if pkg.Ecosystem == "npm" {
			desc = fmt.Sprintf("`%s` no longer takes the parameter `%s` in %s %s: TypeScript rejects a call that passes it.", sym, c.Parameter, pkg.Name, to.Version.ID)
		}
		hint = fmt.Sprintf("Stop passing `%s` to `%s`; its signature in %s is quoted in the source excerpt.", c.Parameter, sym, to.Version.ID)
	case event.PackageChangedDefault:
		subject.Parameter = c.Parameter
		desc = fmt.Sprintf("`%s`'s parameter `%s` has no default in %s %s (it was `%s`): a call that leaves it out raises a TypeError.", sym, c.Parameter, pkg.Name, to.Version.ID, c.OldDefault)
		hint = fmt.Sprintf("Pass `%s=%s` to `%s` to keep the old behavior.", c.Parameter, c.OldDefault, sym)
		if pkg.Ecosystem == "npm" {
			desc = fmt.Sprintf("`%s`'s parameter `%s` is no longer optional in %s %s: TypeScript rejects a call that leaves it out.", sym, c.Parameter, pkg.Name, to.Version.ID)
			hint = fmt.Sprintf("Pass `%s` to `%s`; its signature in %s is quoted in the source excerpt.", c.Parameter, sym, to.Version.ID)
		}
	}
	return event.Event{
		API:             pkg.ID,
		Version:         to.Version.ID,
		PreviousVersion: from.Version.ID,
		Change:          change,
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
	// Also are the other import paths of the same declaration with the
	// same change: one declaration is one change.
	Also []string
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
	renamed, renamedTo := map[string]string{}, map[string]bool{}
	for _, g := range byParent {
		if len(g[0]) == 1 && len(g[1]) == 1 {
			old, now := a[g[0][0]], b[g[1][0]]
			if old.Params != nil && !old.Overloaded && !now.Overloaded && slices.Equal(old.Params, now.Params) && len(callerParams(old.Params)) > 0 {
				renamed[g[0][0]], renamedTo[g[1][0]] = g[1][0], true
			}
		}
	}
	movedUp := map[string]bool{} // "Base.name": a method its subclasses had, now on their base
	for k := range removed {
		if removed[parentOf(k)] {
			continue // its class is reported
		}
		before := a[k]
		if why := notGone(k, before, ra, rb); why != "" {
			if base, ok := strings.CutPrefix(why, "its base class "); ok {
				movedUp[strings.TrimSuffix(base, " defines it")+"."+k[strings.LastIndexByte(k, '.')+1:]] = true
			}
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
	for k, s := range b {
		if _, ok := a[k]; ok || renamedTo[k] {
			continue
		}
		if _, ok := b[parentOf(k)]; ok {
			if _, had := a[parentOf(k)]; !had {
				continue // its class or module is the addition
			}
		}
		if parent, ok := b[parentOf(k)]; ok && movedUp[className(parent.Signature)+"."+k[strings.LastIndexByte(k, '.')+1:]] {
			continue // moved up from its subclasses: not new to them
		}
		after := s
		if why := notGone(k, after, rb, ra); why != "" {
			out = append(out, Change{Symbol: k, Note: fmt.Sprintf("%s: not in the older release, but %s", k, why)})
			continue
		}
		out = append(out, Change{Change: event.PackageAddedSymbol, Kind: s.Kind, Symbol: k, After: &after})
	}
	for k, old := range a {
		now, ok := b[k]
		if !ok || old.Overloaded || now.Overloaded || old.Params == nil || now.Params == nil {
			continue
		}
		before, after := old, now
		if rb.Positional {
			out = append(out, comparePositional(k, old, now, &before, &after)...)
			continue
		}
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
			if np.Kind == "args" || np.Kind == "kwargs" || slices.ContainsFunc(old.Params, func(p pysurface.Param) bool { return p.Name == np.Name }) {
				continue
			}
			if np.HasDef {
				out = append(out, Change{Change: event.PackageAddedParameter, Kind: now.Kind, Symbol: k, Parameter: np.Name, Before: &before, After: &after})
				continue
			}
			out = append(out, Change{Symbol: k, Note: fmt.Sprintf("%s: new required parameter %s (no subject for it in the contract yet)", k, np.Name)})
		}
	}
	out = merge(out)
	slices.SortFunc(out, func(x, y Change) int {
		return strings.Compare(x.Symbol+" "+x.Parameter+" "+x.Note, y.Symbol+" "+y.Parameter+" "+y.Note)
	})
	return out
}

// merge makes the changes one declaration has at several import paths
// (a class defined in one module and re-exported by its package) one
// change: the shortest path is its Symbol, the others its Also.
func merge(cs []Change) []Change {
	type key struct {
		change                     event.PackageChange
		file, name, sig, parameter string
		line                       int
	}
	groups := map[key][]int{}
	var out []Change
	for i, c := range cs {
		// A declaration is where the change is read: the older release's for
		// what's gone or changed, the newer's for what's new.
		at := c.Before
		if c.Change == event.PackageAddedSymbol || c.Change == event.PackageAddedParameter {
			at = c.After
		}
		if at == nil || c.Change == "" || c.Change == event.PackageRenamedSymbol {
			out = append(out, c)
			continue
		}
		k := key{c.Change, at.File, c.Symbol[strings.LastIndexByte(c.Symbol, '.')+1:], at.Signature, c.Parameter, at.Line}
		groups[k] = append(groups[k], i)
	}
	for _, idx := range groups {
		slices.SortFunc(idx, func(x, y int) int {
			a, b := cs[x].Symbol, cs[y].Symbol
			if len(a) != len(b) {
				return len(a) - len(b)
			}
			return strings.Compare(a, b)
		})
		c := cs[idx[0]]
		for _, i := range idx[1:] {
			c.Also = append(c.Also, cs[i].Symbol)
		}
		slices.Sort(c.Also)
		out = append(out, c)
	}
	return out
}

// comparePositional compares two signatures whose arguments are passed by
// position (TypeScript): a parameter gone off the end, when the newer one
// takes no rest parameter, is removed; one optional before and required now,
// under the same name, lost its default (under another name, a parameter
// may have been inserted before it: a note); a rename alone is no change. A required parameter added at the
// end is a note (no subject for it in the contract yet).
func comparePositional(k string, old, now pysurface.Symbol, before, after *pysurface.Symbol) []Change {
	var out []Change
	rest := slices.ContainsFunc(now.Params, func(p pysurface.Param) bool { return p.Kind == "args" })
	for i, op := range old.Params {
		if op.Kind == "args" {
			break
		}
		switch {
		case i >= len(now.Params) && rest:
			out = append(out, Change{Symbol: k, Note: fmt.Sprintf("%s: parameter %s gone, but a rest parameter still takes it", k, op.Name)})
		case i >= len(now.Params):
			out = append(out, Change{Change: event.PackageRemovedParameter, Kind: old.Kind, Symbol: k, Parameter: op.Name, Before: before, After: after})
		case now.Params[i].Kind == "args":
			return out
		case op.HasDef && !now.Params[i].HasDef && now.Params[i].Name == op.Name:
			out = append(out, Change{Change: event.PackageChangedDefault, Kind: old.Kind, Symbol: k, Parameter: op.Name, Before: before, After: after})
		case op.HasDef && !now.Params[i].HasDef:
			// Another name at the place: a parameter inserted before it,
			// or a rename. Which one can't be told for sure.
			out = append(out, Change{Symbol: k, Note: fmt.Sprintf("%s: parameter %d is now %s, required (was %s, optional)", k, i+1, now.Params[i].Name, op.Name)})
		}
	}
	// A new parameter at the end is read only when every older one is still
	// in its place under its name: otherwise one may have moved.
	oldRest := slices.ContainsFunc(old.Params, func(p pysurface.Param) bool { return p.Kind == "args" })
	inPlace := len(now.Params) >= len(old.Params)
	for i := 0; inPlace && i < len(old.Params); i++ {
		inPlace = old.Params[i].Name == now.Params[i].Name
	}
	for i := len(old.Params); i < len(now.Params) && !oldRest; i++ {
		switch np := now.Params[i]; {
		case np.Kind == "args":
		case np.HasDef && inPlace:
			out = append(out, Change{Change: event.PackageAddedParameter, Kind: now.Kind, Symbol: k, Parameter: np.Name, Before: before, After: after})
		case np.HasDef:
		default:
			out = append(out, Change{Symbol: k, Note: fmt.Sprintf("%s: new required parameter %s (no subject for it in the contract yet)", k, np.Name)})
		}
	}
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
	if parent, ok := b.Surface[parentOf(k)]; ok {
		switch parent.Kind {
		case "export":
			return fmt.Sprintf("%s is now re-exported from another package, whose members aren't read", parentOf(k))
		case "variable", "interface", "type":
			return fmt.Sprintf("%s is now a %s, whose members aren't read", parentOf(k), parent.Kind)
		}
	}
	if before.Kind != "method" && before.Kind != "property" {
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
	if b.Positional {
		// TypeScript: a base class from another package isn't read, and
		// may define it.
		for _, base := range b.Classes[className(cls.Signature)].Bases {
			if _, ours := b.Classes[base]; !ours {
				return fmt.Sprintf("its class extends %s, from another package, which may define it", base)
			}
		}
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
