# How it works

A **provider** (Stripe, OpenAI, GitHub…) has one category and one or more **sources**: each one place it announces changes (an OpenAPI spec repo, a changelog, a deprecations page, a feed), read by one **kind** of plugin, at one **layer** of a customer's stack. A source's id is `provider/source` (`stripe/api`, `openai/models`): what its events carry in `api`.

The code is split the same way. **How to read** a kind of source is written once, in `kinds/<kind>/`. **What to read** for a provider is in its own folder, `providers/<provider>/`: one `provider.go` registers the provider and every source of it, whatever their kinds, with its metadata (`plugin.RegisterProvider`), built from the kinds' exported types; code only that provider needs (an OpenAPI adapter, a feed's reader) sits beside it, with its saved pages, golden files, tests and one page, `README.md`. That registration is the only list.

## What a plugin does

0. **List the versions.** A source with versions (a spec's releases, a package on PyPI or npm, a guide per major) lists every one, oldest first (`ListVersions`, from `core/versions`: GitHub releases, tags and commits, PyPI, npm). Each consecutive pair is read and compared once (`ReadAt`, `Extract`), so a customer several versions behind gets every step of the upgrade. A living page with no versions (a deprecations page, a feed) is read as it is now (`Read`), against the snapshot kept from the read before; each read that changed is a dated version.
1. **Fetch.** The plugin asks its environment for the source (`env.Get`); it never opens connections itself. Pages that change in place are also cited through a Web Archive snapshot, so the link in an event keeps showing what the plugin read.
2. **Compare.** It compares two versions of the source: two OpenAPI specs, two versions of a deprecations page, a release's changelog section against the release before. `Extract` does it for any pair: two versions, or a living page's last read and today's.
3. **Emit events.** Each change becomes one event in [patch-it's change-event format](https://github.com/patch-it-dev/patch-it/tree/main/docs/contract): what changed, how bad it is (breaking, deprecation, behavior change, non-breaking), the date it takes effect, and the source URL with a quote. **No source, no event.** When unsure, a plugin emits nothing: a wrong event would become a wrong pull request.

Most sources are an entry in their provider's `provider.go` on an existing kind, not new code. patch-it's hosted ingestion service runs every source on a schedule and serves the events to the patch-it GitHub Action and CLI.

## What's covered

There's no list in this repo: `make plugins` prints every provider and source from the registry as JSON, every release attaches it as `plugins.json`, and [docs.patch-it.dev](https://docs.patch-it.dev/supported-apis/plugins/) shows it.

## Layout

| Path | What's there |
|---|---|
| [`core/plugin/`](../core/plugin/) | The `Plugin` interface (`Extract`) in its two shapes, `Versioned` (`ListVersions`, `ReadAt`) and `Living` (`Read`); `ErrNothingToRead`; the shape guard every kind runs (`Shape`, anchors, `ErrShapeChanged`); the registry of providers and sources (`RegisterProvider`, `Lookup`, `LookupSource`, `Providers`), `Env`, `Result` |
| [`core/versions/`](../core/versions/) | Every version of a source: GitHub releases, tags and commits, PyPI, npm |
| [`kinds/`](../kinds/) | How to read: one package per kind (openapi, changelog, notices, migrationguide, catalog, apisurface), each with its readers and a test helper package; no kind imports a provider |
| [`providers/`](../providers/) | What to read: one folder per provider, `providers/<provider>/` with `provider.go` (its registration), `README.md` (its page: why each source, what it reads), any code only it needs, `testdata/` and its tests; [`providers/all`](../providers/all/) imports every one, and its tests check the folders and the import graph |
| [`cmd/new-plugin/`](../cmd/new-plugin/), [`cmd/list-plugins/`](../cmd/list-plugins/) | The starter for a new provider or source (into `providers/<provider>/`); the list of providers and sources, printed from the registry |
| [`core/kit/`](../core/kit/), [`core/kit/kittest/`](../core/kit/kittest/), [`core/github/`](../core/github/), [`core/extract/`](../core/extract/) | What plugins share (Wayback citing, the newest release or commit on GitHub, RSS/Atom and HTML text helpers) and the test kit (in-memory store, saved pages, golden files) |
| [`core/event/`](../core/event/) | The change event of [patch-it's contract](https://github.com/patch-it-dev/patch-it/tree/main/docs/contract): its types, fingerprint and validation |
| [`core/diff/`](../core/diff/), [`core/normalize/`](../core/normalize/), [`core/hint/`](../core/hint/), [`kinds/openapi/adapter/`](../kinds/openapi/adapter/) | The OpenAPI diff engine: compare two specs, turn the changes into events, attach changelog hints; the interface each provider's spec adapter implements, and what adapters share |
| [`kinds/notices/retire/`](../kinds/notices/retire/), [`kinds/notices/feed/`](../kinds/notices/feed/), [`kinds/notices/eol/`](../kinds/notices/eol/), [`kinds/notices/lifecycle/`](../kinds/notices/lifecycle/), [`kinds/migrationguide/sdkguide/`](../kinds/migrationguide/sdkguide/), [`kinds/migrationguide/members/`](../kinds/migrationguide/members/), [`core/fetch/`](../core/fetch/) | The notices readers (deprecations pages, feeds, end-of-life tables, runtime lifecycle tables) and the migration-guide readers (removed parameters, removed members); the HTTP fetch and cache |
| [`contract/fixtures/`](../contract/) | A copy of patch-it's change-event fixtures |
| [`docs/`](./) | [How to write a plugin](writing-a-plugin.md), the [provider page template](provider-template.md), the [README template](README-template.md) patch-it's public repos share |

## Run the checks

```sh
make check           # what CI runs: tidy, build, gofmt, vet, import and net/http guards, go test -race, no TODO(new-plugin) left, leak guard
```

The tests are offline: every source's test reads a saved copy of it from its provider's `testdata/` and compares the events with a golden file. After an intended change to a source's output, `go test ./providers/<provider> -update` rewrites its golden files; review that diff like code.

`GITHUB_TOKEN` (optional, no scopes) only raises GitHub's rate limit when the OpenAPI adapters look up release tags; the tests never use it. Nothing else is read from the environment.

## Releases

Releases are `v0.x.y` tags on `main` with a [GitHub release](https://github.com/patch-it-dev/patch-it-plugins/releases) and generated notes. The **Release** workflow tags a patch release on its own every Monday and Thursday at 14:00 UTC when something new was merged, and maintainers can run it by hand ([how](../CONTRIBUTING.md#releasing)). patch-it's hosted ingestion service runs the latest release, so a merged plugin is running within a few days. Each release carries `plugins.json`, the list of providers and sources printed from the registry at that tag. Tags are never moved or deleted.

```sh
go get github.com/patch-it-dev/patch-it-plugins@latest
```
