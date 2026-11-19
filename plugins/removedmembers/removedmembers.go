// Package removedmembers is the source plugin for a package release that
// removed members of its top-level module, read from the release's own
// migration guide at its tag: change event v2's `package_release` subject
// . NumPy 2.0 first: its guide lists about 60 members of the main `np`
// namespace that were removed, each with a migration guideline, and code using
// one raises AttributeError on NumPy 2 (checked 2026-11-04 at tag v2.0.0).
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
package removedmembers

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"slices"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/event"
	"github.com/patch-it-dev/patch-it-plugins/plugin"
)

func init() {
	plugin.Register(Plugin{Guide: NumPy2})
	plugin.Register(Plugin{Guide: PostHog8})
	plugin.Register(Plugin{Guide: NewRelic11})
}

// Guide is one package's guide to one release.
type Guide struct {
	Package string // the PyPI name and the events' api: "numpy"
	Module  string // how the guide writes the module in a replacement: "np"
	Release string // "2.0.0"
	Repo    string // owner/name on GitHub
	Path    string // the guide's path in the repo at the release tag
	Section string // the heading the table is under

	// Tag is the release tag; "" is "v" + Release.
	Tag string
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

func (g Guide) tag() string {
	if g.Tag != "" {
		return g.Tag
	}
	return "v" + g.Release
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

// NumPy2 is NumPy's 2.0 migration guide, "Main namespace" table.
var NumPy2 = Guide{
	Package: "numpy",
	Module:  "np",
	Release: "2.0.0",
	Repo:    "numpy/numpy",
	Path:    "doc/source/numpy_2_0_migration_guide.rst",
	Section: "Main namespace",
	Effect:  "code using it raises AttributeError",
}

// URL is the guide as it was at the release tag: its own provenance.
func (g Guide) URL() string {
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", g.Repo, g.tag(), g.Path)
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
		return nil, fmt.Errorf("no %q section: the guide's shape may have changed", g.Section)
	}
	border := -1
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "====") && strings.Contains(lines[i], " ") {
			border = i
			break
		}
	}
	if border < 0 || border+2 >= len(lines) {
		return nil, fmt.Errorf("no table under %q", g.Section)
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
		return nil, fmt.Errorf("the removed members table under %q is empty", g.Section)
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

// Plugin reads one guide.
type Plugin struct {
	Guide Guide
}

func (p Plugin) Name() string { return p.Guide.Package }

// Run reads the guide at its tag. The tag never changes, so a rerun stores
// nothing new: the fingerprints dedupe.
func (p Plugin) Run(ctx context.Context, env plugin.Env) (plugin.Result, error) {
	g := p.Guide
	page, err := env.Get(ctx, g.URL())
	if err != nil {
		return plugin.Result{}, fmt.Errorf("fetching %s: %w", g.URL(), err)
	}
	removals, err := Parse(page, g)
	if err != nil {
		return plugin.Result{}, fmt.Errorf("reading %s: %w", g.URL(), err)
	}
	now := env.Now().UTC()
	var events []event.EventV2
	var rejected []error
	for _, r := range removals {
		if r.Dropped != "" {
			log.Printf("%s %s: dropped %q: %s", g.Package, g.Release, r.Member, r.Dropped)
			continue
		}
		subject := event.Subject{
			Type: event.SubjectPackageRelease, Ecosystem: "pypi", Package: g.Package,
			Change: event.PackageRemovedSymbol, Symbols: []string{r.Member},
		}
		desc := fmt.Sprintf("`%s` was removed in %s %s.", r.Member, g.Package, g.Release)
		if g.Effect != "" {
			desc = fmt.Sprintf("%s.%s was removed in %s %s: %s.", g.Module, r.Member, g.Package, g.Release, g.Effect)
		}
		if r.Replacement != "" && !g.NewReplacements && (g.DropIns == nil || slices.Contains(g.DropIns, r.Member)) {
			subject.Change, subject.Replacement = event.PackageRenamedSymbol, r.Replacement
		}
		e := event.EventV2{
			API:           g.Package,
			Version:       g.Release,
			ChangeType:    event.ChangeTypeBreaking,
			Breaking:      true,
			Subject:       subject,
			Description:   desc,
			MigrationHint: fmt.Sprintf("%s (%s's %s for %s)", r.Guideline, g.Package, g.doc(), g.Release),
			SourceURL:     g.URL(),
			SourceExcerpt: r.Excerpt,
			Confidence:    1.0,
			DetectedAt:    now,
		}
		e.Fingerprint = event.FingerprintV2(e.API, e.Version, e.ChangeType, subject, "")
		if err := e.Validate(); err != nil {
			rejected = append(rejected, fmt.Errorf("%s %s: %w", g.Package, r.Member, err))
			continue
		}
		events = append(events, e)
	}
	return plugin.Result{
		Label:     g.URL(),
		Events:    events,
		Rejected:  rejected,
		Baselines: []plugin.Spec{plugin.Baseline(g.Package, g.Release, g.tag(), g.URL(), page, false, now)},
	}, nil
}
