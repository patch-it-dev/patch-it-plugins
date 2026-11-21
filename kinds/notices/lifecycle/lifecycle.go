// Package lifecycle is the notices kind's reader for a vendor's lifecycle
// table of a runtime or server: one row per version, with the date it
// reaches end of life (kinds/notices reads it, snapshots and cites it). One
// runtime event per version a row gives a date for: the version stops being
// supported on that date, whether the date has passed or not, as a model's
// shutdown date is. A row with no date yet ("TBD", "-") names nothing.
//
// Each provider says how its page reads (Page.Rows): which table, which
// cells hold the version and the date. HTMLTables reads a page's tables
// under their headings for it.
package lifecycle

import (
	"fmt"
	"html"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// Page is one vendor's lifecycle table.
type Page struct {
	API     string // the source id the events carry: "mongodb/server"
	Product string // for sentences: "MongoDB Server"
	URL     string // the page
	// Runtime is the subject's runtime, as configuration writes it: "mongodb".
	Runtime string
	// Platform is where it runs, when that isn't the vendor's own product;
	// "" for a product's own lifecycle.
	Platform string
	// Rows reads the page's rows.
	Rows func(page []byte) ([]Row, error)
}

// Row is one version and its end-of-life date.
type Row struct {
	Version string // "6.0"
	EOL     string // YYYY-MM-DD; "" when the row gives no date yet
	Excerpt string // the table's heading, its header and the row, as the page writes them
}

// Rows are what one read of a page found.
type Rows []Row

// Count is how many versions the page lists.
func (rs Rows) Count() int { return len(rs) }

// Has says whether entry is a version the page gives an end-of-life date.
func (rs Rows) Has(entry string) bool {
	return slices.ContainsFunc(rs, func(r Row) bool { return r.Version == entry && r.EOL != "" })
}

// Slug is the source id the events carry.
func (p Page) Slug() string { return p.API }

// PageURL is the page.
func (p Page) PageURL() string { return p.URL }

// Find reads every row of the page. A page with none is
// plugin.ErrNothingToRead: its shape changed.
func (p Page) Find(page []byte) (kit.Notices, error) {
	rs, err := p.Rows(page)
	if err != nil {
		return nil, err
	}
	if len(rs) == 0 {
		return nil, fmt.Errorf("%s: no version rows found (the page's shape may have changed): %w", p.API, plugin.ErrNothingToRead)
	}
	return Rows(rs), nil
}

// Events are a runtime event per row with a date, citing citeURL. An end of
// life is a dated notice: no previous_version.
func (p Page) Events(found kit.Notices, citeURL, _ string, now time.Time) ([]event.Event, []error) {
	var events []event.Event
	var errs []error
	for _, r := range found.(Rows) {
		if r.EOL == "" {
			continue
		}
		subject := event.Subject{Type: event.SubjectRuntime, Platform: p.Platform, Runtime: p.Runtime, Version: r.Version}
		e := event.Event{
			Fingerprint:   event.Fingerprint(p.API, r.EOL, event.ChangeBreaking, subject, ""),
			API:           p.API,
			Version:       r.EOL,
			Change:        event.ChangeBreaking,
			Subject:       subject,
			Description:   fmt.Sprintf("%s %s reaches end of life on %s: no fixes or support for it from that date.", p.Product, r.Version, r.EOL),
			MigrationHint: fmt.Sprintf("Move to a %s version whose end of life is later; the table lists each one's date.", p.Product),
			SourceURL:     citeURL,
			SourceExcerpt: r.Excerpt,
			Confidence:    1.0,
			DetectedAt:    now,
		}
		if err := e.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("%s %s: %w", p.API, r.Version, err))
			continue
		}
		events = append(events, e)
	}
	return events, errs
}

// Table is one HTML table: the heading above it and its rows of cell texts,
// the header row first.
type Table struct {
	Heading string
	Rows    [][]string
}

var (
	htmlStyle = regexp.MustCompile(`(?is)<(style|script)\b[^>]*>.*?</(style|script)>`)
	htmlTable = regexp.MustCompile(`(?is)<h([1-6])[^>]*>(.*?)</h[1-6]>|<table[^>]*>(.*?)</table>`)
	htmlRow   = regexp.MustCompile(`(?is)<tr[^>]*>(.*?)</tr>`)
	htmlCell  = regexp.MustCompile(`(?is)<t[hd][^>]*>(.*?)</t[hd]>`)
	htmlTag   = regexp.MustCompile(`<[^>]+>`)
	spaces    = regexp.MustCompile(`\s+`)
)

// HTMLTables is every table on an HTML page, each with the nearest heading
// above it. Style and script elements (some pages put CSS inside a cell) are
// dropped first.
func HTMLTables(page []byte) []Table {
	var out []Table
	heading := ""
	s := htmlStyle.ReplaceAllString(string(page), "")
	for _, m := range htmlTable.FindAllStringSubmatch(s, -1) {
		if m[1] != "" {
			heading = text(m[2])
			continue
		}
		t := Table{Heading: heading}
		for _, r := range htmlRow.FindAllStringSubmatch(m[3], -1) {
			var row []string
			for _, c := range htmlCell.FindAllStringSubmatch(r[1], -1) {
				row = append(row, text(c[1]))
			}
			if len(row) > 0 {
				t.Rows = append(t.Rows, row)
			}
		}
		out = append(out, t)
	}
	return out
}

func text(s string) string {
	return strings.TrimSpace(spaces.ReplaceAllString(html.UnescapeString(htmlTag.ReplaceAllString(s, " ")), " "))
}

// Excerpt is a row quoted with its table's heading and header.
func (t Table) Excerpt(row []string) string {
	return t.Heading + "\n" + strings.Join(t.Rows[0], " | ") + "\n" + strings.Join(row, " | ")
}

var monthDay = regexp.MustCompile(`^(January|February|March|April|May|June|July|August|September|October|November|December) (\d{1,2}), (\d{4})$`)

// Date is a cell that is exactly "Month D, YYYY", as YYYY-MM-DD, or "": a
// cell with anything else ("TBD", "-", "TBD, no sooner than October 2026")
// gives no date.
func Date(cell string) string {
	m := monthDay.FindStringSubmatch(strings.TrimSpace(cell))
	if m == nil {
		return ""
	}
	d, _ := strconv.Atoi(m[2])
	t, err := time.Parse("January 2 2006", fmt.Sprintf("%s %d %s", m[1], d, m[3]))
	if err != nil {
		return ""
	}
	return t.Format("2006-01-02")
}
