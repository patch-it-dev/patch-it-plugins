package changelog

import (
	"fmt"
	"regexp"
	"strings"
)

// Section is one titled part of a release's notes ("Removed", "Changed") and
// its bullets, each with its continuation lines.
type Section struct {
	Title   string
	Bullets []Bullet
}

// Bullet is one list item: its text on one line, and its lines as written.
type Bullet struct {
	Text string // continuation lines joined with a space
	Raw  string // as in the file

	ended bool // a blank line closed its first paragraph
}

var (
	// Markdown: "## [1.2.0] - 2024-01-01", "## 1.2.0 (date)", "## v1.2.0".
	mdRelease = regexp.MustCompile(`^##\s+\[?v?([0-9][^\]\s)]*)\]?`)
	mdSection = regexp.MustCompile(`^###\s+(.+?)\s*$`)
	// "**Deprecations**:" on a line of its own (httpx's variant).
	boldSection = regexp.MustCompile(`^\*\*([^*]+)\*\*:?\s*$`)
	// RST: a title line, then an underline of one punctuation character at
	// least as long.
	rstUnderline = regexp.MustCompile(`^([=\-~+^"'*#])+$`)
	rstRelease   = regexp.MustCompile(`^v?([0-9][^\s(]*)`)
	bullet       = regexp.MustCompile(`^[-*+]\s+`)
)

// ReleaseNotes finds release in a changelog and returns its sections. The
// format is told from the file: Markdown headings (Keep a Changelog's
// "## [x]" / "### Removed", and "## x" variants) or reStructuredText titles
// (HISTORY.rst, CHANGES.rst: "2.0.0 (date)" over "=====", sections over
// "-----"). Text before the first section title is not a section. A
// changelog without the release is an error: the file moved or the release
// isn't in it, and neither may read as "no breaking changes".
func ReleaseNotes(file []byte, release string) (heading string, sections []Section, err error) {
	lines := strings.Split(strings.ReplaceAll(string(file), "\r\n", "\n"), "\n")
	heading, body, rst, err := releaseLines(lines, release)
	if err != nil {
		return "", nil, err
	}
	var out []Section
	var cur *Section
	for i := 0; i < len(body); i++ {
		l := body[i]
		title := ""
		switch {
		case rst && i+1 < len(body) && strings.TrimSpace(l) != "" && !bullet.MatchString(l) &&
			rstUnderline.MatchString(strings.TrimSpace(body[i+1])) && len(strings.TrimSpace(body[i+1])) >= len(strings.TrimSpace(l)):
			title = strings.TrimSpace(l)
			i++
		case !rst && mdSection.MatchString(l):
			title = mdSection.FindStringSubmatch(l)[1]
		case boldSection.MatchString(strings.TrimSpace(l)):
			title = boldSection.FindStringSubmatch(strings.TrimSpace(l))[1]
		}
		if title != "" {
			out = append(out, Section{Title: title})
			cur = &out[len(out)-1]
			continue
		}
		if cur == nil {
			continue
		}
		if bullet.MatchString(l) {
			cur.Bullets = append(cur.Bullets, Bullet{Text: strings.TrimSpace(bullet.ReplaceAllString(l, "")), Raw: l})
			continue
		}
		// A continuation: indented, or (some changelogs) flush left right
		// under the bullet. A blank line ends the bullet's first paragraph.
		n := len(cur.Bullets)
		if n == 0 {
			continue
		}
		b := &cur.Bullets[n-1]
		if strings.TrimSpace(l) == "" {
			b.ended = true
		} else if !b.ended {
			b.Text += " " + strings.TrimSpace(l)
			b.Raw += "\n" + l
		}
	}
	return heading, out, nil
}

// releaseLines are release's heading and the lines under it, up to the next
// release's, and whether the file is reStructuredText.
func releaseLines(lines []string, release string) (string, []string, bool, error) {
	// Markdown first: a "## " heading naming the release.
	for i, l := range lines {
		if m := mdRelease.FindStringSubmatch(l); m != nil && m[1] == release {
			end := len(lines)
			for j := i + 1; j < len(lines); j++ {
				if strings.HasPrefix(lines[j], "## ") {
					end = j
					break
				}
			}
			return strings.TrimSpace(l), lines[i+1 : end], false, nil
		}
	}
	// reStructuredText: a title naming the release, underlined.
	for i := 0; i+1 < len(lines); i++ {
		m := rstRelease.FindStringSubmatch(strings.TrimSpace(lines[i]))
		u := strings.TrimSpace(lines[i+1])
		if m == nil || m[1] != release || !rstUnderline.MatchString(u) || len(u) < len(strings.TrimSpace(lines[i])) {
			continue
		}
		level := u[0]
		end := len(lines)
		for j := i + 2; j+1 < len(lines); j++ {
			v := strings.TrimSpace(lines[j+1])
			if rstUnderline.MatchString(v) && v[0] == level && rstRelease.MatchString(strings.TrimSpace(lines[j])) &&
				len(v) >= len(strings.TrimSpace(lines[j])) {
				end = j
				break
			}
		}
		return strings.TrimSpace(lines[i]), lines[i+2 : end], true, nil
	}
	return "", nil, false, fmt.Errorf("no heading for release %s: the changelog may have moved or changed shape", release)
}
