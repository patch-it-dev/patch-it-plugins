# Writing a plugin

A plugin watches one provider source and turns what changed into change events. This guide walks through adding one, from picking a source to the PR. The rules are in [CONTRIBUTING.md](../CONTRIBUTING.md); this is the how-to.

**Most new providers don't need new code.** They are a new entry for an existing format plugin: a new `changelog.File`, a new `retire.Page`, a new `feed.Feed` reader, a new OpenAPI adapter. Write a new format plugin only when none of them reads your source.

## 1. Scaffold it with `new-plugin`

Pick the source and the format plugin that reads it (steps 2 and 3), then let the starter write everything around the provider:

```sh
go run ./cmd/new-plugin --api acme --provider Acme --format openapi
# --format: openapi, retirements, migrationguide, removedmembers, changelog, feed, apichangelog, catalog
# optional: --source https://... (the provider's source), --watches "...", --layer API|SDK|package|runtime|CI|"AI models",
#           --page acme (a second plugin of a provider shares its page in providers/)
```

It writes, and lists:

- the provider's definition, in the package the format reads it from (`plugins/openapi/adapter/`, `plugins/retirements/retire/`, `plugins/migrationguide/sdkguide/` or `plugins/<format>/`), and its `plugin.Register` line in that format's `init` (plus the `core/migrate/` map entry that `retirements` and `migrationguide` need);
- `plugins/<format>/<api>_test.go`: the registration test, a golden test and a pinned drop test, over placeholder sources in `testdata/`, with the golden file written from them;
- a row per saved file in that `testdata/README.md` (URL, date, license, quote to fill in);
- `providers/<api>.md` from the template (or nothing, with `--page`, when the provider already has one);
- the row in `docs/plugins.md`, and `docs/plugins.json` regenerated.

`go test ./...` passes right after; every place to fill in says `TODO(new-plugin)` (`grep -rn 'TODO(new-plugin)' .`), and `make check` (its `todos` step, also in CI_go) fails until none is left. It refuses an id that's taken (registered, in `docs/plugins.md`, or a file it would write exists) and then changes nothing. The steps below are what you fill in.

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

### Pin, and pick a version a repo pins

- **Cite what never changes:** a release tag, or the commit SHA of the release or of a branch's head. A moving branch or a live URL isn't a citation.
- **"Newest" needs care:**
  - a day can have several releases whose tag suffix says nothing about their order (break the tie by commit date);
  - an older major's patch can be released after a newer minor (take the highest version, not GitHub's latest release);
  - the newest branch may be unreleased (read the newest *released* one).
- **`version`** is the API version the change arrived in, per the contract. If the API has none, use a package version a repo pins (an SDK release), or a date: a release date, an end of life.
  - The patch-it client gates a package version as a package. It keeps a date-versioned API's events only if the API is in its list of date-versioned APIs, so **a new date-versioned API needs an entry there too**.
  - Two specs of one version (fixes on a release branch) aren't a release: `RemovalsNeedANewVersion` drops removals between them.

## 3. Pick the format plugin

| Format plugin | Reads | Add a provider by |
|---|---|---|
| `plugins/openapi` | An OpenAPI spec per release, diffed with `core/diff/` and turned into events by `core/normalize/` | An `adapter.Adapter` in `plugins/openapi/adapter/` (latest tag, fetch a tag's spec) and a `plugin.Register(Plugin{Adapter: …})` line |
| `plugins/retirements` | A model provider's deprecations page (a table of date, model, replacement) | A `retire.Page` in `plugins/retirements/retire/providers.go` |
| `plugins/migrationguide` | An SDK's migration guide at a release tag (removed request parameters) | A `sdkguide.Guide` in `plugins/migrationguide/sdkguide/` |
| `plugins/removedmembers` | A package's table of removed members at a release tag (an RST simple table or a Markdown pipe table) | A `removedmembers.Guide` |
| `plugins/changelog` | A `CHANGELOG.md` / `CHANGES.rst` / `HISTORY.rst` at a release tag | A `changelog.File` |
| `plugins/feed` | An RSS or Atom feed | A `feed.Feed` with a reader for that vendor's entries |
| `plugins/catalog` | A flat JSON catalog, one entry per name, with no release tags to pin — a commit SHA stands in | A `catalog.Catalog`, diffed between two snapshots for names that disappeared entirely |
| `plugins/apichangelog` | A vendor's machine-readable API changelog in JSON (per day, endpoint and API version, oasdiff change codes), at the commit that last changed it | An `apichangelog.Log` |
| `plugins/eol` | A vendor's own table of whole API version ranges and their support status (supported, deprecated, retired) | An `eol.Page`; `--format` doesn't scaffold this one yet, write it by hand |

## 4. The interface

A plugin implements `plugin.Plugin` (package [`core/plugin/`](../core/plugin/plugin.go)):

```go
type Plugin interface {
    Name() string                                    // "stripe": the --api id and every event's `api`
    Run(ctx context.Context, env Env) (Result, error) // what changed since the baseline it saved last
}

// Optional: diff two given releases, to recreate a historical change.
type Replayer interface {
    Replay(ctx context.Context, env Env, from, to string) (Result, error)
}

type Env struct {
    Baselines BaselineReader                               // LastSpec(api): what the last run saved
    Get       func(ctx context.Context, url string) ([]byte, error) // the only way to fetch
    Now       func() time.Time
}

type Result struct {
    Label     string          // what was read, for the run summary
    Events    []event.EventV2 // valid events
    Rejected  []error         // changes that failed to become valid events
    Baselines []Spec          // what was read, saved after the events
}
```

A plugin **stores nothing**. It reads its source, compares it with its last baseline (`env.Baselines.LastSpec(api)`), and returns events and the new baseline. Whoever runs it (patch-it's hosted ingestion service, or the test kit) stores the events first and the baseline after, so a crash in between re-reads the same change next time and the fingerprints dedupe it.

Register the plugin in its package's `init`:

```go
func init() {
    plugin.Register(Plugin{File: Textual0860})
}
```

`Register` panics on a duplicate name or the name `all`.

### Fetching: `env.Get` and `core/kit/`

- Fetch **only** with `env.Get`. It caches by `ETag` / `Last-Modified`, retries, and in tests serves saved files. `CI_go` fails a package outside `core/fetch/`, `core/kit/` and `plugins/openapi/adapter/` that imports `net/http`.
- `kit.Unchanged(env, api, body)`: true when the page hashes to the last baseline. Return early: an unchanged source costs one GET and nothing else.
- `kit.Cite(ctx, archiver, url, read, want)`: for a **live page**. Takes a Wayback Machine snapshot, reads it with your own reader and checks it says exactly what the live page said. Make the events from the snapshot and cite its URL. No snapshot, no events.
- `plugin.Baseline(api, version, tag, sourceURL, content, replay, now)`: the baseline to return.
- Older format plugins build v1 `event.ChangeEvent`s and convert them with `migrate.Fresh`; new code builds `event.EventV2` directly.

### Events

The event format is [patch-it's change-event contract](https://github.com/patch-it-dev/patch-it/tree/main/docs/contract); `event/` is its Go binding. Every event has a typed `subject` (`api_operation`, `model`, `package_release`, `ci_runtime`), a `change_type` (`breaking`, `deprecation`, …), a `version`, a `fingerprint` (`event.FingerprintV2`), and **a `source_url` and `source_excerpt`: no source, no event**. Call `Validate()`; an invalid event goes in `Result.Rejected`, not `Events`.

## 5. Save the source

Save a real copy of what the plugin reads under your package's `testdata/` (or the reader package's, like `plugins/retirements/retire/testdata/`). Add a row to that directory's `README.md`: file, URL, date fetched, license, and a short quote of the part the plugin reads.

- **Openly licensed files** (MIT, BSD, Apache, CC BY): keep them whole.
- **Vendor documentation pages**: trim to what the plugin reads (the tables, the sections), copy every kept line unchanged, mark it "excerpt for testing; © the vendor", and say what you left out.
- **A big spec or changelog**: trim to the paths or entries the tests need, and check the trimmed copy gives the same findings as the full file.
  - A YAML or pretty-printed JSON spec can be trimmed by lines, every kept line unchanged.
  - When events point into the file by array index (`/12/paths/3/…`), keep every position and replace what isn't read with `{}`, so each pointer names the same entry as in the real file.

## 6. Write the golden test

The test kit, `core/kit/kittest`, runs the plugin with no network, no database and a fixed clock:

```go
func TestGolden(t *testing.T) {
    got := kittest.Ingest(t, Plugin{File: Textual0860},
        kittest.Pages{Textual0860.URL(): "testdata/textual-v0.86.0-CHANGELOG.md"})
    kittest.Golden(t, "testdata/golden/textual-0.86.0.json", got)
}
```

- `kittest.Pages` maps each URL to a saved file. Fetching any other URL fails the test.
- `kittest.Ingest` runs the plugin **twice** over an in-memory store (`kittest.Mem`): a rejected event, an error, or a second run that stores anything new fails the test.
- `kittest.Golden` compares the stored events with the JSON file. Create or rewrite it with `go test ./plugins/<name> -update`, then read the diff like code: every event in it is a pull request someone may get.
- A live page: build the env with `kittest.Env`, cite through `&kittest.Archive{T: t, Pages: saved}` and run with `kittest.IngestWith`.

## 7. Pin a drop test and the registration

- **One drop test**: a realistic input the plugin must *not* turn into an event (an unrelated section, a behavior change written as prose, a product name in a sentence). Precision over recall: a missed change is a bug we fix later; a wrong change is a wrong pull request in someone's repo. Make it prove something: also run the same input with the filter off, and assert the noise it drops is there.
- **`registered_test.go`**: `kittest.Registered(t, Plugin{…})` checks the package registered that plugin under that name.
- **A new format plugin package**: add its blank import to [`plugins/all/all.go`](../plugins/all/all.go). `TestEveryPluginIsImported` fails until you do; the hosted service runs exactly what `plugins/all` imports.

## 8. Write the provider page

Copy [`providers/_template.md`](../providers/_template.md) to `providers/<api>.md` and fill in every section: what it watches; **why this source** (the link, the date you checked it, why it's the best place this provider announces breaking changes, the alternatives you didn't use and why, one short quote); how the provider announces changes; what the plugin reads and ignores; known gaps.

## 9. Add the row to `docs/plugins.md`

**Not optional** ([AGENTS.md](../AGENTS.md) rule 1). Add one row between the `plugins:start` and `plugins:end` markers: the id linked to its provider page, the provider, what it watches, the format plugin, the source, the status. Absolute links only: the docs site copies the table as it is. Changing a plugin's id, source or what it watches updates its row in the same PR. Then run `make plugins-json` to regenerate `docs/plugins.json`; it fails, naming the problem, if the registry and the table disagree.

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
