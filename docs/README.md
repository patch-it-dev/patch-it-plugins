# Docs

| Page | What's in it |
|---|---|
| [plugins.schema.json](plugins.schema.json) | The shape of `plugins.json`, the list of providers and sources that `go run ./cmd/list-plugins` prints from the registry and each release attaches. No list is committed. |
| [README-template.md](README-template.md) | The shape every public patch-it repo's README follows: sections, badge set, words, images. |
| [writing-a-plugin.md](writing-a-plugin.md) | How to add a provider: the `Plugin` interface, fetching through `env.Get` and `kit/`, saving the source, the golden test with `kittest`, the provider page and the source's metadata in its registration. |

| [../providers/](../providers/) | One page per provider: why that source, what the plugin reads and ignores. |
| [../contract/README.md](../contract/README.md) | The change-event fixtures copied from patch-it. |

The rules for contributing are in [CONTRIBUTING.md](../CONTRIBUTING.md); the rules for AI coding agents are in [AGENTS.md](../AGENTS.md).
