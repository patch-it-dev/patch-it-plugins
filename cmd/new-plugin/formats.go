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
	mapEntry *mapEntry         // an entry the format needs in a map elsewhere
}

// mapEntry is a line added to a map literal in another package: the v1 -> v2
// conversion in migrate/ knows each model retirement page and SDK guide.
type mapEntry struct {
	file, varName, entry string // entry is a template
}

const todo = "TODO(new-plugin)"

// legacyFormats are the old --format names, each the template it is now.
var legacyFormats = map[string]string{
	"feed":           "notices/feed",
	"retirements":    "notices/deprecations",
	"migrationguide": "migrationguide/sdk",
	"removedmembers": "migrationguide/members",
	"changelog":      "changelog/markdown",
	"apichangelog":   "changelog/json",
	"catalog":        "catalog",
	"openapi":        "openapi",
}

// defaultReaders are the reader --kind picks when --reader is left out.
var defaultReaders = map[string]string{
	"notices":        "deprecations",
	"migrationguide": "sdk",
	"changelog":      "markdown",
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
		ingest: `// {{.Lower}}Ingest runs the plugin over a saved copy of the feed, cited
// through a test archive that serves the same copy.
func {{.Lower}}Ingest(t *testing.T, file string) []event.EventV2 {
	t.Helper()
	saved := kittest.Pages{feed.{{.Ident}}.URL: file}
	archive := &kittest.Archive{T: t, Pages: saved}
	return kittest.IngestWith(t, kittest.Env(t, saved), Plugin{Page: feed.{{.Ident}}, Archiver: archive})
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
	API:    "{{.ID}}",
	Vendor: "{{.Provider}}",
	URL:    "{{.Source}}",
	// ` + todo + `: the provider's main call that takes a model name.
	Operation: "POST /v1/TODO",
	// ` + todo + `: keep the model names the operation above takes (text
	// models, not image or embedding ones), as openAIText in providers.go does.
	Keep: func(name string) bool { return strings.HasPrefix(name, "placeholder-text-") },
}
`,
		register: `plugin.Register(Plugin{Page: retire.{{.Ident}}, Archiver: kit.Wayback{}})`,
		mapEntry: &mapEntry{"core/migrate/migrate.go", "providers", `retire.{{.Ident}}.API: "{{.Slug}}"`},
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
		ingest: `// {{.Lower}}Ingest runs the plugin over a saved copy of the page, cited
// through a test archive that serves the same copy.
func {{.Lower}}Ingest(t *testing.T, file string) []event.EventV2 {
	t.Helper()
	saved := kittest.Pages{retire.{{.Ident}}.URL: file}
	archive := &kittest.Archive{T: t, Pages: saved}
	return kittest.IngestWith(t, kittest.Env(t, saved), Plugin{Page: retire.{{.Ident}}, Archiver: archive})
}
`,
		imports: []string{"encoding/json", "strings", "github.com/patch-it-dev/patch-it-plugins/core/event", "github.com/patch-it-dev/patch-it-plugins/plugins/notices/retire"},
	},

	"migrationguide/sdk": {
		kind:   "migrationguide",
		layer:  "sdk",
		defDir: "plugins/migrationguide/sdkguide",
		def: `package sdkguide

// {{.Ident}} is {{.Provider}}'s SDK guide to a major release (MIGRATION.md at
// the release tag).
//
// ` + todo + `: the repo, the major release, the API operation the removed
// parameters were sent to, and the calls the guide's table names. Say what
// you checked, with the date.
var {{.Ident}} = Guide{
	API:       "{{.ID}}",
	Repo:      "TODO-owner/TODO-repo",
	Major:     1,
	Operation: "POST /v1/TODO",
	Methods:   "` + "`TODO.create()`" + `",
	Symbols:   []string{"TODO.create"},
}
`,
		register: `plugin.Register(Plugin{Guide: sdkguide.{{.Ident}}})`,
		mapEntry: &mapEntry{"core/migrate/migrate.go", "guides", `sdkguide.{{.Ident}}.API: sdkguide.{{.Ident}}`},
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
		ingest: `// {{.Lower}}Ingest runs the plugin over a saved copy of the guide.
func {{.Lower}}Ingest(t *testing.T, file string) []event.EventV2 {
	t.Helper()
	return kittest.Ingest(t, Plugin{Guide: sdkguide.{{.Ident}}}, kittest.Pages{sdkguide.{{.Ident}}.URL(): file})
}
`,
		imports: []string{"encoding/json", "strings", "github.com/patch-it-dev/patch-it-plugins/core/event", "github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/sdkguide"},
	},

	"migrationguide/members": {
		kind:   "migrationguide",
		layer:  "package",
		defDir: "plugins/migrationguide/members",
		def: `package members

// {{.Ident}} is {{.Provider}}'s migration guide to a release, with its table
// of removed members.
//
// ` + todo + `: the package, how the guide writes the module, the release,
// the repo, the guide's path at the release tag and the heading the table is
// under. Say what you checked, with the date.
var {{.Ident}} = Guide{
	Package: "{{.ID}}",
	Module:  "{{.ID}}",
	Release: "1.0.0",
	Repo:    "TODO-owner/TODO-repo",
	Path:    "TODO/migration_guide.rst",
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
placeholder_member      Use ` + "``{{.ID}}.placeholder_new``" + ` instead.
======================  ==========================
`},
		dropFile: `.. ` + todo + `: a placeholder. Replace it with a real copy of the guide and say in the test why its other table is no removal.

Removed members
---------------

======================  ==========================
Removed member          Migration guideline
======================  ==========================
placeholder_member      Use ` + "``{{.ID}}.placeholder_new``" + ` instead.
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
		ingest: `// {{.Lower}}Ingest runs the plugin over a saved copy of the guide.
func {{.Lower}}Ingest(t *testing.T, file string) []event.EventV2 {
	t.Helper()
	return kittest.Ingest(t, Plugin{Guide: members.{{.Ident}}}, kittest.Pages{members.{{.Ident}}.URL(): file})
}
`,
		imports: []string{"slices", "github.com/patch-it-dev/patch-it-plugins/core/event", "github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/members"},
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
	Package: "{{.ID}}", Ecosystem: "pypi", Module: "{{.ID}}",
	Repo: "TODO-owner/TODO-repo", TagPrefix: "v", Path: "CHANGELOG.md",
}

// {{.Ident}}Golden is the release the golden test pins.
var {{.Ident}}Golden = {{.Ident}}.At("1.0.0")
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
		ingest: `// {{.Lower}}Ingest runs the plugin over a saved copy of the changelog.
func {{.Lower}}Ingest(t *testing.T, file string) []event.EventV2 {
	t.Helper()
	return kittest.Ingest(t, Plugin{File: {{.Ident}}Golden}, kittest.Pages{ {{.Ident}}Golden.URL(): file})
}
`,
		imports: []string{"github.com/patch-it-dev/patch-it-plugins/core/event"},
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
			"{{.File}}-commits.json":   `[{"sha": "0000000000000000000000000000000000000000"}]` + "\n",
		},
		source: "{{.File}}-changelog.json",
		ingest: `// {{.Lower}}Commit is the commit the saved changelog is at.
// ` + todo + `: the real commit, and its commits response saved alongside.
const {{.Lower}}Commit = "0000000000000000000000000000000000000000"

// {{.Lower}}Ingest runs the plugin over a saved copy of the changelog.
func {{.Lower}}Ingest(t *testing.T, file string) []event.EventV2 {
	t.Helper()
	return kittest.Ingest(t, Plugin{Log: {{.Ident}}}, kittest.Pages{
		{{.Ident}}.CommitsURL():         "testdata/{{.File}}-commits.json",
		{{.Ident}}.URL({{.Lower}}Commit): file,
	})
}
`,
		imports: []string{"github.com/patch-it-dev/patch-it-plugins/core/event"},
	},

	"catalog": {
		kind:   "catalog",
		layer:  "runtime",
		defDir: "plugins/catalog",
		def: `package catalog

// {{.Ident}} is {{.Provider}}'s catalog: one JSON object, one key per command
// or endpoint, at a git ref.
//
// ` + todo + `: the ecosystem and package for the events' subject, the repo,
// the catalog's path and the commits API URL for the newest ref. Say what you
// checked, with the date, as catalog.go does for Redis.
var {{.Ident}} = Catalog{
	API: "{{.ID}}", Ecosystem: "{{.ID}}", Package: "{{.ID}}",
	Repo: "TODO-owner/TODO-repo", Path: "TODO/commands.json",
	RefsURL: "https://api.github.com/repos/TODO-owner/TODO-repo/commits/main",
}
`,
		register: `plugin.Register(Plugin{Catalog: {{.Ident}}})`,
		pluginOf: `Plugin{Catalog: {{.Ident}}}`,
		fixtures: map[string]string{
			"{{.File}}-old.json": "{\"PLACEHOLDER\": {}}\n",
			"{{.File}}-new.json": "{\"PLACEHOLDER\": {}}\n",
		},
		source: "{{.File}}-new.json",
		ingest: `// The two refs the saved snapshots are at.
// ` + todo + `: the real commits, oldest first.
const (
	{{.Lower}}OldRef = "0000000000000000000000000000000000000000"
	{{.Lower}}NewRef = "1111111111111111111111111111111111111111"
)

// {{.Lower}}Ingest replays the saved old snapshot against file.
func {{.Lower}}Ingest(t *testing.T, file string) []event.EventV2 {
	t.Helper()
	env := kittest.Env(t, kittest.Pages{
		{{.Ident}}.URL({{.Lower}}OldRef): "testdata/{{.File}}-old.json",
		{{.Ident}}.URL({{.Lower}}NewRef): file,
	})
	sink := env.Baselines.(kittest.Sink)
	p := Plugin{Catalog: {{.Ident}}}
	if rejected, err := kittest.Replay(context.Background(), env, sink, p, {{.Lower}}OldRef, {{.Lower}}NewRef); err != nil || rejected {
		t.Fatalf("replay: rejected=%v err=%v", rejected, err)
	}
	return kittest.Stored(t, sink)
}
`,
		imports: []string{"context", "github.com/patch-it-dev/patch-it-plugins/core/event"},
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

// {{.Lower}}Ingest replays the saved old spec against file.
func {{.Lower}}Ingest(t *testing.T, file string) []event.EventV2 {
	t.Helper()
	env := kittest.Env(t, nil)
	sink := env.Baselines.(kittest.Sink)
	p := Plugin{Adapter: {{.Lower}}Adapter{"old": "testdata/{{.File}}-old.json", "new": file}}
	if rejected, err := kittest.Replay(context.Background(), env, sink, p, "old", "new"); err != nil || rejected {
		t.Fatalf("replay: rejected=%v err=%v", rejected, err)
	}
	return kittest.Stored(t, sink)
}
`,
		imports: []string{"context", "fmt", "os", "github.com/patch-it-dev/patch-it-plugins/plugins/openapi/adapter", "github.com/patch-it-dev/patch-it-plugins/core/event"},
		next:    "Write LatestTag, Fetch and Versions in plugins/openapi/adapter/{{.File}}.go (fetch.Get only); add Diff/Keep to the Register line if the spec needs them (see vercel.go).",
	},
	"apisurface": {
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
var {{.Ident}} = Package{Name: "{{.PyPI}}", Ecosystem: "pypi", Module: "{{.Module}}"}
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
func {{.Lower}}Ingest(t *testing.T, file string) []event.EventV2 {
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
	from, err := p.ReadAt(ctx, env, vs[0])
	if err != nil {
		t.Fatal(err)
	}
	to, err := p.ReadAt(ctx, env, vs[1])
	if err != nil {
		t.Fatal(err)
	}
	res, err := p.Extract(ctx, env, from, to)
	if err != nil || len(res.Rejected) > 0 {
		t.Fatalf("%v %v", err, res.Rejected)
	}
	return res.Events
}
`,
		imports: []string{"archive/zip", "bytes", "context", "fmt", "github.com/patch-it-dev/patch-it-plugins/core/event"},
		next:    "Save two real wheels of the package and serve them in {{.Lower}}Ingest; the kind reads their .py files, never runs them.",
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
// and read testdata/golden/{{.ID}}.json like code: every event in it is a
// pull request someone may get.
func Test{{.Ident}}Golden(t *testing.T) {
	kittest.Golden(t, "testdata/golden/{{.ID}}.json", {{.Lower}}Ingest(t, {{.Lower}}Source))
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
