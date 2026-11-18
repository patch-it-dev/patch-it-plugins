# AGENTS.md

Rules for AI coding agents (and humans) working in this repo. `CLAUDE.md` imports this file.

## Rule 1: every new or changed plugin goes in `docs/plugins.md`

**If you add a plugin, add its row to [`docs/plugins.md`](docs/plugins.md) in the same PR. If you change a plugin's id, provider, source, what it watches, format plugin or status, update its row in the same PR. If you remove one, remove its row.**

The list of supported APIs on docs.patch-it.dev is generated from that file, so a plugin missing from it doesn't exist as far as users can tell. No CI check enforces this: you are the check. Before you finish, compare the registered plugins with the table and say in the PR that you did.

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
- **`providers/<api>.md`** for every provider.
- **No source, no event**: every event carries the source URL and a quote.
- Don't edit golden files to make a test pass without explaining in the PR why the new output is right.

## Commands

```sh
go build ./...
go vet ./...
gofmt -l .                    # must print nothing
go test -race ./...
python3 scripts/leak_guard.py --all   # the internal-names check over every tracked file
```

Until the plugin code lands there is no `go.mod`, and `CI_go` passes with a notice.

## Layout

```
docs/plugins.md        the list of supported plugins (Rule 1)
docs/                  contributor docs
providers/<api>.md     one page per provider
scripts/leak_guard.py  the hashed internal-names check
.github/               CODEOWNERS, PR template, workflows (CI_go, CI_dco, CI_leak_guard)
```

When the code lands (Go module `github.com/patch-it-dev/patch-it-plugins`): `plugin/` (the interface and registry), `plugins/<format>/` (one package per format plugin, `plugins/all` imports every one), `event/` (the change-event types from [patch-it's contract](https://github.com/patch-it-dev/patch-it/tree/main/docs/contract)), `fetch/`, `kit/`, `diff/`, `normalize/`, `hint/`, `retire/`, `sdkguide/`, `adapter/`, `contract/fixtures/`.

## Commits and PRs

- Sign off every commit: `git commit -s` (DCO; `CI_dco` checks it).
- Subjects short, lowercase, imperative: `add the foo plugin`.
- One plugin or fix per PR; fill in every checkbox of the PR template, including "added to docs/plugins.md".
- AI-assisted commits keep their co-author trailer.
- Never re-run, dispatch or retrigger CI to "see if it passes"; fix the cause and push.
