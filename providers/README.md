# Providers

One page per plugin id: `providers/<api>.md`. Start from [`_template.md`](_template.md). Each page says:

- **What the plugin watches.**
- **Why this source:** the link the plugin reads, when it was checked, why it's the most reliable place this provider announces breaking changes, which other sources were considered and why they weren't used, and one short quote from it.
- **How often** it changes, and how the provider announces breaking changes.
- **What the plugin reads** from it, and what it deliberately ignores (and why).
- **Known gaps**: changes the provider makes that this plugin can't see yet.

Each source's registration names its page (`Page: "providers/<name>.md"`); a provider's sources may share one. There's no list of pages here: `go run ./cmd/list-plugins` prints every provider and source with its page, and each release carries that list as `plugins.json`.
