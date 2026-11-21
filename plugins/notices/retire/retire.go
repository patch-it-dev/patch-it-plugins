// Package retire reads a model provider's deprecations page and turns every
// retired model into change events: a second kind of source next to the
// OpenAPI differ. There is no spec to diff, only the provider's own table of
// shutdown date, model and named replacement, so this reads the table.
//
// A page is a Markdown pipe table (OpenAI's and Anthropic's pages, served as
// markdown at their URL plus ".md", checked 2026-10-13), a Markdoc table
// (Google's Gemini API page, {% table %} with one "* cell" per line, checked
// 2026-11-21), or an HTML table (Google Cloud's Vertex AI model versions,
// checked 2026-11-21). The last two are read into pipe rows first, with
// their headings, so one parser reads every page. A table is a retirement
// table when it has a shutdown or retirement date column, a model column and
// a replacement column, in any order. The event cites a Wayback Machine
// snapshot of the page (kit/), never the live page: the contract requires a
// source that can't change under the event.
package retire

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// Page is one provider's deprecations page and what its models are sent to.
type Page struct {
	API      string // the source id the events carry: "openai/models"
	Provider string // the model subject's provider: "openai"
	Vendor   string // for sentences: "OpenAI"
	URL      string // the page, as markdown or HTML
	// Format is how the page writes its tables: Markdown pipe tables (the
	// default), Markdoc or HTML.
	Format Format
	// Keep says whether a model name belongs to this source: the operations
	// above take text models, not image, audio or embedding ones.
	Keep func(name string) bool
	// Earliest says the page's dates are the earliest a model may be shut
	// down, not the day it is (Google's Gemini API page): the description
	// says so.
	Earliest bool
}

// Format is how a page writes its tables.
type Format string

const (
	FormatMarkdown Format = ""
	FormatMarkdoc  Format = "markdoc"
	FormatHTML     Format = "html"
)

// Retirement is one model name the provider stops serving on Date, with the
// replacement it names. An alias gets its own Retirement, with the row it
// came from.
type Retirement struct {
	Model       string
	Date        string // YYYY-MM-DD
	Replacement string // the replacement cell, as written ("`gpt-5.6-sol` (`reasoning.mode: pro`)")
	Via         []string
	Excerpt     string // the section heading, the table header and the row
}

var (
	dateHeader = regexp.MustCompile(`(?i)^(shutdown|retirement) date$`)
	ticked     = regexp.MustCompile("`([^`]+)`")
)

// Pipes is the page's tables as Markdown pipe tables, each under the
// heading above it: the page itself when it's Markdown.
func (f Format) Pipes(page []byte) []byte {
	switch f {
	case FormatMarkdoc:
		return markdocPipes(page)
	case FormatHTML:
		return htmlPipes(page)
	}
	return page
}

var dateLayouts = []string{"January 2, 2006", "Jan 2, 2006", "2006-01-02"}

// Parse reads every retirement table on the page. A model listed more than
// once keeps its first row: both pages list the most recent announcement
// first, and a later one supersedes an earlier date, including a later row
// that withdraws the date ("To be announced"). A replacement that is
// itself retired is followed to the model that replaces it, so a hint never
// points at a model that no longer works.
func Parse(page []byte, keep func(string) bool) ([]Retirement, error) {
	lines := strings.Split(strings.ReplaceAll(string(page), "\r\n", "\n"), "\n")
	headerAt := 0
	var out []Retirement
	seen := map[string]bool{}
	heading := ""
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "#") {
			heading = line
			continue
		}
		header := cells(line)
		when := -1
		for c, h := range header {
			if dateHeader.MatchString(h) {
				when = c
				break
			}
		}
		if len(header) < 3 || when < 0 {
			continue
		}
		model, repl := column(header, "model"), column(header, "replacement", "substitute", "upgrade")
		if model < 0 || repl < 0 || model == repl {
			continue // an endpoint ("System"), an agent or a training-price table: no model to replace
		}
		headerAt = i
		i++ // the |---| row
		for i+1 < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i+1]), "|") {
			i++
			row := strings.TrimSpace(lines[i])
			c := cells(row)
			if len(c) <= model || len(c) <= repl || len(c) <= when {
				return nil, fmt.Errorf("a short row under %q: %s", heading, row)
			}
			names := ticked.FindAllStringSubmatch(c[model], -1)
			date, err := parseDate(c[when])
			if err != nil {
				// "To be announced": not a retirement yet, and the newer word on
				// these names, so an older row's date for them doesn't count.
				for _, m := range names {
					seen[m[1]] = true
				}
				continue
			}
			excerpt := strings.Join([]string{heading, strings.TrimSpace(lines[headerAt]), row}, "\n")
			for _, m := range names {
				name := m[1]
				if seen[name] || !keep(name) {
					continue
				}
				seen[name] = true
				out = append(out, Retirement{Model: name, Date: date, Replacement: c[repl], Excerpt: excerpt})
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no retirement tables found (the page's shape may have changed): %w", plugin.ErrNothingToRead)
	}
	follow(out)
	return out, nil
}

// follow replaces a replacement that is itself retired with its own
// replacement, recording the hops: gpt-3.5-turbo-0613's named replacement is
// gpt-3.5-turbo, which shuts down on 2026-10-23 for gpt-5.6-terra.
func follow(rs []Retirement) {
	by := map[string]Retirement{}
	for _, r := range rs {
		by[r.Model] = r
	}
	for i := range rs {
		for hops := 0; hops < 10; hops++ {
			next, ok := by[First(rs[i].Replacement)]
			if !ok || next.Model == rs[i].Model {
				break
			}
			rs[i].Via = append(rs[i].Via, next.Model)
			rs[i].Replacement = next.Replacement
		}
	}
}

// First is the first model name a replacement cell gives, "" when it names
// none ("The most capable cyber model available to you.").
func First(replacement string) string {
	if m := ticked.FindStringSubmatch(replacement); m != nil {
		return m[1]
	}
	return ""
}

// cells splits a markdown table row on its unescaped pipes. A cell can hold
// an escaped one: "`gpt-4-0613` \| `gpt-4`".
func cells(row string) []string {
	if !strings.HasPrefix(row, "|") {
		return nil
	}
	row = strings.Trim(row, "|")
	var out []string
	var cur strings.Builder
	for i := 0; i < len(row); i++ {
		if row[i] == '\\' && i+1 < len(row) && row[i+1] == '|' {
			cur.WriteString(`\|`)
			i++
			continue
		}
		if row[i] == '|' {
			out = append(out, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}
		cur.WriteByte(row[i])
	}
	return append(out, strings.TrimSpace(cur.String()))
}

func column(header []string, words ...string) int {
	for _, w := range words {
		for i, h := range header {
			if strings.Contains(strings.ToLower(h), w) {
				return i
			}
		}
	}
	return -1
}

func parseDate(s string) (string, error) {
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, strings.TrimSpace(s)); err == nil {
			return t.Format("2006-01-02"), nil
		}
	}
	return "", fmt.Errorf("not a date: %q", s)
}

// Events turns retirements into change events, one per model name: a model
// subject, the provider and the name, and the replacement when the provider
// names exactly one model. The patch-it client looks for the name wherever
// it reaches the provider's SDK, so one event per name gives one fix per
// repo. version is the shutdown date, so a provider moving the date is a new
// event. sourceURL is the snapshot the retirements were read from.
func Events(p Page, rs []Retirement, sourceURL string, now time.Time) ([]event.Event, []error) {
	var events []event.Event
	var errs []error
	for _, r := range rs {
		h := hint(p.Vendor, r)
		subject := event.Subject{Type: event.SubjectModel, Provider: p.Provider, Name: r.Model, Replacement: replacement(r.Replacement)}
		e := event.Event{
			Fingerprint:   event.Fingerprint(p.API, r.Date, event.ChangeTypeBreaking, subject, ""),
			API:           p.API,
			Version:       r.Date,
			ChangeType:    event.ChangeTypeBreaking,
			Breaking:      true,
			Subject:       subject,
			Description:   describe(p, r),
			MigrationHint: h,
			SourceURL:     sourceURL,
			SourceExcerpt: r.Excerpt,
			Confidence:    1.0,
			DetectedAt:    now,
		}
		if err := e.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("%s %s: %w", p.API, r.Model, err))
			continue
		}
		events = append(events, e)
	}
	return events, errs
}

func describe(p Page, r Retirement) string {
	if p.Earliest {
		return fmt.Sprintf("Model %q can be shut down from %s, the earliest date %s gives: %s requests naming it fail once it is.", r.Model, r.Date, p.Vendor, p.Vendor)
	}
	return fmt.Sprintf("Model %q is shut down on %s: %s requests naming it fail from that date.", r.Model, r.Date, p.Vendor)
}

// replacement is the one model the provider names, when its replacement cell
// is exactly one model name: not none, not several, not a model with a
// setting (`gpt-5.6-sol` (`reasoning.mode: pro`)). The patch-it client opens
// an issue for the rest.
func replacement(cell string) string {
	cell = strings.TrimSpace(cell)
	name := First(cell)
	if name == "" || cell != "`"+name+"`" {
		return ""
	}
	return name
}

func hint(vendor string, r Retirement) string {
	repl := strings.TrimSpace(r.Replacement)
	if repl == "" {
		return fmt.Sprintf("%s names no replacement model.", vendor)
	}
	if First(repl) == "" {
		return fmt.Sprintf("%s names no replacement model: %s", vendor, repl)
	}
	h := fmt.Sprintf("Use %s instead: %s's recommended replacement.", repl, vendor)
	if len(r.Via) > 0 {
		h += fmt.Sprintf(" (%s first named %s, which is itself retired.)", vendor, strings.Join(r.Via, ", then "))
	}
	return h
}
