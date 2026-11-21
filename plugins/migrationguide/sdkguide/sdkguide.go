// Package sdkguide reads an SDK's own migration guide for the request parameters
// a major release removed: a third kind of source, next to the OpenAPI differ and
// the deprecations pages. The guide is read at the release tag of the major it
// describes (`v1.0.0` for "Migrating to v1"), a URL that never changes, so it is
// its own provenance.
//
// The first is anthropic-sdk-python 1.0.0 (2026-08-20): its MIGRATION.md lists
// `temperature`, `top_p` and `top_k` as removed from `messages.create()` and its
// siblings, passing them is a `TypeError`, and it says what to do instead (checked
// 2026-10-17). A repo with `anthropic` unpinned gets 1.x on a fresh install.
package sdkguide

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// Guide is one SDK's migration guide for one major release.
type Guide struct {
	API     string // the source id the events carry: "anthropic/python-sdk"
	Package string // its PyPI name: "anthropic"
	Repo    string // its GitHub repository, owner/name
	Major   int    // pinned: the release the guide describes, 1 for "Migrating to v1"; 0 lists them
	// Methods is what a table row's method cell must name for its parameters to
	// count: the plain call, so a row about beta-only methods (where the same
	// name can still be valid on a helper) isn't read as a removal.
	Methods string
	// Symbols are the calls the removed parameters can no longer be passed to, as
	// code writes them after the import: the plain methods the guide's table names
	// first (`messages.create()`, `messages.stream()`, `messages.parse()`).
	Symbols []string
}

// Anthropic is anthropic-sdk-python's MIGRATION.md, unpinned: read at the
// first release of each major (Major 0; At pins it).
var Anthropic = Guide{
	API:     "anthropic/python-sdk",
	Package: "anthropic",
	Repo:    "anthropics/anthropic-sdk-python",
	Methods: "`messages.create()`",
	Symbols: []string{"messages.create", "messages.stream", "messages.parse"},
}

// AnthropicV1 is anthropic-sdk-python's guide to 1.0.0: what the golden pins.
var AnthropicV1 = Anthropic.At(1)

// At is g pinned to one major.
func (g Guide) At(major int) Guide {
	g.Major = major
	return g
}

// Version is the release that removed the parameters.
func (g Guide) Version() string { return fmt.Sprintf("%d.0.0", g.Major) }

// URL is the guide as it was at that release's tag.
func (g Guide) URL() string {
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/v%s/MIGRATION.md", g.Repo, g.Version())
}

// Page is the same guide for a person to read.
func (g Guide) Page() string {
	return fmt.Sprintf("https://github.com/%s/blob/v%s/MIGRATION.md", g.Repo, g.Version())
}

// Removal is one parameter the release removed, with what the guide says to do.
type Removal struct {
	Param   string
	Instead string // the guide's "use instead" cell, as written
	Excerpt string // the section heading, the table header and the row
}

var ticked = regexp.MustCompile("`([^`]+)`")

// Parse reads the guide's removed-parameter tables: every row whose method cell
// names g.Methods gives one Removal per backticked parameter. A guide for another
// major, or one with no such row, is an error, never zero removals.
func Parse(page []byte, g Guide) ([]Removal, error) {
	text := strings.ReplaceAll(string(page), "\r\n", "\n")
	if !strings.Contains(text, fmt.Sprintf("# Migrating to v%d", g.Major)) {
		return nil, fmt.Errorf("not the guide to v%d: %w", g.Major, plugin.ErrNothingToRead)
	}
	lines := strings.Split(text, "\n")
	var out []Removal
	heading := ""
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "#") {
			heading = line
			continue
		}
		header := cells(line)
		param, instead := column(header, "removed parameter"), column(header, "instead")
		if param < 0 || instead < 0 {
			continue
		}
		i++ // the |---| row
		for i+1 < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i+1]), "|") {
			i++
			row := strings.TrimSpace(lines[i])
			c := cells(row)
			if len(c) <= param || len(c) <= instead || !strings.Contains(c[0], g.Methods) {
				continue
			}
			excerpt := cut(strings.Join([]string{heading, line, row}, "\n"), 2000)
			for _, m := range ticked.FindAllStringSubmatch(c[param], -1) {
				out = append(out, Removal{Param: m[1], Instead: c[instead], Excerpt: excerpt})
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no removed-parameter row naming %s (the guide's shape may have changed): %w", g.Methods, plugin.ErrNothingToRead)
	}
	return out, nil
}

func cells(row string) []string {
	if !strings.HasPrefix(row, "|") {
		return nil
	}
	parts := strings.Split(strings.Trim(row, "|"), "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func column(header []string, word string) int {
	for i, h := range header {
		if strings.Contains(strings.ToLower(h), word) {
			return i
		}
	}
	return -1
}

func cut(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}

// Events turns removals into change events: one breaking removed_parameter
// event per parameter, on the SDK calls it can no longer be passed to, with
// previous (the release before) as previous_version.
func Events(g Guide, rs []Removal, previous string, now time.Time) ([]event.Event, []error) {
	var events []event.Event
	var errs []error
	for _, r := range rs {
		subject := event.Subject{
			Type: event.SubjectPackageRelease, Ecosystem: "pypi", Package: g.Package,
			Change: event.PackageRemovedParameter, Parameter: r.Param, Symbols: g.Symbols,
		}
		e := event.Event{
			Fingerprint:     event.Fingerprint(g.API, g.Version(), event.ChangeTypeBreaking, subject, ""),
			API:             g.API,
			Version:         g.Version(),
			PreviousVersion: previous,
			ChangeType:      event.ChangeTypeBreaking,
			Breaking:        true,
			Subject:         subject,
			Description:     fmt.Sprintf("Parameter %s removed from %s and its sibling methods in the %s SDK %s: passing it raises a TypeError.", r.Param, g.Methods, g.Package, g.Version()),
			MigrationHint:   fmt.Sprintf("%s Source: %s", r.Instead, g.Page()),
			SourceURL:       g.URL(),
			SourceExcerpt:   r.Excerpt,
			Confidence:      1.0,
			DetectedAt:      now,
		}
		if err := e.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("%s %s: %w", g.API, r.Param, err))
			continue
		}
		events = append(events, e)
	}
	return events, errs
}
