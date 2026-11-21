package main

// formatSpec is how a provider is added to one format plugin: where its
// definition goes, the line that registers it, the placeholder sources its
// test reads, and how the test runs it.
type formatSpec struct {
	kind     string            // the kind: the plugins/<kind> package that registers and tests it
	layer    string            // the source's default layer (plugin.Layers())
	defDir   string            // the package directory the definition goes in
	def      string            // the definition file (template)
	register string            // the plugin.Register line added to plugins/<format>'s init (template)
	ref      string            // how the test and the Register line name the definition
	pluginOf string            // the plugin literal for the registered test (template)
	fixtures map[string]string // placeholder sources under plugins/<format>/testdata: name (template) -> content (template)
	source   string            // the fixture the golden test reads; the drop test reads a copy, <name>-drop.<ext>
	ingest   string            // the test's <lower>Ingest func and anything it needs (template)
	imports  []string          // the test's imports besides testing and kittest
	next     string            // a format-specific line for the next steps
	drop     string            // the drop test, when "no event at all" can't be it (template)
	dropFile string            // the drop test's placeholder, when not a copy of the source's
}

const todo = "TODO(new-plugin)"

// defaultReaders are the reader --kind picks when --reader is left out.
var defaultReaders = map[string]string{
	"notices":        "deprecations",
	"migrationguide": "sdk",
	"changelog":      "markdown",
	"apisurface":     "pypi", // npm when a --package is npm:
}

var formats = map[string]formatSpec{
	"notices/feed": {
		kind:   "notices",
		layer:  "api",
		defDir: "plugins/notices/feed",
		def: `package feed

// {{.Ident}} is {{.Provider}}'s feed of changes.
//
// ` + todo + `: say what the feed is and what you checked, with the date, as
// the feeds in readers.go do: RSS or Atom, how many entries it keeps, how
// {{.Provider}} marks a breaking change or a deprecation.
var {{.Ident}} = Feed{
	API:    "{{.ID}}",
	Vendor: "{{.Provider}}",
	URL:    "{{.Source}}",
	Read:   read{{.Ident}},
}

// read{{.Ident}} turns one entry of {{.Provider}}'s feed into findings.
//
// ` + todo + `: return one Finding per thing the entry says stops or is
// deprecated, with a typed Subject, the date in Version and the entry's own
// lines in Excerpt. For an entry that is marked but that you can't read for
// sure, return a Finding with only Entry and Dropped set. Return nothing for
// an entry that isn't a breaking change or a deprecation. Precision over
// recall: readCloudflareAPI in readers.go is a worked example.
func read{{.Ident}}(e Entry) []Finding {
	return nil
}
`,
		register: `plugin.Register(Plugin{Page: feed.{{.Ident}}, Archiver: kit.Wayback{}})`,
		pluginOf: `Plugin{Page: feed.{{.Ident}}}`,
		fixtures: map[string]string{"{{.File}}.rss.xml": `<?xml version="1.0" encoding="UTF-8"?>
<!-- ` + todo + `: a placeholder. Replace it with a real copy of {{.Source}} (curl -o) and list it in README.md. -->
<rss version="2.0"><channel><title>placeholder</title><link>https://example.com/</link><description>placeholder</description>
<item><title>Placeholder entry</title><link>https://example.com/placeholder</link><pubDate>Sat, 14 Nov 2026 12:00:00 GMT</pubDate><description>Not a change.</description></item>
</channel></rss>
`},
		source: "{{.File}}.rss.xml",
		ingest: `// {{.Lower}}Ingest is a first read of a saved copy of the feed (a living
// page), cited through a test archive that serves the same copy.
func {{.Lower}}Ingest(t *testing.T, file string) []event.Event {
	t.Helper()
	saved := kittest.Pages{feed.{{.Ident}}.URL: file}
	archive := &kittest.Archive{T: t, Pages: saved}
	return kittest.Page(t, kittest.Env(t, saved), Plugin{Page: feed.{{.Ident}}, Archiver: archive})
}
`,
		imports: []string{"github.com/patch-it-dev/patch-it-plugins/core/event", "github.com/patch-it-dev/patch-it-plugins/plugins/notices/feed"},
	},

	"notices/deprecations": {
		kind:   "notices",
		layer:  "ai-models",
		defDir: "plugins/notices/retire",
		def: `package retire

import "strings"

// {{.Ident}} is {{.Provider}}'s model deprecations page.
//
// ` + todo + `: say what the page is and what you checked, with the date, as
// providers.go does: served as markdown, the table's columns (a "Shutdown
// date" or "Retirement date" column first, then the model and its
// replacement), how far back it goes.
var {{.Ident}} = Page{
	API:      "{{.ID}}",
	Provider: "{{.ProviderID}}",
	Vendor:   "{{.Provider}}",
	URL:      "{{.Source}}",
	// ` + todo + `: keep the model names the provider's calls take as text
	// models (not image or embedding ones), as openAIText in providers.go does.
	Keep: func(name string) bool { return strings.HasPrefix(name, "placeholder-text-") },
}
`,
		register: `plugin.Register(Plugin{Page: retire.{{.Ident}}, Archiver: kit.Wayback{}})`,
		pluginOf: `Plugin{Page: retire.{{.Ident}}}`,
		fixtures: map[string]string{"{{.File}}-deprecations.md": `# Placeholder deprecations page

` + todo + `: a placeholder. Replace it with a real copy of {{.Source}} (as markdown) and list it in README.md.

## Placeholder

| Shutdown date | Model | Replacement |
| ------------- | ----- | ----------- |
| November 14, 2026 | ` + "`placeholder-text-model`" + ` | ` + "`placeholder-text-model-2`" + ` |
`},
		dropFile: `# Placeholder deprecations page

` + todo + `: a placeholder. Replace it with a real copy of {{.Source}} (as markdown) and list it in README.md.

## Placeholder

| Shutdown date | Model | Replacement |
| ------------- | ----- | ----------- |
| November 14, 2026 | ` + "`placeholder-text-model`" + ` | ` + "`placeholder-text-model-2`" + ` |
| November 14, 2026 | ` + "`placeholder-image-model`" + ` | ` + "`placeholder-image-model-2`" + ` |
`,
		drop: `// The pinned drop test: a model the operation doesn't take (Keep says no)
// must not become an event, even with a shutdown date.
// ` + todo + `: save a page with such a row (an image, audio or embedding
// model) and name the model here.
func Test{{.Ident}}DropsANonChange(t *testing.T) {
	for _, e := range {{.Lower}}Ingest(t, {{.Lower}}Drop) {
		if b, _ := json.Marshal(e); strings.Contains(string(b), "placeholder-image-model") {
			t.Errorf("event for placeholder-image-model, which Keep drops: %s", e.Description)
		}
	}
}
`,
		source: "{{.File}}-deprecations.md",
		ingest: `// {{.Lower}}Ingest is a first read of a saved copy of the page (a living
// page), cited through a test archive that serves the same copy.
func {{.Lower}}Ingest(t *testing.T, file string) []event.Event {
	t.Helper()
	saved := kittest.Pages{retire.{{.Ident}}.URL: file}
	archive := &kittest.Archive{T: t, Pages: saved}
	return kittest.Page(t, kittest.Env(t, saved), Plugin{Page: retire.{{.Ident}}, Archiver: archive})
}
`,
		imports: []string{"encoding/json", "strings", "github.com/patch-it-dev/patch-it-plugins/core/event", "github.com/patch-it-dev/patch-it-plugins/plugins/notices/retire"},
	},

	"migrationguide/sdk": {
		kind:   "migrationguide",
		layer:  "sdk",
		defDir: "plugins/migrationguide/sdkguide",
		def: `package sdkguide

// {{.Ident}} is {{.Provider}}'s SDK guide to each major release (MIGRATION.md
// at the major's first tag), its releases listed from PyPI.
//
// ` + todo + `: the PyPI name, the repo and the calls the guide's table
// names. Say what you checked, with the date.
var {{.Ident}} = Guide{
	API:     "{{.ID}}",
	Package: "{{.PyPI}}",
	Repo:    "TODO-owner/TODO-repo",
	Methods: "` + "`TODO.create()`" + `",
	Symbols: []string{"TODO.create"},
}
`,
		register: `plugin.Register(Plugin{Guide: sdkguide.{{.Ident}}})`,
		pluginOf: `Plugin{Guide: sdkguide.{{.Ident}}}`,
		fixtures: map[string]string{"{{.File}}-MIGRATION.md": `# Migrating to v1

` + todo + `: a placeholder. Replace it with a real copy of the guide at the release tag and list it in README.md.

## Removed parameters

| Method | Removed parameter | Use instead |
| ------ | ----------------- | ----------- |
| ` + "`TODO.create()`" + ` | ` + "`placeholder_param`" + ` | Remove it. |
`},
		dropFile: `# Migrating to v1

` + todo + `: a placeholder. Replace it with a real copy of the guide at the release tag and list it in README.md.

## Removed parameters

| Method | Removed parameter | Use instead |
| ------ | ----------------- | ----------- |
| ` + "`TODO.create()`" + ` | ` + "`placeholder_param`" + ` | Remove it. |
| ` + "`TODO.beta.create()`" + ` | ` + "`placeholder_elsewhere`" + ` | Not the plain call. |
`,
		drop: `// The pinned drop test: a row about another call than Methods names must
// not become an event.
// ` + todo + `: save a guide with such a row (a beta-only method, a helper)
// and name its parameter here.
func Test{{.Ident}}DropsANonChange(t *testing.T) {
	for _, e := range {{.Lower}}Ingest(t, {{.Lower}}Drop) {
		if b, _ := json.Marshal(e); strings.Contains(string(b), "placeholder_elsewhere") {
			t.Errorf("event for placeholder_elsewhere, a row about another call: %s", e.Description)
		}
	}
}
`,
		source: "{{.File}}-MIGRATION.md",
		ingest: `// {{.Lower}}Previous is the release before the guide's major on PyPI: the
// pair's older side, which has no guide.
// ` + todo + `: the real release before it.
const {{.Lower}}Previous = "0.0.0"

// {{.Lower}}Ingest is the version pair {{.Lower}}Previous -> the guide's
// major over a saved copy of the guide.
// ` + todo + `: the major the saved guide is for.
func {{.Lower}}Ingest(t *testing.T, file string) []event.Event {
	t.Helper()
	g := sdkguide.{{.Ident}}.At(1)
	env := kittest.Env(t, kittest.Pages{g.URL(): file})
	return kittest.Versions(t, env, Plugin{Guide: g},
		plugin.Version{ID: {{.Lower}}Previous, Ref: "v" + {{.Lower}}Previous}, plugin.Version{ID: g.Version(), Ref: "v" + g.Version()})
}
`,
		imports: []string{"encoding/json", "strings", "github.com/patch-it-dev/patch-it-plugins/core/event", "github.com/patch-it-dev/patch-it-plugins/core/plugin", "github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/sdkguide"},
	},

	"migrationguide/members": {
		kind:   "migrationguide",
		layer:  "package",
		defDir: "plugins/migrationguide/members",
		def: `package members

// {{.Ident}} is {{.Provider}}'s migration guide to each major, with its table
// of removed members, read at the major's first tag.
//
// ` + todo + `: the package, how the guide writes the module, the repo, the
// guide's path at a major's tag ({major} for its number, {prev} for the one
// before) and the heading the table is under. Say what you checked, with the
// date.
var {{.Ident}} = Guide{
	API:     "{{.ID}}",
	Package: "{{.PyPI}}",
	Module:  "{{.Module}}",
	Repo:    "TODO-owner/TODO-repo",
	Path:    "TODO/migration_guide_{major}.rst",
	Section: "Removed members",
}
`,
		register: `plugin.Register(Plugin{Guide: members.{{.Ident}}})`,
		pluginOf: `Plugin{Guide: members.{{.Ident}}}`,
		fixtures: map[string]string{"{{.File}}-migration-guide.rst": `.. ` + todo + `: a placeholder. Replace it with a real copy of the guide at the release tag and list it in README.md.

Removed members
---------------

======================  ==========================
Removed member          Migration guideline
======================  ==========================
placeholder_member      Use ` + "``{{.Module}}.placeholder_new``" + ` instead.
======================  ==========================
`},
		dropFile: `.. ` + todo + `: a placeholder. Replace it with a real copy of the guide and say in the test why its other table is no removal.

Removed members
---------------

======================  ==========================
Removed member          Migration guideline
======================  ==========================
placeholder_member      Use ` + "``{{.Module}}.placeholder_new``" + ` instead.
======================  ==========================

Other members
-------------

======================  ==========================
Removed member          Migration guideline
======================  ==========================
placeholder_elsewhere   Not in the main table.
======================  ==========================
`,
		drop: `// The pinned drop test: every row of the table under Section is a removal,
// so what must not become an event is a member listed anywhere else.
// ` + todo + `: save a guide with such a table (another namespace, a
// deprecation table) and name one of its members here.
func Test{{.Ident}}DropsANonChange(t *testing.T) {
	for _, e := range {{.Lower}}Ingest(t, {{.Lower}}Drop) {
		if slices.Contains(e.Subject.Symbols, "placeholder_elsewhere") {
			t.Errorf("event for placeholder_elsewhere, which isn't under %q", members.{{.Ident}}.Section)
		}
	}
}
`,
		source: "{{.File}}-migration-guide.rst",
		ingest: `// {{.Lower}}Previous is the release before the guide's on PyPI: the pair's
// older side, which has no guide.
// ` + todo + `: the real release before it.
const {{.Lower}}Previous = "0.0.0"

// {{.Lower}}Release is the major's first release the saved guide is for.
// ` + todo + `: the real release.
const {{.Lower}}Release = "1.0.0"

// {{.Lower}}Ingest is the version pair {{.Lower}}Previous ->
// {{.Lower}}Release over a saved copy of the guide.
func {{.Lower}}Ingest(t *testing.T, file string) []event.Event {
	t.Helper()
	g := members.{{.Ident}}
	env := kittest.Env(t, kittest.Pages{g.RawURL("v"+{{.Lower}}Release, 1): file})
	return kittest.Versions(t, env, Plugin{Guide: g}, plugin.Version{ID: {{.Lower}}Previous}, plugin.Version{ID: {{.Lower}}Release})
}
`,
		imports: []string{"slices", "github.com/patch-it-dev/patch-it-plugins/core/event", "github.com/patch-it-dev/patch-it-plugins/core/plugin", "github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/members"},
	},

	"changelog/markdown": {
		kind:   "changelog",
		layer:  "package",
		defDir: "plugins/changelog",
		def: `package changelog

// {{.Ident}} is {{.Provider}}'s changelog, read release by release (its
// releases listed from the ecosystem's registry, each at TagPrefix+version).
//
// ` + todo + `: the ecosystem, the import name, the repo, the tag prefix and
// the changelog's path. Say what you checked, with the date, as changelog.go
// does for Textual and urllib3.
var {{.Ident}} = File{
	API: "{{.ID}}", Package: "{{.PyPI}}", Ecosystem: "pypi", Module: "{{.Module}}",
	Repo: "TODO-owner/TODO-repo", TagPrefix: "v", Path: "CHANGELOG.md",
}
`,
		register: `plugin.Register(Plugin{File: {{.Ident}}})`,
		pluginOf: `Plugin{File: {{.Ident}}}`,
		fixtures: map[string]string{"{{.File}}-CHANGELOG.md": `# Changelog

<!-- ` + todo + `: a placeholder. Replace it with a real copy of the changelog at the release tag and list it in README.md. -->

## [1.0.0] - 2026-11-14

### Added

- A placeholder entry.
`},
		source: "{{.File}}-CHANGELOG.md",
		ingest: `// {{.Ident}}Golden is the release the golden test pins: only a test names
// one.
// ` + todo + `: the real release.
var {{.Ident}}Golden = {{.Ident}}.At("1.0.0")

// {{.Lower}}Previous is the release before {{.Ident}}Golden's on the
// registry: the pair's older side.
// ` + todo + `: the real release before it.
const {{.Lower}}Previous = "0.0.0"

// {{.Lower}}Ingest is the version pair {{.Lower}}Previous -> {{.Ident}}Golden
// over a saved copy of the changelog. A changelog's Extract reads only the
// newer release's section, so the older side is named, not read.
func {{.Lower}}Ingest(t *testing.T, file string) []event.Event {
	t.Helper()
	f := {{.Ident}}Golden
	env := kittest.Env(t, kittest.Pages{f.URL(): file})
	p := Plugin{File: f}
	to, err := p.ReadAt(context.Background(), env, plugin.Version{ID: f.Release, Ref: f.Tag})
	if err != nil {
		t.Fatal(err)
	}
	from := plugin.Snapshot{Version: plugin.Version{ID: {{.Lower}}Previous, Ref: f.TagPrefix + {{.Lower}}Previous}}
	return kittest.Extracted(t, env, p, from, to)
}
`,
		imports: []string{"context", "github.com/patch-it-dev/patch-it-plugins/core/event", "github.com/patch-it-dev/patch-it-plugins/core/plugin"},
	},

	"changelog/json": {
		kind:   "changelog",
		layer:  "api",
		defDir: "plugins/changelog",
		def: `package changelog

// {{.Ident}} is {{.Provider}}'s machine-readable API changelog.
//
// ` + todo + `: the repo and the file's path, and what you checked, with the
// date, as json.go does for MongoDB Atlas.
var {{.Ident}} = Log{
	API: "{{.ID}}", Vendor: "{{.Provider}}", Name: "TODO: the API's name, for sentences",
	Repo: "TODO-owner/TODO-repo", Path: "changelog/changelog.json",
}
`,
		register: `plugin.Register(Plugin{Log: {{.Ident}}})`,
		pluginOf: `Plugin{Log: {{.Ident}}}`,
		fixtures: map[string]string{
			"{{.File}}-changelog.json": `[{"date": "2026-11-14", "paths": [{"path": "/placeholder", "httpMethod": "GET", "versions": [{"version": "2026-01-01", "changes": [{"change": "placeholder: not a life-cycle change", "changeCode": "response-property-added", "backwardCompatible": true}]}]}]}]` + "\n",
		},
		source: "{{.File}}-changelog.json",
		ingest: `// The commit the saved changelog is at, and the one that changed the file
// before it (each version's ID is its commit's date).
// ` + todo + `: the real commits.
var (
	{{.Lower}}Commit = plugin.Version{ID: "2026-11-14", Ref: "1111111111111111111111111111111111111111"}
	{{.Lower}}Before = plugin.Version{ID: "2026-11-01", Ref: "0000000000000000000000000000000000000000"}
)

// {{.Lower}}Ingest is the version pair {{.Lower}}Before -> {{.Lower}}Commit
// over a saved copy of the changelog at {{.Lower}}Commit. The commit before
// is named, not read: every change in file is extracted.
func {{.Lower}}Ingest(t *testing.T, file string) []event.Event {
	t.Helper()
	env := kittest.Env(t, kittest.Pages{ {{.Ident}}.URL({{.Lower}}Commit.Ref): file})
	p := Plugin{Log: {{.Ident}}}
	to, err := p.ReadAt(context.Background(), env, {{.Lower}}Commit)
	if err != nil {
		t.Fatal(err)
	}
	return kittest.Extracted(t, env, p, plugin.Snapshot{Version: {{.Lower}}Before}, to)
}
`,
		imports: []string{"context", "github.com/patch-it-dev/patch-it-plugins/core/event", "github.com/patch-it-dev/patch-it-plugins/core/plugin"},
	},

	"catalog": {
		kind:   "catalog",
		layer:  "runtime",
		defDir: "plugins/catalog",
		def: `package catalog

import "regexp"

// {{.Ident}} is {{.Provider}}'s command catalog: one JSON file per command in a
// directory of its repo, read at each release tag, as Redis's is (redis.go).
//
// ` + todo + `: the ecosystem and package for the events' subject, the repo,
// the directory and the release tags' shape. Say what you checked, with the
// date, as redis.go does for Redis.
var {{.Ident}} = Commands{
	API: "{{.ID}}", Ecosystem: "{{.ProviderID}}", Package: "{{.ProviderID}}",
	Repo: "TODO-owner/TODO-repo", Dir: "TODO/commands",
	Release: regexp.MustCompile(` + "`" + `^v?(\d+\.\d+\.\d+)$` + "`" + `),
}
`,
		register: `plugin.Register({{.Ident}})`,
		pluginOf: `{{.Ident}}`,
		fixtures: map[string]string{
			"{{.File}}-old.json":         "[{\"name\": \"placeholder.json\", \"type\": \"file\"}, {\"name\": \"kept.json\", \"type\": \"file\"}]\n",
			"{{.File}}-new.json":         "[{\"name\": \"kept.json\", \"type\": \"file\"}]\n",
			"{{.File}}-placeholder.json": "{\"PLACEHOLDER\": {\"summary\": \"a placeholder\"}}\n",
		},
		dropFile: "[{\"name\": \"placeholder.json\", \"type\": \"file\"}, {\"name\": \"kept.json\", \"type\": \"file\"}, {\"name\": \"added.json\", \"type\": \"file\"}]\n",
		source:   "{{.File}}-new.json",
		ingest: `// The two releases the saved listings are at.
// ` + todo + `: the real tags, oldest first.
const (
	{{.Lower}}OldRef = "v1.0.0"
	{{.Lower}}NewRef = "v2.0.0"
)

// {{.Lower}}Ingest is the version pair from the saved old listing to file,
// the newer release's listing; a file gone from it is read at the old tag.
func {{.Lower}}Ingest(t *testing.T, file string) []event.Event {
	t.Helper()
	env := kittest.Env(t, kittest.Pages{
		{{.Ident}}.listingURL({{.Lower}}OldRef): "testdata/{{.File}}-old.json",
		{{.Ident}}.listingURL({{.Lower}}NewRef): file,
		{{.Ident}}.FileURL({{.Lower}}OldRef, "placeholder.json"): "testdata/{{.File}}-placeholder.json",
	})
	return kittest.Versions(t, env, {{.Ident}},
		plugin.Version{ID: {{.Lower}}OldRef, Ref: {{.Lower}}OldRef}, plugin.Version{ID: {{.Lower}}NewRef, Ref: {{.Lower}}NewRef})
}
`,
		imports: []string{"github.com/patch-it-dev/patch-it-plugins/core/event", "github.com/patch-it-dev/patch-it-plugins/core/plugin"},
	},

	"openapi": {
		kind:   "openapi",
		layer:  "api",
		defDir: "plugins/openapi/adapter",
		def: `package adapter

import (
	"context"
	"fmt"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// {{.Ident}} adapts {{.Provider}}'s OpenAPI spec for the openapi plugin.
//
// ` + todo + `: LatestTag finds the newest release (a tag, or the commit
// that last changed the spec) and Fetch downloads the spec at it with
// fetch.Get, as supabase.go and vercel.go do. Say where the spec is
// published and what you checked, with the date.
type {{.Ident}} struct{}

var _ Adapter = {{.Ident}}{}

func ({{.Ident}}) API() string { return "{{.ID}}" }

// LatestTag is the newest release of the spec.
func ({{.Ident}}) LatestTag(ctx context.Context) (string, error) {
	return "", fmt.Errorf("{{.ID}}: LatestTag isn't written yet (plugins/openapi/adapter/{{.File}}.go)")
}

// Fetch downloads the spec at tag.
func ({{.Ident}}) Fetch(ctx context.Context, tag string) (Spec, error) {
	return Spec{}, fmt.Errorf("{{.ID}}: Fetch isn't written yet (plugins/openapi/adapter/{{.File}}.go)")
}

// Versions are every release of the spec, oldest first, each with the Ref
// Fetch takes: what lets every pair of releases be diffed, not only the
// newest. ` + todo + `: list them with core/versions (GitHubReleases,
// GitHubCommits, GitHubTags), as versions.go does for the other adapters.
func ({{.Ident}}) Versions(ctx context.Context) ([]plugin.Version, error) {
	return nil, fmt.Errorf("{{.ID}}: Versions isn't written yet (plugins/openapi/adapter/{{.File}}.go)")
}
`,
		register: `plugin.Register(Plugin{Adapter: adapter.{{.Ident}}{}})`,
		pluginOf: `Plugin{Adapter: adapter.{{.Ident}}{}}`,
		fixtures: map[string]string{
			"{{.File}}-old.json": `{"openapi": "3.0.3", "info": {"title": "placeholder", "version": "1"}, "paths": {}}` + "\n",
			"{{.File}}-new.json": `{"openapi": "3.0.3", "info": {"title": "placeholder", "version": "2"}, "paths": {}}` + "\n",
		},
		source: "{{.File}}-new.json",
		ingest: `// {{.Lower}}Adapter serves saved copies of the spec: tag -> file.
// ` + todo + `: name the tags after the real releases you saved.
type {{.Lower}}Adapter map[string]string

func ({{.Lower}}Adapter) API() string { return "{{.ID}}" }

func ({{.Lower}}Adapter) LatestTag(context.Context) (string, error) { return "new", nil }

func (a {{.Lower}}Adapter) Fetch(_ context.Context, tag string) (adapter.Spec, error) {
	file, ok := a[tag]
	if !ok {
		return adapter.Spec{}, fmt.Errorf("no saved copy for %s", tag)
	}
	body, err := os.ReadFile(file)
	if err != nil {
		return adapter.Spec{}, err
	}
	return adapter.Spec{Content: body, Version: tag, SourceURL: "{{.Source}}"}, nil
}

// {{.Lower}}Ingest extracts the change from the saved old spec to file.
func {{.Lower}}Ingest(t *testing.T, file string) []event.Event {
	t.Helper()
	p := Plugin{Adapter: {{.Lower}}Adapter{"old": "testdata/{{.File}}-old.json", "new": file}}
	return kittest.Versions(t, kittest.Env(t, nil), p, plugin.Version{ID: "old", Ref: "old"}, plugin.Version{ID: "new", Ref: "new"})
}
`,
		imports: []string{"context", "fmt", "os", "github.com/patch-it-dev/patch-it-plugins/plugins/openapi/adapter", "github.com/patch-it-dev/patch-it-plugins/core/event", "github.com/patch-it-dev/patch-it-plugins/core/plugin"},
		next:    "Write LatestTag, Fetch and Versions in plugins/openapi/adapter/{{.File}}.go (fetch.Get only); add Diff/Keep to the Register line if the spec needs them (see vercel.go).",
	},
	"apisurface/pypi": {
		kind:   "apisurface",
		layer:  "sdk",
		defDir: "plugins/apisurface",
		def: `package apisurface

// {{.Ident}} is {{.Provider}}'s package on PyPI, its public API read from each
// release's wheel.
//
// ` + todo + `: check the name on PyPI and the top-level module code
// imports (from --package, or guessed). Say
// what you checked, with the date: that the wheel holds the package's .py
// (or .pyi) files.
var {{.Ident}} = Package{ID: "{{.ID}}", Name: "{{.PyPI}}", Ecosystem: "pypi", Module: "{{.Module}}"}
`,
		register: `plugin.Register(Plugin{Package: {{.Ident}}})`,
		pluginOf: `Plugin{Package: {{.Ident}}}`,
		fixtures: map[string]string{
			"{{.File}}-old.py": "# ` + todo + `: a placeholder for the older release's module. Save the real release's wheel instead (see apisurface_test.go).\ndef placeholder(a, b=1):\n    pass\n",
			"{{.File}}-new.py": "# ` + todo + `: a placeholder for the newer release's module: it drops b, one event.\ndef placeholder(a):\n    pass\n",
		},
		// The drop test's newer release: the older one again, nothing gone.
		dropFile: "# ` + todo + `: a placeholder: a release that only adds or changes private names.\ndef placeholder(a, b=1):\n    pass\n\ndef _private(x):\n    pass\n",
		source:   "{{.File}}-new.py",
		ingest: `// {{.Lower}}Ingest compares two releases: 1.0.0, a wheel holding
// testdata/{{.File}}-old.py as {{.Module}}/__init__.py, and 1.1.0, one
// holding file. The wheels are built here so the placeholders are text.
// ` + todo + `: save the two real wheels (trimmed if they're large, as
// apisurface_test.go's are) and serve them instead.
func {{.Lower}}Ingest(t *testing.T, file string) []event.Event {
	t.Helper()
	wheel := func(src string) []byte {
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		w, err := zw.Create("{{.Module}}/__init__.py")
		if err != nil {
			t.Fatal(err)
		}
		w.Write(kittest.Read(t, src))
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}
	files := map[string][]byte{
		"https://pypi.org/pypi/{{.PyPI}}/json":       []byte(` + "`" + `{"releases": {"1.0.0": [{"upload_time_iso_8601": "2026-01-01T00:00:00Z"}], "1.1.0": [{"upload_time_iso_8601": "2026-02-01T00:00:00Z"}]}}` + "`" + `),
		"https://pypi.org/pypi/{{.PyPI}}/1.0.0/json": []byte(` + "`" + `{"urls": [{"packagetype": "bdist_wheel", "filename": "{{.Module}}-1.0.0-py3-none-any.whl", "url": "https://example.com/{{.Module}}-1.0.0-py3-none-any.whl"}]}` + "`" + `),
		"https://pypi.org/pypi/{{.PyPI}}/1.1.0/json": []byte(` + "`" + `{"urls": [{"packagetype": "bdist_wheel", "filename": "{{.Module}}-1.1.0-py3-none-any.whl", "url": "https://example.com/{{.Module}}-1.1.0-py3-none-any.whl"}]}` + "`" + `),
		"https://example.com/{{.Module}}-1.0.0-py3-none-any.whl": wheel("testdata/{{.File}}-old.py"),
		"https://example.com/{{.Module}}-1.1.0-py3-none-any.whl": wheel(file),
	}
	env := kittest.Env(t, nil)
	env.Get = func(_ context.Context, url string) ([]byte, error) {
		if b, ok := files[url]; ok {
			return b, nil
		}
		t.Errorf("fetched %s, which the test didn't save", url)
		return nil, fmt.Errorf("no saved copy of %s", url)
	}
	ctx, p := context.Background(), Plugin{Package: {{.Ident}}}
	vs, err := p.ListVersions(ctx, env)
	if err != nil || len(vs) != 2 {
		t.Fatalf("versions %v: %v", vs, err)
	}
	return kittest.Versions(t, env, p, vs[0], vs[1])
}
`,
		imports: []string{"archive/zip", "bytes", "context", "fmt", "github.com/patch-it-dev/patch-it-plugins/core/event"},
		next:    "Save two real wheels of the package and serve them in {{.Lower}}Ingest; the kind reads their .py files, never runs them.",
	},
	"apisurface/npm": {
		kind:   "apisurface",
		layer:  "sdk",
		defDir: "plugins/apisurface",
		def: `package apisurface

// {{.Ident}} is {{.Provider}}'s package on npm, its public API read from each
// release's TypeScript declarations.
//
// ` + todo + `: check the name on npm, and that its tarball has .d.ts
// files (package.json's "types" or "exports"). Say what you checked, with
// the date.
var {{.Ident}} = npmPackage("{{.ID}}", "{{.NPM}}")
`,
		register: `plugin.Register(Plugin{Package: {{.Ident}}})`,
		pluginOf: `Plugin{Package: {{.Ident}}}`,
		fixtures: map[string]string{
			"{{.File}}-old.d.ts": "// ` + todo + `: a placeholder for the older release's declarations. Save the real release's tarball instead (see npm_test.go).\nexport declare function placeholder(a: string, b?: number): void;\n",
			"{{.File}}-new.d.ts": "// ` + todo + `: a placeholder for the newer release's declarations: b is no longer optional, one event.\nexport declare function placeholder(a: string, b: number): void;\n",
		},
		// The drop test's newer release: the older one again, a name added.
		dropFile: "// ` + todo + `: a placeholder: a release that only adds.\nexport declare function placeholder(a: string, b?: number): void;\nexport declare function added(): void;\n",
		source:   "{{.File}}-new.d.ts",
		ingest: `// {{.Lower}}Ingest compares two releases: 1.0.0, a tarball holding
// testdata/{{.File}}-old.d.ts as index.d.ts, and 1.1.0, one holding file.
// The tarballs are built here so the placeholders are text.
// ` + todo + `: save the two real tarballs (trimmed if they're large, as
// npm_test.go's are) and serve them instead.
func {{.Lower}}Ingest(t *testing.T, file string) []event.Event {
	t.Helper()
	tarball := func(src string) []byte {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gz)
		for name, body := range map[string][]byte{
			"package/package.json": []byte(` + "`" + `{"name": "{{.NPM}}", "types": "index.d.ts"}` + "`" + `),
			"package/index.d.ts":   kittest.Read(t, src),
		} {
			if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
				t.Fatal(err)
			}
			tw.Write(body)
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		gz.Close()
		return buf.Bytes()
	}
	reg := "https://registry.npmjs.org/" + strings.ReplaceAll("{{.NPM}}", "/", "%2F")
	files := map[string][]byte{
		reg:            []byte(` + "`" + `{"versions": {"1.0.0": {}, "1.1.0": {}}, "time": {"1.0.0": "2026-01-01T00:00:00Z", "1.1.0": "2026-02-01T00:00:00Z"}}` + "`" + `),
		reg + "/1.0.0": []byte(` + "`" + `{"dist": {"tarball": "https://example.com/{{.File}}-1.0.0.tgz"}}` + "`" + `),
		reg + "/1.1.0": []byte(` + "`" + `{"dist": {"tarball": "https://example.com/{{.File}}-1.1.0.tgz"}}` + "`" + `),
		"https://example.com/{{.File}}-1.0.0.tgz": tarball("testdata/{{.File}}-old.d.ts"),
		"https://example.com/{{.File}}-1.1.0.tgz": tarball(file),
	}
	env := kittest.Env(t, nil)
	env.Get = func(_ context.Context, url string) ([]byte, error) {
		if b, ok := files[url]; ok {
			return b, nil
		}
		t.Errorf("fetched %s, which the test didn't save", url)
		return nil, fmt.Errorf("no saved copy of %s", url)
	}
	ctx, p := context.Background(), Plugin{Package: {{.Ident}}}
	vs, err := p.ListVersions(ctx, env)
	if err != nil || len(vs) != 2 {
		t.Fatalf("versions %v: %v", vs, err)
	}
	return kittest.Versions(t, env, p, vs[0], vs[1])
}
`,
		imports: []string{"archive/tar", "bytes", "compress/gzip", "context", "fmt", "strings", "github.com/patch-it-dev/patch-it-plugins/core/event"},
		next:    "Save two real tarballs of the package and serve them in {{.Lower}}Ingest; the kind reads their .d.ts files, never runs them.",
	},
}

// testFile is the generated test, the same for every format.
const testFile = `package {{.Kind}}

import (
{{range .Imports}}	"{{.}}"
{{end}})

// The saved sources the tests read.
// ` + todo + `: replace the placeholders with real copies, and list each in
// testdata/README.md (URL, date fetched, license, a short quote).
const (
	{{.Lower}}Source = "testdata/{{.SourceFile}}"
	{{.Lower}}Drop   = "testdata/{{.DropFile}}"
)

func Test{{.Ident}}RegistersItself(t *testing.T) {
	kittest.Registered(t, {{.PluginOf}})
}

// The golden test: the saved source in, every event out.
// ` + todo + `: once the source is real, run
//
//	go test ./plugins/{{.Kind}} -run Test{{.Ident}}Golden -update
//
// and read testdata/golden/{{.Name}}.json like code: every event in it is a
// pull request someone may get.
func Test{{.Ident}}Golden(t *testing.T) {
	kittest.Golden(t, "testdata/golden/{{.Name}}.json", {{.Lower}}Ingest(t, {{.Lower}}Source))
}

{{if .Drop}}{{.Drop}}{{else}}// The pinned drop test: a realistic input that must not become an event.
// ` + todo + `: save one (an unrelated section, a behavior change written as
// prose, a product name in a sentence) and say here why it's no change.
func Test{{.Ident}}DropsANonChange(t *testing.T) {
	if got := {{.Lower}}Ingest(t, {{.Lower}}Drop); len(got) != 0 {
		t.Errorf("%d events from %s, want none", len(got), {{.Lower}}Drop)
	}
}
{{end}}
{{.Ingest}}`
