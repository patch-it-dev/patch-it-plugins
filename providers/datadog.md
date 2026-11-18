# Datadog (`datadog`)

## What it watches

Datadog's API v2, release by release: endpoints, request and response properties and schemas removed or changed between two versions of the OpenAPI spec Datadog generates its official clients from.

Format plugin: `openapi` (`plugins/openapi`), with `adapter.Datadog`.

## Why this source

- **Source:** `.generator/schemas/v2/openapi.yaml` in [DataDog/datadog-api-client-python](https://github.com/DataDog/datadog-api-client-python), pinned to a release tag (`https://raw.githubusercontent.com/DataDog/datadog-api-client-python/<tag>/.generator/schemas/v2/openapi.yaml`); the newest tag from the GitHub Releases API. Checked on 2026-11-13.
- **Why this one:** Datadog publishes no standalone spec repo. Its generator writes the same spec, byte for byte, into every official client repo on every release, so this file *is* the API as Datadog's own clients see it, versioned by tag.
- **Alternatives considered:** the same file in the Go, Java, Ruby or TypeScript client repos (identical content; reading more than one would report every change twice); the repo's `CHANGELOG.md` (one section per release with `Removed` / `Deprecated` bullets linking each PR; prose, not wired as a hint source yet); the API reference on docs.datadoghq.com (rendered by JavaScript, not readable with a plain fetch).
- **Quote:** "Remove deprecated queried_tags_window_seconds and related_asset_tags fields" ([PR #4045](https://github.com/DataDog/datadog-api-client-python/pull/4045), released in 2.61.0).

## How the provider announces changes

- REST paths are versioned by prefix only (`/api/v1/…`, `/api/v2/…`), one spec file each. `info.version` is a fixed `"1.0"` that never moves.
- **Breaking changes are forced**: a removed property is gone from the next release, with no account-level version to stay on. Some go through a `Deprecated` changelog entry first; not all do.
- The client's release tag (`2.61.0`) is the only version signal, and it's what users pin (`datadog-api-client==2.61.0`). So the event's `version` is the release tag. Releases come roughly weekly.
- The spec is YAML only (~10 MB). The adapter converts it to JSON before diffing; excerpts are re-serialized from the node they quote either way.

## What the plugin reads and ignores

- Reads: the v2 spec at the newest tag, diffed against the last one.
- Ignores: changes to OpenAPI `tags` (they only rename generated SDK methods; the REST API is unchanged). The v1 spec isn't read yet.
- Replays a historical pair (`Replay`, from `2.60.0` to `2.61.0`) and recreates the removal of two properties from `TagIndexingRuleDynamicTags` (the golden test uses trimmed excerpts of both specs).

## Known gaps

- API v1 (`v1/openapi.yaml`) isn't watched.
- Deprecations announced only in the changelog (with no spec change yet) aren't events.
