package github

import (
	"fmt"
	"regexp"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/extract"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/feed"
)

// Actions is the GitHub changelog's Actions label, RSS 2.0 (checked
// 2026-11-14: the latest 10 posts, each with its labels as <category>, the
// post's HTML in content:encoded). A retirement is labelled "Retired"; a
// runner image retirement names the date ("The macOS 14 runner image will be
// retired on November 2, 2026.") and lists the `runs-on` labels it removes.
var Actions = feed.Feed{
	API:    "github/actions",
	Vendor: "GitHub",
	URL:    "https://github.blog/changelog/label/actions/feed/",
	Read:   readGitHubActions,
}

var runnerImage = regexp.MustCompile(`(?i)\brunner images?\b`)

var labelsList = regexp.MustCompile(`(?is)<p>([^<]*following labels[^<]*)</p>\s*<ul>(.*?)</ul>`)

func readGitHubActions(e extract.Entry) []feed.Finding {
	if !feed.Marked(e.Categories, "Retired", "Deprecation", "Deprecated", "Breaking change", "Breaking changes") {
		return nil
	}
	drop := func(why string) []feed.Finding { return []feed.Finding{{Entry: e.Link, Dropped: why}} }
	if !runnerImage.MatchString(e.Title) {
		return drop("not a runner image retirement: no ci_runtime this reader can name")
	}
	line, date, ct := feed.Dated(extract.Lines(e.HTML))
	if date == "" {
		return drop("no sentence giving the date the image is removed")
	}
	m := labelsList.FindStringSubmatch(e.HTML)
	if m == nil {
		return drop("lists no runner labels")
	}
	intro := extract.Text(m[1])
	var out []feed.Finding
	for _, li := range feed.ListItem.FindAllStringSubmatch(m[2], -1) {
		codes := extract.Codes(li[1])
		if len(codes) != 1 || extract.Text(li[1]) != codes[0] {
			out = append(out, feed.Finding{Entry: e.Link, Dropped: fmt.Sprintf("label item %q isn't one plain label", extract.Text(li[1]))})
			continue
		}
		label := codes[0]
		desc := fmt.Sprintf("GitHub-hosted runner label %q is retired on %s: jobs that run on it fail from that date.", label, date)
		if ct == event.ChangeDeprecated {
			desc = fmt.Sprintf("GitHub-hosted runner label %q is deprecated; GitHub names %s for it.", label, date)
		}
		out = append(out, feed.Finding{
			ChangeType:  ct,
			Version:     date,
			Subject:     event.Subject{Type: event.SubjectCIRuntime, Platform: "github-actions", Runtime: label},
			Description: desc,
			Hint:        "Move `runs-on` to a supported label; GitHub's announcement lists them: " + e.Link,
			Excerpt:     feed.Excerpt(e.Title, line, intro, label),
			Entry:       e.Link,
		})
	}
	if len(out) == 0 {
		return drop("lists no runner labels")
	}
	return out
}
