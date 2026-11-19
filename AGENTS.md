# AGENTS.md

Rules for AI coding agents (and humans) working in this repo. `CLAUDE.md` imports this file.

## Rule 1: every new or changed plugin goes in `docs/plugins.md`

**If you add a plugin, add its row to [`docs/plugins.md`](docs/plugins.md) in the same PR. If you change a plugin's id, provider, source, what it watches, format plugin or status, update its row in the same PR. If you remove one, remove its row.**

The list of supported APIs on docs.patch-it.dev is generated from that file, so a plugin missing from it doesn't exist as far as users can tell. Then run `make plugins-json`: it regenerates [`docs/plugins.json`](docs/plugins.json) from the plugin registry and the table, and fails if a registered plugin has no row (or a row has no plugin, a wrong format or no provider page). `go test` fails while `docs/plugins.json` is out of date. The check can't tell whether a row's words are right: you are that check. Say in the PR that you compared them.

Keep the table between the `<!-- plugins:start -->` and `<!-- plugins:end -->` markers a plain Markdown table, one row per plugin, absolute links only (the docs site copies it verbatim).

## Rule 2: this repo is public

Everything you write here is public forever, including commit messages, branch names, PR titles and descriptions, and test data. Never write:

- credentials, tokens, keys, or values from any `.env`;
- internal hostnames, internal service names, or links to private repos, dashboards or tickets;
- names or emails of customers, users or leads, or anything about sales or outreach;
- links to AI chat or agent sessions.

Refer to the service that runs these plugins as "patch-it's hosted ingestion service". `CI_leak_guard` checks every PR for secrets and for a list of known internal names (committed only as SHA-256 hashes), but it can't catch everything.

## Rules for plugin code

- **Fetch only through the environment** (`env.Get` or `kit/`). No `net/http` outside the fetch layer, no filesystem outside `testdata/`, no database.
- **Golden test + dated source quote**: a saved copy of the real source in `testdata/`, a row in `testdata/README.md` (URL, date fetched, license, quote), and `testdata/golden/<api>.json` with the exact events.
- **Precision over recall.** When unsure whether something is a change, don't emit an event.
- **One pinned drop test** per plugin: a realistic input that must produce no event.
- **`providers/<api>.md`** for every provider, copied from `providers/_template.md`, including **Why this source**: the link, the date it was checked, why it's the best source, the alternatives not used, and a short quote.
- **No source, no event**: every event carries the source URL and a quote.
- Don't edit golden files to make a test pass without explaining in the PR why the new output is right.

## Commands

**Run `make check` before you say you're done.** It runs exactly what `CI_go` runs (each CI step is a Makefile target) plus the internal-names check over every tracked file; if it passes locally, those checks pass in CI.

```sh
make check          # tidy, build, gofmt, vet, the import allowlist, the net/http guard, go test -race, leak guard
make check PATCH_IT=../patch-it   # also CI_contract: contract/fixtures/ against a patch-it checkout
make plugins-json   # regenerate docs/plugins.json after adding or changing a plugin or its row
```

One target at a time: `make tidy`, `make build`, `make fmt`, `make vet`, `make imports`, `make nethttp`, `make test`, `make leak-guard`, `make contract`. Use `make GO=/path/to/go check` for another Go.

## Layout

Go module `github.com/patch-it-dev/patch-it-plugins`; the only dependency is `gopkg.in/yaml.v3`.

```
plugin/                the Plugin interface, the registry, Env, Result, Spec (baselines)
cmd/list-plugins/      writes docs/plugins.json (make plugins-json); internal/pluginlist/ is its logic
plugins/<format>/      one package per format plugin: openapi, retirements, migrationguide,
                       removedmembers, changelog, feed, apichangelog, catalog; plugins/all imports every one
kit/  kit/kittest/     Unchanged, Cite, Wayback; the test kit (Mem, Pages, Ingest, Golden)
event/  migrate/       the change-event types from patch-it's contract (v1 + v2), v1 -> v2
fetch/                 the HTTP fetch with retries and the ETag / Last-Modified cache
diff/ normalize/ hint/ adapter/   the OpenAPI diff engine and the spec adapters
retire/  sdkguide/     deprecations-page and migration-guide readers
contract/fixtures/     a copy of patch-it's change-event fixtures (CI_contract checks it)
docs/plugins.md        the list of supported plugins (Rule 1); docs/plugins.json is generated from it
Makefile               make check (what CI_go runs, plus the leak guard), make plugins-json
docs/                  contributor docs, including writing-a-plugin.md
providers/<api>.md     one page per provider
scripts/               leak_guard.py (hashed internal names), check_contract.py
.github/               CODEOWNERS, PR template, release-notes categories, workflows
                       (CI_go, CI_dco, CI_leak_guard, CI_contract, Release)
```

The [change-event contract](https://github.com/patch-it-dev/patch-it/tree/main/docs/contract) is owned by patch-it: don't change `contract/fixtures/` here first.

## Commits and PRs

- Sign off every commit: `git commit -s` (DCO; `CI_dco` checks it).
- Subjects short, lowercase, imperative: `add the foo plugin`.
- One plugin or fix per PR; fill in every checkbox of the PR template, including "added to docs/plugins.md".
- AI-assisted commits keep their co-author trailer.
- Never re-run, dispatch or retrigger CI to "see if it passes"; fix the cause and push.

## Releases

Agents never tag or release. The **Release** workflow tags a patch release on its own every Monday and Thursday when something new was merged, and a maintainer can run it by hand (see [CONTRIBUTING.md](CONTRIBUTING.md#releasing)). Never delete or move a tag.
