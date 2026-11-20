// Package eol is the source plugin for a vendor's own API end-of-life page: a
// table of whole version ranges, each with a support status. One event per
// version number a row says is already retired (breaking) or scheduled to be
// (deprecation); a row that only says "Supported." names nothing to report.
//
// There is no spec to diff, like retirements and feed: the page is read as it
// is now, and (as with them) the events cite a Wayback Machine snapshot of
// it, taken after a successful read of the live page, so the citation always
// says what the plugin actually saw.
//
// Salesforce's docs site is the one difference from retirements and feed: it
// answers 403 to a plain GET (checked 2026-11-20, including with a browser
// User-Agent), so Run's direct read of the live page fails every time. The
// Wayback Machine's own save-page-now request succeeds against it anyway,
// and its capture parses to the same findings the live page would (checked
// the same day), so Run falls back to a fresh snapshot on a 401 or 403,
// used directly as both the content and its own citation, kit.Cite's
// double-check skipped: there is no separate live read left to agree with.
//
// Unlike retirements (one named model per row) and feed (an RSS or Atom
// entry), a row here names a whole range of version numbers and no
// replacement: moving up to any newer version clears it.
package eol

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/kit"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

func init() {
	plugin.Register(Plugin{Page: Salesforce, Archiver: kit.Wayback{}})
}

// Page is one vendor's end-of-life table.
type Page struct {
	API    string // the events' api: "salesforce"
	Vendor string // for sentences: "Salesforce"
	URL    string // the page, as HTML
	// Operation names the one real, documented, version-rooted resource a
	// retired version takes down: Salesforce's "List Available REST
	// Resources" call, GET /services/data/vXX.X/. Every other resource under
	// that version is gone too (the page says so), but this is the one this
	// plugin can point at honestly, without inventing a wildcard operation
	// nothing calls.
	Operation func(version string) string
}

// Salesforce is the REST API's end-of-life policy page (checked 2026-11-20):
// a table of version ranges, each "Supported.", already retired ("As of
// Summer '25, these versions are retired and unavailable."), or scheduled
// ("Deprecated and unsupported from Summer '27. Retired from Summer '28.").
// Salesforce gives no exact date, only the named release ("Summer '27"); the
// policy text above the table says the notice is at least a year ahead of
// retirement.
var Salesforce = Page{
	API: "salesforce", Vendor: "Salesforce",
	URL:       "https://developer.salesforce.com/docs/platform/api-rest/guide/api-rest-eol.html",
	Operation: func(v string) string { return "GET /services/data/v" + v + "/" },
}

// Row is one version range and what its status cell says, cleaned of HTML.
type Row struct {
	From, To int    // inclusive, both ends; From == To for a single version
	Label    string // the first cell's own text, word for word ("Versions 31.0 through 40.0")
	Status   string // every <p> in the status cell, joined with a space
}

// Finding is one version number a row's status names, or a row that names
// none (Dropped set).
type Finding struct {
	ChangeType event.ChangeType
	Version    string // "21.0"
	Season     string // "Summer ’25", or "Summer ’27, retired from Summer ’28"
	Excerpt    string
	Row        string // the row's version-range text, for Dropped's log line
	Dropped    string
}

var (
	table = regexp.MustCompile(`(?is)<tbody>(.*?)</tbody>`)
	row   = regexp.MustCompile(`(?is)<tr>(.*?)</tr>`)
	// cell tolerates an attribute (<td class="…">): a future style change
	// shouldn't make every row unreadable.
	cell = regexp.MustCompile(`(?is)<td[^>]*>(.*?)</td>`)
	tag  = regexp.MustCompile(`<[^>]+>`)
	// versions reads only the whole-number part of a version ("31.0 through
	// 40.0" -> 31, 40): every version on the page today is a whole number.
	// Salesforce has never shipped a minor REST API version, but this would
	// silently drop one ("31.5") rather than fail loudly if that changed.
	versions = regexp.MustCompile(`(?i)Versions?\s+(\d+)(?:\.\d+)?(?:\s+through\s+(\d+)(?:\.\d+)?)?`)
	retired  = regexp.MustCompile(`(?i)As of ([^,]+), these versions are retired and unavailable\.`)
	deprecd  = regexp.MustCompile(`(?i)Deprecated and unsupported from ([^.]+)\.\s*Retired from ([^.]+)\.`)
)

// text strips tags and normalizes whitespace, the way a <td>'s content (bare
// text, or several <p>s) reads as one sentence.
func text(fragment string) string {
	return strings.Join(strings.Fields(tag.ReplaceAllString(fragment, " ")), " ")
}

// Parse reads every row of the page's first table.
func Parse(page []byte) ([]Row, error) {
	m := table.FindStringSubmatch(string(page))
	if m == nil {
		return nil, fmt.Errorf("no table found: the page's shape may have changed")
	}
	var out []Row
	for _, r := range row.FindAllStringSubmatch(m[1], -1) {
		cells := cell.FindAllStringSubmatch(r[1], -1)
		if len(cells) < 2 {
			continue
		}
		label := text(cells[0][1])
		vm := versions.FindStringSubmatch(label)
		if vm == nil {
			continue
		}
		from, _ := strconv.Atoi(vm[1])
		to := from
		if vm[2] != "" {
			to, _ = strconv.Atoi(vm[2])
		}
		out = append(out, Row{From: from, To: to, Label: label, Status: text(cells[1][1])})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no version rows found: the page's shape may have changed")
	}
	return out, nil
}

// Findings classifies every row and names every version number a breaking or
// deprecated row covers. A row that's merely "Supported." names none.
func Findings(page []byte) ([]Finding, error) {
	rows, err := Parse(page)
	if err != nil {
		return nil, err
	}
	var out []Finding
	for _, r := range rows {
		label := r.Label
		excerpt := label + " | " + r.Status

		var ct event.ChangeType
		var season string
		switch {
		case retired.MatchString(r.Status):
			ct, season = event.ChangeTypeBreaking, retired.FindStringSubmatch(r.Status)[1]
		case deprecd.MatchString(r.Status):
			m := deprecd.FindStringSubmatch(r.Status)
			ct, season = event.ChangeTypeDeprecation, m[1]+", retired from "+m[2]
		default:
			out = append(out, Finding{Row: label, Dropped: fmt.Sprintf("status %q names neither a retirement nor a deprecation", r.Status)})
			continue
		}
		for v := r.From; v <= r.To; v++ {
			out = append(out, Finding{
				ChangeType: ct,
				Version:    fmt.Sprintf("%d.0", v),
				Season:     season,
				Excerpt:    excerpt,
				Row:        label,
			})
		}
	}
	return out, nil
}

// Plugin reads one vendor's end-of-life page.
type Plugin struct {
	Page     Page
	Archiver kit.Archiver
}

func (p Plugin) Name() string { return p.Page.API }

// Run reads the page as it is now. An unchanged page since the last run costs
// one GET and no snapshot, unless the live page is blocked: see liveOrArchived.
func (p Plugin) Run(ctx context.Context, env plugin.Env) (plugin.Result, error) {
	pg := p.Page
	live, archived, err := liveOrArchived(ctx, env, p.Archiver, pg.URL)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("fetching %s: %w", pg.URL, err)
	}
	hash := plugin.SHA256(live)
	last, same, err := kit.Unchanged(env, pg.API, live)
	if err != nil {
		return plugin.Result{}, err
	}
	if same {
		log.Printf("%s: no change since %s", pg.API, last.SourceURL)
		return plugin.Result{}, nil
	}

	want, err := Findings(live)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("reading %s: %w", pg.URL, err)
	}

	snapURL, snap, got := archived, live, want
	if snapURL == "" {
		snapURL, snap, got, err = kit.Cite(ctx, p.Archiver, pg.URL, Findings, want)
		if err != nil {
			return plugin.Result{}, err
		}
	}

	now := env.Now().UTC()
	var events []event.EventV2
	var rejected []error
	for _, f := range got {
		if f.Dropped != "" {
			log.Printf("%s: dropped %s: %s", pg.API, f.Row, f.Dropped)
			continue
		}
		e := event.EventV2{
			API:        pg.API,
			Version:    f.Version,
			ChangeType: f.ChangeType,
			Breaking:   f.ChangeType == event.ChangeTypeBreaking,
			Subject: event.Subject{
				Type:      event.SubjectAPIOperation,
				Kind:      event.AffectedKindOperation,
				Side:      event.SideRequest,
				Operation: pg.Operation(f.Version),
			},
			Description:   describe(pg, f),
			MigrationHint: fmt.Sprintf("Pin a newer %s REST API version: %s names which ones are still supported.", pg.Vendor, snapURL),
			SourceURL:     snapURL,
			SourceExcerpt: f.Excerpt,
			Confidence:    1.0,
			DetectedAt:    now,
		}
		e.Fingerprint = event.FingerprintV2(e.API, e.Version, e.ChangeType, e.Subject, "")
		if err := e.Validate(); err != nil {
			rejected = append(rejected, fmt.Errorf("%s %s: %w", pg.API, f.Version, err))
			continue
		}
		events = append(events, e)
	}
	baseline := plugin.Baseline(pg.API, now.Format("2006-01-02"), snapURL, snapURL, snap, false, now)
	baseline.SHA256 = hash
	return plugin.Result{Label: snapURL, Events: events, Rejected: rejected, Baselines: []plugin.Spec{baseline}}, nil
}

// liveOrArchived reads pageURL directly. When the provider's own site
// blocks a plain fetch with 401 or 403 (Salesforce's docs site, every time,
// checked 2026-11-20; confirmed the same day that a real Wayback Machine
// save-page-now capture of this exact page parses to the same findings as
// the live page would), it falls back to a fresh Wayback Machine snapshot,
// taken right now by the archive's own crawler rather than patch-it's, and
// returns that content with its own permanent URL in archivedURL: it needs
// no further citation step, having come from the archive already.
//
// Only 401 and 403 fall back. A 404 or 410 means the page moved, not that
// it's blocked: asking the archive to snapshot a page that's gone would
// waste the call and hide the real cause behind the archive's own error.
// A 429 or a 5xx is returned as is: retrying it against the archive would
// hide a transient failure as a real page, not fix one.
func liveOrArchived(ctx context.Context, env plugin.Env, a kit.Archiver, pageURL string) (body []byte, archivedURL string, err error) {
	body, err = env.Get(ctx, pageURL)
	if err == nil {
		return body, "", nil
	}
	var se *fetch.StatusError
	if !errors.As(err, &se) || (se.StatusCode != 401 && se.StatusCode != 403) {
		return nil, "", err
	}
	archivedURL, body, archErr := a.Snapshot(ctx, pageURL)
	if archErr != nil {
		return nil, "", fmt.Errorf("fetching it directly failed (%w), and so did asking the Wayback Machine to: %w", err, archErr)
	}
	return body, archivedURL, nil
}

func describe(pg Page, f Finding) string {
	if f.ChangeType == event.ChangeTypeBreaking {
		return fmt.Sprintf("%s REST API version %s is retired, as of %s: it returns 410:GONE for every resource.", pg.Vendor, f.Version, f.Season)
	}
	return fmt.Sprintf("%s REST API version %s is deprecated and scheduled for retirement: unsupported from %s.", pg.Vendor, f.Version, f.Season)
}
