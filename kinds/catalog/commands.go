package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
)

// Commands is a server's command catalog kept as one JSON file per command
// in a directory of its own repo, read at each release tag: Redis's
// src/commands/, which the server's command table (and redis.io's command
// reference) is generated from. Each file is one object, keyed by the
// command's name, a subcommand naming its parent as "container"
// ({"KILL": {"container": "CLIENT", ...}} is CLIENT KILL).
//
// A version is a release: its tag lists the directory (one GitHub API
// call), and a command whose file is gone from the next release's listing
// is removed; one whose file is new in it is added. Only a removed or added
// file is fetched, at the tag that has it, for the command's name and the
// entry quoted as evidence. A name that leaves in one file and arrives in
// another is a file moved: neither.
type Commands struct {
	API       string // the source id the events carry: "redis/commands"
	Ecosystem string // the package_release subject's ecosystem: "redis"
	Package   string // the package_release subject's package: "redis"
	Repo      string // owner/name on GitHub
	Dir       string // the directory of command files in the repo
	// Release matches a final release's tag; its first submatch is the
	// version.
	Release *regexp.Regexp

	shape *plugin.Shape // the source's, set by RegisterProvider (WithShape); nil: no guard
}

// CommandsMinRatio is the shape guard's ratio for a command catalog: a
// release that keeps fewer than 80% of the command files of the one before
// moved or reorganized the directory; read as removals, it would be most of
// the catalog.
const CommandsMinRatio = 0.8

func (c Commands) Name() string { return c.API }

// WithShape is c guarding its pairs with s.
func (c Commands) WithShape(s plugin.Shape) plugin.Plugin {
	c.shape = &s
	return c
}

// ListingURL is GitHub's listing of the directory at a ref.
func (c Commands) ListingURL(ref string) string {
	return fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", github.API, c.Repo, c.Dir, url.QueryEscape(ref))
}

// FileURL is one command's file at a ref: its own provenance, as a tag
// never changes.
func (c Commands) FileURL(ref, file string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", github.Raw, c.Repo, ref, c.Dir, file)
}

// ListVersions is every final release on GitHub, ordered by version, not by
// publication date: Redis patches older lines after a newer one ships
// (7.2.10 after 8.0.0), and a pair across lines would read the older line's
// missing commands as removals.
func (c Commands) ListVersions(ctx context.Context, env plugin.Env) ([]plugin.Version, error) {
	vs, err := versions.GitHubReleases(ctx, env.Get, c.Repo, c.Release.MatchString)
	if err != nil {
		return nil, err
	}
	for i := range vs {
		vs[i].ID = c.Release.FindStringSubmatch(vs[i].Ref)[1]
	}
	slices.SortStableFunc(vs, func(a, b plugin.Version) int { return versions.Compare(a.ID, b.ID) })
	return vs, nil
}

// ReadAt lists the command files at v's tag, as a sorted JSON array of file
// names. A release without the directory (before 7.0.0) has nothing to
// read.
func (c Commands) ReadAt(ctx context.Context, env plugin.Env, v plugin.Version) (plugin.Snapshot, error) {
	u := c.ListingURL(v.Ref)
	body, err := env.Get(ctx, u)
	var se *fetch.StatusError
	if errors.As(err, &se) && se.StatusCode == 404 {
		return plugin.Snapshot{}, fmt.Errorf("no %s at %s: %w", c.Dir, v.Ref, plugin.ErrNothingToRead)
	}
	if err != nil {
		return plugin.Snapshot{}, fmt.Errorf("listing %s at %s: %w", c.Dir, v.Ref, err)
	}
	var entries []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &entries); err != nil {
		return plugin.Snapshot{}, fmt.Errorf("parsing the listing of %s at %s: %w", c.Dir, v.Ref, err)
	}
	var files []string
	for _, e := range entries {
		if e.Type == "file" && strings.HasSuffix(e.Name, ".json") {
			files = append(files, e.Name)
		}
	}
	if len(files) == 0 {
		return plugin.Snapshot{}, fmt.Errorf("no command files in %s at %s: %w", c.Dir, v.Ref, plugin.ErrNothingToRead)
	}
	sort.Strings(files)
	content, _ := json.Marshal(files)
	return plugin.Snapshot{Version: v, Content: content, SourceURL: fmt.Sprintf("https://github.com/%s/tree/%s/%s", c.Repo, v.Ref, c.Dir)}, nil
}

// Extract is a breaking event per command whose file is in from's listing
// and not in to's, citing the file at from's tag, and an added one per
// command whose file is in to's and not in from's, citing it at to's tag,
// with previous_version from's release. A zero from has nothing to compare.
func (c Commands) Extract(ctx context.Context, env plugin.Env, from, to plugin.Snapshot) (plugin.Result, error) {
	if from.IsZero() {
		return plugin.Result{}, nil
	}
	var before, after []string
	if err := json.Unmarshal(from.Content, &before); err != nil {
		return plugin.Result{}, fmt.Errorf("reading %s: %w", from.SourceURL, err)
	}
	if err := json.Unmarshal(to.Content, &after); err != nil {
		return plugin.Result{}, fmt.Errorf("reading %s: %w", to.SourceURL, err)
	}
	if err := c.shape.CheckCount("command files", CommandsMinRatio, len(before), len(after)); err != nil {
		return plugin.Result{}, fmt.Errorf("reading %s: %w", to.SourceURL, err)
	}
	type found struct{ name, entry, url string }
	read := func(ref string, files, other []string) ([]found, error) {
		var out []found
		for _, file := range files {
			if slices.Contains(other, file) {
				continue
			}
			u := c.FileURL(ref, file)
			body, err := env.Get(ctx, u)
			if err != nil {
				return nil, fmt.Errorf("fetching %s: %w", u, err)
			}
			name, entry, err := command(body)
			if err != nil {
				return nil, fmt.Errorf("reading %s: %w", u, err)
			}
			out = append(out, found{name, entry, u})
		}
		return out, nil
	}
	gone, err := read(from.Version.Ref, before, after)
	if err != nil {
		return plugin.Result{}, err
	}
	added, err := read(to.Version.Ref, after, before)
	if err != nil {
		return plugin.Result{}, err
	}
	moved := map[string]bool{}
	for _, g := range gone {
		for _, a := range added {
			if g.name == a.name {
				moved[g.name] = true
			}
		}
	}
	var made []event.Event
	for _, f := range gone {
		if moved[f.name] {
			continue
		}
		name, entry, u := f.name, f.entry, f.url
		made = append(made, event.Event{
			API: c.API, Version: to.Version.ID, PreviousVersion: from.Version.ID,
			Change: event.ChangeBreaking,
			Subject: event.Subject{
				Type: event.SubjectPackageRelease, Ecosystem: c.Ecosystem, Package: c.Package,
				Change: event.PackageRemovedSymbol, Symbols: []string{name},
			},
			Description:   fmt.Sprintf("%s is gone in Redis %s (it was in %s): calling it fails.", name, to.Version.ID, from.Version.ID),
			SourceURL:     u,
			SourceExcerpt: excerpt(entry),
			Confidence:    1.0,
			DetectedAt:    env.Now().UTC(),
		})
	}
	for _, f := range added {
		if moved[f.name] {
			continue
		}
		made = append(made, event.Event{
			API: c.API, Version: to.Version.ID, PreviousVersion: from.Version.ID,
			Change: event.ChangeAdded,
			Subject: event.Subject{
				Type: event.SubjectPackageRelease, Ecosystem: c.Ecosystem, Package: c.Package,
				Change: event.PackageAddedSymbol, Symbols: []string{f.name},
			},
			Description:   fmt.Sprintf("%s is new in Redis %s (it wasn't in %s).", f.name, to.Version.ID, from.Version.ID),
			SourceURL:     f.url,
			SourceExcerpt: excerpt(f.entry),
			Confidence:    1.0,
			DetectedAt:    env.Now().UTC(),
		})
	}
	events, rejected := finish(c.API, made, nil)
	return plugin.Result{Events: events, Rejected: rejected}, nil
}

// command reads one command file: its name ("CLIENT KILL" for a subcommand)
// and the file's entry, quoted as the file writes it, whitespace between
// tokens left out.
func command(body []byte) (name, entry string, err error) {
	var file map[string]json.RawMessage
	if err := json.Unmarshal(body, &file); err != nil {
		return "", "", err
	}
	if len(file) != 1 {
		return "", "", fmt.Errorf("%d commands in one file, want 1", len(file))
	}
	for key, raw := range file {
		var meta struct {
			Container string `json:"container"`
		}
		if err := json.Unmarshal(raw, &meta); err != nil {
			return "", "", err
		}
		name = key
		if meta.Container != "" {
			name = meta.Container + " " + key
		}
		var compact bytes.Buffer
		if err := json.Compact(&compact, body); err != nil {
			return "", "", err
		}
		entry = compact.String()
	}
	return name, entry, nil
}

var (
	_ plugin.Versioned = Commands{}
	_ plugin.Guarded   = Commands{}
)
