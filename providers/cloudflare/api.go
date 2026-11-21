package cloudflare

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/extract"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/feed"
)

// API is Cloudflare's API deprecations page as RSS 2.0 (checked
// 2026-11-19: every announcement since 2020, 50 items, newest first, no
// categories). Every item is a deprecation, so none needs marking. An item
// names what it deprecates under "Deprecated APIs:", one `METHOD /path` per
// list item, kept apart from "Affected APIs:" (endpoints that only change),
// and gives "Deprecation date:" and, for most since 2025, "End of life date:".
var API = feed.Feed{
	API:    "cloudflare/api",
	Vendor: "Cloudflare",
	URL:    "https://developers.cloudflare.com/fundamentals/api/reference/deprecations/index.xml",
	Read:   readCloudflareAPI,
}

func readCloudflareAPI(e extract.Entry) []feed.Finding {
	title := strings.TrimPrefix(e.Title, "API deprecations - ")
	drop := func(why string) []feed.Finding {
		return []feed.Finding{{Entry: e.Link, Dropped: fmt.Sprintf("%q: %s", title, why)}}
	}
	m := cfDeprecatedList.FindStringSubmatch(e.HTML)
	if m == nil {
		// Behavior changes, fields or settings, cipher suites: no operation
		// is deprecated, so there's no api_operation to name.
		return drop(`no "Deprecated APIs:" list of operations`)
	}

	lines := extract.Lines(e.HTML)
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
		if l := extract.Text(p[1]); cfStops.MatchString(l) {
			stopLine = l
			break
		}
	}
	ct, date, dateLine := event.ChangeDeprecated, extract.Date(depLine), depLine
	if d := extract.Date(eolLine); d != "" && stopLine != "" {
		ct, date, dateLine = event.ChangeBreaking, d, eolLine
	}
	if date == "" {
		if t, err := time.Parse(time.RFC1123, strings.TrimSpace(e.Published)); err == nil {
			date, dateLine = t.UTC().Format("2006-01-02"), ""
		}
	}
	if date == "" {
		return drop("no date: neither an end of life, a deprecation date nor a publication date")
	}

	var out []feed.Finding
	for _, li := range feed.ListItem.FindAllStringSubmatch(m[1], -1) {
		codes := extract.Codes(li[1])
		text := extract.Text(li[1])
		// One operation, in code, first, and at most a name after it: "GET
		// /path — List domains". Not "GET and POST /path", a hostname, nor
		// "POST /path — creating rules with action: audit_ssh is no longer
		// accepted", which deprecates a value, not the operation.
		if len(codes) < 1 || !strings.HasPrefix(text, codes[0]) || cfSentence.MatchString(strings.TrimPrefix(text, codes[0])) {
			out = append(out, feed.Finding{Entry: e.Link, Dropped: fmt.Sprintf("%q: list item %q isn't one operation", title, text)})
			continue
		}
		op := cfOperation.FindStringSubmatch(codes[0])
		if op == nil {
			out = append(out, feed.Finding{Entry: e.Link, Dropped: fmt.Sprintf("%q: list item %q isn't one operation", title, text)})
			continue
		}
		path := "/" + strings.TrimSuffix(cfColonParam.ReplaceAllString(op[2], "{$1}"), "/")
		operation := op[1] + " " + path

		desc := fmt.Sprintf("Cloudflare API operation %s is deprecated (%s).", operation, date)
		if ct == event.ChangeBreaking {
			desc = fmt.Sprintf("Cloudflare API operation %s reaches end of life on %s, the date Cloudflare stops serving it.", operation, date)
		}
		quote := []string{e.Title}
		if dateLine != "" {
			quote = append(quote, dateLine)
		}
		if ct == event.ChangeBreaking {
			quote = append(quote, stopLine)
		}
		out = append(out, feed.Finding{
			ChangeType:  ct,
			Version:     date,
			Subject:     event.Subject{Type: event.SubjectAPIOperation, Kind: event.KindOperation, Side: event.SideRequest, Operation: operation},
			Description: desc,
			Hint:        "Move to the replacement Cloudflare names in its deprecation notice: " + e.Link,
			Excerpt:     feed.Excerpt(append(quote, "Deprecated APIs:", text)...),
			Entry:       e.Link,
		})
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
