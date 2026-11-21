// Package members is the migration-guide kind's reader for a package release
// that removed members of its top-level module, read from the release's own
// migration guide at its tag: change event v2's `package_release` subject.
// The kind (kinds/migrationguide) reads the guide and calls Read; this
// package knows where the guide is and how its table reads. A guide is read
// at the first release of each major (X.0.0), listed from PyPI: nothing in
// the code names a release. NumPy first: its 2.0 guide lists about 60
// members of the main `np` namespace that were removed, each with a
// migration guideline, and code using one raises AttributeError on NumPy 2
// (checked 2026-11-04 at tag v2.0.0).
//
// A row whose guideline is exactly one replacement in the module ("Use
// “np.float64“ instead.") becomes a `renamed_symbol` with that replacement:
// the rename works on NumPy 1.x too, so a fix holds before and after the
// upgrade. Every other row (no replacement, a builtin, an expression, a
// sentence of advice) becomes a `removed_symbol`, which the patch-it client
// opens as an issue: precision over recall.
//
// A guide's table is an RST simple table (NumPy) or a Markdown pipe table
// (PostHog, New Relic, markdown.go). A Markdown row may list several members in one cell;
// each is its own event, as each NumPy row is. A row the reader can't read as
// members of the package (prose, an argument value, an environment variable)
// is dropped and logged, never guessed at.
package members

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
)

// Guide is one package's guide to each of its majors.
type Guide struct {
	API     string // the source id the events carry: "numpy/numpy"
	Package string // the PyPI name: "numpy"
	Module  string // how the guide writes the module in a replacement: "np"
	Repo    string // owner/name on GitHub
	// Path is the guide's path in the repo for a major, "{major}" standing
	// for the major's number and "{prev}" for the one before:
	// "doc/source/numpy_{major}_0_migration_guide.rst" is
	// doc/source/numpy_2_0_migration_guide.rst for 2.0.0.
	Path    string
	Section string // the heading the table is under

	// TagPrefix makes a release's git tag: TagPrefix + version ("" is "v":
	// v2.0.0; PostHog's is "posthog-v"). A major's guide is read at the tag
	// of its first release.
	TagPrefix string
	// Untagged is a guide that isn't in the package's own repo but in a docs
	// repo with no release tags (New Relic's release notes): it is read at
	// the newest commit that changed its path, looked up when it's read.
	Untagged bool
	// Format is how the table is written: FormatRST (the default) or
	// FormatMarkdown.
	Format Format
	// Header is the table's first column heading, matched case-insensitively
	// (as a prefix in an RST table); "" is "removed member".
	Header string
	// Effect is what code using any removed member gets, said in every
	// event's description after "<Module>.<member> was removed": only for a
	// guide whose members are all attributes of Module and all fail the same
	// way. "" says only that it was removed.
	Effect string
	// EnvVars are names in the table that are environment variables, not
	// members of the package: no code imports one, so they're dropped. Listed
	// by hand from the code, not guessed from a prefix (a package can have
	// constants named like its environment variables).
	EnvVars []string
	// NewReplacements says the release's replacements are new in it: a
	// rename fix wouldn't run on the release before, and may need a different
	// import. A row with one replacement is then a removed_symbol, with the
	// replacement in the migration hint (the guide's own cell), not a
	// renamed_symbol the client would apply as a rename. NumPy 2.0's
	// replacements already worked on 1.x, so it doesn't set this.
	NewReplacements bool
	// Paths gives a name the table writes in shorthand the path code imports
	// it by, after Module ("EVENTS_ENDPOINT" -> "request.EVENTS_ENDPOINT"),
	// checked by hand against the package at the release before. A guide's
	// cell can list names that share a prefix only the first one spells out.
	// It applies to a removed name and to a replacement alike.
	Paths map[string]string
	// DropIns, when set, are the only removed members (as the events name
	// them, after Paths) whose one replacement is a drop-in: it existed in the
	// release before, at the same path, with the same call signature and
	// behavior, checked by hand against the code. Only they become a
	// renamed_symbol, which the client applies as a rename; every other row
	// with one replacement is a removed_symbol, the replacement in the
	// migration hint (the guide's own cell). nil leaves it to the table: every
	// row with one replacement is a rename (NumPy's guide says each works on
	// 1.x), unless NewReplacements.
	DropIns []string
	// Doc is what the hint calls the source: "<Package>'s <Doc> for
	// <Release>". "" is "migration guide".
	Doc string
	// LinkBase makes the guide's site-relative links absolute in the hint
	// ("](/docs/…)" -> "](<LinkBase>/docs/…)"): the hint is shown outside the
	// vendor's site, in a pull request or an issue, where a relative link
	// breaks. "" leaves links as written.
	LinkBase string
	// StillWork are names the table lists as removed but the same guide says
	// still work or have no effect: no event, with the guide's reason.
	StillWork map[string]string
}

// Format is how a guide writes its table.
type Format string

const (
	FormatRST      Format = ""
	FormatMarkdown Format = "markdown"
)

// TagOf is the git tag of a release of the package.
func (g Guide) TagOf(release string) string {
	if g.TagPrefix != "" {
		return g.TagPrefix + release
	}
	return "v" + release
}

func (g Guide) doc() string {
	if g.Doc != "" {
		return g.Doc
	}
	return "migration guide"
}

func (g Guide) header() string {
	if g.Header != "" {
		return g.Header
	}
	return "removed member"
}

// PathAt is the guide's path for a major.
func (g Guide) PathAt(major int) string {
	return strings.NewReplacer("{major}", strconv.Itoa(major), "{prev}", strconv.Itoa(major-1)).Replace(g.Path)
}

// CommitsURL lists the newest commit that changed a major's guide.
func (g Guide) CommitsURL(major int) string {
	return fmt.Sprintf("%s/repos/%s/commits?path=%s&per_page=1", github.API, g.Repo, url.QueryEscape(g.PathAt(major)))
}

// RawURL is the guide for a major at a git ref (a tag or a commit): its own
// provenance, as it never changes.
func (g Guide) RawURL(ref string, major int) string {
	return fmt.Sprintf("%s/%s/%s/%s", github.Raw, g.Repo, ref, g.PathAt(major))
}

// Removal is one removed member and the guide's guideline for it.
type Removal struct {
	Member      string
	Guideline   string
	Replacement string // the one member to use instead, or ""
	Excerpt     string // the table's header and the row
	Dropped     string // why it isn't an event; "" when it is
}

// Parse reads the section's table: one Removal per member, and one per row or
// name it drops, with the reason in Dropped.
func Parse(page []byte, g Guide) ([]Removal, error) {
	if g.Format == FormatMarkdown {
		return parseMarkdown(page, g)
	}
	return parseRST(page, g)
}

// parseRST reads an RST simple table, the member in the first column, the
// guideline in the second, a guideline continued on the next lines indented to
// the second column.
func parseRST(page []byte, g Guide) ([]Removal, error) {
	lines := strings.Split(strings.ReplaceAll(string(page), "\r\n", "\n"), "\n")
	start := -1
	for i, l := range lines {
		if strings.TrimSpace(l) == g.Section && i+1 < len(lines) && strings.HasPrefix(lines[i+1], "---") {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("no %q section (the guide's shape may have changed): %w", g.Section, plugin.ErrNothingToRead)
	}
	border := -1
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "====") && strings.Contains(lines[i], " ") {
			border = i
			break
		}
	}
	if border < 0 || border+2 >= len(lines) {
		return nil, fmt.Errorf("no table under %q: %w", g.Section, plugin.ErrNothingToRead)
	}
	col := strings.Index(lines[border], " ") + 1
	for col < len(lines[border]) && lines[border][col] == ' ' {
		col++
	}
	header := lines[border+1]
	if !strings.HasPrefix(strings.ToLower(header), strings.ToLower(g.header())) {
		return nil, fmt.Errorf("the table under %q isn't the removed members: %q", g.Section, header)
	}
	head := strings.Join([]string{lines[border], header, lines[border+2]}, "\n")
	var out []Removal
	var raw []string
	for i := border + 3; i < len(lines) && !strings.HasPrefix(lines[i], "===="); i++ {
		l := lines[i]
		if len(l) > col && strings.TrimSpace(l[:col]) == "" && len(out) > 0 {
			out[len(out)-1].Guideline += " " + strings.TrimSpace(l)
			raw[len(raw)-1] += "\n" + l
			continue
		}
		if strings.TrimSpace(l) == "" {
			continue
		}
		member, guide := l, ""
		if len(l) > col {
			member, guide = l[:col], l[col:]
		}
		out = append(out, Removal{
			Member:    strings.ReplaceAll(strings.TrimSpace(member), `\_`, "_"),
			Guideline: strings.TrimSpace(guide),
		})
		raw = append(raw, l)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("the removed members table under %q is empty: %w", g.Section, plugin.ErrNothingToRead)
	}
	plain := regexp.MustCompile("^Use ``" + regexp.QuoteMeta(g.Module) + `\.([A-Za-z_][A-Za-z0-9_]*)` + "`` instead\\.$")
	for i := range out {
		if m := plain.FindStringSubmatch(out[i].Guideline); m != nil {
			out[i].Replacement = m[1]
		}
		out[i].Excerpt = g.Section + "\n" + head + "\n" + raw[i]
	}
	return out, nil
}

// Slug is the source id the events carry: "numpy/numpy".
func (g Guide) Slug() string { return g.API }

// Releases is every final release of the package on PyPI, yanked ones
// included, oldest first, each with its tag. A yanked release stays: it can
// be the one a guide documents (New Relic's 11.0.0, yanked for a
// server-side configuration issue), and a pin on a later one still crosses
// it.
func (g Guide) Releases(ctx context.Context, env plugin.Env) ([]plugin.Version, error) {
	vs, err := versions.PyPIReleasesWithYanked(ctx, env.Get, g.Package)
	if err != nil {
		return nil, err
	}
	for i := range vs {
		vs[i].Ref = g.TagOf(vs[i].ID)
	}
	return vs, nil
}

// GuideURL is the guide for v's major when v is its first release (X.0.0),
// at v's tag or, Untagged, at the newest commit that changed the guide's
// path; any other release has no guide (""). A major whose guide's path has
// no commit has nothing to read (plugin.ErrNothingToRead).
func (g Guide) GuideURL(ctx context.Context, env plugin.Env, v plugin.Version) (string, error) {
	if !versions.FirstOfMajor(v.ID) {
		return "", nil
	}
	major := versions.Major(v.ID)
	if !g.Untagged {
		return g.RawURL(g.TagOf(v.ID), major), nil
	}
	sha, err := github.LatestCommit(ctx, env.Get, g.CommitsURL(major))
	if err != nil {
		return "", fmt.Errorf("the commit %s's guide for %s is read at: %w", g.Package, v.ID, err)
	}
	return g.RawURL(sha, major), nil
}

// Read is an event per removed member in the guide to's release documents,
// read from sourceURL, with previous_version from's release.
func (g Guide) Read(page []byte, sourceURL string, from, to plugin.Version, now time.Time) (events []event.Event, rejected []error, err error) {
	removals, err := Parse(page, g)
	if err != nil {
		return nil, nil, err
	}
	for _, r := range removals {
		if r.Dropped != "" {
			log.Printf("%s %s: dropped %q: %s", g.Package, to.ID, r.Member, r.Dropped)
			continue
		}
		subject := event.Subject{
			Type: event.SubjectPackageRelease, Ecosystem: "pypi", Package: g.Package,
			Change: event.PackageRemovedSymbol, Symbols: []string{r.Member},
		}
		desc := fmt.Sprintf("`%s` was removed in %s %s.", r.Member, g.Package, to.ID)
		if g.Effect != "" {
			desc = fmt.Sprintf("%s.%s was removed in %s %s: %s.", g.Module, r.Member, g.Package, to.ID, g.Effect)
		}
		if r.Replacement != "" && !g.NewReplacements && (g.DropIns == nil || slices.Contains(g.DropIns, r.Member)) {
			subject.Change, subject.Replacement = event.PackageRenamedSymbol, r.Replacement
		}
		e := event.Event{
			API:             g.API,
			PreviousVersion: from.ID,
			Version:         to.ID,
			ChangeType:      event.ChangeTypeBreaking,
			Breaking:        true,
			Subject:         subject,
			Description:     desc,
			MigrationHint:   fmt.Sprintf("%s (%s's %s for %s)", g.absoluteLinks(r.Guideline), g.Package, g.doc(), to.ID),
			SourceURL:       sourceURL,
			SourceExcerpt:   r.Excerpt,
			Confidence:      1.0,
			DetectedAt:      now,
		}
		e.Fingerprint = event.Fingerprint(e.API, e.Version, e.ChangeType, subject, "")
		if err := e.Validate(); err != nil {
			rejected = append(rejected, fmt.Errorf("%s %s: %w", g.Package, r.Member, err))
			continue
		}
		events = append(events, e)
	}
	return events, rejected, nil
}

// absoluteLinks prefixes LinkBase to every Markdown link target in s that
// starts with "/".
func (g Guide) absoluteLinks(s string) string {
	if g.LinkBase == "" {
		return s
	}
	return strings.ReplaceAll(s, "](/", "]("+strings.TrimSuffix(g.LinkBase, "/")+"/")
}
