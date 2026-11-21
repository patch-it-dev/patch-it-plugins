package openai

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// Deprecations is the endpoints on OpenAI's deprecations page (checked
// 2026-11-21): the "Feature and endpoint deprecations" table and the "Past
// deprecations" table name an endpoint by its path alone ("`/v1/prompts`",
// "`/v1/edits`"), with its shutdown date. A row is read only when its
// affected cell starts with a path in a code span; a feature named in words
// ("Agent Builder", "Evals dashboard, API, and graders") names no operation,
// and models are openai/models'.
var Deprecations = EndpointsPage{
	API: "openai/deprecations",
	URL: "https://developers.openai.com/api/docs/deprecations.md",
}

// EndpointsPage is OpenAI's deprecations page read for endpoints.
type EndpointsPage struct {
	API string // the source id the events carry
	URL string // the page, as markdown
}

// Endpoint is one endpoint the page says shuts down on Date.
type Endpoint struct {
	Path        string // "/v1/prompts"
	Date        string // YYYY-MM-DD
	Replacement string // the one path the page names instead, or ""
	Excerpt     string // the section heading, the table header and the row
}

// Endpoints are what one read of the page found.
type Endpoints []Endpoint

// Count is how many endpoints the page lists.
func (es Endpoints) Count() int { return len(es) }

// Has says whether entry is an endpoint's path.
func (es Endpoints) Has(entry string) bool {
	return slices.ContainsFunc(es, func(e Endpoint) bool { return e.Path == entry })
}

// Slug is the source id the events carry.
func (p EndpointsPage) Slug() string { return p.API }

// PageURL is the page, as markdown.
func (p EndpointsPage) PageURL() string { return p.URL }

var (
	// leadingPath is an affected cell that starts with a path in a code span.
	leadingPath = regexp.MustCompile("^`(/v[0-9]+/[A-Za-z0-9_./{}-]+)`")
	// onlyPath is a replacement cell that is one path: in a code span, or a
	// link whose text is the path (in a code span or not).
	onlyPath = regexp.MustCompile("^(?:`(/v[0-9]+/[A-Za-z0-9_./{}-]+)`|\\[`?(/v[0-9]+/[A-Za-z0-9_./{}-]+)`?\\]\\([^)]*\\))$")
	shutdown = regexp.MustCompile(`(?i)^shutdown date$`)
	dateFmts = []string{"Jan 2, 2006", "January 2, 2006"}
)

// Find reads every endpoint row of the page's tables with a shutdown date
// column and an "Affected ..." column.
func (p EndpointsPage) Find(page []byte) (kit.Notices, error) {
	var out Endpoints
	heading := ""
	lines := strings.Split(strings.ReplaceAll(string(page), "\r\n", "\n"), "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "#") {
			heading = line
			continue
		}
		header := pipeCells(line)
		when, what, repl := -1, -1, -1
		for c, h := range header {
			switch {
			case shutdown.MatchString(h):
				when = c
			case strings.HasPrefix(strings.ToLower(h), "affected"):
				what = c
			case strings.Contains(strings.ToLower(h), "replacement"):
				repl = c
			}
		}
		if when < 0 || what < 0 {
			continue
		}
		for i += 2; i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "|"); i++ {
			row := pipeCells(strings.TrimSpace(lines[i]))
			if len(row) != len(header) {
				continue
			}
			m := leadingPath.FindStringSubmatch(row[what])
			date := shutdownDate(row[when])
			if m == nil || date == "" {
				continue
			}
			e := Endpoint{Path: m[1], Date: date,
				Excerpt: heading + "\n" + strings.Join(header, " | ") + "\n" + strings.Join(row, " | ")}
			if repl >= 0 {
				if r := onlyPath.FindStringSubmatch(row[repl]); r != nil {
					e.Replacement = r[1] + r[2]
				}
			}
			out = append(out, e)
		}
		i--
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: no endpoint rows found (the page's shape may have changed): %w", p.API, plugin.ErrNothingToRead)
	}
	return out, nil
}

// pipeCells is a Markdown pipe table row's cells, trimmed; nil for a line
// that isn't a row.
func pipeCells(line string) []string {
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return nil
	}
	var cells []string
	var cur strings.Builder
	for i := 1; i < len(line)-1; i++ {
		switch {
		case line[i] == '\\' && i+1 < len(line)-1 && line[i+1] == '|':
			cur.WriteByte('|')
			i++
		case line[i] == '|':
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteByte(line[i])
		}
	}
	return append(cells, strings.TrimSpace(cur.String()))
}

// shutdownDate is a date cell ("**Nov 30, 2026**") as YYYY-MM-DD, or "" for
// anything but one plain date ("Jun 13, 2024 Earliest estimate").
func shutdownDate(cell string) string {
	cell = strings.TrimSpace(strings.Trim(strings.TrimSpace(cell), "*"))
	for _, f := range dateFmts {
		if t, err := time.Parse(f, cell); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return ""
}

// Events are an api_operation event per endpoint, citing citeURL: every
// method on the path stops on the shutdown date. A version pair: previous is
// the read it was compared with.
func (p EndpointsPage) Events(found kit.Notices, citeURL, previous string, now time.Time) ([]event.Event, []error) {
	var events []event.Event
	var errs []error
	for _, ep := range found.(Endpoints) {
		subject := event.Subject{Type: event.SubjectAPIOperation, Kind: event.KindOperation, Side: event.SideRequest, Operation: ep.Path}
		hint := "OpenAI names no one endpoint to use instead; see the page's recommended replacement."
		if ep.Replacement != "" {
			hint = fmt.Sprintf("Use %s instead: OpenAI's recommended replacement.", ep.Replacement)
		}
		e := event.Event{
			Fingerprint:     event.Fingerprint(p.API, ep.Date, event.ChangeBreaking, subject, ""),
			API:             p.API,
			Version:         ep.Date,
			PreviousVersion: previous,
			Change:          event.ChangeBreaking,
			Subject:         subject,
			Description:     fmt.Sprintf("OpenAI shuts down %s on %s: requests to it, whatever the method, fail from that date.", ep.Path, ep.Date),
			MigrationHint:   hint,
			SourceURL:       citeURL,
			SourceExcerpt:   ep.Excerpt,
			Confidence:      1.0,
			DetectedAt:      now,
		}
		if err := e.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("%s %s: %w", p.API, ep.Path, err))
			continue
		}
		events = append(events, e)
	}
	return events, errs
}
