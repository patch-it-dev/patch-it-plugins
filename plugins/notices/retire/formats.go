package retire

import (
	"regexp"
	"strings"
)

var (
	// markdocCell is one cell of a Markdoc table: "  * `gemini-2.0-flash`".
	markdocCell    = regexp.MustCompile(`^\s*\* ?(.*)$`)
	markdocHeading = regexp.MustCompile(`^(#+ .*?)\s*(\{%.*%\})?\s*$`)
)

// markdocPipes reads a Markdoc page's tables ({% table %}, a "* cell" per
// line, rows split by "---", the first row the header) into pipe tables,
// each under the heading above it. Nothing else on the page is kept.
func markdocPipes(page []byte) []byte {
	var out []string
	var rows [][]string
	var cur []string
	in := false
	flush := func() {
		if len(cur) > 0 {
			rows = append(rows, cur)
			cur = nil
		}
	}
	for _, line := range strings.Split(strings.ReplaceAll(string(page), "\r\n", "\n"), "\n") {
		t := strings.TrimSpace(line)
		switch {
		case !in && strings.HasPrefix(t, "#"):
			if m := markdocHeading.FindStringSubmatch(t); m != nil {
				out = append(out, m[1])
			}
		case !in && t == "{% table %}":
			in, rows, cur = true, nil, nil
		case in && t == "{% /table %}":
			flush()
			out = append(out, pipeTable(rows)...)
			in = false
		case in && t == "---":
			flush()
		case in && strings.HasPrefix(t, "*"):
			cur = append(cur, strings.TrimSpace(markdocCell.FindStringSubmatch(line)[1]))
		case in && t != "" && len(cur) > 0:
			// A cell's text continued on the next line.
			cur[len(cur)-1] += " " + t
		}
	}
	return []byte(strings.Join(out, "\n") + "\n")
}

// pipeTable writes rows (the first the header) as a Markdown pipe table,
// escaping a pipe inside a cell.
func pipeTable(rows [][]string) []string {
	if len(rows) == 0 {
		return nil
	}
	line := func(cells []string) string {
		esc := make([]string, len(cells))
		for i, c := range cells {
			esc[i] = strings.ReplaceAll(c, "|", `\|`)
		}
		return "| " + strings.Join(esc, " | ") + " |"
	}
	out := []string{"", line(rows[0]), "|" + strings.Repeat(" --- |", len(rows[0]))}
	for _, r := range rows[1:] {
		out = append(out, line(r))
	}
	return append(out, "")
}
