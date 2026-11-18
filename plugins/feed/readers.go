package feed

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/event"
)

// GitHubActions is the GitHub changelog's Actions label, RSS 2.0 (checked
// 2026-11-14: the latest 10 posts, each with its labels as <category>, the
// post's HTML in content:encoded). A retirement is labelled "Retired"; a
// runner image retirement names the date ("The macOS 14 runner image will be
// retired on November 2, 2026.") and lists the `runs-on` labels it removes.
var GitHubActions = Feed{
	API:    "github-actions",
	Vendor: "GitHub",
	URL:    "https://github.blog/changelog/label/actions/feed/",
	Read:   readGitHubActions,
}

// VertexAIModels is Google Cloud's "Generative AI on Vertex AI" release
// notes, Atom (checked 2026-11-14: one entry per day, each change under an
// <h3> naming its kind: Feature, Change, Announcement, Deprecated). A model
// deprecation is a "Deprecated" section with a table of model ids in <code>
// (discontinued, recommended migration) and the date to move by.
var VertexAIModels = Feed{
	API:    "vertex-ai-models",
	Vendor: "Google",
	URL:    "https://docs.cloud.google.com/feeds/generative-ai-on-vertex-ai-release-notes.xml",
	Read:   readVertexAI,
}

var (
	runnerImage  = regexp.MustCompile(`(?i)\brunner images?\b`)
	willStop     = regexp.MustCompile(`(?i)\bwill be (retired|removed|shut down|discontinued)\b`)
	willSoften   = regexp.MustCompile(`(?i)\b(will be deprecated|is deprecated|are deprecated|before)\b`)
	labelsList   = regexp.MustCompile(`(?is)<p>([^<]*following labels[^<]*)</p>\s*<ul>(.*?)</ul>`)
	listItem     = regexp.MustCompile(`(?is)<li>(.*?)</li>`)
	tableRow     = regexp.MustCompile(`(?is)<tr>(.*?)</tr>`)
	tableCell    = regexp.MustCompile(`(?is)<t[dh][^>]*>(.*?)</t[dh]>`)
	plainModelID = regexp.MustCompile(`^[a-z0-9][a-z0-9.@_-]*$`)
)

func marked(labels []string, words ...string) bool {
	for _, l := range labels {
		for _, w := range words {
			if strings.EqualFold(l, w) {
				return true
			}
		}
	}
	return false
}

// dated is the first line holding a date and saying what happens on it:
// breaking when the thing stops on the date, deprecation when the vendor only
// asks to move by it. "" when no line says either.
func dated(lines []string) (line, date string, ct event.ChangeType) {
	for _, l := range lines {
		// The date after the words, not the first on the line: "deprecated
		// as of February 23, 2026 and will be shut down on August 23, 2026".
		if m := willStop.FindStringIndex(l); m != nil {
			if d := Date(l[m[1]:]); d != "" {
				return l, d, event.ChangeTypeBreaking
			}
		}
		if m := willSoften.FindStringIndex(l); m != nil {
			if d := Date(l[m[1]:]); d != "" {
				return l, d, event.ChangeTypeDeprecation
			}
		}
	}
	return "", "", ""
}

func excerpt(lines ...string) string {
	s := strings.Join(lines, "\n")
	if r := []rune(s); len(r) > 2000 {
		s = string(r[:1999]) + "…"
	}
	return s
}

func readGitHubActions(e Entry) []Finding {
	if !marked(e.Categories, "Retired", "Deprecation", "Deprecated", "Breaking change", "Breaking changes") {
		return nil
	}
	drop := func(why string) []Finding { return []Finding{{Entry: e.Link, Dropped: why}} }
	if !runnerImage.MatchString(e.Title) {
		return drop("not a runner image retirement: no ci_runtime this reader can name")
	}
	line, date, ct := dated(Lines(e.HTML))
	if date == "" {
		return drop("no sentence giving the date the image is removed")
	}
	m := labelsList.FindStringSubmatch(e.HTML)
	if m == nil {
		return drop("lists no runner labels")
	}
	intro := Text(m[1])
	var out []Finding
	for _, li := range listItem.FindAllStringSubmatch(m[2], -1) {
		codes := Codes(li[1])
		if len(codes) != 1 || Text(li[1]) != codes[0] {
			out = append(out, Finding{Entry: e.Link, Dropped: fmt.Sprintf("label item %q isn't one plain label", Text(li[1]))})
			continue
		}
		label := codes[0]
		desc := fmt.Sprintf("GitHub-hosted runner label %q is retired on %s: jobs that run on it fail from that date.", label, date)
		if ct == event.ChangeTypeDeprecation {
			desc = fmt.Sprintf("GitHub-hosted runner label %q is deprecated; GitHub names %s for it.", label, date)
		}
		out = append(out, Finding{
			ChangeType:  ct,
			Version:     date,
			Subject:     event.Subject{Type: event.SubjectCIRuntime, Platform: "github-actions", Runtime: label},
			Description: desc,
			Hint:        "Move `runs-on` to a supported label; GitHub's announcement lists them: " + e.Link,
			Excerpt:     excerpt(e.Title, line, intro, label),
			Entry:       e.Link,
		})
	}
	if len(out) == 0 {
		return drop("lists no runner labels")
	}
	return out
}

func readVertexAI(e Entry) []Finding {
	var out []Finding
	for _, s := range SplitBy(e.HTML, 3) {
		if !marked([]string{s.Heading}, "Deprecated", "Breaking") {
			continue
		}
		lines := Lines(s.HTML)
		drop := func(why string) {
			title := ""
			if len(lines) > 0 {
				title = lines[0]
			}
			out = append(out, Finding{Entry: e.Link, Dropped: fmt.Sprintf("%q: %s", title, why)})
		}
		rows := tableRow.FindAllStringSubmatch(s.HTML, -1)
		if len(rows) < 2 {
			drop("names no model id in a table")
			continue
		}
		header := cells(rows[0][1])
		if len(header) < 2 || !strings.Contains(strings.ToLower(header[0]), "discontinued") {
			drop(fmt.Sprintf("a table that isn't discontinued models: %q", header))
			continue
		}
		line, date, ct := dated(lines)
		if date == "" {
			drop("no sentence giving the date to move by")
			continue
		}
		for _, r := range rows[1:] {
			raw := tableCell.FindAllStringSubmatch(r[1], -1)
			if len(raw) < 2 {
				continue
			}
			names := Codes(raw[0][1])
			if len(names) != 1 || !plainModelID.MatchString(names[0]) || Text(raw[0][1]) != names[0] {
				drop(fmt.Sprintf("row %q doesn't name one model id", Text(raw[0][1])))
				continue
			}
			name, migrate := names[0], Text(raw[1][1])
			subject := event.Subject{Type: event.SubjectModel, Provider: "google", Name: name}
			hint := fmt.Sprintf("Google recommends: %s.", migrate)
			// Exactly one plain model, and not the model itself (the feed has a
			// row whose migration is its own name): anything else stays a hint.
			if repl := Codes(raw[1][1]); len(repl) == 1 && Text(raw[1][1]) == repl[0] && repl[0] != name {
				subject.Replacement = repl[0]
				hint = fmt.Sprintf("Use `%s` instead: Google's recommended migration.", repl[0])
			}
			desc := fmt.Sprintf("Model %q is deprecated on Vertex AI: Google asks to move off it before %s to avoid service disruption.", name, date)
			if ct == event.ChangeTypeBreaking {
				desc = fmt.Sprintf("Model %q is shut down on Vertex AI on %s: requests naming it fail from that date.", name, date)
			}
			out = append(out, Finding{
				ChangeType:  ct,
				Version:     date,
				Subject:     subject,
				Description: desc,
				Hint:        hint,
				Excerpt:     excerpt(s.Heading, lines[0], line, strings.Join(header, " | "), name+" | "+migrate),
				Entry:       e.Link,
			})
		}
	}
	return out
}

func cells(row string) []string {
	var out []string
	for _, c := range tableCell.FindAllStringSubmatch(row, -1) {
		out = append(out, Text(c[1]))
	}
	return out
}
