# Contributing

Thanks for helping patch-it watch more APIs. This file is the rulebook; [docs/writing-a-plugin.md](docs/writing-a-plugin.md) is the how-to.

## The rules

1. **Add your plugin to [docs/plugins.md](docs/plugins.md).** Every new plugin gets a row; every change to a plugin's id, provider, source, what it watches or status updates its row. The list on docs.patch-it.dev is built from that file, so a plugin that isn't listed is invisible to users.
2. **Fetch only through the environment.** Plugins get their sources from `env.Get` or the helpers in `kit/`, never from `net/http` directly, the filesystem or a database. That keeps tests offline and lets the hosted service cache, retry and rate-limit politely.
3. **Every plugin has a golden test** over a real saved copy of its source: `testdata/<file>` in, `testdata/golden/<api>.json` out. `testdata/README.md` lists each file with its URL, the date you fetched it, its license, and a short quote of the part your plugin reads. Keep openly licensed files whole; trim vendor documentation pages to what the test needs.
4. **Precision over recall.** A missed change is a bug we can fix later. A wrong change becomes a wrong pull request in someone's repo. When in doubt, don't emit.
5. **One pinned drop test per plugin**: a realistic input your plugin must *not* turn into an event (an unrelated section, a renamed heading, a non-breaking addition).
6. **Document the provider** in `providers/<api>.md`, copied from [`providers/_template.md`](providers/_template.md): what it watches, **why you chose that source** (and the alternatives you didn't), how the provider announces changes, what the plugin reads and ignores, and known gaps.
7. **Nothing private.** This repo is public. No credentials, tokens, internal hostnames or links to private repos or systems, in code, tests, commit messages or PR descriptions. CI scans every PR for secrets and for known internal names.

## Sign your commits (DCO)

We use the [Developer Certificate of Origin](https://developercertificate.org/). Sign off every commit:

```sh
git commit -s -m "add the foo plugin"
```

That adds `Signed-off-by: Your Name <you@example.com>`, matching the commit's author. `CI_dco` fails a PR with an unsigned commit. To fix one: `git commit --amend -s` (last commit) or `git rebase --signoff origin/main` (all of them), then force-push your branch.

## Pull requests

- One plugin or one fix per PR. Fill in the PR template's checklist.
- Commit subjects: short, lowercase, imperative (`add the foo plugin`, `fix stripe hint parsing`).
- Run the tests before you push (see [README](README.md#run-the-tests)).
- A maintainer listed in `.github/CODEOWNERS` reviews every PR.
- **Reviews:** Claude also reviews each PR against the rules above, once, when it's opened or marked ready. PRs from forks get no secrets, so for those a maintainer comments `@claude review` and Claude reviews it read-only (it never runs a fork's code). Only maintainers can call `@claude`. Claude's review helps; a maintainer still decides.

## CI

| Workflow | What it checks |
|---|---|
| `CI_go` | `go mod tidy`, build, vet, gofmt, `go test -race`; plugins import only this module, the standard library and approved dependencies; `net/http` only in the fetch layer |
| `CI_dco` | Every commit in the PR is signed off by its author |
| `CI_leak_guard` | gitleaks over the PR's commits, and a check for known internal names (stored as hashes only). Also runs weekly over the whole repo |
| `Claude Code Review` | Claude reviews the PR against the rules above, once (PRs from this repo's branches; forks: a maintainer comments `@claude review`). Advisory, not a required check |
| `Release` | Not a check: tags a release on its own twice a week, or when a maintainer runs it (see [Releasing](#releasing)) |

## Releasing

Releases are tags on `main` (`v0.x.y`), made only by the **Release** workflow. Nothing is tagged on push, and agents never tag or release.

**Automatic, twice a week:** every Monday and Thursday at 14:00 UTC the workflow makes a **patch** release, but only if something was merged since the last tag; otherwise it does nothing. So a merged plugin reaches patch-it's hosted ingestion service within a few days.

**By hand, any time** (an urgent fix, or a **minor** release):

1. Merge the PRs that should go out.
2. **Actions → Release → Run workflow** (branch `main`), pick the bump:
   - **patch** (`v0.4.1` → `v0.4.2`): fixes, a new provider on an existing format plugin, docs;
   - **minor** (`v0.4.1` → `v0.5.0`): a new format plugin, new exported API, anything users should read about.
3. The workflow refuses if `main`'s HEAD is already tagged, then runs, on that exact commit, the internal-names check over every file, gitleaks over the whole history, `go build`, `go vet`, `gofmt` and `go test -race`. Any failure stops it before a tag exists.
4. When everything passes it pushes an annotated tag and publishes a GitHub release with generated notes (grouped by the PR's label: `plugin`/`enhancement`, `bug`, `documentation`; see [`.github/release.yml`](.github/release.yml)).

`v1.0.0` (and any later major) is not a button: it is a deliberate decision a maintainer makes and tags by hand.

**Never delete or move a pushed tag.** Go's module proxy and checksum database keep every published version forever, so a moved tag breaks everyone who already fetched it. If a release is wrong, fix it on `main` and release the next patch version.

## License

By contributing you agree that your contribution is licensed under [Apache-2.0](LICENSE).
