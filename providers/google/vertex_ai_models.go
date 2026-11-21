package google

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/feed"
)

// VertexAIModels is Google Cloud's "Generative AI on Vertex AI" release
// notes, Atom (checked 2026-11-14: one entry per day, each change under an
// <h3> naming its kind: Feature, Change, Announcement, Deprecated). A model
// deprecation is a "Deprecated" section with a table of model ids in <code>
// (discontinued, recommended migration) and the date to move by.
var VertexAIModels = feed.Feed{
	API:    "google/vertex-ai-models",
	Vendor: "Google",
	URL:    "https://docs.cloud.google.com/feeds/generative-ai-on-vertex-ai-release-notes.xml",
	Read:   readVertexAI,
}

var tableRow = regexp.MustCompile(`(?is)<tr>(.*?)</tr>`)

var tableCell = regexp.MustCompile(`(?is)<t[dh][^>]*>(.*?)</t[dh]>`)

var plainModelID = regexp.MustCompile(`^[a-z0-9][a-z0-9.@_-]*$`)

func readVertexAI(e feed.Entry) []feed.Finding {
	var out []feed.Finding
	for _, s := range feed.SplitBy(e.HTML, 3) {
		if !feed.Marked([]string{s.Heading}, "Deprecated", "Breaking") {
			continue
		}
		lines := feed.Lines(s.HTML)
		drop := func(why string) {
			title := ""
			if len(lines) > 0 {
				title = lines[0]
			}
			out = append(out, feed.Finding{Entry: e.Link, Dropped: fmt.Sprintf("%q: %s", title, why)})
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
		line, date, ct := feed.Dated(lines)
		if date == "" {
			drop("no sentence giving the date to move by")
			continue
		}
		for _, r := range rows[1:] {
			raw := tableCell.FindAllStringSubmatch(r[1], -1)
			if len(raw) < 2 {
				continue
			}
			names := feed.Codes(raw[0][1])
			if len(names) != 1 || !plainModelID.MatchString(names[0]) || feed.Text(raw[0][1]) != names[0] {
				drop(fmt.Sprintf("row %q doesn't name one model id", feed.Text(raw[0][1])))
				continue
			}
			name, migrate := names[0], feed.Text(raw[1][1])
			subject := event.Subject{Type: event.SubjectModel, Provider: "google", Name: name}
			hint := fmt.Sprintf("Google recommends: %s.", migrate)
			// Exactly one plain model, and not the model itself (the feed has a
			// row whose migration is its own name): anything else stays a hint.
			if repl := feed.Codes(raw[1][1]); len(repl) == 1 && feed.Text(raw[1][1]) == repl[0] && repl[0] != name {
				subject.Replacement = repl[0]
				hint = fmt.Sprintf("Use `%s` instead: Google's recommended migration.", repl[0])
			}
			desc := fmt.Sprintf("Model %q is deprecated on Vertex AI: Google asks to move off it before %s to avoid service disruption.", name, date)
			if ct == event.ChangeTypeBreaking {
				desc = fmt.Sprintf("Model %q is shut down on Vertex AI on %s: requests naming it fail from that date.", name, date)
			}
			out = append(out, feed.Finding{
				ChangeType:  ct,
				Version:     date,
				Subject:     subject,
				Description: desc,
				Hint:        hint,
				Excerpt:     feed.Excerpt(s.Heading, lines[0], line, strings.Join(header, " | "), name+" | "+migrate),
				Entry:       e.Link,
			})
		}
	}
	return out
}

func cells(row string) []string {
	var out []string
	for _, c := range tableCell.FindAllStringSubmatch(row, -1) {
		out = append(out, feed.Text(c[1]))
	}
	return out
}
