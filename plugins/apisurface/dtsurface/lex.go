// Package dtsurface reads an npm package's public API from its TypeScript
// declaration files (.d.ts, .d.mts, .d.cts) as text: the names each entry
// point of the package exports, and for classes, functions and namespaces
// what they hold, with every parameter and whether it's optional. Nothing in
// the package is run: declaration files hold declarations only, so a
// tokenizer and a parser of declarations, which skips over type expressions
// by their brackets, read all a caller can see.
package dtsurface

import (
	"strings"
)

// token is one lexical token of a declaration file.
type token struct {
	kind string // "id", "str" (its value unquoted), "num", "punct", "eof"
	text string
	line int
	// doc is the JSDoc comment (/** ... */) right before the token, if any:
	// where @internal is said.
	doc string
}

// lex splits src into tokens. Comments are dropped (a JSDoc comment is kept
// on the token after it); strings and template literals are one token each;
// "=>", "...", "?." and "?:" are one token; every other punctuation mark is
// its own token, so "Array<Array<T>>" closes with two ">".
func lex(src string) []token {
	var out []token
	line, doc := 1, ""
	i := 0
	for i < len(src) {
		c := src[i]
		switch {
		case c == '\n':
			line++
			i++
		case c == ' ' || c == '\t' || c == '\r' || c == '\f' || c == '\v' || c == 0xEF || c == 0xBB || c == 0xBF:
			i++ // whitespace, and a UTF-8 byte order mark
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				end = len(src) - i - 2
			}
			body := src[i : i+2+end]
			if strings.HasPrefix(body, "/**") {
				doc = body
			}
			line += strings.Count(body, "\n")
			i += 2 + end + 2
		case c == '"' || c == '\'':
			j := i + 1
			var b strings.Builder
			for j < len(src) && src[j] != c && src[j] != '\n' {
				if src[j] == '\\' && j+1 < len(src) {
					j++
				}
				b.WriteByte(src[j])
				j++
			}
			out = append(out, token{kind: "str", text: b.String(), line: line, doc: doc})
			doc = ""
			i = j + 1
		case c == '`':
			// A template literal type: skipped whole, ${...} nesting
			// included.
			j, depth := i+1, 0
			for j < len(src) {
				if src[j] == '\\' {
					j += 2
					continue
				}
				if src[j] == '\n' {
					line++
				}
				if depth == 0 && src[j] == '`' {
					break
				}
				if src[j] == '$' && j+1 < len(src) && src[j+1] == '{' {
					depth++
					j += 2
					continue
				}
				if depth > 0 && src[j] == '}' {
					depth--
				}
				j++
			}
			out = append(out, token{kind: "str", text: src[i:min(j+1, len(src))], line: line, doc: doc})
			doc = ""
			i = j + 1
		case isIdentStart(c):
			j := i + 1
			for j < len(src) && isIdentPart(src[j]) {
				j++
			}
			out = append(out, token{kind: "id", text: src[i:j], line: line, doc: doc})
			doc = ""
			i = j
		case c >= '0' && c <= '9':
			j := i + 1
			for j < len(src) && (isIdentPart(src[j]) || src[j] == '.') {
				j++
			}
			out = append(out, token{kind: "num", text: src[i:j], line: line, doc: doc})
			doc = ""
			i = j
		default:
			p := string(c)
			for _, two := range []string{"=>", "...", "?.", "?:"} {
				if strings.HasPrefix(src[i:], two) {
					p = two
					break
				}
			}
			out = append(out, token{kind: "punct", text: p, line: line, doc: doc})
			doc = ""
			i += len(p)
		}
	}
	return append(out, token{kind: "eof", line: line})
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '$' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80
}

func isIdentPart(c byte) bool { return isIdentStart(c) || c >= '0' && c <= '9' }
