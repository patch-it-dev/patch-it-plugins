package catalog

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
)

// CompatFlags is a runtime's compatibility flags declared in a Cap'n Proto
// file in its repo, read at each commit that changed it: Cloudflare
// workerd's src/workerd/io/compatibility-date.capnp. Each flag is a Bool
// field of a struct, annotated with its name as configuration writes it
// ($compatEnableFlag("temporal")), its opt-out name ($compatDisableFlag)
// and, when it turns on by default, the compatibility date it does so from
// ($compatEnableDate("2026-12-01")).
//
// A version is a commit (its date, its SHA). A flag in to's file with a
// default date that from's file didn't have with that date is a newly
// announced default: a breaking flag event whose version is that date (a
// Worker whose compatibility_date is on or after it gets the new behavior),
// a dated notice with no previous_version. A date before to's commit isn't
// an announcement (a flag dated in the past when it lands, or a branch
// merged late), and an experimental flag ($experimental, usable only with
// workerd's --experimental) isn't one either.
type CompatFlags struct {
	API      string // the source id the events carry: "cloudflare/workers-compat-flags"
	Vendor   string // for sentences: "Cloudflare Workers"
	Provider string // the flag subject's provider: "cloudflare"
	Setting  string // where configuration sets a flag: "compatibility_flags"
	Repo     string // owner/name on GitHub
	Path     string // the file's path in the repo

	shape *plugin.Shape // the source's, set by RegisterProvider (WithShape); nil: no guard
}

// CompatFlagsMinRatio is the shape guard's ratio for a flags file: flags are never removed
// from the file (a retired one keeps its field number), so a read with
// under 90% of the last one's dated flags changed shape.
const CompatFlagsMinRatio = 0.9

func (c CompatFlags) Name() string { return c.API }

// WithShape is c guarding its pairs with s.
func (c CompatFlags) WithShape(s plugin.Shape) plugin.Plugin {
	c.shape = &s
	return c
}

// FileURL is the file at a commit: its own provenance.
func (c CompatFlags) FileURL(sha string) string {
	return fmt.Sprintf("%s/%s/%s/%s", github.Raw, c.Repo, sha, c.Path)
}

// ListVersions is every commit that changed the file, oldest first.
func (c CompatFlags) ListVersions(ctx context.Context, env plugin.Env) ([]plugin.Version, error) {
	return versions.GitHubCommits(ctx, env.Get, c.Repo, c.Path)
}

// ReadAt fetches the file at v's commit.
func (c CompatFlags) ReadAt(ctx context.Context, env plugin.Env, v plugin.Version) (plugin.Snapshot, error) {
	url := c.FileURL(v.Ref)
	body, err := env.Get(ctx, url)
	if err != nil {
		return plugin.Snapshot{}, fmt.Errorf("fetching %s: %w", url, err)
	}
	return plugin.Snapshot{Version: v, Content: body, SourceURL: url}, nil
}

// Flag is one flag with a default date.
type Flag struct {
	Name    string // as compatibility_flags writes it: "temporal"
	Date    string // the compatibility date it turns on from, YYYY-MM-DD
	OptOut  string // its $compatDisableFlag name ("no_temporal"), or ""
	Excerpt string // the field and its annotations, as the file writes them
}

var (
	// boolField is a field of the struct and its annotations, up to the ";".
	boolField   = regexp.MustCompile(`(\w+)\s+@\d+\s*:Bool((?:\s*\$\w+(?:\([^)]*\))?)*)\s*;`)
	enableFlag  = regexp.MustCompile(`\$compatEnableFlag\("([^"]+)"\)`)
	disableFlag = regexp.MustCompile(`\$compatDisableFlag\("([^"]+)"\)`)
	enableDate  = regexp.MustCompile(`\$compatEnableDate\("(\d{4}-\d{2}-\d{2})"\)`)
	isExperim   = regexp.MustCompile(`\$experimental\b`)
)

// Flags reads every non-experimental flag with a default date.
func Flags(file []byte) []Flag {
	var out []Flag
	for _, m := range boolField.FindAllStringSubmatch(string(file), -1) {
		name, date := enableFlag.FindStringSubmatch(m[2]), enableDate.FindStringSubmatch(m[2])
		if name == nil || date == nil || isExperim.MatchString(m[2]) {
			continue
		}
		f := Flag{Name: name[1], Date: date[1], Excerpt: strings.Join(strings.Fields(m[0]), " ")}
		if off := disableFlag.FindStringSubmatch(m[2]); off != nil {
			f.OptOut = off[1]
		}
		out = append(out, f)
	}
	return out
}

// Extract is a flag event per flag in to's file with a default date from's
// didn't have, dated on or after to's commit (see CompatFlags). A zero from
// has nothing to compare.
func (c CompatFlags) Extract(_ context.Context, env plugin.Env, from, to plugin.Snapshot) (plugin.Result, error) {
	if from.IsZero() {
		return plugin.Result{}, nil
	}
	before, after := Flags(from.Content), Flags(to.Content)
	if len(after) == 0 {
		return plugin.Result{}, fmt.Errorf("reading %s: no flag with a default date: %w", to.SourceURL, plugin.ErrShapeChanged)
	}
	if err := c.shape.CheckCount("dated flags", CompatFlagsMinRatio, len(before), len(after)); err != nil {
		return plugin.Result{}, fmt.Errorf("reading %s: %w", to.SourceURL, err)
	}
	had := map[string]bool{}
	for _, f := range before {
		had[f.Name+"@"+f.Date] = true
	}
	var events []event.Event
	var rejected []error
	for _, f := range after {
		if had[f.Name+"@"+f.Date] || f.Date < to.Version.ID {
			continue
		}
		hint := fmt.Sprintf("Check the code against the new behavior before moving its compatibility date to %s or later.", f.Date)
		if f.OptOut != "" {
			hint += fmt.Sprintf(" To move the date without it, add %q to %s.", f.OptOut, c.Setting)
		}
		subject := event.Subject{Type: event.SubjectFlag, Provider: c.Provider, Setting: c.Setting, Name: f.Name}
		e := event.Event{
			Fingerprint: event.Fingerprint(c.API, f.Date, event.ChangeBreaking, subject, ""),
			API:         c.API, Version: f.Date, Change: event.ChangeBreaking, Subject: subject,
			Description:   fmt.Sprintf("%s turn on %s by default from compatibility date %s: code whose compatibility date is on or after it gets the new behavior.", c.Vendor, f.Name, f.Date),
			MigrationHint: hint,
			SourceURL:     to.SourceURL,
			SourceExcerpt: f.Excerpt,
			Confidence:    1.0,
			DetectedAt:    env.Now().UTC(),
		}
		if err := e.Validate(); err != nil {
			rejected = append(rejected, fmt.Errorf("%s %s: %w", c.API, f.Name, err))
			continue
		}
		events = append(events, e)
	}
	return plugin.Result{Events: events, Rejected: rejected}, nil
}

var (
	_ plugin.Versioned = CompatFlags{}
	_ plugin.Guarded   = CompatFlags{}
)
