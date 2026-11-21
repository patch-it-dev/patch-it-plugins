package feed

import (
	"regexp"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/extract"
)

var (
	willStop   = regexp.MustCompile(`(?i)\bwill be (retired|removed|shut down|discontinued)\b`)
	willSoften = regexp.MustCompile(`(?i)\b(will be deprecated|is deprecated|are deprecated|before)\b`)

	// ListItem is an HTML list item; its text is the first group.
	ListItem = regexp.MustCompile(`(?is)<li>(.*?)</li>`)
)

// Marked reports whether one of an entry's labels is one of words, in any
// case.
func Marked(labels []string, words ...string) bool {
	for _, l := range labels {
		for _, w := range words {
			if strings.EqualFold(l, w) {
				return true
			}
		}
	}
	return false
}

// Dated is the first line holding a date and saying what happens on it:
// breaking when the thing stops on the date, deprecation when the vendor only
// asks to move by it. "" when no line says either.
func Dated(lines []string) (line, date string, ct event.Change) {
	for _, l := range lines {
		// The date after the words, not the first on the line: "deprecated
		// as of February 23, 2026 and will be shut down on August 23, 2026".
		if m := willStop.FindStringIndex(l); m != nil {
			if d := extract.Date(l[m[1]:]); d != "" {
				return l, d, event.ChangeBreaking
			}
		}
		if m := willSoften.FindStringIndex(l); m != nil {
			if d := extract.Date(l[m[1]:]); d != "" {
				return l, d, event.ChangeDeprecated
			}
		}
	}
	return "", "", ""
}

// Excerpt is lines as one quote, capped at the contract's 2000 characters.
func Excerpt(lines ...string) string {
	s := strings.Join(lines, "\n")
	if r := []rune(s); len(r) > 2000 {
		s = string(r[:1999]) + "…"
	}
	return s
}
