package dtsurface

import (
	"strings"
)

// Param is one parameter of a function, method or constructor, in order. A
// TypeScript call passes arguments by position, so a parameter's name is
// for reading only: Optional is what a caller relies on.
type Param struct {
	Name     string
	Optional bool // marked ?, or the rest parameter
	Rest     bool // ...args
}

// decl is one declaration of a name in a scope. Declarations of the same
// name merge (a class and a namespace, a function's overloads).
type decl struct {
	kind     string // "class", "function", "interface", "type", "enum", "variable", "namespace"
	name     string
	file     string
	line     int
	sig      string
	exported bool
	params   []Param
	hasParam bool // params were read (a function, a class's constructor)
	ctors    int  // a class's constructor declarations
	bases    []string
	members  []*member
	ns       *scope // a namespace's (or a module declaration's) body
}

// member is one public member of a class: a method or a property.
type member struct {
	kind   string // "method", "property"
	name   string
	line   int
	sig    string
	params []Param
	static bool
}

// scope is a module (a declaration file), a namespace's body or an ambient
// module declaration's body.
type scope struct {
	file    string
	parent  *scope
	decls   map[string][]*decl
	order   []string
	exports map[string]exportRef
	stars   []starRef
	imports map[string]importRef
	aliases map[string]string // import X = A.B.C
	// assign is `export = X`: the module is X.
	assign string
	// explicit is true once the scope says export anywhere: in an ambient
	// namespace, only then are its unexported declarations hidden.
	explicit bool
	ambient  bool // a namespace or module declaration's body
}

type exportRef struct {
	local string // `export { local as name }`, or a declaration's own name
	spec  string // `export { name } from "spec"`: the module it's from
	name  string // its name there; "*" for `export * as name from`
	line  int
	sig   string
}

type starRef struct {
	spec string
	line int
}

type importRef struct {
	spec string
	name string // "*" for `import * as X`, "default", "=" for `import X = require()`
}

func newScope(file string, parent *scope, ambient bool) *scope {
	return &scope{file: file, parent: parent, decls: map[string][]*decl{}, exports: map[string]exportRef{}, imports: map[string]importRef{}, aliases: map[string]string{}, ambient: ambient}
}

func (s *scope) add(d *decl) {
	if _, ok := s.decls[d.name]; !ok {
		s.order = append(s.order, d.name)
	}
	s.decls[d.name] = append(s.decls[d.name], d)
}

// parser reads one file's tokens.
type parser struct {
	toks    []token
	pos     int
	file    string
	modules map[string]*scope // `declare module "spec" { }` bodies, by spec
}

func (p *parser) peek() token    { return p.toks[p.pos] }
func (p *parser) at(n int) token { return p.toks[min(p.pos+n, len(p.toks)-1)] }
func (p *parser) next() token    { t := p.toks[p.pos]; p.pos = min(p.pos+1, len(p.toks)-1); return t }
func (p *parser) is(s string) bool {
	t := p.peek()
	return (t.kind == "punct" || t.kind == "id") && t.text == s
}
func (p *parser) eat(s string) bool {
	if p.is(s) {
		p.next()
		return true
	}
	return false
}

// parseFile reads a declaration file into its module scope; ambient module
// declarations ("declare module 'x'") go to modules, merged by spec.
func parseFile(file, src string, modules map[string]*scope) *scope {
	p := &parser{toks: lex(src), file: file, modules: modules}
	sc := newScope(file, nil, false)
	p.block(sc, false)
	return sc
}

// block reads statements until "}" (when inBraces) or the end.
func (p *parser) block(sc *scope, inBraces bool) {
	for {
		t := p.peek()
		if t.kind == "eof" {
			return
		}
		if inBraces && t.kind == "punct" && t.text == "}" {
			p.next()
			return
		}
		start := p.pos
		p.statement(sc)
		if p.pos == start {
			p.next() // never stall
		}
	}
}

var modifiers = map[string]bool{"declare": true, "abstract": true, "async": true}

// statement reads one statement of a scope.
func (p *parser) statement(sc *scope) {
	t := p.peek()
	if t.kind == "punct" && t.text == ";" {
		p.next()
		return
	}
	internal := strings.Contains(t.doc, "@internal")
	line := t.line
	exported, isDefault := false, false
	if p.is("export") {
		sc.explicit = true
		p.next()
		exported = true
		switch {
		case p.is("{") || (p.is("type") && p.at(1).text == "{"):
			p.eat("type")
			p.exportList(sc, line)
			return
		case p.is("*"):
			p.next()
			if p.eat("as") {
				name := p.next().text
				p.eat("from")
				spec := p.next().text
				sc.exports[name] = exportRef{spec: spec, name: "*", line: line, sig: "export * as " + name + " from " + quote(spec)}
			} else {
				p.eat("from")
				sc.stars = append(sc.stars, starRef{spec: p.next().text, line: line})
			}
			p.eat(";")
			return
		case p.is("="):
			p.next()
			sc.assign = p.dotted()
			p.eat(";")
			return
		case p.is("as") && p.at(1).text == "namespace":
			p.skipStatement()
			return
		case p.is("import"):
			p.next()
			name := p.next().text
			if p.eat("=") {
				sc.aliases[name] = p.dotted()
				sc.exports[name] = exportRef{local: name, line: line, sig: "export import " + name + " = " + sc.aliases[name]}
			}
			p.eat(";")
			return
		case p.is("default"):
			p.next()
			isDefault = true
			if !p.isDeclStart() {
				local := p.dotted()
				if local != "" {
					sc.exports["default"] = exportRef{local: local, line: line, sig: "export default " + local}
				}
				p.skipStatement()
				return
			}
		}
	}
	if p.is("import") {
		p.importStatement(sc)
		return
	}
	for p.peek().kind == "id" && modifiers[p.peek().text] && p.at(1).kind == "id" {
		p.next()
	}
	kw := p.peek()
	if kw.kind != "id" {
		p.skipStatement()
		return
	}
	var ds []*decl
	switch kw.text {
	case "class":
		ds = []*decl{p.class()}
	case "function":
		ds = []*decl{p.function()}
	case "interface":
		ds = []*decl{p.interfaceDecl()}
	case "type":
		ds = []*decl{p.typeAlias()}
	case "enum":
		ds = []*decl{p.enum()}
	case "const":
		if p.at(1).text == "enum" {
			p.next()
			ds = []*decl{p.enum()}
		} else {
			ds = p.variables()
		}
	case "let", "var":
		ds = p.variables()
	case "namespace", "module":
		if p.at(1).kind == "str" {
			p.next()
			spec := p.next().text
			if p.is("{") {
				p.next()
				m := p.modules[spec]
				if m == nil {
					m = newScope(p.file, nil, true)
					p.modules[spec] = m
				}
				m.file = p.file
				p.block(m, true)
			} else {
				p.eat(";")
			}
			return
		}
		ds = []*decl{p.namespace(sc)}
	case "global":
		p.next()
		p.skipBalanced()
		return
	default:
		p.skipStatement()
		return
	}
	for _, d := range ds {
		if d == nil || d.name == "" || internal {
			continue
		}
		d.line = line
		// `export default class X` exports X as default, not by its name.
		d.exported = exported && !isDefault
		if isDefault {
			sc.exports["default"] = exportRef{local: d.name, line: line, sig: d.sig}
		}
		sc.add(d)
	}
}

// isDeclStart is a declaration after `export default`.
func (p *parser) isDeclStart() bool {
	i := 0
	for p.at(i).kind == "id" && modifiers[p.at(i).text] {
		i++
	}
	switch p.at(i).text {
	case "class", "function", "interface", "enum":
		return p.at(i).kind == "id"
	}
	return false
}

// exportList reads `{ a, b as c, type d } [from "spec"];`.
func (p *parser) exportList(sc *scope, line int) {
	p.next() // {
	type item struct{ name, as string }
	var items []item
	for !p.is("}") && p.peek().kind != "eof" {
		p.eat("type")
		n := p.next().text
		as := n
		if p.eat("as") {
			as = p.next().text
		}
		items = append(items, item{n, as})
		p.eat(",")
	}
	p.eat("}")
	spec := ""
	if p.eat("from") {
		spec = p.next().text
	}
	p.eat(";")
	for _, it := range items {
		if it.as == "" {
			continue
		}
		sig := "export { " + it.name
		if it.as != it.name {
			sig += " as " + it.as
		}
		sig += " }"
		if spec != "" {
			sc.exports[it.as] = exportRef{spec: spec, name: it.name, line: line, sig: sig + " from " + quote(spec)}
		} else {
			sc.exports[it.as] = exportRef{local: it.name, line: line, sig: sig}
		}
	}
}

// importStatement reads every form of import into sc's imports.
func (p *parser) importStatement(sc *scope) {
	p.next() // import
	if p.peek().kind == "str" {
		p.skipStatement()
		return
	}
	p.eat("type")
	if p.peek().kind == "id" && p.at(1).text == "=" {
		name := p.next().text
		p.next() // =
		if p.eat("require") {
			p.eat("(")
			spec := p.next().text
			p.eat(")")
			sc.imports[name] = importRef{spec: spec, name: "="}
		} else {
			sc.aliases[name] = p.dotted()
		}
		p.eat(";")
		return
	}
	var pending []func(spec string)
	for !p.is("from") && p.peek().kind != "eof" && !p.is(";") {
		switch {
		case p.is("*"):
			p.next()
			p.eat("as")
			n := p.next().text
			pending = append(pending, func(spec string) { sc.imports[n] = importRef{spec: spec, name: "*"} })
		case p.is("{"):
			p.next()
			for !p.is("}") && p.peek().kind != "eof" {
				p.eat("type")
				n := p.next().text
				as := n
				if p.eat("as") {
					as = p.next().text
				}
				pending = append(pending, func(spec string) { sc.imports[as] = importRef{spec: spec, name: n} })
				p.eat(",")
			}
			p.eat("}")
		case p.peek().kind == "id":
			n := p.next().text
			pending = append(pending, func(spec string) { sc.imports[n] = importRef{spec: spec, name: "default"} })
		default:
			p.next()
		}
		p.eat(",")
	}
	if p.eat("from") {
		spec := p.next().text
		for _, f := range pending {
			f(spec)
		}
	}
	p.skipStatement()
}

// dotted reads a name like A.B.C.
func (p *parser) dotted() string {
	var parts []string
	for p.peek().kind == "id" {
		parts = append(parts, p.next().text)
		if !p.eat(".") {
			break
		}
	}
	return strings.Join(parts, ".")
}

// class reads a class declaration and its public members.
func (p *parser) class() *decl {
	start := p.pos
	p.next() // class
	d := &decl{kind: "class", file: p.file}
	if p.peek().kind == "id" && p.peek().text != "extends" && p.peek().text != "implements" {
		d.name = p.next().text
	}
	if p.is("<") {
		p.skipBalanced()
	}
	if p.eat("extends") {
		base := ""
		for p.peek().kind == "id" {
			base = p.next().text
			if !p.eat(".") {
				break
			}
		}
		if p.is("(") {
			base = "" // a mixin call: its members aren't known
		}
		for !p.is("{") && !p.is("implements") && p.peek().kind != "eof" {
			p.skipOne()
		}
		if base != "" {
			d.bases = append(d.bases, base)
		}
	}
	for !p.is("{") && p.peek().kind != "eof" {
		p.skipOne()
	}
	d.sig = p.text(start, p.pos)
	p.next() // {
	p.classBody(d)
	return d
}

var memberModifiers = map[string]bool{"public": true, "static": true, "readonly": true, "abstract": true, "declare": true, "override": true, "accessor": true, "async": true}

// classBody reads members until "}".
func (p *parser) classBody(d *decl) {
	for !p.is("}") && p.peek().kind != "eof" {
		t := p.peek()
		if t.kind == "punct" && t.text == ";" {
			p.next()
			continue
		}
		start := p.pos
		internal := strings.Contains(t.doc, "@internal")
		hidden, static := false, false
		for p.peek().kind == "id" && (memberModifiers[p.peek().text] || p.peek().text == "private" || p.peek().text == "protected") {
			n := p.at(1)
			if n.kind == "punct" && (n.text == "(" || n.text == ":" || n.text == "?" || n.text == ";" || n.text == "<" || n.text == "?:") {
				break // a member named like a modifier
			}
			switch p.peek().text {
			case "private", "protected":
				hidden = true
			case "static":
				static = true
			}
			p.next()
		}
		accessor := ""
		if (p.is("get") || p.is("set")) && (p.at(1).kind == "id" || p.at(1).kind == "str") {
			accessor = p.next().text
		}
		name := p.peek()
		switch {
		case name.kind == "punct" && name.text == "[":
			p.skipMember()
			continue
		case name.kind == "punct" && name.text == "#":
			p.skipMember()
			continue
		case name.kind != "id" && name.kind != "str" && name.kind != "num":
			p.skipMember()
			continue
		}
		p.next()
		p.eat("?")
		p.eat("!")
		m := &member{name: name.text, line: t.line, static: static}
		switch {
		case p.is("(") || p.is("<"):
			if p.is("<") {
				p.skipBalanced()
			}
			params := p.params()
			p.skipMember()
			m.sig = p.text(start, p.pos)
			if name.text == "constructor" && !static {
				if !hidden && !internal {
					d.ctors++
					d.params, d.hasParam = params, true
					d.sig = strings.TrimSuffix(d.sig, " {") + " / " + m.sig
				}
				continue
			}
			m.kind, m.params = "method", params
			if accessor != "" {
				m.kind, m.params = "property", nil
			}
		default:
			p.skipMember()
			m.kind = "property"
			m.sig = p.text(start, p.pos)
		}
		if hidden || internal || accessor == "set" && hasMember(d, m.name) {
			continue
		}
		d.members = append(d.members, m)
	}
	p.eat("}")
}

func hasMember(d *decl, name string) bool {
	for _, m := range d.members {
		if m.name == name {
			return true
		}
	}
	return false
}

// skipMember skips to the end of a class member: its ";" (or the "}" that
// closes the class, left unread), types in brackets skipped whole.
func (p *parser) skipMember() {
	for {
		t := p.peek()
		switch {
		case t.kind == "eof":
			return
		case t.kind == "punct" && t.text == ";":
			p.next()
			return
		case t.kind == "punct" && t.text == "}":
			return
		case t.kind == "punct" && isOpen(t.text):
			p.skipBalanced()
		default:
			line := t.line
			p.next()
			// A member ends at its line's end when the file leaves out ";".
			if n := p.peek(); n.line > line && n.kind == "id" && !continues(t) && memberStart(p) {
				return
			}
		}
	}
}

// memberStart is a token that starts a new member: a name followed by
// "(", ":", "?" or ";", or a modifier.
func memberStart(p *parser) bool {
	t, n := p.peek(), p.at(1)
	if memberModifiers[t.text] || t.text == "private" || t.text == "protected" || t.text == "constructor" {
		return true
	}
	return n.kind == "punct" && (n.text == "(" || n.text == ":" || n.text == "?" || n.text == "?:" || n.text == ";" || n.text == "<")
}

// continues is a token after which a type goes on: an operator, a comma, a
// colon.
func continues(t token) bool {
	if t.kind == "punct" {
		switch t.text {
		case "|", "&", ":", ",", "=", "=>", ".", "?", "?:", "<", "(", "[", "{":
			return true
		}
	}
	switch t.text {
	case "keyof", "typeof", "extends", "infer", "is", "new", "readonly", "unique", "asserts":
		return true
	}
	return false
}

// params reads "( ... )": each parameter's name, whether it's optional or
// the rest; a `this` parameter (not an argument) is left out.
func (p *parser) params() []Param {
	out := []Param{}
	if !p.eat("(") {
		return out
	}
	for !p.is(")") && p.peek().kind != "eof" {
		for p.peek().kind == "id" && (memberModifiers[p.peek().text] || p.peek().text == "private" || p.peek().text == "protected") && p.at(1).kind == "id" {
			p.next()
		}
		var prm Param
		if p.eat("...") {
			prm.Rest, prm.Optional = true, true
		}
		switch t := p.peek(); {
		case t.kind == "punct" && (t.text == "{" || t.text == "["):
			p.skipBalanced()
			prm.Name = "(destructured)"
		default:
			prm.Name = p.next().text
		}
		if p.eat("?") || p.is("?:") {
			prm.Optional = true
			if p.is("?:") {
				p.next()
				p.skipType(",", ")")
			}
		}
		if p.eat(":") {
			p.skipType(",", ")")
		}
		if p.eat("=") {
			prm.Optional = true
			p.skipType(",", ")")
		}
		if prm.Name != "this" {
			out = append(out, prm)
		}
		p.eat(",")
	}
	p.eat(")")
	return out
}

// skipType skips a type up to one of stops at depth 0, left unread.
func (p *parser) skipType(stops ...string) {
	for {
		t := p.peek()
		if t.kind == "eof" {
			return
		}
		if t.kind == "punct" {
			for _, s := range stops {
				if t.text == s {
					return
				}
			}
			if isOpen(t.text) {
				p.skipBalanced()
				continue
			}
			if isClose(t.text) {
				return
			}
		}
		p.next()
	}
}

// function reads `function name<...>(params): T;`.
func (p *parser) function() *decl {
	start := p.pos
	p.next() // function
	p.eat("*")
	d := &decl{kind: "function", file: p.file}
	if p.peek().kind == "id" {
		d.name = p.next().text
	}
	if p.is("<") {
		p.skipBalanced()
	}
	d.params, d.hasParam = p.params(), true
	p.skipStatement()
	d.sig = p.text(start, p.pos)
	return d
}

func (p *parser) interfaceDecl() *decl {
	start := p.pos
	p.next()
	d := &decl{kind: "interface", file: p.file, name: p.next().text}
	for !p.is("{") && p.peek().kind != "eof" {
		p.skipOne()
	}
	d.sig = p.text(start, p.pos)
	p.skipBalanced()
	return d
}

func (p *parser) typeAlias() *decl {
	start := p.pos
	p.next()
	d := &decl{kind: "type", file: p.file, name: p.next().text}
	if p.is("<") {
		p.skipBalanced()
	}
	d.sig = p.text(start, p.pos)
	p.skipStatement()
	return d
}

func (p *parser) enum() *decl {
	start := p.pos
	p.next()
	d := &decl{kind: "enum", file: p.file, name: p.next().text}
	d.sig = p.text(start, p.pos)
	if p.is("{") {
		p.skipBalanced()
	}
	return d
}

// variables reads `const a: A, b: B;`.
func (p *parser) variables() []*decl {
	start := p.pos
	p.next()
	var out []*decl
	for p.peek().kind == "id" {
		d := &decl{kind: "variable", file: p.file, name: p.next().text}
		out = append(out, d)
		p.skipType(",", ";", "}")
		if !p.eat(",") {
			break
		}
	}
	p.eat(";")
	sig := p.text(start, p.pos)
	for _, d := range out {
		d.sig = sig
	}
	return out
}

// namespace reads `namespace A.B { ... }` as A, with B a namespace inside.
func (p *parser) namespace(parent *scope) *decl {
	start := p.pos
	p.next()
	names := strings.Split(p.dotted(), ".")
	d := &decl{kind: "namespace", file: p.file, name: names[0]}
	d.sig = p.text(start, p.pos)
	if !p.is("{") {
		p.skipStatement()
		return d
	}
	p.next()
	outer := newScope(p.file, parent, true)
	d.ns = outer
	in := outer
	for _, n := range names[1:] {
		inner := &decl{kind: "namespace", file: p.file, name: n, sig: d.sig, exported: true}
		inner.ns = newScope(p.file, in, true)
		in.add(inner)
		in = inner.ns
	}
	p.block(in, true)
	return d
}

func isOpen(s string) bool  { return s == "(" || s == "[" || s == "{" || s == "<" }
func isClose(s string) bool { return s == ")" || s == "]" || s == "}" || s == ">" }

// skipBalanced skips from an opening bracket to its match.
func (p *parser) skipBalanced() {
	depth := 0
	for {
		t := p.next()
		if t.kind == "eof" {
			return
		}
		if t.kind != "punct" {
			continue
		}
		if isOpen(t.text) {
			depth++
		} else if isClose(t.text) {
			depth--
			if depth <= 0 {
				return
			}
		}
	}
}

// skipOne skips a token, or a bracketed group whole.
func (p *parser) skipOne() {
	if t := p.peek(); t.kind == "punct" && isOpen(t.text) {
		p.skipBalanced()
		return
	}
	p.next()
}

// skipStatement skips to the end of a statement: its ";", or the line's
// end where the file leaves ";" out, or a "}" closing the scope (unread).
func (p *parser) skipStatement() {
	for {
		t := p.peek()
		switch {
		case t.kind == "eof":
			return
		case t.kind == "punct" && t.text == ";":
			p.next()
			return
		case t.kind == "punct" && t.text == "}":
			return
		case t.kind == "punct" && isOpen(t.text):
			p.skipBalanced()
			if n := p.peek(); n.line > t.line && n.kind == "id" && statementStart[n.text] {
				return
			}
		default:
			p.next()
			if n := p.peek(); n.line > t.line && n.kind == "id" && statementStart[n.text] && !continues(t) {
				return
			}
		}
	}
}

var statementStart = map[string]bool{"export": true, "declare": true, "interface": true, "class": true, "function": true, "type": true, "const": true, "let": true, "var": true, "namespace": true, "module": true, "enum": true, "import": true, "abstract": true}

// text is the tokens from a to b as written, spaced simply, cut at 400
// characters: a signature to quote.
func (p *parser) text(a, b int) string {
	var sb strings.Builder
	prev := ""
	for i := a; i < b && i < len(p.toks); i++ {
		t := p.toks[i]
		s := t.text
		if t.kind == "str" && !strings.HasPrefix(s, "`") {
			s = quote(s)
		}
		if sb.Len() > 0 && needsSpace(prev, s) {
			sb.WriteByte(' ')
		}
		sb.WriteString(s)
		prev = s
		if sb.Len() > 400 {
			return sb.String()[:400] + "…"
		}
	}
	return strings.TrimSuffix(sb.String(), ";")
}

func needsSpace(prev, s string) bool {
	switch s {
	case ",", ")", "]", ">", ".", ";", "?", ":", "?:", "(", "<", "!", "[":
		return (s == "(" || s == "[") && (prev == "=>" || prev == ":" || prev == "|" || prev == "," || prev == "=")
	}
	switch prev {
	case "(", "[", "<", ".", "...", "!":
		return false
	}
	return true
}

func quote(s string) string { return `"` + s + `"` }
