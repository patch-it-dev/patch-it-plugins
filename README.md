# patch-it-plugins

The source plugins for [patch-it](https://github.com/patch-it-dev/patch-it), the tool that turns third-party API changes into pull requests in your repo. Each plugin reads one provider's own sources (API specs, changelogs, deprecation pages, SDK migration guides, release feeds) and turns what changed into change events.

> **Just moved.** The plugins were moved here from patch-it's hosted ingestion service, which imports this module at a tagged release (`v0.x.y`) and runs them every day.

## How a plugin works

```
provider source ──fetch──▶ plugin ──▶ change events ──▶ patch-it's change feed ──▶ your CI
 (spec, page, feed)        reads, diffs,   (validated, each       (GET /v1/changes)       (patch-it opens
                           normalizes       with a source link                              the PR)
                                            and a quote)
```

1. **Fetch.** The plugin asks its environment for the source (`env.Get`); it never opens connections itself. Pages that change in place are also cited through a Web Archive snapshot, so the link in an event keeps showing what the plugin read.
2. **Compare.** It compares the source to the last version it saw (the baseline): two OpenAPI specs, two versions of a deprecations page, a new release's changelog section.
3. **Emit events.** Each change becomes one event in [patch-it's change-event format](https://github.com/patch-it-dev/patch-it/tree/main/docs/contract): what changed, how bad it is (breaking, deprecation, behavior change, non-breaking), the date it takes effect, and the source URL with a quote. **No source, no event.**

Plugins are small: most providers are an entry for a shared format plugin (`openapi`, `retirements`, `migrationguide`, `removedmembers`, `changelog`, `feed`), not new code. patch-it's hosted service runs every plugin daily and serves the events to the patch-it GitHub Action and CLI.

## Layout

| Path | What's there |
|---|---|
| [`plugin/`](plugin/) | The `Plugin` interface, the registry (`Register`, `Lookup`, `Names`), `Env`, `Result` |
| [`plugins/`](plugins/) | The format plugins: `openapi`, `retirements`, `migrationguide`, `removedmembers`, `changelog`, `feed`. `plugins/all` imports every one |
| [`kit/`](kit/), [`kit/kittest/`](kit/kittest/) | What plugins share (the unchanged-page check, Wayback citing) and the test kit (in-memory store, saved pages, golden files) |
| [`event/`](event/), [`migrate/`](migrate/) | The change-event types of [patch-it's contract](https://github.com/patch-it-dev/patch-it/tree/main/docs/contract), and the v1 → v2 conversion |
| [`diff/`](diff/), [`normalize/`](normalize/), [`hint/`](hint/), [`adapter/`](adapter/) | The OpenAPI diff engine: compare two specs, turn the changes into events, attach changelog hints; one adapter per spec provider |
| [`retire/`](retire/), [`sdkguide/`](sdkguide/), [`fetch/`](fetch/) | The deprecations-page and migration-guide readers; the HTTP fetch and cache |
| [`contract/fixtures/`](contract/) | A copy of patch-it's change-event fixtures |
| [`providers/`](providers/) | One page per provider: why that source, what the plugin reads |
| [`docs/`](docs/) | [The plugin list](docs/plugins.md) and [how to write a plugin](docs/writing-a-plugin.md) |

## Supported plugins

See **[docs/plugins.md](docs/plugins.md)**. The same list is on [docs.patch-it.dev](https://docs.patch-it.dev/supported-apis/plugins/).

## Run the tests

Go (the version in `go.mod`), no other setup:

```sh
go build ./...
go vet ./...
gofmt -l .           # prints nothing when formatted
go test -race ./...
```

The tests are offline: every plugin's test reads a saved copy of its source from `testdata/` and compares the events with a golden file. One package: `go test ./plugins/feed`. After an intended change to a plugin's output, `go test ./plugins/<name> -update` rewrites its golden files; review that diff like code.

### Environment

| Variable | Needed | What it does |
|---|---|---|
| `GITHUB_TOKEN` | No | Optional. A GitHub token (no scopes needed) sent to the GitHub API when the `openapi` adapters look up release tags (Stripe's and Datadog's releases, GitHub's API versions). It only raises GitHub's rate limit for unauthenticated calls (60 an hour per IP); the tests never use it. |

Nothing else is read from the environment.

## Contributing

New providers and fixes are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md) first (DCO sign-off, golden tests, adding your plugin to `docs/plugins.md`). AI coding agents: [AGENTS.md](AGENTS.md).

User docs for patch-it itself: [docs.patch-it.dev](https://docs.patch-it.dev).

## License

[Apache-2.0](LICENSE), copyright patch-it contributors.
