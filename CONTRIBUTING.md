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

## CI

| Workflow | What it checks |
|---|---|
| `CI_go` | `go mod tidy`, build, vet, gofmt, `go test -race`; plugins import only this module, the standard library and approved dependencies; `net/http` only in the fetch layer |
| `CI_dco` | Every commit in the PR is signed off by its author |
| `CI_leak_guard` | gitleaks over the PR's commits, and a check for known internal names (stored as hashes only). Also runs weekly over the whole repo |

## License

By contributing you agree that your contribution is licensed under [Apache-2.0](LICENSE).
