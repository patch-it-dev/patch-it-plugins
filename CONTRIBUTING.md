# Contributing

Thanks for helping patch-it watch more APIs. This file is the rulebook; [docs/writing-a-plugin.md](docs/writing-a-plugin.md) is the how-to.

**Start a new provider or source with the starter:** `make new-plugin ARGS='--provider <id> --source <id> --kind <kind>'` ([which kind?](docs/which-kind.md); in a terminal it asks for the rest) writes everything into the provider's folder, `providers/<id>/`: the source's definition, its registration with its metadata in `provider.go` (the provider's one `RegisterProvider` block), a test over placeholder sources, the testdata rows, and the source's section of the provider's page; you fill in the `TODO(new-plugin)`s ([how](docs/writing-a-plugin.md#1-scaffold-it-with-new-plugin)).

## The rules

1. **Give every provider and source complete metadata in its registration.** A source exists when its provider's `providers/<provider>/provider.go` registers it with `plugin.RegisterProvider` (once per provider, every source in it, built from the kinds' exported types): the provider's id, name, category and homepage, and the source's id, layer, what it watches, the URLs it reads and the packages it covers; its page is the provider's `README.md`. That registration is the only list: `make plugins` prints it, each release carries it as `plugins.json`, and docs.patch-it.dev reads it from there. Nothing else to update.
2. **Fetch only through the environment.** Plugins get their sources from `env.Get` or the helpers in `core/kit/`, never from `net/http` directly, the filesystem or a database. That keeps tests offline and lets the hosted service cache, retry and rate-limit politely.
3. **Every source has a golden test** over a real saved copy of it: `providers/<provider>/testdata/<file>` in, `providers/<provider>/testdata/golden/<provider>-<source>.json` out. The provider's `testdata/README.md` lists each file with its URL, the date you fetched it, its license, and a short quote of the part your plugin reads. Keep openly licensed files whole; trim vendor documentation pages to what the test needs.
4. **Precision over recall.** A missed change is a bug we can fix later. A wrong change becomes a wrong pull request in someone's repo. When in doubt, don't emit.
5. **One pinned drop test per source**: a realistic input your plugin must *not* turn into an event (an unrelated section, a renamed heading, a non-breaking addition).
6. **Document the source** in its section of the provider's page, `providers/<provider>/README.md` (a new provider's from [`docs/provider-template.md`](docs/provider-template.md)): what it watches, **why you chose that source** (and the alternatives you didn't), how the provider announces changes, what the plugin reads and ignores, and known gaps.
7. **Keep how apart from what.** `kinds/<kind>/` says how to read a kind of source, written once, and imports no provider; `providers/<provider>/` says what to read for one provider and imports only `core/` and `kinds/`, never another provider. Code specific to one provider (an OpenAPI adapter, a feed's reader) lives in its folder. `go test ./providers/all` checks the import graph.
8. **Nothing private.** This repo is public. No credentials, tokens, internal hostnames or links to private repos or systems, in code, tests, commit messages or PR descriptions. CI scans every PR for secrets and for known internal names.

## Sign your commits (DCO)

We use the [Developer Certificate of Origin](https://developercertificate.org/). Sign off every commit:

```sh
git commit -s -m "add the foo plugin"
```

That adds `Signed-off-by: Your Name <you@example.com>`, matching the commit's author. `CI_dco` fails a PR with an unsigned commit. To fix one: `git commit --amend -s` (last commit) or `git rebase --signoff origin/main` (all of them), then force-push your branch.

## Pull requests

- One plugin or one fix per PR. Fill in the PR template's checklist.
- Commit subjects: short, lowercase, imperative (`add the foo plugin`, `fix stripe hint parsing`).
- **Run `make check` before you push.** It runs exactly what `CI_go` runs (tidy, build, gofmt, vet, the import allowlist, the `net/http` guard, `go test -race`) plus the internal-names check, so a green `make check` means those CI checks pass.
- A maintainer listed in `.github/CODEOWNERS` reviews every PR.
- **Reviews:** Claude also reviews each PR against the rules above, once, when it's opened or marked ready. PRs from forks get no secrets, so for those a maintainer comments `@claude review` and Claude reviews it read-only (it never runs a fork's code). Only maintainers can call `@claude`. Claude's review helps; a maintainer still decides.

## CI

| Workflow | What it checks |
|---|---|
| `CI_go` | `go mod tidy`, build, vet, gofmt, `go test -race` (including that every provider's folder is imported and registers it, every registered source has its metadata and is named on its provider's page, and the import graph: kinds import no provider, a provider only `core/` and `kinds/`); plugins import only this module, the standard library and approved dependencies; `net/http` only in the fetch layer. Each step is a [Makefile](Makefile) target: `make check` runs them all locally |
| `CI_dco` | Every commit in the PR is signed off by its author |
| `CI_leak_guard` | gitleaks over the PR's commits, and a check for known internal names (stored as hashes only). Also runs weekly over the whole repo |
| `Claude Code Review` | Claude reviews the PR against the rules above, once (PRs from this repo's branches; forks: a maintainer comments `@claude review`). Advisory, not a required check |
| `Release` | Not a check: tags a release on its own twice a week, or when a maintainer runs it (see [Releasing](#releasing)) |

## Releasing

Releases are tags on `main` (`v0.x.y`), made only by the **Release** workflow. Nothing is tagged on push, and agents never tag or release.

**Automatic, twice a week:** every Monday and Thursday at 14:00 UTC the workflow makes a **patch** release, but only if something was merged since the last tag; otherwise it does nothing. If the merges add a **new kind** (a new `kinds/<kind>/` directory), the scheduled run skips with a warning instead: run it by hand with **minor**. So a merged plugin reaches patch-it's hosted ingestion service within a few days.

**By hand, any time** (an urgent fix, or a **minor** release):

1. Merge the PRs that should go out.
2. **Actions → Release → Run workflow** (branch `main`), pick the bump:
   - **patch** (`v0.4.1` → `v0.4.2`): fixes, a new source on an existing kind, docs;
   - **minor** (`v0.4.1` → `v0.5.0`): a new kind, new exported API, anything users should read about.
3. The workflow refuses if `main`'s HEAD is already tagged, then runs, on that exact commit, the internal-names check over every file, gitleaks over the whole history, `go build`, `go vet`, `gofmt` and `go test -race`. Any failure stops it before a tag exists.
4. When everything passes it pushes an annotated tag and publishes a GitHub release with generated notes (grouped by the PR's label: `plugin`/`enhancement`, `bug`, `documentation`; see [`.github/release.yml`](.github/release.yml)). The release carries one asset, `plugins.json`: the list of providers and sources printed from the registry at that tag (`go run ./cmd/list-plugins`). It is the only published list; nothing in the repo keeps one, and docs.patch-it.dev reads `https://github.com/patch-it-dev/patch-it-plugins/releases/latest/download/plugins.json`.

`v1.0.0` (and any later major) is not a button: it is a deliberate decision a maintainer makes and tags by hand.

**Never delete or move a pushed tag.** Go's module proxy and checksum database keep every published version forever, so a moved tag breaks everyone who already fetched it. If a release is wrong, fix it on `main` and release the next patch version.

## License

By contributing you agree that your contribution is licensed under [Apache-2.0](LICENSE).
