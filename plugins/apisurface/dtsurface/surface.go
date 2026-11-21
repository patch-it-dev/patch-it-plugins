package dtsurface

import (
	"encoding/json"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/plugins/apisurface/pysurface"
)

// File is one file of the package, its path from the package's root
// ("index.d.ts", "package.json"), and its text.
type File struct {
	Path string
	Text string
}

// MaxDepth is how deep namespaces and classes are followed from an entry
// point: "openai.OpenAI.Chat.Completions.create" is four below the entry.
const MaxDepth = 5

// Read is the public surface of the package whose files these are: for
// each entry point package.json declares (its "exports", or its "types"),
// every name it exports, and below a class or namespace what it holds (a
// class's public methods and properties, a namespace's exports), keyed by
// the import path code writes ("openai.OpenAI.chat", "openai/resources.Chat")
// and in the same shape as the Python reader's, so one comparison serves
// both. Parameters are positional (Kind "positional", or "args" for the
// rest); HasDef is "optional".
//
// What isn't read, on purpose: names marked @internal, private and
// protected members, interface and object-type members (an interface is
// one name), interfaces and type aliases inside a namespace (a generated
// SDK re-exports each of its thousands of types through namespaces at
// several paths; an entry point's own exports are read, and a type is only
// a type: the code that runs is the classes and functions), names an entry point re-exports from another package (one
// name each, nothing below), and entry points under a private path ("_" or
// "internal"). Bindings are what each entry point exports, Star when it
// re-exports everything of another package (whose names aren't read).
func Read(pkgName string, files []File) (pysurface.Surface, pysurface.Classes, pysurface.Bindings) {
	r := newReader(pkgName, files)
	out := pysurface.Surface{}
	binds := pysurface.Bindings{}
	for _, e := range r.entries() {
		sc := e.scope
		names, star := r.exportedNames(sc, 0, map[*scope]bool{})
		slices.Sort(names)
		binds[e.spec] = pysurface.Binding{Names: names, Star: star}
		for _, n := range names {
			t := r.lookupExport(sc, n, 0)
			if n == "default" {
				// The default export is the same declaration as a
				// named one, almost always: its name, not again all
				// it holds.
				if t.ok {
					out[e.spec+".default"] = r.head(t)
				}
				continue
			}
			r.expand(out, e.spec+"."+n, t, 0, map[any]bool{})
		}
	}
	return out, r.classes(), binds
}

type reader struct {
	name    string
	files   map[string]string // declaration files and package.json, by path
	mods    map[string]*scope // parsed declaration files, by path
	ambient map[string]*scope // `declare module "x"` bodies, by spec
}

func newReader(pkgName string, files []File) *reader {
	r := &reader{name: pkgName, files: map[string]string{}, mods: map[string]*scope{}, ambient: map[string]*scope{}}
	for _, f := range files {
		r.files[f.Path] = f.Text
	}
	var paths []string
	for p := range r.files {
		if isDTS(p) {
			paths = append(paths, p)
		}
	}
	slices.Sort(paths)
	for _, p := range paths {
		r.mods[p] = parseFile(p, r.files[p], r.ambient)
	}
	return r
}

func isDTS(p string) bool {
	return strings.HasSuffix(p, ".d.ts") || strings.HasSuffix(p, ".d.mts") || strings.HasSuffix(p, ".d.cts")
}

// entry is one entry point: the import path code writes, and its module.
type entry struct {
	spec  string
	scope *scope
}

// entries are the package's entry points: package.json's "exports" (each
// subpath, wildcards matched against the files), else its "types" or
// "typings", else index.d.ts. A subpath under a private path, or one that
// names a .js file (a duplicate of the extensionless one), isn't an entry.
func (r *reader) entries() []entry {
	var pj struct {
		Types   string          `json:"types"`
		Typings string          `json:"typings"`
		Main    string          `json:"main"`
		Exports json.RawMessage `json:"exports"`
	}
	_ = json.Unmarshal([]byte(r.files["package.json"]), &pj) // no package.json: index.d.ts
	subpaths := map[string]string{}                          // subpath -> declaration file
	var exp any
	if len(pj.Exports) > 0 && json.Unmarshal(pj.Exports, &exp) == nil {
		m, isMap := exp.(map[string]any)
		if !isMap || !hasSubpaths(m) {
			exp = map[string]any{".": exp}
			m = exp.(map[string]any)
		}
		for sub, target := range m {
			if !strings.HasPrefix(sub, ".") || privatePath(sub) || strings.HasSuffix(sub, ".json") || jsExt.MatchString(sub) || sub == "./index" || strings.HasSuffix(sub, "/index") {
				continue
			}
			file := typesOf(target)
			if file == "" {
				continue
			}
			if strings.Contains(sub, "*") {
				for k, v := range r.wildcard(sub, file) {
					subpaths[k] = v
				}
				continue
			}
			if f := r.dtsFor(file); f != "" {
				subpaths[sub] = f
			}
		}
	} else {
		for _, f := range []string{pj.Types, pj.Typings, pj.Main, "index.d.ts"} {
			if f = r.dtsFor(f); f != "" {
				subpaths["."] = f
				break
			}
		}
	}
	var out []entry
	for sub, file := range subpaths {
		spec := r.name + strings.TrimPrefix(sub, ".")
		sc := r.mods[file]
		if a, ok := r.ambient[spec]; ok && len(sc.exports) == 0 && len(sc.stars) == 0 && !hasExported(sc) {
			sc = a // the file only declares the module
		}
		out = append(out, entry{spec: spec, scope: sc})
	}
	// The ambient module of the package itself, when no file is its entry.
	if len(out) == 0 {
		if a, ok := r.ambient[r.name]; ok {
			out = append(out, entry{spec: r.name, scope: a})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].spec < out[j].spec })
	return out
}

func hasExported(sc *scope) bool {
	for _, ds := range sc.decls {
		for _, d := range ds {
			if d.exported {
				return true
			}
		}
	}
	return false
}

var jsExt = regexp.MustCompile(`\.(m|c)?js$`)

func hasSubpaths(m map[string]any) bool {
	for k := range m {
		if strings.HasPrefix(k, ".") {
			return true
		}
	}
	return false
}

// privatePath is a subpath or file path with a part starting with "_" or
// named internal or src.
func privatePath(p string) bool {
	for _, part := range strings.Split(strings.TrimPrefix(p, "./"), "/") {
		if strings.HasPrefix(part, "_") || part == "internal" || part == "src" {
			return true
		}
	}
	return false
}

// typesOf is the file an "exports" target resolves to for TypeScript: its
// "types" condition, else (in order) import, require, node, default, else
// any condition, a .js target read as its declaration file.
func typesOf(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []any:
		if len(t) > 0 {
			return typesOf(t[0])
		}
	case map[string]any:
		for _, k := range []string{"types", "import", "require", "node", "default"} {
			if x, ok := t[k]; ok {
				if s := typesOf(x); s != "" {
					return s
				}
			}
		}
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			if s := typesOf(t[k]); s != "" {
				return s
			}
		}
	}
	return ""
}

// dtsFor is the declaration file for a path in the package: itself, or a
// .js file's .d.ts beside it (.mjs .d.mts, .cjs .d.cts), or a directory's
// index; "" when there's none.
func (r *reader) dtsFor(p string) string {
	if p == "" {
		return ""
	}
	p = path.Clean(strings.TrimPrefix(p, "./"))
	base := jsExt.ReplaceAllString(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(p, ".d.ts"), ".d.mts"), ".d.cts"), "")
	var cands []string
	if isDTS(p) {
		cands = append(cands, p)
	}
	switch {
	case strings.HasSuffix(p, ".mjs"):
		cands = append(cands, base+".d.mts")
	case strings.HasSuffix(p, ".cjs"):
		cands = append(cands, base+".d.cts")
	}
	cands = append(cands, base+".d.ts", base+".d.mts", base+".d.cts", base+"/index.d.ts", base+"/index.d.mts", base+"/index.d.cts")
	for _, c := range cands {
		if _, ok := r.mods[c]; ok {
			return c
		}
	}
	return ""
}

// wildcard matches a subpath pattern ("./resources/*") against the files
// its target pattern ("./resources/*.mjs") names.
func (r *reader) wildcard(sub, target string) map[string]string {
	out := map[string]string{}
	t := path.Clean(strings.TrimPrefix(target, "./"))
	pre, post, _ := strings.Cut(t, "*")
	post = jsExt.ReplaceAllString(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(post, ".d.ts"), ".d.mts"), ".d.cts"), "")
	for f := range r.mods {
		if !strings.HasPrefix(f, pre) {
			continue
		}
		rest := strings.TrimPrefix(f, pre)
		for _, ext := range []string{".d.ts", ".d.mts", ".d.cts"} {
			if stem, ok := strings.CutSuffix(rest, post+ext); ok && stem != "" {
				s := strings.Replace(sub, "*", stem, 1)
				if privatePath(s) || s == "./index" || strings.HasSuffix(s, "/index") {
					continue
				}
				if have, ok := out[s]; !ok || f < have {
					out[s] = r.dtsFor(f)
				}
			}
		}
	}
	return out
}

// target is what a name resolves to.
type target struct {
	decls    []*decl
	module   *scope    // a namespace import, `export * as`
	external exportRef // a name from another package
	from     string    // the file that names it
	ok       bool
}

// module resolves a module specifier from a file: a relative path to the
// package's declaration file, the package's own name to its entry, an
// ambient module; nil for another package.
func (r *reader) module(from, spec string) *scope {
	if a, ok := r.ambient[spec]; ok {
		return a
	}
	if strings.HasPrefix(spec, ".") {
		if f := r.dtsFor(path.Join(path.Dir(from), spec)); f != "" {
			return r.mods[f]
		}
	}
	return nil
}

// exportedNames are the names a scope exports, and whether it re-exports
// all of another package (names it can't list).
func (r *reader) exportedNames(sc *scope, hops int, seen map[*scope]bool) ([]string, bool) {
	if sc == nil || seen[sc] || hops > 8 {
		return nil, false
	}
	seen[sc] = true
	set := map[string]bool{}
	star := false
	for n := range sc.exports {
		set[n] = true
	}
	for _, n := range sc.order {
		for _, d := range sc.decls[n] {
			if d.exported || (sc.ambient && !sc.explicit) {
				set[n] = true
			}
		}
	}
	if sc.assign != "" {
		// `export = X`: the module is X's namespace members.
		t := r.lookupLocal(sc, sc.assign, 0)
		for _, d := range t.decls {
			if d.ns != nil {
				ns, s := r.exportedNames(d.ns, hops+1, seen)
				for _, n := range ns {
					set[n] = true
				}
				star = star || s
			}
		}
	}
	for _, st := range sc.stars {
		m := r.module(sc.file, st.spec)
		if m == nil {
			star = true
			continue
		}
		ns, s := r.exportedNames(m, hops+1, seen)
		for _, n := range ns {
			if n != "default" {
				set[n] = true
			}
		}
		star = star || s
	}
	out := make([]string, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	return out, star
}

// lookupExport resolves a name a scope exports.
func (r *reader) lookupExport(sc *scope, name string, hops int) target {
	if sc == nil || hops > 12 {
		return target{}
	}
	if e, ok := sc.exports[name]; ok {
		if e.spec != "" {
			m := r.module(sc.file, e.spec)
			if m == nil {
				return target{external: e, from: sc.file, ok: true}
			}
			if e.name == "*" {
				return target{module: m, from: sc.file, ok: true}
			}
			return r.lookupExport(m, e.name, hops+1)
		}
		return r.lookupLocal(sc, e.local, hops+1)
	}
	var ds []*decl
	for _, d := range sc.decls[name] {
		if d.exported || (sc.ambient && !sc.explicit) {
			ds = append(ds, d)
		}
	}
	if len(ds) > 0 {
		return target{decls: ds, from: sc.file, ok: true}
	}
	if sc.assign != "" {
		for _, d := range r.lookupLocal(sc, sc.assign, hops+1).decls {
			if d.ns != nil {
				if t := r.lookupExport(d.ns, name, hops+1); t.ok {
					return t
				}
			}
		}
	}
	if name != "default" {
		for _, st := range sc.stars {
			if m := r.module(sc.file, st.spec); m != nil {
				if t := r.lookupExport(m, name, hops+1); t.ok {
					return t
				}
			}
		}
	}
	return target{}
}

// lookupLocal resolves a name as written in a scope: its declarations, an
// import, an alias (import X = A.B), else the enclosing scope's.
func (r *reader) lookupLocal(sc *scope, dotted string, hops int) target {
	if sc == nil || hops > 12 {
		return target{}
	}
	first, rest, more := strings.Cut(dotted, ".")
	var t target
	switch {
	case len(sc.decls[first]) > 0:
		t = target{decls: sc.decls[first], from: sc.file, ok: true}
	case sc.imports[first].spec != "":
		imp := sc.imports[first]
		m := r.module(sc.file, imp.spec)
		switch {
		case m == nil:
			t = target{external: exportRef{spec: imp.spec, name: imp.name, sig: "import " + first + " from " + quote(imp.spec)}, from: sc.file, ok: true}
		case imp.name == "*":
			t = target{module: m, from: sc.file, ok: true}
		case imp.name == "=":
			if m.assign != "" {
				t = r.lookupLocal(m, m.assign, hops+1)
			} else {
				t = target{module: m, from: sc.file, ok: true}
			}
		default:
			t = r.lookupExport(m, imp.name, hops+1)
		}
	case sc.aliases[first] != "":
		t = r.lookupLocal(sc, sc.aliases[first], hops+1)
	default:
		return r.lookupLocal(sc.parent, dotted, hops+1)
	}
	if !more {
		return t
	}
	for _, part := range strings.Split(rest, ".") {
		switch {
		case t.module != nil:
			t = r.lookupExport(t.module, part, hops+1)
		case len(t.decls) > 0:
			var next target
			for _, d := range t.decls {
				if d.ns != nil {
					if next = r.lookupExport(d.ns, part, hops+1); next.ok {
						break
					}
				}
			}
			t = next
		default:
			return target{}
		}
		if !t.ok {
			return t
		}
	}
	return t
}

// kindOrder is which declaration of a merged name is the symbol.
var kindOrder = []string{"class", "function", "enum", "variable", "interface", "type", "namespace"}

// expand adds the symbol a target is at key, and what it holds.
func (r *reader) expand(out pysurface.Surface, key string, t target, depth int, seen map[any]bool) {
	if !t.ok || depth > MaxDepth {
		return
	}
	switch {
	case t.external.spec != "":
		out[key] = r.head(t)
	case t.module != nil:
		if seen[t.module] {
			return
		}
		seen[t.module] = true
		defer delete(seen, t.module)
		out[key] = r.head(t)
		names, _ := r.exportedNames(t.module, 0, map[*scope]bool{})
		for _, n := range names {
			r.expand(out, key+"."+n, r.lookupExport(t.module, n, 0), depth+1, seen)
		}
	case len(t.decls) > 0:
		if seen[t.decls[0]] {
			return
		}
		if sym := symbol(t.decls); depth > 0 && (sym.Kind == "interface" || sym.Kind == "type") {
			return // a type inside a namespace: see Read
		}
		seen[t.decls[0]] = true
		defer delete(seen, t.decls[0])
		out[key] = symbol(t.decls)
		for _, d := range t.decls {
			if d.kind == "class" {
				byName := map[string][]*member{}
				for _, m := range d.members {
					byName[m.name] = append(byName[m.name], m)
				}
				for n, ms := range byName {
					out[key+"."+n] = memberSymbol(d.file, ms)
				}
			}
			if d.ns != nil {
				names, _ := r.exportedNames(d.ns, 0, map[*scope]bool{})
				for _, n := range names {
					r.expand(out, key+"."+n, r.lookupExport(d.ns, n, 0), depth+1, seen)
				}
			}
		}
	}
}

// head is a target's own symbol, without what it holds.
func (r *reader) head(t target) pysurface.Symbol {
	switch {
	case t.external.spec != "":
		return pysurface.Symbol{Kind: "export", File: t.from, Line: t.external.line, Signature: t.external.sig}
	case t.module != nil:
		return pysurface.Symbol{Kind: "namespace", File: t.module.file, Line: 1, Signature: "module " + t.module.file}
	}
	return symbol(t.decls)
}

// symbol is a merged name's symbol: its first declaration of the first
// kind in kindOrder; a function declared more than once is overloaded, a
// class's parameters are its constructor's.
func symbol(ds []*decl) pysurface.Symbol {
	var primary *decl
	for _, k := range kindOrder {
		for _, d := range ds {
			if d.kind == k {
				primary = d
				break
			}
		}
		if primary != nil {
			break
		}
	}
	s := pysurface.Symbol{Kind: primary.kind, File: primary.file, Line: primary.line, Signature: primary.sig}
	switch primary.kind {
	case "function":
		n := 0
		for _, d := range ds {
			if d.kind == "function" {
				n++
			}
		}
		if n > 1 {
			s.Overloaded = true
		} else {
			s.Params = params(primary.params)
		}
	case "class":
		switch {
		case primary.ctors > 1:
			s.Overloaded = true
		case primary.hasParam:
			s.Params = params(primary.params)
		}
	}
	return s
}

func memberSymbol(file string, ms []*member) pysurface.Symbol {
	m := ms[0]
	s := pysurface.Symbol{Kind: m.kind, File: file, Line: m.line, Signature: m.sig}
	if m.kind == "method" {
		if len(ms) > 1 {
			s.Overloaded = true
		} else {
			s.Params = params(m.params)
		}
	}
	return s
}

func params(ps []Param) []pysurface.Param {
	out := make([]pysurface.Param, 0, len(ps))
	for _, p := range ps {
		k := "positional"
		if p.Rest {
			k = "args"
		}
		out = append(out, pysurface.Param{Name: p.Name, Kind: k, HasDef: p.Optional})
	}
	return out
}

// classes are every class the package declares, by name: its base and its
// public members, for following a method that moved to a base class.
func (r *reader) classes() pysurface.Classes {
	out := pysurface.Classes{}
	var walk func(sc *scope)
	walk = func(sc *scope) {
		for _, n := range sc.order {
			for _, d := range sc.decls[n] {
				if d.kind == "class" {
					c := out[n]
					for _, b := range d.bases {
						if !slices.Contains(c.Bases, b) {
							c.Bases = append(c.Bases, b)
						}
					}
					for _, m := range d.members {
						if !slices.Contains(c.Members, m.name) {
							c.Members = append(c.Members, m.name)
						}
					}
					slices.Sort(c.Bases)
					slices.Sort(c.Members)
					out[n] = c
				}
				if d.ns != nil {
					walk(d.ns)
				}
			}
		}
	}
	for _, sc := range r.mods {
		walk(sc)
	}
	for _, sc := range r.ambient {
		walk(sc)
	}
	return out
}
