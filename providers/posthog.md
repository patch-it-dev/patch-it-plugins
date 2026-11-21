# PostHog (`posthog/python-sdk`)

## What it watches

Members a new major of the `posthog` Python SDK removes, from that major's migration guide: in 8.0.0, `CaptureMode`, `posthog.capture_v1`, `posthog.capture_v1.CaptureV1Error`, `posthog.request.batch_post`… (code using one breaks on the upgrade). One event per member, with the major's first release as `version` and the release before it on PyPI as `previous_version`. The releases are listed from PyPI; the guide is read at each major's first tag, so nothing in the code names a release.

Kind: `migrationguide` (`plugins/migrationguide`), with the removed-members reader (`plugins/migrationguide/members`), reading a Markdown table.

## Why this source

- **Source:** `docs/migration-<X-1>.x-to-<X>.0.md` at tag `posthog-v<X>.0.0`, read raw at the tag; the one so far is [`docs/migration-7.x-to-8.0.md` at posthog-v8.0.0](https://github.com/PostHog/posthog-python/blob/posthog-v8.0.0/docs/migration-7.x-to-8.0.md) (commit `6edff41425188056e48f6119457dc9bc0669ca99`). A major with no guide at its tag has nothing to read (the pair is skipped). Releases from [PyPI](https://pypi.org/project/posthog/). Checked on 2026-11-19. The guide isn't in the repo at posthog-v7.91.4, the release before, and is the same at posthog-v8.0.2. MIT.
- **Why this one:** it's PostHog's own migration guide for the SDK's 8.0.0, and its "Fix removed APIs" section is a table of what 8.0.0 removed and what to use instead. A tag never changes, so the URL is its own citation, and 8.0.0 is a version a repo pins.
- **Alternatives considered:**
  - **`references/public_api_snapshot.txt`** in the same repo: a generated listing of the public API at every tag. Diffing it between two releases gives every removed name with its full path, but no reason, replacement or announcement. The plugin reads the guide, and the listing was used to check the paths of the guide's names.
  - **PostHog's REST API spec.** It's served only live, at <https://us.posthog.com/api/schema/>, and isn't committed anywhere, so there's no release to pin or replay. Its history, measured on 2026-11-19 over the two Wayback Machine snapshots of the schema (app.posthog.com, 2024-05-06 and 2025-11-09) and the live copy (about two and a half years):
    - operations: 391, 815, then 2,787;
    - deprecations: 0 in either snapshot. The live copy marks 8 operations `deprecated: true`, all on organization routes (`advanced_activity_logs`, `desktop_beta_terms`, `integrations`) that weren't in the 2025-11-09 snapshot: they arrived already deprecated, so no operation a caller used was deprecated in the window;
    - removals: 155 between the snapshots and 397 between the 2025 snapshot and live, none marked deprecated first. 308 of the 397 are `/api/environments/` operations, and 254 of those have a `/api/projects/` twin in the live spec: PostHog moving routes, not removing them.

    So the `deprecationsOnly` rule would find nothing, and a removal diff would be mostly route moves.
  - **The SDK changelogs** (`posthog/CHANGELOG.md`): the 8.0.0 entry files its breaking changes under "Major changes", which the `changelog` plugin doesn't read as breaking, and in long sentences rather than plain removals.
  - **The rendered docs** on posthog.com, and `references/posthog-python-migration.md` in the same repo: the same guide, as a live page and with posthog.com's front matter.
- **Quote:** "8.x.x removes these options and APIs:" (Fix removed APIs), with rows such as "| `CaptureV1Error` | `CaptureError` |".

## How the provider announces changes

- Package versions (SemVer). posthog-python releases often (7.89.0 to 8.0.0 in five days). 8.0.0 shipped on 2026-11-17.
- **Forced on a fresh install** for an unpinned or open pin (`posthog>=7`), otherwise when the team upgrades.
- A major release gets a migration guide in `docs/`; this is the first one there.

## What the plugin reads and ignores

- Reads the table under "Fix removed APIs", `| Removed | Use instead |`. A row becomes one event per name, but only when its first cell is nothing but code spans naming members. Names are written as code writes them after the import: `posthog.` is stripped, other prefixes kept.
- `removed_symbol` for every name, with the "Use instead" cell in the migration hint. Not `renamed_symbol`, even when that cell is one name: every replacement is new in 8.0.0, so a renamed import would fail on 7.x, and `CaptureError` also moves module (below). The 7 rows, checked against the code at posthog-v7.91.4 and posthog-v8.0.0:

  | Row | Events |
  |---|---|
  | `capture_mode`, `CaptureMode`, `POSTHOG_CAPTURE_MODE` → Nothing | `removed_symbol` `CaptureMode` (importable from `posthog`). `capture_mode` dropped: the guide's own "Rename deprecated settings" table says `posthog.capture_mode` still works or has no effect in 8.x.x, and the name also covers a `Client` argument and a module, which the row doesn't tell apart. `POSTHOG_CAPTURE_MODE` is an environment variable, not a member: dropped |
  | `gzip=True` → `capture_compression` | Dropped: an argument value of a call the row doesn't name (`removed_parameter` needs the call) |
  | `CaptureV1Error` → `CaptureError` | `removed_symbol` `capture_v1.CaptureV1Error` (its module), with `CaptureError` in the hint |
  | `posthog.capture_v1` → `posthog.capture_event` and `posthog.capture_send` | `removed_symbol` `capture_v1` (two replacements) |
  | `request.batch_post`, `async_batch_post`, `EVENTS_ENDPOINT`, `AI_EVENTS_ENDPOINT` → Send events through a client | `removed_symbol` for each of the four, at the module the code has them in: `request.batch_post`, `_async_request.async_batch_post` (a private module), `request.EVENTS_ENDPOINT`, `request.AI_EVENTS_ENDPOINT`. The cell spells out `request.` only for the first. Checked against `references/public_api_snapshot.txt` and the code at posthog-v7.91.4 |
  | The `gzip` parameter of `request.post` and `request.flags` → Nothing | Dropped: prose. Not a `removed_parameter` either: both take `**kwargs` in 8.0.0, so `gzip=` raises nothing and is sent in the request body, as the guide says |
  | The `backoff` dependency → Your own requirements… | Dropped: prose, and a dependency |

  7 events. The event's description says only that the member was removed: what code gets differs by member (an `ImportError` for most, an `AttributeError` for a module attribute).
- **`CaptureV1Error` → `CaptureError` moves module too:** from `posthog.capture_v1` to `posthog`, and `CaptureError` is new in 8.0.0. The row says only the names; the guide says the rest a few paragraphs below ("Replace `from posthog.capture_v1 import CaptureV1Error` with `from posthog import CaptureError`. There is no alias.").
- Ignores the guide's eight other tables (endpoints, AI lane arguments, error fields, retries, SDK identity, dropped events, options, and "Rename deprecated settings", whose old names still work in 8.x.x) and its prose (behavior changes such as property precedence and retries).

## Known gaps

- PostHog's REST API itself (see above: no pinned spec, and nothing deprecated in its history).
- **Importing the `posthog.capture_mode` module directly** (`from posthog.capture_mode import …`) breaks in 8.0.0 and gets no event of its own: the row's `capture_mode` is dropped because the guide says setting it still works. The usual use of that module, `CaptureMode`, is an event.
- posthog-js and posthog-node, and PostHog's other SDKs.
- Later majors of posthog-python: each is a new entry in this plugin.
- Behavior changes in 8.0.0 that aren't removals (keyword-only arguments, property precedence, retries, routing of AI events).
- The `posthoganalytics` package, which the guide says has the same changes.
