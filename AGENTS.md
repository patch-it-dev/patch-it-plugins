# AGENTS.md

Rules for AI coding agents (and humans) working in this repo. `CLAUDE.md` imports this file.

## Rule 1: every new or changed provider or source has complete metadata in its registration

**A source exists when its provider's `providers/<provider>/provider.go` registers it with `plugin.RegisterProvider` (once per provider, every source of it, whatever the kind), and its metadata lives only there: the provider's id, name, category and homepage, and each source's id, layer, what it watches, the URLs it reads and the packages it covers (`pypi:stripe`). Its page is the provider's, `providers/<provider>/README.md`, one section per source.** Adding, changing or removing a source is a change to that registration (and to its section of the page) in the same PR. There is no list to update: nothing in this repo, the docs site or anywhere else keeps one by hand.

`go run ./cmd/list-plugins` prints the list from the registry (JSON), and the Release workflow attaches it to every release as `plugins.json`; the docs site and patch-it's hosted ingestion service read it from there. `RegisterProvider` refuses incomplete metadata or a second registration of a provider, and `go test` fails when a provider's folder isn't imported by `providers/all`, or a source isn't named on its provider's page, but no check can tell whether the words are right: you are that check. Say in the PR that you read them.

## Rule 2: this repo is public

Public text (README, docs, descriptions) follows [docs/README-template.md](docs/README-template.md): its order, its words, its layer statuses.

Everything you write here is public forever, including commit messages, branch names, PR titles and descriptions, and test data. Never write:

- credentials, tokens, keys, or values from any `.env`;
- internal hostnames, internal service names, or links to private repos, dashboards or tickets;
- names or emails of customers, users or leads, or anything about sales or outreach;
- links to AI chat or agent sessions.

Refer to the service that runs these plugins as "patch-it's hosted ingestion service". `CI_leak_guard` checks every PR for secrets and for a list of known internal names (committed only as SHA-256 hashes), but it can't catch everything.

## Rules for plugin code

- **How to read is a kind; what to read is a provider.** `kinds/<kind>/` reads one kind of source, written once, and imports no provider. `providers/<provider>/` holds one provider and nothing about another: `provider.go` (its registration, built from the kinds' exported types), `README.md` (its page), any code only it needs (`<source>.go`: an OpenAPI adapter, a feed's reader), `testdata/` and its tests. A provider imports only `core/` and `kinds/`. No folder per source, no forwarders: when code moves, the old path goes in the same change, and no package keeps a type alias (`type X = …`), a re-export (`var X = otherpkg.Y`) or a func that only calls another package's (a genuine one says why with `alias-ok: <reason>` in its doc comment). `go test ./providers/all` checks all of this.

- **Start a new source with `make new-plugin ARGS='--provider <id> --name <Name> --source <id> --kind <kind>'`** (`go run ./cmd/new-plugin`; [which kind](docs/which-kind.md)): it writes into `providers/<id>/`, a new folder or one more source in the existing one. Then fill in every `TODO(new-plugin)`; `make todos` (part of `make check` and CI_go) fails while one is left.
- **Fetch only through the environment** (`env.Get` or `core/kit/`). No `net/http` outside the fetch layer, no filesystem outside `testdata/`, no database.
- **Golden test + dated source quote**: a saved copy of the real source in `providers/<provider>/testdata/`, a row in its `testdata/README.md` (URL, date fetched, license, quote), and `testdata/golden/<provider>-<source>.json` with the exact events.
- **Precision over recall.** When unsure whether something is a change, don't emit an event.
- **One pinned drop test** per source: a realistic input that must produce no event.
- **A section of `providers/<provider>/README.md`** for every source (a new provider's page from `docs/provider-template.md`), including **Why this source**: the link, the date it was checked, why it's the best source, the alternatives not used, and a short quote.
- **No source, no event**: every event carries the source URL and a quote.
- Don't edit golden files to make a test pass without explaining in the PR why the new output is right.

## Commands

**Run `make check` before you say you're done.** It runs exactly what `CI_go` runs (each CI step is a Makefile target) plus the internal-names check over every tracked file; if it passes locally, those checks pass in CI.

```sh
make check          # tidy, build, gofmt, vet, the import allowlist, the net/http guard, go test -race, no TODO(new-plugin) left, leak guard
make check PATCH_IT=../patch-it   # also CI_contract: contract/fixtures/ against a patch-it checkout
make plugins        # print the list of providers and sources from the registry (what each release attaches as plugins.json)
make new-plugin ARGS='--provider acme --source api --kind openapi'   # scaffold a source (asks for what's missing in a terminal)
```

One target at a time: `make tidy`, `make build`, `make fmt`, `make vet`, `make imports`, `make nethttp`, `make test`, `make todos`, `make leak-guard`, `make contract`. Use `make GO=/path/to/go check` for another Go.

## Layout

Go module `github.com/patch-it-dev/patch-it-plugins`; the only dependency is `gopkg.in/yaml.v3`.

```
core/plugin/           the Plugin interface (Extract) in its two shapes, Versioned (ListVersions, ReadAt) and Living (Read);
                       ErrNothingToRead; the shape guard (Shape, anchors, ErrShapeChanged); the registry, Env, Result
core/fetch/            the HTTP fetch with retries and the ETag / Last-Modified cache
core/github/           the newest release / commit on GitHub, and a commit's SHA and date (one place)
core/versions/         every version of a source: GitHub releases, tags and commits, PyPI, npm
core/kit/  core/kit/kittest/   Cite, Wayback, Notices; the test kit (Mem, Pages, Versions, Extracted, Page, Golden)
core/extract/          what readers of prose share: RSS/Atom entries, HTML as text, dates, code spans
core/event/            the change event of patch-it's contract: types, fingerprint, validation
core/diff/ core/normalize/ core/hint/   the OpenAPI diff engine and its helpers
kinds/<kind>/          how to read, one package per kind, written once, no provider in it: openapi (+ adapter/,
                       the interface a provider's spec adapter implements), notices (+ retire/, feed/, eol/, lifecycle/),
                       migrationguide (+ sdkguide/, members/), changelog (Markdown, RST, GitHub release notes
                       or JSON), catalog, apisurface (+ pysurface/, dtsurface/); each with a <kind>test/ package
                       its providers' tests share
providers/<provider>/  what to read, one folder per provider: provider.go (its RegisterProvider: the provider
                       and every source), README.md (its page, a section per source), <source>.go only for code
                       only it needs, testdata/ (+ README.md, golden/), its golden and drop tests
providers/all/         imports every provider; its tests check the folders, the metadata and the import graph
cmd/list-plugins/      prints the list from the registry (make plugins); internal/pluginlist/ is its logic
cmd/new-plugin/        scaffolds a source into providers/<provider>/ (make new-plugin ARGS=...; go run ./cmd/new-plugin --help)
contract/fixtures/     a copy of patch-it's change-event fixtures (CI_contract checks it)
Makefile               make check (what CI_go runs, plus the leak guard), make plugins
docs/                  contributor docs, including writing-a-plugin.md and provider-template.md
scripts/               leak_guard.py (hashed internal names), check_contract.py
.github/               CODEOWNERS, PR template, release-notes categories, workflows
                       (CI_go, CI_dco, CI_leak_guard, CI_contract, Release)
```

The [change-event contract](https://github.com/patch-it-dev/patch-it/tree/main/docs/contract) is owned by patch-it: don't change `contract/fixtures/` here first.

## Commits and PRs

- Sign off every commit: `git commit -s` (DCO; `CI_dco` checks it).
- Subjects short, lowercase, imperative: `add the foo plugin`.
- One plugin or fix per PR; fill in every checkbox of the PR template, including "metadata complete in the registration".
- AI-assisted commits keep their co-author trailer.
- Never re-run, dispatch or retrigger CI to "see if it passes"; fix the cause and push.

## Releases

Agents never tag or release. The **Release** workflow tags a patch release on its own every Monday and Thursday when something new was merged, and a maintainer can run it by hand (see [CONTRIBUTING.md](CONTRIBUTING.md#releasing)). Never delete or move a tag.
