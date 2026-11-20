# How it works

A **provider** (Stripe, OpenAI, GitHub…) has one category and one or more **sources**: each one place it announces changes (an OpenAPI spec repo, a changelog, a deprecations page, a feed), read by one **kind** of plugin (`plugins/<kind>/`), at one **layer** of a customer's stack. A source's id is `provider/source` (`stripe/api`, `openai/models`); its old slug (`stripe`, `openai-models`) is what events carry in `api` until patch-it's change-event contract switches to source ids. Every provider and source is registered in code with its metadata (`plugin.RegisterProvider`): that registration is the only list.

## What a plugin does

0. **List the versions.** A source with versions (a spec's releases, a package on PyPI or npm, a guide per major) lists every one, oldest first (`ListVersions`, from `core/versions`: GitHub releases, tags and commits, PyPI, npm, a living page's Web Archive captures). Each consecutive pair is read and compared once (`ReadAt`, `Extract`), so a customer several versions behind gets every step of the upgrade. A living page with no versions (a deprecations page, a feed) is read as it is now; each change to it is a dated version, and its past ones are its Web Archive captures.
1. **Fetch.** The plugin asks its environment for the source (`env.Get`); it never opens connections itself. Pages that change in place are also cited through a Web Archive snapshot, so the link in an event keeps showing what the plugin read.
2. **Compare.** It compares two versions of the source: two OpenAPI specs, two versions of a deprecations page, a release's changelog section against the release before. `Run` does the newest version against the last one it saw (the baseline); `Extract` does any pair.
3. **Emit events.** Each change becomes one event in [patch-it's change-event format](https://github.com/patch-it-dev/patch-it/tree/main/docs/contract): what changed, how bad it is (breaking, deprecation, behavior change, non-breaking), the date it takes effect, and the source URL with a quote. **No source, no event.** When unsure, a plugin emits nothing: a wrong event would become a wrong pull request.

Most providers are an entry for an existing kind, not new code. patch-it's hosted ingestion service runs every source on a schedule and serves the events to the patch-it GitHub Action and CLI.

## What's covered

There's no list in this repo: `make plugins` prints every provider and source from the registry as JSON, every release attaches it as `plugins.json`, and [docs.patch-it.dev](https://docs.patch-it.dev/supported-apis/plugins/) shows it.

## Layout

| Path | What's there |
|---|---|
| [`core/plugin/`](../core/plugin/) | The `Plugin` interface (`Run`; `ListVersions`, `ReadAt`, `Extract`), the registry of providers and sources (`RegisterProvider`, `Lookup`, `LookupSource`, `Providers`), `Env`, `Result` |
| [`core/versions/`](../core/versions/) | Every version of a source: GitHub releases, tags and commits, PyPI, npm, a page's Web Archive captures (all, or the one at a date) |
| [`plugins/`](../plugins/) | One package per kind, with its providers (openapi, changelog, notices, migrationguide, catalog, apisurface); `plugins/all` imports every one |
| [`cmd/new-plugin/`](../cmd/new-plugin/), [`cmd/list-plugins/`](../cmd/list-plugins/) | The starter for a new provider; the list of providers and sources, printed from the registry |
| [`core/kit/`](../core/kit/), [`core/kit/kittest/`](../core/kit/kittest/), [`core/github/`](../core/github/), [`core/extract/`](../core/extract/) | What plugins share (the unchanged-page check, Wayback citing, the newest release or commit on GitHub, RSS/Atom and HTML text helpers) and the test kit (in-memory store, saved pages, golden files) |
| [`core/event/`](../core/event/), [`core/migrate/`](../core/migrate/) | The change-event types of [patch-it's contract](https://github.com/patch-it-dev/patch-it/tree/main/docs/contract), and the v1 → v2 conversion |
| [`core/diff/`](../core/diff/), [`core/normalize/`](../core/normalize/), [`core/hint/`](../core/hint/), [`plugins/openapi/adapter/`](../plugins/openapi/adapter/) | The OpenAPI diff engine: compare two specs, turn the changes into events, attach changelog hints; one adapter per spec provider |
| [`plugins/notices/retire/`](../plugins/notices/retire/), [`plugins/notices/feed/`](../plugins/notices/feed/), [`plugins/notices/eol/`](../plugins/notices/eol/), [`plugins/migrationguide/sdkguide/`](../plugins/migrationguide/sdkguide/), [`plugins/migrationguide/members/`](../plugins/migrationguide/members/), [`core/fetch/`](../core/fetch/) | The notices readers (deprecations pages, feeds, end-of-life tables) and the migration-guide readers (removed parameters, removed members); the HTTP fetch and cache |
| [`contract/fixtures/`](../contract/) | A copy of patch-it's change-event fixtures |
| `plugins/removedmembers/`, `plugins/apichangelog/`, `plugins/retirements/`, `plugins/feed/`, `plugins/eol/` | Deprecated: the old import paths, forwarding to the packages above until code that imports them moves |
| [`providers/`](../providers/) | One page per provider: why that source, what the plugin reads |
| [`docs/`](./) | [How to write a plugin](writing-a-plugin.md), the [README template](README-template.md) patch-it's public repos share |

## Run the checks

```sh
make check           # what CI runs: tidy, build, gofmt, vet, import and net/http guards, go test -race, no TODO(new-plugin) left, leak guard
```

The tests are offline: every plugin's test reads a saved copy of its source from `testdata/` and compares the events with a golden file. After an intended change to a plugin's output, `go test ./plugins/<name> -update` rewrites its golden files; review that diff like code.

`GITHUB_TOKEN` (optional, no scopes) only raises GitHub's rate limit when the `openapi` adapters look up release tags; the tests never use it. Nothing else is read from the environment.

## Releases

Releases are `v0.x.y` tags on `main` with a [GitHub release](https://github.com/patch-it-dev/patch-it-plugins/releases) and generated notes. The **Release** workflow tags a patch release on its own every Monday and Thursday at 14:00 UTC when something new was merged, and maintainers can run it by hand ([how](../CONTRIBUTING.md#releasing)). patch-it's hosted ingestion service runs the latest release, so a merged plugin is running within a few days. Each release carries `plugins.json`, the list of providers and sources printed from the registry at that tag. Tags are never moved or deleted.

```sh
go get github.com/patch-it-dev/patch-it-plugins@latest
```
