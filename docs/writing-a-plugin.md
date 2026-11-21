# Writing a plugin

A **provider** (Stripe, OpenAI, Kubernetes…) has a **category** and one or more **sources**. A source is one place the provider announces changes; it is read by one **kind** of plugin (the `plugins/<kind>` package: `openapi`, `changelog`, `notices`, `migrationguide`, `catalog`, `apisurface`), covers one **layer** (`api`, `sdk`, `package`, `runtime`, `ci`, `ai-models`, `infra`), and has **versions**. Its id is `provider/source` (`stripe/api`, `algolia/search-api`). This guide walks through adding a source, from picking it to the PR. The rules are in [CONTRIBUTING.md](../CONTRIBUTING.md); this is the how-to. How the pieces fit: [how-it-works.md](how-it-works.md).

**Most new sources don't need new code.** They are a new entry for an existing kind: a new `changelog.File`, a new `retire.Page`, a new `feed.Feed` reader, a new OpenAPI adapter, a new `apisurface.Package`. Write a new kind only when none of them reads your source. There is no "Kubernetes kind" or "Terraform kind": Kubernetes API removals are a migration-guide or notices source, a client SDK is a package source, a Terraform provider is a changelog source.

## 1. Scaffold it with `new-plugin`

Pick the source and the kind that reads it (steps 2 and 3, and [which kind fits my provider?](which-kind.md)), then let the starter write everything around it:

```sh
make new-plugin ARGS='--provider acme --name Acme --category devtools --homepage https://acme.example \
  --source api --kind openapi --url https://github.com/acme/openapi --package pypi:acme --package npm:acme'
# the same: go run ./cmd/new-plugin --provider acme ...
# --kind: openapi, changelog, notices, migrationguide, catalog, apisurface
# --reader, for a kind with several: changelog markdown|json, notices deprecations|feed, migrationguide sdk|members
# optional: --layer api|sdk|package|runtime|ci|ai-models|infra (default: the kind's), --watches "...",
#           --page acme (share an existing providers/ page; default <provider>-<source>)
```

Run in a terminal without the flags, it asks for them: the provider's id, then (for a new provider) its name, category and homepage, then the source's id, kind, reader, layer, URLs, packages and what it watches. A second source of a registered provider needs only `--provider`, `--source` and `--kind`: the name, category and homepage come from the registry. One run adds one source; run it again for the next.

It writes, and lists:

- the source's definition in the package its kind reads it from (`plugins/openapi/adapter/`, `plugins/notices/retire/`, `plugins/notices/feed/`, `plugins/migrationguide/sdkguide/`, `plugins/migrationguide/members/` or `plugins/<kind>/`), with the version wiring the kind needs from it: an OpenAPI adapter's `LatestTag`, `Fetch` and `Versions` to write; every other kind lists, reads and extracts versions itself;
- its `plugin.RegisterProvider` block (the provider and the source with every metadata field, packages included) in the kind's `init`, and the kind's import in `plugins/all` if it's a new kind;
- `plugins/<kind>/<provider>_<source>_test.go`: the registration test, a golden test and a pinned drop test, over placeholder sources in `testdata/`, with the golden file written from them;
- a row per saved file in that `testdata/README.md` (URL, date, license, quote to fill in);
- `providers/<provider>-<source>.md` from the template (or nothing, with `--page`, when the provider already has one);

`go test ./...` passes right after; every place to fill in says `TODO(new-plugin)` (`grep -rn 'TODO(new-plugin)' .`), and `make check` (its `todos` step, also in CI_go) fails until none is left. It refuses a `provider/source` id that's taken, or a file it would write that exists, and then changes nothing.

## 2. Pick the source

Use the provider's own, most machine-readable source. In order of preference:

1. **An official machine-readable spec** (OpenAPI), diffed between releases. Precise, typed, no guessing.
2. **A file at a release tag** (a changelog, a migration guide) on GitHub or the package registry. A tag never changes, so its URL is a permanent citation.
3. **An official RSS / Atom feed** of changes. Built to be machine-read.
4. **An official documentation page** (a deprecations page), ideally served as markdown. A live page changes, so events cite a Wayback Machine snapshot of it.

Never scrape what the provider's `robots.txt` or terms disallow, and never read a forum or a third-party blog. Write down what you chose and why in the provider page (step 8).

### Measure the source over its history before trusting it

A source that looks right on today's copy can be wrong as a history of changes. Before picking what the plugin keeps, download the source at every release (or a sample plus every release of a recent window), run the format's diff or reader over each consecutive pair, and count what comes out:

- **Deprecations:** how many, and how the provider marks them. A flag (`deprecated: true`), words in a summary ("(deprecated)", "Deprecated:"), an extension (`x-cfDeprecation`), a change code. A flag the provider rarely sets misses most of them; a regex that matches "deprecated" anywhere matches operations *about* deprecated things.
- **Removals, split by whether they were marked deprecated first.** In the specs measured so far, most removals with no deprecation before them weren't removals from the API:
  - the spec moved or renamed the path (a path parameter renamed, `/site/` corrected to `/sites/`, a doubled base path fixed);
  - an operation went to an internal spec, or was left out as not for direct use;
  - an operation came back a few releases later;
  - a release branch cut before an older branch's last minor didn't have it yet;
  - it described only an API client: a helper or a "custom request" method, marked `x-helper` or similar.

  So the default rule for a spec is the `deprecationsOnly` filter (`plugins/openapi/vercel.go`): operation deprecations, plus removals of operations marked deprecated first. Keep an unmarked removal only if the history shows they're real.
- **Withdrawals:** a deprecation the provider takes back (`endpoint-reactivated`), a removed version that reappears. Replay the history in order, and let a withdrawal cancel what it withdraws.
- **Property-level changes,** request and response. Response properties of a generated spec often churn by the hundred per release. Count before deciding; leaving them out is a fine first version.

Then check what's left against the provider's own words:

- **A shutdown needs the provider to say the thing stops.** An "end of life date" field alone isn't enough when the item's text says direct callers need no action, or names only a field or a value.
- **Cross-check a removal with the release notes** ("Remove unfreeze REST endpoint" for the endpoint that left the spec).
- **Probing the live API from outside rarely settles it.** Many APIs answer 401 or 403 to every path without credentials, real or made up.

Write the numbers on the provider page, as `providers/vercel.md` does for its spec's history, so a reviewer can check the rule against them.

### Cite a version, and pick one a repo pins

- **Cite what never changes:** a release tag, or the commit SHA of the release or of a branch's head. A moving branch or a live URL isn't a citation.
- **"Newest" needs care:**
  - a day can have several releases whose tag suffix says nothing about their order (break the tie by commit date);
  - an older major's patch can be released after a newer minor (take the highest version, not GitHub's latest release);
  - the newest branch may be unreleased (read the newest *released* one).
- **`version`** is the API version the change arrived in, per the contract. If the API has none, use a package version a repo pins (an SDK release), or a date: a release date, an end of life.
  - The patch-it client gates a package version as a package. It keeps a date-versioned API's events only if the API is in its list of date-versioned APIs, so **a new date-versioned API needs an entry there too**.
  - Two specs of one version (fixes on a release branch) aren't a release: `RemovalsNeedANewVersion` drops removals between them.

## 3. Pick the kind

[Which kind fits my provider?](which-kind.md) asks the questions; the table says what each kind reads.

| Kind | Reads | Add a source by |
|---|---|---|
| `plugins/openapi` | An OpenAPI spec per release, diffed with `core/diff/` and turned into events by `core/normalize/` | An `adapter.Adapter` in `plugins/openapi/adapter/` (latest tag, fetch a tag's spec; `Versions` to list every release) and a `plugin.RegisterProvider` block |
| `plugins/notices` | A live page of dated notices: a model provider's deprecations page (a table of date, model, replacement), a vendor's RSS or Atom feed, a vendor's table of whole API version ranges and their support status | A `retire.Page` in `plugins/notices/retire/providers.go`, a `feed.Feed` with a reader for that vendor's entries in `plugins/notices/feed/`, or an `eol.Page` in `plugins/notices/eol/` (`new-plugin` doesn't scaffold this reader yet) |
| `plugins/migrationguide` | A guide that lists what a release removed, at the release's tag: an SDK's migration guide per new major (removed request parameters), or a package's table of removed members (an RST simple table or a Markdown pipe table) | A `sdkguide.Guide` in `plugins/migrationguide/sdkguide/`, or a `members.Guide` in `plugins/migrationguide/members/` |
| `plugins/changelog` | A changelog: a `CHANGELOG.md` / `CHANGES.rst` / `HISTORY.rst`, each release's section at its tag (releases from PyPI or npm), or a vendor's machine-readable API changelog in JSON (per day, endpoint and API version, oasdiff change codes), commit by commit | A `changelog.File` with its `TagPrefix`, or a `changelog.Log` |
| `plugins/catalog` | A provider's own structured list of what it offers: a command catalog kept as one JSON file per command in its repo, read at each release tag and compared for commands that disappeared entirely | A `catalog.Commands` (repo, directory, release tag pattern); a catalog of another shape is a new reader type in the package |
| `plugins/apisurface` | A published package's public API, compared between two releases: the classes, functions, methods and parameters in each release's files (Python first: the wheel or sdist from PyPI) | A `apisurface.Package` (registry name, ecosystem, top-level module) and its `plugin.RegisterProvider` block |

## 4. The interface

A plugin implements `plugin.Plugin` (package [`core/plugin/`](../core/plugin/plugin.go)) and **exactly one** of two shapes: `plugin.Versioned` (a source with versions) or `plugin.Living` (a page with no versions of its own). `RegisterProvider` refuses a plugin that is both or neither. Either way the changes are what `Extract` finds between two snapshots, so whoever runs it treats every source as pairs.

```go
type Plugin interface {
    Name() string                                                  // the source's id, provider/source: every event's `api`
    Extract(ctx context.Context, env Env, from, to Snapshot) (Result, error) // the changes from one snapshot to the next
}

type Versioned interface { // tags, releases, commits, a guide per major
    Plugin
    ListVersions(ctx context.Context, env Env) ([]Version, error)   // every version, oldest first
    ReadAt(ctx context.Context, env Env, v Version) (Snapshot, error) // the source at one version
}

type Living interface { // a deprecations page, a feed
    Plugin
    Read(ctx context.Context, env Env, last Snapshot) (Snapshot, error) // the page now, against the snapshot kept from the read before
}

type Env struct {
    Get func(ctx context.Context, url string) ([]byte, error) // the only way to fetch
    Now func() time.Time
}

type Snapshot struct {
    Version     Version // ID, Released, Ref, URL
    Content     []byte
    SourceURL   string  // the exact document read: the events' citation
    CompareHash string  // a living page only: how its next Read tells it unchanged
}

type Result struct {
    Events   []event.Event // valid events
    Rejected []error       // changes that failed to become valid events
}
```

A plugin **stores nothing** and reads nothing back: whoever runs it (patch-it's hosted ingestion service, or the test kit) stores the events and keeps the snapshot the next comparison starts from.

When a release or a page has no part the reader can use (a changelog with no section for the release, a migration guide with no table of removals, a page with no table of notices), return `plugin.ErrNothingToRead`, wrapped with what was missing (`fmt.Errorf("no heading for release %s: %w", v, plugin.ErrNothingToRead)`). For a release that's normal, and the runner skips the pair instead of retrying it; it finds the error with `errors.Is`. Any other error is a failure, retried next run.

### A source with versions: `plugin.Versioned`

A source that has versions (a spec's releases, a package's releases, a guide per major, the commits that changed a file) is read version by version, so a customer several versions behind gets every step: the runner lists the versions, reads each consecutive pair with `ReadAt` and extracts it once (a released version never changes).

- **List** with [`core/versions`](../core/versions/versions.go): `GitHubReleases`, `GitHubTags`, `GitHubCommits` (the commits that changed a file), `PyPIReleases` (`PyPIReleasesWithYanked` when a change can be documented at a release later yanked), `NPMReleases`, `GitHubPages` (any other GitHub list, page by page). An OpenAPI adapter lists its own versions (`adapter.Lister`): every adapter does. Never pin a tag in code: a version you need for a golden test is a pinned *case* in the test (`changelog.Urllib3.At("2.0.0")`, in a `_test.go` file), not the registered source; `plugins/all`'s `TestNoPinnedReleaseInPluginCode` fails on a release number or commit SHA in any plugin's non-test code.
- **Read at** a version's `Ref` (a tag, a commit SHA) and return the exact URL read as the snapshot's `SourceURL`: it's the events' citation.
- **Extract** sets `previous_version` to `from`'s version: every `api_operation` and `package_release` event must name it (the contract's rule 13). `from` is zero only for the first version read, which has nothing to compare: patch-it's hosted ingestion service reads that version as its first snapshot and extracts nothing from it, so a version pair always starts at a real `from` (an `api_operation` or `package_release` event extracted from zero would be rejected).

### A page with no versions: `plugin.Living`

A living page (a deprecations page, a feed: a `notices` source) only shows its latest state. The runner keeps the snapshot of the last read and hands it to `Read`:

- **Read** fetches the page. When it hashes as it did at `last` (`CompareHash`), return `last` as is: an unchanged page costs one GET (a 304 with the HTTP cache) and nothing else. Otherwise cite it (`kit.Cite`: a Wayback Machine snapshot that must say what the live page said) and return a new snapshot, dated now (`Version.ID` is the read's time, RFC 3339), whose content is the snapshot's bytes and whose `CompareHash` is the live page's.
- **Extract** from the kept snapshot to the new one is an event per notice the new one has that the kept one didn't (every one on a first read), citing the new snapshot. A dated notice (`model`, `ci_runtime`) has no `previous_version`; an `api_operation` notice names the read it was compared with (on a first read, its own).

### The `apisurface` kind: reading a package's API without running it

`plugins/apisurface` downloads a release's wheel (or sdist) through `env.Get`, opens it in memory (`archive/zip`, `archive/tar`: nothing is written to disk) and hands the `.py` and `.pyi` files to `plugins/apisurface/pysurface`, which reads them as text: logical lines, indentation, and the headers of `def`, `class`, `__all__` and `from … import`. Nothing in the package is imported, compiled or run. That's the safest way to read code someone else publishes, and it needs no dependency: a Python interpreter (the `ast` module) would mean running a second runtime next to the plugins, and the pure-Go Python parsers available follow old grammars and pull in a large module for what is a few hundred lines here. The cost is that only signatures are read, never what a body does, and anything the reader can't be sure of is left out (see `providers/openai-python-sdk.md`). Archive and file sizes are capped (`MaxFileBytes`, `MaxSourceBytes`), so a crafted release can't exhaust the runner.

An event cites the newer release's file, and its excerpt quotes the definition's path and line in both releases. npm packages (`.d.ts` files) are next.

### Register the provider and its source

In the kind's package `init`, with the metadata ([AGENTS.md](../AGENTS.md) rule 1):

```go
func init() {
    plugin.RegisterProvider(plugin.Provider{
        ID: "urllib3", Name: "urllib3", Category: plugin.CategoryPackages, Homepage: "https://urllib3.readthedocs.io",
        Sources: []plugin.Source{{
            ID: "urllib3", Layer: plugin.LayerPackage, Page: "providers/urllib3.md",
            Watches: "Names removed or deprecated in each release, …",
            URLs:    []string{"https://github.com/urllib3/urllib3/blob/main/CHANGES.rst"},
            Packages: []string{"pypi:urllib3"},
            Plugin:  Plugin{File: Urllib3},
        }},
    })
}
```

`RegisterProvider` panics on missing metadata, a duplicate id, or a provider registered twice with a different name, category or homepage (a provider with sources in two kinds repeats the same ones). `plugin.Lookup` and `plugin.LookupSource` find a source by its id, `provider/source`. The plugin's `Name` is that id: what every event carries in `api`, and what the fingerprint hashes.

`Packages` are the published packages the source covers, `<ecosystem>:<name>` (`pypi:stripe`, `npm:@vercel/sdk`): the package itself for a package or SDK source, the provider's official SDKs for an API source. `plugin.ForPackage` finds a package's sources, so a customer's lockfile maps to sources with no table kept anywhere else. List only names you checked; none is fine.

### Fetching: `env.Get` and `core/kit/`

- Fetch **only** with `env.Get`. It caches by `ETag` / `Last-Modified`, retries, and in tests serves saved files. `CI_go` fails a package outside `core/fetch/`, `core/kit/` and `plugins/openapi/adapter/` that imports `net/http`.
- `kit.Cite(ctx, archiver, url, read, want)`: for a **living page**'s `Read`. Takes a Wayback Machine snapshot, reads it with your own reader and checks it says exactly what the live page said. Make the events from the snapshot and cite its URL. No snapshot, no events.
- `plugin.SHA256(body)`: the hash a living page's `CompareHash` holds.
- `github.LatestRelease`, `github.LatestCommit`, `github.Commit`, `github.Head` ([`core/github`](../core/github/github.go)): the newest release or commit on GitHub, one place; don't decode GitHub's responses yourself.

### Events

The event format is [patch-it's change-event contract](https://github.com/patch-it-dev/patch-it/tree/main/docs/contract); `core/event/` is its Go binding. Every event has a typed `subject` (`api_operation`, `model`, `package_release`, `ci_runtime`), a `change_type` (`breaking`, `deprecation`, …), a `version`, a `fingerprint` (`event.Fingerprint`), and **a `source_url` and `source_excerpt`: no source, no event**. Call `Validate()`; an invalid event goes in `Result.Rejected`, not `Events`.

## 5. Save the source

Save a real copy of what the plugin reads under your package's `testdata/` (or the reader package's, like `plugins/notices/retire/testdata/`). Add a row to that directory's `README.md`: file, URL, date fetched, license, and a short quote of the part the plugin reads.

- **Openly licensed files** (MIT, BSD, Apache, CC BY): keep them whole.
- **Vendor documentation pages**: trim to what the plugin reads (the tables, the sections), copy every kept line unchanged, mark it "excerpt for testing; © the vendor", and say what you left out.
- **A big spec or changelog**: trim to the paths or entries the tests need, and check the trimmed copy gives the same findings as the full file.
  - A YAML or pretty-printed JSON spec can be trimmed by lines, every kept line unchanged.
  - When events point into the file by array index (`/12/paths/3/…`), keep every position and replace what isn't read with `{}`, so each pointer names the same entry as in the real file.

## 6. Write the golden test

The test kit, `core/kit/kittest`, runs the plugin with no network, no database and a fixed clock:

```go
func TestGolden(t *testing.T) {
    env := kittest.Env(t, kittest.Pages{
        Redis.listingURL("8.8-m03"):              "testdata/redis-contents-src-commands-8.8-m03.json",
        Redis.listingURL("8.8-rc1"):              "testdata/redis-contents-src-commands-8.8-rc1.json",
        Redis.FileURL("8.8-m03", "gcra.json"):    "testdata/redis-8.8-m03-gcra.json",
        Redis.FileURL("8.8-m03", "gcrasetvalue.json"): "testdata/redis-8.8-m03-gcrasetvalue.json",
    })
    got := kittest.Versions(t, env, Redis,
        plugin.Version{ID: "8.8-m03", Ref: "8.8-m03"}, plugin.Version{ID: "8.8-rc1", Ref: "8.8-rc1"})
    kittest.Golden(t, "testdata/golden/redis-commands.json", got)
}
```

- `kittest.Pages` maps each URL to a saved file; `kittest.Env` serves them with a fixed clock. Fetching any other URL fails the test.
- A versioned source: `kittest.Versions(t, env, plugin, from, to)` reads both versions with `ReadAt` and extracts the pair, as the hosted service does. When the older side needn't be read (a changelog's or a migration guide's `Extract` reads only the newer release's own section), build the snapshots yourself and call `kittest.Extracted(t, env, plugin, from, to)`, naming the real release before as `from`.
- A living page: `kittest.Page(t, env, plugin)` is its first read, cited through `&kittest.Archive{T: t, Pages: saved}`; a second read of the same page must return the kept snapshot and store nothing.
- Each stores into an in-memory store (`kittest.Mem`) and extracts **twice**: a rejected event, an error, or a second extract that stores anything new fails the test. `kittest.Pair` and `kittest.ReadPage` are the same steps returning errors, for tests that expect one.
- `kittest.Golden` compares the stored events with the JSON file. Create or rewrite it with `go test ./plugins/<name> -update`, then read the diff like code: every event in it is a pull request someone may get.

## 7. Pin a drop test and the registration

- **One drop test**: a realistic input the plugin must *not* turn into an event (an unrelated section, a behavior change written as prose, a product name in a sentence). Precision over recall: a missed change is a bug we fix later; a wrong change is a wrong pull request in someone's repo. Make it prove something: also run the same input with the filter off, and assert the noise it drops is there.
- **`registered_test.go`**: `kittest.Registered(t, Plugin{…})` checks the package registered that plugin under that name.
- **A new kind (a new package under plugins/)**: add its blank import to [`plugins/all/all.go`](../plugins/all/all.go). `TestEveryPluginIsImported` fails until you do; the hosted service runs exactly what `plugins/all` imports.

## 8. Write the provider page

Copy [`providers/_template.md`](../providers/_template.md) to `providers/<api>.md` and fill in every section: what it watches; **why this source** (the link, the date you checked it, why it's the best place this provider announces breaking changes, the alternatives you didn't use and why, one short quote); how the provider announces changes; what the plugin reads and ignores; known gaps.

## 9. Fill in the metadata in the registration

**Not optional** ([AGENTS.md](../AGENTS.md) rule 1). The `plugin.RegisterProvider` block the starter wrote is the only place the source is listed: the provider's id, name, category (`plugin.Categories()`) and homepage, and the source's id (`<provider>/<source>`), layer (`plugin.Layers()`), what it watches, the URLs it reads and its page. A second source of a provider repeats the provider's id, name, category and homepage exactly. `make plugins` prints the list with it; each release attaches that list as `plugins.json`, which docs.patch-it.dev and patch-it's hosted ingestion service read. There's nothing else to update.

## 10. Check and open the PR

```sh
make check                                # what CI runs, plus the leak guard
git commit -s -m "add the foo plugin"     # DCO sign-off
```

Fill in every box of the PR template. One plugin per PR.

Before opening it, have the work reviewed by someone who re-derives it, not by rereading your own diff:
- the history numbers, from the real source;
- every claim on the provider page and in comments, against the code and the data;
- whether any event would be wrong for a caller.

Most of the bugs found in plugins so far were found this way: a deprecation the provider later withdrew, an end of life read as a shutdown, a count that didn't match the data.
