package members

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

var (
	codeSpan = regexp.MustCompile("`([^`]+)`")
	ident    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$`)
	// A cell that is only a list of code spans: "§", "§, §", "§ and §",
	// "§, §, and §" (each § a code span).
	nameList  = regexp.MustCompile(`^§(?:(?:,| and|, and| or|, or) §)*$`)
	delimCell = regexp.MustCompile(`^:?-{3,}:?$`)
	// A cell that is only a link around one code span: "[`name`](url)".
	linkedSpan = regexp.MustCompile("^\\[(`[^`]+`)\\]\\([^()\\s]*\\)$")
)

// parseMarkdown reads a Markdown pipe table: the first one under the heading
// g.Section, before the next heading. Its first column is what was removed, its
// second what to use instead.
//
// A row becomes one Removal per name in its first cell, but only when that
// cell is nothing but code spans naming members ("`A`, `B`"). A cell of prose
// ("The `backoff` dependency") or a span that isn't a name code writes
// ("`gzip=True`") drops the whole row; a name in g.EnvVars drops that name. The replacement is the second cell
// only when it's exactly one code span naming a member, or a link around one;
// g.Paths applies to it as to the removed names.
func parseMarkdown(page []byte, g Guide) ([]Removal, error) {
	lines := strings.Split(strings.ReplaceAll(string(page), "\r\n", "\n"), "\n")
	inFence := false
	// heading reports a line's heading text, outside fenced code (a "# 7.x.x"
	// comment in a Python block isn't a heading).
	heading := func(l string) (string, bool) {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			inFence = !inFence
			return "", false
		}
		if inFence {
			return "", false
		}
		t := strings.TrimLeft(l, "#")
		if n := len(l) - len(t); n == 0 || n > 6 || !strings.HasPrefix(t, " ") {
			return "", false
		}
		return strings.TrimSpace(strings.TrimRight(strings.TrimSpace(t), "#")), true
	}
	start := -1
	for i, l := range lines {
		if h, ok := heading(l); ok && h == g.Section {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("no %q section (the guide's shape may have changed): %w", g.Section, plugin.ErrNothingToRead)
	}
	table := -1
	for i := start + 1; i+1 < len(lines); i++ {
		if _, ok := heading(lines[i]); ok {
			break
		}
		if !inFence && isRow(lines[i]) && isDelimiter(lines[i+1]) {
			table = i
			break
		}
	}
	if table < 0 {
		return nil, fmt.Errorf("no table under %q: %w", g.Section, plugin.ErrNothingToRead)
	}
	header := pipeCells(lines[table])
	if len(header) < 2 || !strings.EqualFold(header[0], g.header()) {
		return nil, fmt.Errorf("the table under %q isn't the removed members: %q", g.Section, lines[table])
	}
	head := g.Section + "\n" + lines[table] + "\n" + lines[table+1]
	var out []Removal
	for i := table + 2; i < len(lines) && isRow(lines[i]); i++ {
		cells := pipeCells(lines[i])
		excerpt := head + "\n" + lines[i]
		if len(cells) != len(header) {
			out = append(out, Removal{Member: lines[i], Excerpt: excerpt,
				Dropped: fmt.Sprintf("%d cells, the header has %d", len(cells), len(header))})
			continue
		}
		guideline := header[1] + ": " + cells[1]
		names, why := memberNames(cells[0], g.Module)
		if why != "" {
			out = append(out, Removal{Member: cells[0], Guideline: guideline, Excerpt: excerpt, Dropped: why})
			continue
		}
		repl := soleName(cells[1], g.Module)
		replPath := repl
		if p, ok := g.Paths[repl]; ok {
			replPath = p
		}
		for _, n := range names {
			r := Removal{Member: n, Guideline: guideline, Excerpt: excerpt}
			if p, ok := g.Paths[n]; ok {
				r.Member = p
			}
			switch {
			case slices.Contains(g.EnvVars, n):
				r.Dropped = "an environment variable, not a member of the package"
			case g.StillWork[n] != "":
				r.Dropped = g.StillWork[n]
			case repl != "" && repl != n:
				r.Replacement = replPath
			}
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("the removed members table under %q is empty: %w", g.Section, plugin.ErrNothingToRead)
	}
	return out, nil
}

func isRow(l string) bool {
	return strings.HasPrefix(strings.TrimSpace(l), "|")
}

func isDelimiter(l string) bool {
	if !isRow(l) {
		return false
	}
	cells := pipeCells(l)
	for _, c := range cells {
		if !delimCell.MatchString(c) {
			return false
		}
	}
	return len(cells) > 0
}

// pipeCells splits a table row on the pipes outside code spans, each cell
// trimmed. An escaped pipe (\|) is part of its cell.
func pipeCells(row string) []string {
	row = strings.TrimSpace(row)
	row = strings.TrimPrefix(row, "|")
	if strings.HasSuffix(row, "|") && !strings.HasSuffix(row, `\|`) {
		row = row[:len(row)-1]
	}
	var cells []string
	var cur strings.Builder
	inCode := false
	for i := 0; i < len(row); i++ {
		c := row[i]
		switch {
		case c == '\\' && i+1 < len(row) && row[i+1] == '|':
			cur.WriteByte('|')
			i++
		case c == '`':
			inCode = !inCode
			cur.WriteByte(c)
		case c == '|' && !inCode:
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	return append(cells, strings.TrimSpace(cur.String()))
}

// memberNames reads a first cell that is only a list of code spans, each a
// name as code writes it after importing the package (the package's own
// prefix, "posthog.", stripped, as the changelog plugin does). why says why a
// cell isn't one.
func memberNames(cell, module string) (names []string, why string) {
	spans := codeSpan.FindAllStringSubmatch(cell, -1)
	if len(spans) == 0 {
		return nil, "no code span: not a member's name"
	}
	rest := strings.Join(strings.Fields(codeSpan.ReplaceAllString(cell, "§")), " ")
	if !nameList.MatchString(rest) {
		return nil, "prose around the names, not a list of members (a parameter, a dependency…)"
	}
	for _, s := range spans {
		n, ok := cleanName(s[1], module)
		if !ok {
			return nil, fmt.Sprintf("`%s` isn't a name code writes", s[1])
		}
		names = append(names, n)
	}
	return names, ""
}

// soleName is the cell's one name when the cell is exactly one code span
// naming a member, or a link around one ("[`name`](url)": New Relic's
// release notes link each replacement to its docs page), or "".
func soleName(cell, module string) string {
	cell = strings.TrimSpace(cell)
	if l := linkedSpan.FindStringSubmatch(cell); l != nil {
		cell = l[1]
	}
	m := codeSpan.FindStringSubmatch(cell)
	if m == nil || m[0] != cell {
		return ""
	}
	n, ok := cleanName(m[1], module)
	if !ok {
		return ""
	}
	return n
}

func cleanName(s, module string) (string, bool) {
	s = strings.TrimSuffix(strings.TrimSpace(s), "()")
	if module != "" {
		s = strings.TrimPrefix(s, module+".")
	}
	return s, ident.MatchString(s) && s != module
}
