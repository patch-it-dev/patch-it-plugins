# Anthropic Python SDK (`anthropic`)

## What it watches

Request parameters removed from `messages.create()` and its siblings in anthropic-sdk-python 1.0.0 (`temperature`, `top_p`, `top_k`): passing one is a `TypeError`. One breaking event per parameter. Pinned to that release.

Format plugin: `migrationguide` (`plugins/migrationguide`), reading with `sdkguide/`.

## Why this source

- **Source:** [MIGRATION.md at v1.0.0](https://github.com/anthropics/anthropic-sdk-python/blob/v1.0.0/MIGRATION.md), read raw at the tag (`https://raw.githubusercontent.com/anthropics/anthropic-sdk-python/v1.0.0/MIGRATION.md`). Checked on 2026-10-17.
- **Why this one:** it's the SDK maintainers' own guide to the major release, with a table of the methods, the removed parameter and what to use instead. A tag never changes, so the URL is its own permanent citation.
- **Alternatives considered:** the GitHub release notes and `CHANGELOG.md` (list commits, not "what to use instead"); diffing the SDK's code between versions (finds signature changes but not the maintainers' advice, and is a lot more machinery); Anthropic's model deprecations page (mentions the change, but it's the models page).
- **Quote:** "Removed: deprecated request parameters" (section heading), with the row "`temperature`, `top_p`, `top_k` | Remove them."

## How the provider announces changes

- Package versions (PEP 440). 1.0.0 shipped on 2026-08-20.
- **Forced on a fresh install** for a repo whose `anthropic` pin is open above 1.0, otherwise when the team upgrades.
- The fix works on both sides of the upgrade: removing the parameter is right for current models, and `extra_body` passes it through for an older one.

## What the plugin reads and ignores

- The guide must be the "Migrating to v1" guide. Every table with a "Removed parameter" and a "Use instead" column; a row counts only when its method cell names the plain `messages.create()`.
- Ignores the beta-only `output_format` row: `output_format=Model` is still valid on the `parse()` / `stream()` helpers, so it isn't a removal everywhere.
- `migration_hint` is the guide's "Use instead" cell, as written.

## Known gaps

- Only 1.0.0. A later major's guide is a new entry.
- Removed type aliases, exports and helper arguments in the same guide aren't events yet.
