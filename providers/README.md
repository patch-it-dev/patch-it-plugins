# Providers

One page per plugin id: `providers/<api>.md`. Start from [`_template.md`](_template.md). Each page says:

- **What the plugin watches.**
- **Why this source:** the link the plugin reads, when it was checked, why it's the most reliable place this provider announces breaking changes, which other sources were considered and why they weren't used, and one short quote from it.
- **How often** it changes, and how the provider announces breaking changes.
- **What the plugin reads** from it, and what it deliberately ignores (and why).
- **Known gaps**: changes the provider makes that this plugin can't see yet.

The pages arrive with the plugin code. The list of plugins is [docs/plugins.md](../docs/plugins.md).
