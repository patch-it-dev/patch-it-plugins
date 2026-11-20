// Package pysurface reads the public API of a Python package from its source
// files, statically: the modules, classes, functions and methods code can
// import, with each one's parameters and their defaults. It never imports,
// compiles or runs the package's code: it reads the text, the way a person
// reading the files would.
//
// It is not a Python parser. It reads only what a signature needs: logical
// lines (comments dropped, strings kept whole, brackets and backslashes
// joining physical lines), their indentation, and the headers of `def`,
// `async def` and `class` statements, `__all__ = [...]` and
// `from ... import ...`. Function bodies are skipped. Whatever it can't read
// for sure (a dynamic `__all__`, a star import it can't resolve, an
// overloaded function's parameters) it leaves out: precision over recall.
package pysurface

import (
	"strings"
)

// line is one logical line of Python: its text with comments dropped and
// physical lines joined, its indentation, and the physical line it starts on.
type line struct {
	text   string
	indent int
	num    int // 1-based
}

// logicalLines splits a Python source file into logical lines. Strings
// (single, double, triple quoted, with any prefix) are kept verbatim, so a
// default value reads as written; a `#` inside one is not a comment.
func logicalLines(src string) []line {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	var out []line
	var cur strings.Builder
	depth := 0
	startNum, num := 0, 1
	indent := -1
	atLineStart := true
	col := 0
	flush := func() {
		t := strings.TrimSpace(cur.String())
		if t != "" {
			out = append(out, line{text: t, indent: indent, num: startNum})
		}
		cur.Reset()
		indent = -1
		depth = 0
	}
	for i := 0; i < len(src); i++ {
		c := src[i]
		if atLineStart {
			// Measure the indentation of a new logical line.
			if c == ' ' || c == '\t' {
				if c == '\t' {
					col += 8 - col%8
				} else {
					col++
				}
				continue
			}
			atLineStart = false
			if indent < 0 && c != '\n' && c != '#' {
				indent, startNum = col, num
			}
		}
		switch {
		case c == '#':
			for i < len(src) && src[i] != '\n' {
				i++
			}
			i-- // let the newline be handled
		case c == '\'' || c == '"':
			end := stringEnd(src, i)
			lit := src[i:end]
			num += strings.Count(lit, "\n")
			cur.WriteString(lit)
			i = end - 1
		case c == '\\' && i+1 < len(src) && src[i+1] == '\n':
			cur.WriteByte(' ')
			i++
			num++
		case c == '(' || c == '[' || c == '{':
			depth++
			cur.WriteByte(c)
		case c == ')' || c == ']' || c == '}':
			if depth > 0 {
				depth--
			}
			cur.WriteByte(c)
		case c == '\n':
			num++
			if depth > 0 {
				cur.WriteByte(' ')
				continue
			}
			flush()
			atLineStart, col = true, 0
		default:
			cur.WriteByte(c)
		}
	}
	flush()
	return out
}

// stringEnd is the index just past the string literal that starts at i (a
// quote): triple or single quoted, with backslash escapes. An unterminated
// one ends at the end of its line (single) or the file (triple).
func stringEnd(src string, i int) int {
	q := src[i]
	triple := strings.HasPrefix(src[i:], strings.Repeat(string(q), 3))
	j := i + 1
	if triple {
		j = i + 3
	}
	for j < len(src) {
		switch {
		case src[j] == '\\':
			j += 2
			continue
		case triple && strings.HasPrefix(src[j:], strings.Repeat(string(q), 3)):
			return j + 3
		case !triple && src[j] == q:
			return j + 1
		case !triple && src[j] == '\n':
			return j
		}
		j++
	}
	return len(src)
}

// splitTop splits s on sep where it isn't inside brackets or a string.
func splitTop(s string, sep byte) []string {
	var out []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '\'' || c == '"':
			i = stringEnd(s, i) - 1
		case c == '(' || c == '[' || c == '{':
			depth++
		case c == ')' || c == ']' || c == '}':
			depth--
		case c == sep && depth == 0:
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

// closing is the index of the bracket that closes the one at open, or -1.
func closing(s string, open int) int {
	depth := 0
	for i := open; i < len(s); i++ {
		switch c := s[i]; {
		case c == '\'' || c == '"':
			i = stringEnd(s, i) - 1
		case c == '(' || c == '[' || c == '{':
			depth++
		case c == ')' || c == ']' || c == '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
