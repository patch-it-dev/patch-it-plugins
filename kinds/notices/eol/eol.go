// Package eol is the notices kind's reader for a vendor's own API
// end-of-life page (plugins/notices reads it, snapshots and cites it): a
// table of whole version ranges, each with a support status. One event per
// version number a row says is already retired (breaking) or scheduled to be
// (deprecation); a row that only says "Supported." names nothing to report.
//
// There is no spec to diff, like retirements and feed: the page is read as it
// is now, and (as with them) the events cite a Wayback Machine snapshot of
// it, taken after a successful read of the live page, so the citation always
// says what the plugin actually saw.
//
// Salesforce's docs site is the one difference from the other notices
// readers: it blocks a plain GET, with 403 (checked 2026-11-20, including
// with a browser User-Agent) or, from its CDN (AkamaiNetStorage), 503
// (checked 2026-11-21), so a direct read of the live page fails every time.
// The Wayback Machine's own crawler reads it anyway (its capture of
// 2026-11-20 parses to the same findings the live page would), so the kind
// falls back to a fresh snapshot on those statuses (ArchiveWhenBlocked),
// used directly as both the content and its own citation, kit.Cite's
// double-check skipped: there is no separate live read left to agree with.
// A snapshot that can't be taken, or holds no table, fails the run.
//
// Unlike retirements (one named model per row) and feed (an RSS or Atom
// entry), a row here names a whole range of version numbers and no
// replacement: moving up to any newer version clears it.
package eol

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// Page is one vendor's end-of-life table.
type Page struct {
	API    string // the source id the events carry: "salesforce/rest-api"
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
		return nil, fmt.Errorf("no table found (the page's shape may have changed): %w", plugin.ErrNothingToRead)
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
		return nil, fmt.Errorf("no version rows found (the page's shape may have changed): %w", plugin.ErrNothingToRead)
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

// What the notices kind (plugins/notices) asks of an end-of-life page.

// Read is what one read of a page found, dropped rows included.
type Read []Finding

// Count is how many findings the page gave.
func (r Read) Count() int { return len(r) }

// Slug is the source id the events carry: "salesforce/rest-api".
func (pg Page) Slug() string { return pg.API }

// PageURL is the page, as HTML.
func (pg Page) PageURL() string { return pg.URL }

// Find reads every row of the page.
func (pg Page) Find(page []byte) (kit.Notices, error) {
	fs, err := Findings(page)
	return Read(fs), err
}

// ArchiveWhenBlocked is the statuses Salesforce's docs site blocks a plain
// GET with (401 and 403; 503 from its CDN), so the kind reads a fresh
// Wayback Machine snapshot instead (see the package comment).
func (Page) ArchiveWhenBlocked() []int { return []int{401, 403, 503} }

// Events are an event per version number a row retires or deprecates,
// citing citeURL, with previous as previous_version (each is an
// api_operation: a version pair); a dropped row is logged with why.
func (pg Page) Events(found kit.Notices, citeURL, previous string, now time.Time) ([]event.Event, []error) {
	var events []event.Event
	var rejected []error
	for _, f := range found.(Read) {
		if f.Dropped != "" {
			log.Printf("%s: dropped %s: %s", pg.API, f.Row, f.Dropped)
			continue
		}
		e := event.Event{
			API:             pg.API,
			Version:         f.Version,
			PreviousVersion: previous,
			ChangeType:      f.ChangeType,
			Breaking:        f.ChangeType == event.ChangeTypeBreaking,
			Subject: event.Subject{
				Type:      event.SubjectAPIOperation,
				Kind:      event.KindOperation,
				Side:      event.SideRequest,
				Operation: pg.Operation(f.Version),
			},
			Description:   describe(pg, f),
			MigrationHint: fmt.Sprintf("Pin a newer %s REST API version: %s names which ones are still supported.", pg.Vendor, citeURL),
			SourceURL:     citeURL,
			SourceExcerpt: f.Excerpt,
			Confidence:    1.0,
			DetectedAt:    now,
		}
		e.Fingerprint = event.Fingerprint(e.API, e.Version, e.ChangeType, e.Subject, "")
		if err := e.Validate(); err != nil {
			rejected = append(rejected, fmt.Errorf("%s %s: %w", pg.API, f.Version, err))
			continue
		}
		events = append(events, e)
	}
	return events, rejected
}

func describe(pg Page, f Finding) string {
	if f.ChangeType == event.ChangeTypeBreaking {
		return fmt.Sprintf("%s REST API version %s is retired, as of %s: it returns 410:GONE for every resource.", pg.Vendor, f.Version, f.Season)
	}
	return fmt.Sprintf("%s REST API version %s is deprecated and scheduled for retirement: unsupported from %s.", pg.Vendor, f.Version, f.Season)
}
