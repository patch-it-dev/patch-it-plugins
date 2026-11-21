package feed

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
)

// GitHubActions is the GitHub changelog's Actions label, RSS 2.0 (checked
// 2026-11-14: the latest 10 posts, each with its labels as <category>, the
// post's HTML in content:encoded). A retirement is labelled "Retired"; a
// runner image retirement names the date ("The macOS 14 runner image will be
// retired on November 2, 2026.") and lists the `runs-on` labels it removes.
var GitHubActions = Feed{
	API:    "github/actions",
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
	API:    "google/vertex-ai-models",
	Vendor: "Google",
	URL:    "https://docs.cloud.google.com/feeds/generative-ai-on-vertex-ai-release-notes.xml",
	Read:   readVertexAI,
}

// CloudflareAPI is Cloudflare's API deprecations page as RSS 2.0 (checked
// 2026-11-19: every announcement since 2020, 50 items, newest first, no
// categories). Every item is a deprecation, so none needs marking. An item
// names what it deprecates under "Deprecated APIs:", one `METHOD /path` per
// list item, kept apart from "Affected APIs:" (endpoints that only change),
// and gives "Deprecation date:" and, for most since 2025, "End of life date:".
var CloudflareAPI = Feed{
	API:    "cloudflare/api",
	Vendor: "Cloudflare",
	URL:    "https://developers.cloudflare.com/fundamentals/api/reference/deprecations/index.xml",
	Read:   readCloudflareAPI,
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

var (
	// cfDeprecatedList is the "Deprecated APIs:" paragraph and the list after
	// it. Only the first: no item on the feed (2026-11-19) has two.
	cfDeprecatedList = regexp.MustCompile(`(?is)<p>\s*Deprecated APIs?:\s*</p>\s*<ul>(.*?)</ul>`)
	// cfOperation is a list item that starts with one operation in code:
	// "GET /zones/{zone_id}/settings", or as the oldest entries write it,
	// "GET zones/:zone_identifier/analytics/dashboard" or "GET/zones/...".
	cfOperation  = regexp.MustCompile(`^(GET|POST|PUT|PATCH|DELETE) ?/?([A-Za-z0-9_{}:@.\-]+(?:/[A-Za-z0-9_{}:@.\-]*)*)$`)
	cfColonParam = regexp.MustCompile(`:([A-Za-z_][A-Za-z0-9_]*)`)
	cfEOL        = regexp.MustCompile(`(?i)^End of life date:`)
	cfDeprecated = regexp.MustCompile(`(?i)^Deprecation date:`)
	// cfStops is the item saying the endpoints stop, with the endpoints as
	// what stops: "After this date, the legacy endpoints will no longer be
	// available." Only a paragraph counts: a list item under "Affected APIs:"
	// saying "Mirage parameter will be removed" is about a field. And not a
	// sentence that only mentions the API: "Service Key authentication for
	// the Cloudflare API is deprecated and will be removed" is about the
	// authentication.
	cfStops     = regexp.MustCompile(`(?i)\b(endpoints?|APIs?|operations?) will (no longer be (available|accessible|served)|be (removed|shut down|turned off))\b`)
	cfParagraph = regexp.MustCompile(`(?is)<p>(.*?)</p>`)
	// cfNoAction is an item telling direct API callers nothing changes for
	// them: "For users calling the API directly (not via the Cloudflare SDK),
	// no action is required."
	cfNoAction = regexp.MustCompile(`(?i)calling the API directly[^.]*no action is required`)
	// cfSentence is text after an operation that says something about it
	// ("— creating rules with action: audit_ssh is no longer accepted"), as
	// opposed to a name for it ("— List domains", "(DNS Analytics API)").
	cfSentence = regexp.MustCompile(`(?i)\b(is|are|was|were|will|no longer|must|only|with)\b`)
)

func readCloudflareAPI(e Entry) []Finding {
	title := strings.TrimPrefix(e.Title, "API deprecations - ")
	drop := func(why string) []Finding {
		return []Finding{{Entry: e.Link, Dropped: fmt.Sprintf("%q: %s", title, why)}}
	}
	m := cfDeprecatedList.FindStringSubmatch(e.HTML)
	if m == nil {
		// Behavior changes, fields or settings, cipher suites: no operation
		// is deprecated, so there's no api_operation to name.
		return drop(`no "Deprecated APIs:" list of operations`)
	}

	lines := Lines(e.HTML)
	for _, l := range lines {
		if cfNoAction.MatchString(l) {
			// The URL keeps working; only an SDK method moves (the Radar
			// summary endpoints, folded into /summary/{dimension}).
			return drop("says direct API callers need no action: " + l)
		}
	}

	// Breaking needs the item to say the endpoints stop ("will no longer be
	// available") and a full end-of-life date. An end-of-life date alone isn't
	// enough: of the 11 items with one on 2026-11-19, only 2 say the
	// endpoints stop, and one tells API callers no action is required.
	// Anything else is a deprecation, dated by the deprecation date, else the
	// day the item was published.
	var eolLine, depLine, stopLine string
	for _, l := range lines {
		switch {
		case eolLine == "" && cfEOL.MatchString(l):
			eolLine = l
		case depLine == "" && cfDeprecated.MatchString(l):
			depLine = l
		}
	}
	for _, p := range cfParagraph.FindAllStringSubmatch(e.HTML, -1) {
		if l := Text(p[1]); cfStops.MatchString(l) {
			stopLine = l
			break
		}
	}
	ct, date, dateLine := event.ChangeTypeDeprecation, Date(depLine), depLine
	if d := Date(eolLine); d != "" && stopLine != "" {
		ct, date, dateLine = event.ChangeTypeBreaking, d, eolLine
	}
	if date == "" {
		if t, err := time.Parse(time.RFC1123, strings.TrimSpace(e.Published)); err == nil {
			date, dateLine = t.UTC().Format("2006-01-02"), ""
		}
	}
	if date == "" {
		return drop("no date: neither an end of life, a deprecation date nor a publication date")
	}

	var out []Finding
	for _, li := range listItem.FindAllStringSubmatch(m[1], -1) {
		codes := Codes(li[1])
		text := Text(li[1])
		// One operation, in code, first, and at most a name after it: "GET
		// /path — List domains". Not "GET and POST /path", a hostname, nor
		// "POST /path — creating rules with action: audit_ssh is no longer
		// accepted", which deprecates a value, not the operation.
		if len(codes) < 1 || !strings.HasPrefix(text, codes[0]) || cfSentence.MatchString(strings.TrimPrefix(text, codes[0])) {
			out = append(out, Finding{Entry: e.Link, Dropped: fmt.Sprintf("%q: list item %q isn't one operation", title, text)})
			continue
		}
		op := cfOperation.FindStringSubmatch(codes[0])
		if op == nil {
			out = append(out, Finding{Entry: e.Link, Dropped: fmt.Sprintf("%q: list item %q isn't one operation", title, text)})
			continue
		}
		path := "/" + strings.TrimSuffix(cfColonParam.ReplaceAllString(op[2], "{$1}"), "/")
		operation := op[1] + " " + path

		desc := fmt.Sprintf("Cloudflare API operation %s is deprecated (%s).", operation, date)
		if ct == event.ChangeTypeBreaking {
			desc = fmt.Sprintf("Cloudflare API operation %s reaches end of life on %s, the date Cloudflare stops serving it.", operation, date)
		}
		quote := []string{e.Title}
		if dateLine != "" {
			quote = append(quote, dateLine)
		}
		if ct == event.ChangeTypeBreaking {
			quote = append(quote, stopLine)
		}
		out = append(out, Finding{
			ChangeType:  ct,
			Version:     date,
			Subject:     event.Subject{Type: event.SubjectAPIOperation, Kind: event.KindOperation, Side: event.SideRequest, Operation: operation},
			Description: desc,
			Hint:        "Move to the replacement Cloudflare names in its deprecation notice: " + e.Link,
			Excerpt:     excerpt(append(quote, "Deprecated APIs:", text)...),
			Entry:       e.Link,
		})
	}
	return out
}
