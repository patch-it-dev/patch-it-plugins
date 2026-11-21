package pysurface

import (
	"path"
	"regexp"
	"slices"
	"strings"
)

// Surface is a package's public API at one version: every public symbol by
// its import path ("openai.OpenAI", "openai.OpenAI.with_options").
type Surface map[string]Symbol

// Symbol is one class, function or method code can reach.
type Symbol struct {
	Kind string `json:"kind"` // "class", "function", "method"
	File string `json:"file"` // where it's defined, in the artifact
	Line int    `json:"line"` // its def or class line there
	// Signature is the header as written, whitespace collapsed: "def
	// create(self, *, model: str, n: int | None = None) -> Completion". A
	// class's is its class line, then its __init__ when it has one.
	Signature string `json:"signature"`
	// Params are what a call passes, in order (a class's are its
	// __init__'s); nil when they can't be read for sure (Overloaded).
	Params []Param `json:"params,omitempty"`
	// Overloaded is a function written as @overload variants: its
	// parameters aren't compared.
	Overloaded bool `json:"overloaded,omitempty"`
}

// Param is one parameter.
type Param struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`              // "positional" (before /), "either", "keyword" (after *), "args" (*a), "kwargs" (**k)
	Default string `json:"default,omitempty"` // as written; "" with HasDefault false is no default
	HasDef  bool   `json:"has_default,omitempty"`
}

// File is one source file of the package: its path in the artifact, with the
// package's top directory first ("openai/_client.py"), and its text.
type File struct {
	Path string
	Text string
}

// Class is what one class's body defines, for following a method that left
// a class: its bases as written (the last part of a dotted name, without
// [...]: "SyncAPIResource", "Generic"), and every name its body defines (defs
// of any name, nested classes, attributes assigned or annotated).
type Class struct {
	Bases   []string `json:"bases,omitempty"`
	Members []string `json:"members,omitempty"`
}

// Classes are the package's module-level classes by name (private modules
// and names included): where a method that left a class may still be, in
// the class itself (now an attribute) or in a base. Classes of the same
// name in two modules are merged (their members together), which can only
// make a moved method more likely to be found, never less.
type Classes map[string]Class

// module is one file read: its own definitions and how it re-exports others.
type module struct {
	name    string            // "openai.types.chat"
	file    string            // its path
	defs    map[string]Symbol // by name in the module ("OpenAI", "OpenAI.with_options")
	order   []string          // defs' names, in file order
	all     []string          // __all__, when it's a plain list of strings
	hasAll  bool
	imports map[string][2]string // local name -> {module, name} from "from M import name as local"
	stars   []string             // modules of "from M import *"
	classes map[string]*Class    // module-level classes by name
	bound   map[string]bool      // every name bound at the top level
	extStar bool                 // a "from M import *" from outside the package
}

// Binding is what one public module binds at its top level: every name
// (classes and functions, assignments and annotations, imports from
// anywhere), and Star when names can't all be listed: it star-imports a
// module outside the package (or not in the release's files), or defines a
// module-level __getattr__.
type Binding struct {
	Names []string `json:"names,omitempty"`
	Star  bool     `json:"star,omitempty"`
}

// Bindings are the public modules' Bindings by module name ("acme.client"):
// what tells a class or function that is gone from a module's surface from
// a name the module still binds (an alias, a re-export, an import), which
// code can still import.
type Bindings map[string]Binding

var (
	defLine   = regexp.MustCompile(`^(?:async\s+)?def\s+([A-Za-z_]\w*)\s*\(`)
	classLine = regexp.MustCompile(`^class\s+([A-Za-z_]\w*)\s*[(:]`)
	allLine   = regexp.MustCompile(`^__all__\s*(?::[^=]*)?=\s*[\[(](.*)[\])]\s*$`)
	fromLine  = regexp.MustCompile(`^from\s+(\.*[\w.]*)\s+import\s+(.+)$`)
	plainImp  = regexp.MustCompile(`^import\s+(.+)$`)
	strLit    = regexp.MustCompile(`^[rRuU]?(?:"([^"\\]*)"|'([^'\\]*)')$`)
	name      = regexp.MustCompile(`^[A-Za-z_]\w*$`)
	spaces    = regexp.MustCompile(`\s+`)
	attrLine  = regexp.MustCompile(`^([A-Za-z_]\w*)\s*(?::|=[^=])`)
	keywords  = map[string]bool{"else": true, "try": true, "finally": true, "except": true, "elif": true, "if": true, "while": true, "for": true, "with": true, "match": true, "case": true}
)

// Read is the public surface of the package whose files these are. A module
// is public when no part of its path starts with "_" (__init__ aside); its
// public names are its own definitions and, in a package's __init__, the
// names it imports from the package (re-exports, "from ._client import
// OpenAI"), less any starting with "_"; only those in __all__ when it has
// one (re-exports included, in any module). A .pyi stub next to a
// .py file is read instead of it.
func Read(files []File) Surface {
	mods := parseAll(files)
	out := Surface{}
	for _, m := range mods {
		if private(m.name) {
			continue
		}
		for _, n := range m.public(mods) {
			for k, s := range resolve(mods, m, n, 0) {
				out[m.name+"."+k] = s
			}
		}
	}
	return out
}

// ReadClasses is the package's Classes: every module-level class.
func ReadClasses(files []File) Classes {
	all := map[string]*Class{}
	for _, m := range parseAll(files) {
		for n, c := range m.classes {
			have := all[n]
			if have == nil {
				have = &Class{}
				all[n] = have
			}
			have.Bases = appendNew(have.Bases, c.Bases...)
			have.Members = appendNew(have.Members, c.Members...)
		}
	}
	out := Classes{}
	for n, c := range all {
		slices.Sort(c.Bases)
		slices.Sort(c.Members)
		out[n] = *c
	}
	return out
}

// ReadBindings is the package's Bindings: every public module's top-level
// names. A star import from another module of the package adds that
// module's public names.
func ReadBindings(files []File) Bindings {
	mods := parseAll(files)
	out := Bindings{}
	for _, m := range mods {
		if private(m.name) {
			continue
		}
		// A module-level __getattr__ (PEP 562) can hand out any name.
		b := Binding{Star: m.extStar || m.bound["__getattr__"]}
		names := map[string]bool{}
		for n := range m.bound {
			if !strings.HasPrefix(n, "_") {
				names[n] = true
			}
		}
		for _, st := range m.stars {
			t, ok := mods[st]
			if !ok {
				b.Star = true
				continue
			}
			if t != m {
				for _, n := range t.public(mods) {
					names[n] = true
				}
			}
		}
		for n := range names {
			b.Names = append(b.Names, n)
		}
		slices.Sort(b.Names)
		if len(b.Names) > 0 || b.Star {
			out[m.name] = b
		}
	}
	return out
}

// appendNew appends the names list doesn't have yet.
func appendNew(list []string, names ...string) []string {
	for _, n := range names {
		if !slices.Contains(list, n) {
			list = append(list, n)
		}
	}
	return list
}

// Bases are a class's bases as written in its signature ("class X(a.B[T],
// C, metaclass=M):" is B, C): the last part of each dotted name, without
// [...]; keyword arguments left out.
func Bases(signature string) []string {
	sig, _, _ := strings.Cut(signature, " / ") // a class's __init__ follows
	if !strings.HasPrefix(sig, "class ") {
		return nil
	}
	open := strings.IndexByte(sig, '(')
	if open < 0 {
		return nil
	}
	end := closing(sig, open)
	if end < 0 {
		return nil
	}
	var out []string
	for _, raw := range splitTop(sig[open+1:end], ',') {
		b := strings.TrimSpace(raw)
		if b == "" || indexTop(b, '=') >= 0 {
			continue
		}
		if i := strings.IndexByte(b, '['); i >= 0 {
			b = b[:i]
		}
		b = strings.TrimSpace(b[strings.LastIndexByte(b, '.')+1:])
		if b != "" {
			out = append(out, b)
		}
	}
	return out
}

// parseAll parses every .py and .pyi file, by module name; a .pyi stub next
// to a .py file is read instead of it.
func parseAll(files []File) map[string]*module {
	byPath := map[string]File{}
	for _, f := range files {
		byPath[f.Path] = f
	}
	mods := map[string]*module{}
	for _, f := range files {
		if !strings.HasSuffix(f.Path, ".py") && !strings.HasSuffix(f.Path, ".pyi") {
			continue
		}
		if strings.HasSuffix(f.Path, ".py") {
			if _, stub := byPath[f.Path+"i"]; stub {
				continue
			}
		}
		m := parse(f)
		mods[m.name] = m
	}
	return mods
}

// moduleName is "openai.types.chat" for "openai/types/chat/__init__.py".
func moduleName(p string) string {
	p = strings.TrimSuffix(strings.TrimSuffix(p, ".pyi"), ".py")
	p = strings.TrimSuffix(p, "/__init__")
	return strings.ReplaceAll(p, "/", ".")
}

// private is a module with a part starting with "_" or named internal or
// vendor (vendored code), or a test module.
func private(mod string) bool {
	for _, part := range strings.Split(mod, ".") {
		if strings.HasPrefix(part, "_") || part == "internal" || part == "vendor" || part == "tests" || part == "test" || strings.HasPrefix(part, "test_") {
			return true
		}
	}
	return false
}

// public are the module's public top-level names, in a stable order.
func (m *module) public(mods map[string]*module) []string {
	if m.hasAll {
		return m.all
	}
	var out []string
	seen := map[string]bool{}
	add := func(n string) {
		if !strings.HasPrefix(n, "_") && !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	for _, n := range m.order {
		if !strings.Contains(n, ".") {
			add(n)
		}
	}
	// Imported names are the package's API only where a package says so:
	// its __init__ (re-exports from its own modules), __all__, or a star
	// import, which exists to re-export ("from . import *"). Elsewhere an
	// import is the module's own business.
	if m.isInit() {
		locals := make([]string, 0, len(m.imports))
		for n, imp := range m.imports {
			if imp[0] == m.name || strings.HasPrefix(imp[0], m.name+".") {
				locals = append(locals, n) // from its own modules: a re-export
			}
		}
		slices.Sort(locals)
		for _, n := range locals {
			add(n)
		}
	}
	for _, s := range m.stars {
		if t, ok := mods[s]; ok && t != m {
			for _, n := range t.public(mods) {
				add(n)
			}
		}
	}
	return out
}

// isInit is a package's __init__.py or __init__.pyi.
func (m *module) isInit() bool {
	return path.Base(strings.TrimSuffix(strings.TrimSuffix(m.file, "i"), ".py")) == "__init__"
}

// resolve is name in m and its members ("OpenAI", "OpenAI.with_options"),
// following re-exports a few hops; nothing for a name that isn't a class or
// function the package defines (a constant, a type alias, a module).
func resolve(mods map[string]*module, m *module, n string, hops int) map[string]Symbol {
	if hops > 4 {
		return nil
	}
	if s, ok := m.defs[n]; ok {
		out := map[string]Symbol{n: s}
		for k, v := range m.defs {
			if strings.HasPrefix(k, n+".") {
				out[k] = v
			}
		}
		return out
	}
	if imp, ok := m.imports[n]; ok {
		if t, ok := mods[imp[0]]; ok {
			return rekey(resolve(mods, t, imp[1], hops+1), imp[1], n)
		}
		return nil
	}
	for _, s := range m.stars {
		if t, ok := mods[s]; ok && t != m {
			if got := resolve(mods, t, n, hops+1); got != nil {
				return got
			}
		}
	}
	return nil
}

// rekey renames the symbols of name from to name to: a re-export under
// another name ("AsyncClient as Async").
func rekey(syms map[string]Symbol, from, to string) map[string]Symbol {
	if from == to || syms == nil {
		return syms
	}
	out := make(map[string]Symbol, len(syms))
	for k, v := range syms {
		out[to+strings.TrimPrefix(k, from)] = v
	}
	return out
}

// block is an open def or class while its body is read.
type block struct {
	indent int
	kind   string // "class", "def", "other"
	name   string // the def's or class's path in the module: "OpenAI.with_options"
}

// parse reads one file's definitions, __all__ and imports.
func parse(f File) *module {
	m := &module{name: moduleName(f.Path), file: f.Path, defs: map[string]Symbol{}, imports: map[string][2]string{}, classes: map[string]*Class{}, bound: map[string]bool{}}
	var stack []block
	var decorators []string
	for _, l := range logicalLines(f.Text) {
		for len(stack) > 0 && l.indent <= stack[len(stack)-1].indent {
			stack = stack[:len(stack)-1]
		}
		// Where this line is: at module level (only if/try/with blocks
		// around it), in a class body at module level, or anywhere else.
		parent, inDef := "", false
		for _, b := range stack {
			switch b.kind {
			case "def":
				inDef = true
			case "class":
				parent = b.name
			}
		}
		nested := strings.Count(parent, ".") > 0 // a class inside a class: not read
		t := l.text
		opens := strings.HasSuffix(t, ":")
		if c := m.classes[parent]; c != nil && !inDef {
			// A name the class body defines: a def, a nested class, an
			// attribute.
			var n string
			switch {
			case defLine.MatchString(t):
				n = defLine.FindStringSubmatch(t)[1]
			case classLine.MatchString(t):
				n = classLine.FindStringSubmatch(t)[1]
			case attrLine.MatchString(t) && !keywords[attrLine.FindStringSubmatch(t)[1]]:
				n = attrLine.FindStringSubmatch(t)[1]
			}
			if n != "" {
				c.Members = appendNew(c.Members, n)
			}
		}
		if parent == "" && !inDef {
			m.bind(t)
		}
		switch {
		case strings.HasPrefix(t, "@"):
			decorators = append(decorators, strings.TrimSpace(t[1:]))
			continue
		case defLine.MatchString(t):
			n := defLine.FindStringSubmatch(t)[1]
			path := n
			if parent != "" {
				path = parent + "." + n
			}
			if !inDef && !nested {
				m.addDef(path, parent, l, t, decorators)
			}
			stack = append(stack, block{indent: l.indent, kind: "def", name: path})
		case classLine.MatchString(t):
			n := classLine.FindStringSubmatch(t)[1]
			path := n
			if parent != "" {
				path = parent + "." + n
			}
			if !inDef && parent == "" {
				m.defs[path] = Symbol{Kind: "class", File: f.Path, Line: l.num, Signature: header(t)}
				m.order = append(m.order, path)
				if m.classes[path] == nil {
					m.classes[path] = &Class{}
				}
				m.classes[path].Bases = appendNew(m.classes[path].Bases, Bases(header(t))...)
			}
			kind := "class"
			if inDef {
				kind = "def"
			}
			stack = append(stack, block{indent: l.indent, kind: kind, name: path})
		case opens && !strings.Contains(t, "lambda"):
			// if/else/try/except/with/for: what's inside is still at
			// this level.
			stack = append(stack, block{indent: l.indent, kind: "other", name: parent})
		case parent == "" && !inDef && allLine.MatchString(t):
			m.readAll(allLine.FindStringSubmatch(t)[1])
		case parent == "" && !inDef && fromLine.MatchString(t):
			fm := fromLine.FindStringSubmatch(t)
			m.readImport(fm[1], fm[2])
		}
		decorators = nil
	}
	return m
}

// bind records the names a top-level line binds: a def or class, an
// assignment or annotation, an import of any module.
func (m *module) bind(t string) {
	switch {
	case defLine.MatchString(t):
		m.bound[defLine.FindStringSubmatch(t)[1]] = true
	case classLine.MatchString(t):
		m.bound[classLine.FindStringSubmatch(t)[1]] = true
	case fromLine.MatchString(t):
		fm := fromLine.FindStringSubmatch(t)
		names := strings.TrimSpace(strings.Trim(strings.TrimSpace(fm[2]), "()"))
		if names == "*" {
			if absolute(m, fm[1]) == "" {
				m.extStar = true
			}
			return
		}
		for _, item := range splitTop(names, ',') {
			f := strings.Fields(item)
			switch {
			case len(f) == 1 && name.MatchString(f[0]):
				m.bound[f[0]] = true
			case len(f) == 3 && f[1] == "as" && name.MatchString(f[2]):
				m.bound[f[2]] = true
			}
		}
	case plainImp.MatchString(t):
		for _, item := range splitTop(plainImp.FindStringSubmatch(t)[1], ',') {
			f := strings.Fields(item)
			switch {
			case len(f) == 1:
				m.bound[strings.SplitN(f[0], ".", 2)[0]] = true
			case len(f) == 3 && f[1] == "as" && name.MatchString(f[2]):
				m.bound[f[2]] = true
			}
		}
	case attrLine.MatchString(t) && !keywords[attrLine.FindStringSubmatch(t)[1]]:
		m.bound[attrLine.FindStringSubmatch(t)[1]] = true
	}
}

// addDef records a function, or a method of a module-level class; a
// class's __init__ becomes the class's own parameters. A property's setter
// or deleter isn't recorded (the getter is the symbol), and a property's
// parameters aren't read: callers never pass them.
func (m *module) addDef(path, parent string, l line, text string, decorators []string) {
	n := path[strings.LastIndex(path, ".")+1:]
	params, ok := params(text)
	overloaded := slices.ContainsFunc(decorators, func(d string) bool { return d == "overload" || d == "typing.overload" })
	if !ok {
		params = nil
	}
	if parent != "" && n == "__init__" {
		c := m.defs[parent]
		c.Signature = strings.TrimSuffix(c.Signature, ":") + " / " + header(text)
		if overloaded || c.Overloaded {
			c.Params, c.Overloaded = nil, true
		} else {
			c.Params = params
		}
		m.defs[parent] = c
		return
	}
	if strings.HasPrefix(n, "_") {
		return
	}
	if slices.ContainsFunc(decorators, func(d string) bool {
		return strings.HasSuffix(d, ".setter") || strings.HasSuffix(d, ".deleter")
	}) {
		return
	}
	if slices.ContainsFunc(decorators, func(d string) bool {
		return d == "property" || d == "cached_property" || strings.HasSuffix(d, ".cached_property")
	}) {
		params = nil
	}
	kind := "function"
	if parent != "" {
		kind = "method"
	}
	if have, dup := m.defs[path]; dup && (overloaded || have.Overloaded) {
		// @overload variants, then the implementation: one symbol, its
		// parameters not compared.
		have.Overloaded, have.Params = true, nil
		if !overloaded {
			have.Line, have.Signature = l.num, header(text)
		}
		m.defs[path] = have
		return
	}
	s := Symbol{Kind: kind, File: m.file, Line: l.num, Signature: header(text), Params: params, Overloaded: overloaded}
	if overloaded {
		s.Params = nil
	}
	if _, dup := m.defs[path]; !dup {
		m.order = append(m.order, path)
	}
	m.defs[path] = s
}

// header is a def or class line without its body, whitespace collapsed.
func header(text string) string {
	if open := strings.IndexByte(text, '('); open >= 0 {
		if end := closing(text, open); end > 0 {
			rest := text[end+1:]
			if colon := indexTop(rest, ':'); colon >= 0 {
				rest = rest[:colon]
			}
			return spaces.ReplaceAllString(strings.TrimSpace(text[:end+1]+rest), " ")
		}
	}
	if colon := strings.IndexByte(text, ':'); colon >= 0 {
		text = text[:colon]
	}
	return spaces.ReplaceAllString(strings.TrimSpace(text), " ")
}

// indexTop is the first sep outside brackets and strings, or -1.
func indexTop(s string, sep byte) int {
	parts := splitTop(s, sep)
	if len(parts) < 2 {
		return -1
	}
	return len(parts[0])
}

// params reads a def line's parameter list. ok is false when the list
// can't be read for sure.
func params(text string) ([]Param, bool) {
	open := strings.IndexByte(text, '(')
	end := closing(text, open)
	if open < 0 || end < 0 {
		return nil, false
	}
	inner := strings.TrimSpace(text[open+1 : end])
	if inner == "" {
		return []Param{}, true
	}
	var out []Param
	kind := "either"
	for _, raw := range splitTop(inner, ',') {
		p := strings.TrimSpace(raw)
		switch {
		case p == "":
			continue // a trailing comma
		case p == "/":
			for i := range out {
				out[i].Kind = "positional"
			}
			continue
		case p == "*":
			kind = "keyword"
			continue
		}
		var prm Param
		if def := indexTop(p, '='); def >= 0 {
			prm.Default, prm.HasDef = spaces.ReplaceAllString(strings.TrimSpace(p[def+1:]), " "), true
			p = p[:def]
		}
		if colon := indexTop(p, ':'); colon >= 0 {
			p = p[:colon]
		}
		p = strings.TrimSpace(p)
		switch {
		case strings.HasPrefix(p, "**"):
			prm.Kind, p = "kwargs", p[2:]
		case strings.HasPrefix(p, "*"):
			prm.Kind, p = "args", p[1:]
			kind = "keyword"
		default:
			prm.Kind = kind
		}
		if !name.MatchString(p) {
			return nil, false
		}
		prm.Name = p
		out = append(out, prm)
	}
	return out, true
}

// readAll reads `__all__ = ["a", "b"]`; anything but plain strings leaves
// it unread (every public name counts then).
func (m *module) readAll(list string) {
	var names []string
	for _, item := range splitTop(list, ',') {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		s := strLit.FindStringSubmatch(item)
		if s == nil {
			return
		}
		names = append(names, s[1]+s[2])
	}
	m.all, m.hasAll = names, true
}

// readImport reads `from M import a, b as c` (or `*`), keeping imports from
// the package itself: relative ones, and absolute ones under its top module.
func (m *module) readImport(from, names string) {
	target := absolute(m, from)
	if target == "" {
		return
	}
	names = strings.TrimSpace(strings.Trim(strings.TrimSpace(names), "()"))
	if names == "*" {
		m.stars = append(m.stars, target)
		return
	}
	for _, item := range splitTop(names, ',') {
		f := strings.Fields(item)
		switch {
		case len(f) == 1 && name.MatchString(f[0]):
			m.imports[f[0]] = [2]string{target, f[0]}
		case len(f) == 3 && f[1] == "as" && name.MatchString(f[0]) && name.MatchString(f[2]):
			m.imports[f[2]] = [2]string{target, f[0]}
		}
	}
}

// absolute is the module a from-import names, or "" for one outside the
// package.
func absolute(m *module, from string) string {
	top := strings.SplitN(m.name, ".", 2)[0]
	dots := len(from) - len(strings.TrimLeft(from, "."))
	if dots == 0 {
		if from == top || strings.HasPrefix(from, top+".") {
			return from
		}
		return ""
	}
	// The package a relative import starts from: the module's own for
	// __init__, its parent otherwise; each further dot goes up one.
	base := strings.Split(m.name, ".")
	if !m.isInit() {
		base = base[:len(base)-1]
	}
	if dots-1 > len(base) {
		return ""
	}
	base = base[:len(base)-(dots-1)]
	rest := from[dots:]
	if rest != "" {
		base = append(base, strings.Split(rest, ".")...)
	}
	return strings.Join(base, ".")
}
