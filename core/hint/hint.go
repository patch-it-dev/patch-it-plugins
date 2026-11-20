// Package hint fills a change event's migration_hint from the vendor's own
// changelog. The spec diff knows that a parameter disappeared; only the
// vendor's prose says what replaced it, and that is the difference between a
// fix that deletes the usage and one that keeps the code's behavior.
//
// A changelog is one entry per change, each with the text a human reads
// ("What's new") and a table mapping parameter, change kind and endpoint. The
// table is what makes matching deterministic: an event whose operation and
// parameter path, or schema and property path, appear in a row gets that
// entry's text and link. No model, no guessing, and the link is the hint's own
// provenance.
//
// Hints are optional. A changelog that can't be fetched or an entry that can't
// be parsed costs the hint, never the event.
package hint

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
)

// hintLimit caps a hint at the contract's excerpt cap, counted in characters.
const hintLimit = 2000

// Source knows one vendor's changelog: where a version's entries are and how
// their endpoint references name our operations.
type Source interface {
	Index(ctx context.Context, version string) (*Index, error)
}

// Entry is one changelog entry: one change, as the vendor describes it.
type Entry struct {
	Title    string
	URL      string // the entry's page, for a human
	WhatsNew string // the entry's "What's new" section, as published (markdown)
	Rows     []Row  // the REST API table
}

// Row is one line of an entry's REST API table.
type Row struct {
	Parameter string   // the parameter's own name, e.g. "payment_method_types"
	Change    string   // "Removed", "Added", "Changed"...
	Targets   []Target // where the row applies, each with the full parameter path there
}

// Target is one endpoint or object a row names, in the contract's terms, and
// the full dotted path of the parameter at that place: the same name can be a
// top-level parameter in one row and a nested one in another
// ("trial_settings.end_behavior.billing_cycle_anchor"), and they are
// different parameters.
type Target struct {
	Operation string // "POST /v1/payment_intents/{}": path parameters normalized
	Schema    string // "payment_intent": an object (a response schema)
	// Path is the full dotted parameter path at that place, "[]" dropped:
	// "trial_settings.end_behavior.billing_cycle_anchor". Required: a target
	// without one is keyed on an empty path and matches no event.
	Path string
}

// Index is every entry of one version, keyed for lookup by what an event affects.
type Index struct {
	Entries []Entry
	byKey   map[string][]match
}

type match struct {
	entry  int
	change string
}

// NewIndex keys entries by (operation or schema, parameter path). Two entries
// naming the same key is logged: Lookup takes the first on the release page and
// the second's guidance is never seen, which should be visible rather than
// silent (it hasn't happened on a Stripe release yet).
func NewIndex(entries []Entry) *Index {
	ix := &Index{Entries: entries, byKey: map[string][]match{}}
	for i, e := range entries {
		for _, r := range e.Rows {
			for _, t := range r.Targets {
				k := key(t.Operation, t.Schema, t.Path)
				if prior := ix.byKey[k]; len(prior) > 0 && prior[0].entry != i {
					log.Printf("hint: %s %s %s is named by %q and %q; the first wins",
						t.Operation+t.Schema, t.Path, r.Change, entries[prior[0].entry].Title, e.Title)
				}
				ix.byKey[k] = append(ix.byKey[k], match{i, r.Change})
			}
		}
	}
	return ix
}

// Lookup finds the entry whose table names what the event affects: the same
// operation (request side) or schema (response side), the same parameter path
// ("[]" dropped on both sides), and a change kind that fits the event (a row
// that *adds* a parameter never explains a breaking change to it). When
// several entries name the same key, the first on the release page wins.
func (ix *Index) Lookup(e event.ChangeEvent) (Entry, bool) {
	a := e.Affected
	if ix == nil || a.Property == "" {
		return Entry{}, false
	}
	var k string
	if a.Operation != "" {
		k = key(NormalizeOperation(a.Operation), "", path(a.Property))
	} else {
		k = key("", a.Schema, path(a.Property))
	}
	for _, m := range ix.byKey[k] {
		if compatible(m.change, e.ChangeType) {
			return ix.Entries[m.entry], true
		}
	}
	return Entry{}, false
}

// compatible keeps a row's text off an event going the other way: an "Added" row
// only explains an addition, a "Removed" row never does. Anything else
// ("Changed", ...) can explain either.
func compatible(rowChange string, ct event.ChangeType) bool {
	switch {
	case strings.EqualFold(rowChange, "Added"):
		return ct == event.ChangeTypeNewFeature
	case strings.EqualFold(rowChange, "Removed"):
		return ct != event.ChangeTypeNewFeature
	}
	return true
}

// Attach adds the changelog's text to every event an entry matches and returns
// how many. The normalizer's own hint, when it has one ("Handle X as Y", "Stop
// sending V"), is the precise sentence and stays first; the vendor's prose
// follows it. A match only ever adds information. An event nothing matches is
// left as it was.
func (ix *Index) Attach(events []event.ChangeEvent) int {
	n := 0
	for i := range events {
		e, ok := ix.Lookup(events[i])
		if !ok {
			continue
		}
		events[i].MigrationHint = capped(join(events[i].MigrationHint, Text(e)))
		n++
	}
	return n
}

// Text is the hint one entry gives: its "What's new" as plain prose (the
// markdown links' URLs dropped, their text kept: a 100-character docs URL per
// link is what pushed the useful sentence past the cap) and its own link.
func Text(e Entry) string {
	text := strings.TrimSpace(plainLinks(e.WhatsNew))
	return join(text, "Source: "+e.URL)
}

func join(first, second string) string {
	if strings.TrimSpace(first) == "" {
		return second
	}
	return first + "\n\n" + second
}

// capped keeps a hint within the limit, cutting the prose and never the
// "Source:" line at the end.
func capped(hint string) string {
	if utf8.RuneCountInString(hint) <= hintLimit {
		return hint
	}
	i := strings.LastIndex(hint, "\n\nSource: ")
	if i < 0 {
		return string([]rune(hint)[:hintLimit-1]) + "…"
	}
	source := hint[i:]
	room := hintLimit - utf8.RuneCountInString(source) - 1
	return string([]rune(hint[:i])[:room]) + "…" + source
}

// plainLinks turns "[text](url)" into "text".
func plainLinks(markdown string) string {
	return linkRE.ReplaceAllString(markdown, "$1")
}

var pathParam = regexp.MustCompile(`\{[^}]*\}`)

// NormalizeOperation makes "POST /v1/payment_intents/{intent}" comparable to
// "POST /v1/payment_intents/{id}": a path parameter's name is the spec's choice.
func NormalizeOperation(op string) string {
	return pathParam.ReplaceAllString(strings.TrimSpace(op), "{}")
}

// path is a parameter path as both sides write it, "[]" dropped from every
// segment: "payment_settings.payment_method_types[]" ->
// "payment_settings.payment_method_types".
func path(property string) string {
	return strings.ReplaceAll(strings.TrimSpace(property), "[]", "")
}

func key(operation, schema, parameter string) string {
	return operation + "\x00" + schema + "\x00" + parameter
}

var (
	titleLine    = regexp.MustCompile(`(?m)^# (.+)$`)
	whatsNewRE   = regexp.MustCompile(`(?m)^## What.s new\s*$`)
	sectionRE    = regexp.MustCompile(`(?m)^## `)
	restRE       = regexp.MustCompile(`(?m)^#### REST API\s*$`)
	subheadRE    = regexp.MustCompile(`(?m)^#### `)
	linkRE       = regexp.MustCompile(`\[([^\]]*)\]\(([^)\s]+)\)`)
	tableRowRE   = regexp.MustCompile(`(?m)^\|(.+)\|\s*$`)
	backtickedRE = regexp.MustCompile("`([^`]*)`")
)

// Resolve turns one reference of a table row into where it applies: the
// endpoints or objects (`ref` is the text, e.g. "Subscription#update" or
// "Subscription#update.trial_settings.end_behavior"; `href` its link, empty
// for a bare reference) and the parent path the reference carries. Each
// vendor's docs name endpoints their own way; the Source supplies it.
type Resolve func(ref, href string) (targets []Target, parent string, ok bool)

// Parse reads one entry page: its title, its "What's new" section and the rows
// of its REST API tables. A row names a parameter in its first cell, either a
// bare name (`payment_method_types`) with the places it applies linked in the
// last cell, or the full reference (`Subscription#update.billing_cycle_anchor`)
// with no link at all. A row with no parameter (a resource or an endpoint
// added) is skipped. A page with no REST API table parses to an entry with no
// rows (a Stripe.js change).
func Parse(url string, markdown string, resolve Resolve) (Entry, error) {
	e := Entry{URL: url}
	if m := titleLine.FindStringSubmatch(markdown); m != nil {
		e.Title = strings.TrimSpace(m[1])
	}
	if e.Title == "" {
		return Entry{}, fmt.Errorf("%s: no title", url)
	}
	e.WhatsNew = section(markdown, whatsNewRE, sectionRE)
	table := section(markdown, restRE, subheadRE)
	for _, m := range tableRowRE.FindAllStringSubmatch(table, -1) {
		cells := strings.Split(m[1], "|")
		if len(cells) < 3 {
			continue
		}
		first := strings.TrimSpace(cells[0])
		bt := backtickedRE.FindStringSubmatch(first)
		if bt == nil || strings.HasPrefix(first, "---") {
			continue // a header, a separator, or a row with no parameter
		}
		name := strings.TrimSpace(bt[1])
		row := Row{Change: strings.TrimSpace(cells[1])}
		if strings.Contains(name, "#") || isReference(name) {
			// The whole reference in one cell: the parameter is its last segment.
			targets, parent, ok := resolve(name, "")
			if !ok || parent == "" {
				continue
			}
			row.Parameter = parent[strings.LastIndex(parent, ".")+1:]
			for _, t := range targets {
				t.Path = parent
				row.Targets = append(row.Targets, t)
			}
		} else {
			row.Parameter = name
			for _, l := range linkRE.FindAllStringSubmatch(cells[len(cells)-1], -1) {
				targets, parent, ok := resolve(l[1], l[2])
				if !ok {
					continue
				}
				for _, t := range targets {
					t.Path = name
					if parent != "" {
						t.Path = parent + "." + name
					}
					row.Targets = append(row.Targets, t)
				}
			}
		}
		if row.Parameter != "" && len(row.Targets) > 0 {
			e.Rows = append(e.Rows, row)
		}
	}
	return e, nil
}

// isReference: "Subscription.trial_settings.end_behavior" names an object and
// a path; a bare parameter is all lowercase.
func isReference(name string) bool {
	first, _, found := strings.Cut(name, ".")
	return found && first != "" && first[0] >= 'A' && first[0] <= 'Z'
}

// section is the text after the first line matching start, up to the next
// line matching end (or the end of the document).
func section(markdown string, start, end *regexp.Regexp) string {
	loc := start.FindStringIndex(markdown)
	if loc == nil {
		return ""
	}
	rest := markdown[loc[1]:]
	if stop := end.FindStringIndex(rest); stop != nil {
		rest = rest[:stop[0]]
	}
	return strings.TrimSpace(rest)
}
