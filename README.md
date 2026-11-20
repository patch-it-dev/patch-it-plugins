<img align="right" src="https://raw.githubusercontent.com/patch-it-dev/.github/main/profile/assets/bot-found-one.svg" alt="Patchy, the patch-it bot, holding up the lines it found" width="120">

# patch-it-plugins

**How patch-it notices, on its own, that something your code depends on changed.** Each provider (Stripe, OpenAI, GitHub…) has sources (a spec, a changelog, a shutdown notice, a feed); a plugin reads every version of a source and turns what changed into a change event, with the source linked and quoted.

[![CI](https://github.com/patch-it-dev/patch-it-plugins/actions/workflows/CI_go.yml/badge.svg)](https://github.com/patch-it-dev/patch-it-plugins/actions/workflows/CI_go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/patch-it-dev/patch-it-plugins.svg)](https://pkg.go.dev/github.com/patch-it-dev/patch-it-plugins)
[![Go version](https://img.shields.io/github/go-mod/go-version/patch-it-dev/patch-it-plugins)](go.mod)
[![License: Apache-2.0](https://img.shields.io/badge/license-Apache--2.0-3f5bf0)](LICENSE)
[![Docs](https://img.shields.io/badge/docs-docs.patch--it.dev-3f5bf0)](https://docs.patch-it.dev)
[![Discussions](https://img.shields.io/github/discussions/patch-it-dev/patch-it-plugins)](https://github.com/patch-it-dev/patch-it-plugins/discussions)

## Add a provider in 15 minutes

You need Go (the version in `go.mod`), `make` and Python 3.

1. **Scaffold it**, on the kind that reads its source ([which kind](docs/writing-a-plugin.md#3-pick-the-kind)):
   ```sh
   go run ./cmd/new-plugin --api acme --provider Acme --format feed --source https://acme.example/changelog.xml
   ```
2. **Fill in every `TODO(new-plugin)`**: the reader, the provider and source metadata in its `plugin.RegisterProvider` block, a saved copy of the source in `testdata/`, and "Why this source" in `providers/acme.md`. Then write the golden file and read it like code:
   ```sh
   go test ./plugins/notices -run TestAcme -update
   ```
3. **Check it the way CI will, and open the PR** (`git commit -s`, one provider per PR):
   ```sh
   make check
   ```

## More

- [How to write a plugin](docs/writing-a-plugin.md): providers, sources, kinds, versions, the test kit, step by step.
- [How it works](docs/how-it-works.md): what a plugin does, the layout, the checks, releases.
- What's covered: `make plugins` prints every provider and source from the code; each [release](https://github.com/patch-it-dev/patch-it-plugins/releases) carries it as `plugins.json`, and [docs.patch-it.dev](https://docs.patch-it.dev/supported-apis/plugins/) shows it.
- [CONTRIBUTING.md](CONTRIBUTING.md) (the rules), [AGENTS.md](AGENTS.md) (for AI coding agents), [Discussions](https://github.com/patch-it-dev/patch-it-plugins/discussions), [good first issues](https://github.com/patch-it-dev/patch-it-plugins/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22).
- What patch-it does with the events: [patch-it.dev](https://patch-it.dev) and [docs.patch-it.dev](https://docs.patch-it.dev). [Apache-2.0](LICENSE), copyright patch-it contributors.
