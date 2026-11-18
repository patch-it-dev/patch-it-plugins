# patch-it-plugins

The source plugins for [patch-it](https://github.com/patch-it-dev/patch-it), the tool that turns third-party API changes into pull requests in your repo. Each plugin reads one provider's own sources (API specs, changelogs, deprecation pages, SDK migration guides, release feeds) and turns what changed into change events.

> **Under construction.** Plugins are being moved here from patch-it's hosted ingestion service. Until they land, this repo holds the rules, the CI and [the list of supported plugins](docs/plugins.md); the Go code follows.

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

## Supported plugins

See **[docs/plugins.md](docs/plugins.md)**. The same list is on [docs.patch-it.dev](https://docs.patch-it.dev/supported-apis/plugins/).

## Run the tests

Once the code has landed (Go, see `go.mod`):

```sh
go build ./...
go vet ./...
gofmt -l .           # prints nothing when formatted
go test -race ./...
```

The tests are offline: every plugin's test reads a saved copy of its source from `testdata/` and compares the events with a golden file.

## Contributing

New providers and fixes are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md) first (DCO sign-off, golden tests, adding your plugin to `docs/plugins.md`). AI coding agents: [AGENTS.md](AGENTS.md).

User docs for patch-it itself: [docs.patch-it.dev](https://docs.patch-it.dev).

## License

[Apache-2.0](LICENSE), copyright patch-it contributors.
