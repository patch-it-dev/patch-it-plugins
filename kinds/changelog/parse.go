package changelog

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
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
	// Markdown: "## [1.2.0] - 2024-01-01", "## 1.2.0 (date)", "## v1.2.0",
	// Stripe's "## <a id="16-1-0"></a>16.1.0 - date", Datadog's
	// "## 2.67.0/2026-10-13", and semantic-release's "# [5.0.0](...)" for a
	// major.
	mdRelease = regexp.MustCompile(`^##?\s+(?:<a [^>]*></a>\s*)?\[?v?([0-9][^\]\s)/]*)\]?`)
	mdSection = regexp.MustCompile(`^###\s+(.+?)\s*$`)
	// A release's notes on their own use any level.
	anyHeading = regexp.MustCompile(`^#{1,6}\s+(.+?)\s*$`)
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
func ReleaseNotes(file []byte, release string) (string, []Section, error) {
	lines := strings.Split(strings.ReplaceAll(string(file), "\r\n", "\n"), "\n")
	heading, body, rst, err := releaseLines(lines, release)
	if err != nil {
		return "", nil, err
	}
	return heading, sections(body, rst, mdSection), nil
}

// NotesSections are the sections of one release's notes on their own (a
// GitHub release's body): every Markdown heading, whatever its level, starts
// one, and so does a bold line.
func NotesSections(body string) []Section {
	return sections(strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n"), false, anyHeading)
}

// sections splits a release's lines into titled sections of bullets: a
// Markdown heading (heading) or a reStructuredText title (rst) starts one.
func sections(body []string, rst bool, heading *regexp.Regexp) []Section {
	var out []Section
	var cur *Section
	fenced := false
	for i := 0; i < len(body); i++ {
		l := body[i]
		// A fenced code block is an example, never a title or a bullet.
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		title := ""
		switch {
		case rst && i+1 < len(body) && strings.TrimSpace(l) != "" && !bullet.MatchString(l) &&
			rstUnderline.MatchString(strings.TrimSpace(body[i+1])) && len(strings.TrimSpace(body[i+1])) >= len(strings.TrimSpace(l)):
			title = strings.TrimSpace(l)
			i++
		case !rst && heading.MatchString(l):
			title = heading.FindStringSubmatch(l)[1]
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
	return out
}

// releaseLines are release's heading and the lines under it, up to the next
// release's, and whether the file is reStructuredText.
func releaseLines(lines []string, release string) (string, []string, bool, error) {
	// Markdown first: a "## " (or "# ") heading naming the release, up to
	// the next one.
	for i, l := range lines {
		if m := mdRelease.FindStringSubmatch(l); m != nil && m[1] == release {
			end := len(lines)
			for j := i + 1; j < len(lines); j++ {
				if strings.HasPrefix(lines[j], "## ") || strings.HasPrefix(lines[j], "# ") {
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
	return "", nil, false, fmt.Errorf("no heading for release %s (the changelog may have moved or changed shape): %w", release, plugin.ErrNothingToRead)
}
