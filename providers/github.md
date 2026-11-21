# GitHub REST API (`github/rest-api`)

## What it watches

GitHub's REST API between dated API versions (`2022-11-28` → `2026-03-10`): properties, schemas, parameters and operations removed or changed in a new version.

Kind: `openapi` (`plugins/openapi`), with `adapter.GitHub`.

## Why this source

- **Source:** [github/rest-api-description](https://github.com/github/rest-api-description), one file per supported dated version: `descriptions/api.github.com/api.github.com.<version>.json` on `main`. The supported versions come from `GET https://api.github.com/versions`. Checked on 2026-11-14.
- **Why this one:** it's GitHub's own OpenAPI description, generated from the same source as docs.github.com's reference and the Octokit SDKs. A dated version's file is frozen, so two of them diff exactly.
- **Alternatives considered:** the undated `api.github.com.json` (the in-progress spec, moves several times a day with additive changes; not a version anyone pins); the `dereferenced/` variants (every `$ref` inlined, so one schema change would show up on every endpoint using it); the repo's `CHANGELOG.md` (one section per dated version listing affected endpoints; good prose, not wired as a hint source yet); the [Breaking changes](https://docs.github.com/en/rest/about-the-rest-api/breaking-changes) docs page (prose).
- **Quote:** "Remove deprecated `rate` property from rate limit endpoint." ([CHANGELOG.md](https://github.com/github/rest-api-description/blob/main/descriptions/api.github.com/CHANGELOG.md), Version 2026-03-10).

## How the provider announces changes

- Clients pick a version per request with `X-GitHub-Api-Version: <date>`; without it GitHub uses `2022-11-28`.
- **Breaking changes only ship in a new dated version**, which is rare: `2022-11-28` was the only one for over three years.
- **Opt-in now, forced later**: a version keeps working for 24 months after a newer one ships, then returns `410 Gone` (with `Deprecation` / `Sunset` headers before that).
- The event's `version` is the dated version string, not `info.version` (a fixed package version, `1.1.4`).

## What the plugin reads and ignores

- Reads: the newest dated version's spec, diffed against the last one it saw. `GET /versions`' order isn't documented, so the adapter sorts the versions itself.
- The real `2022-11-28` → `2026-03-10` diff is 108 breaking events. The golden test replays a trimmed pair that keeps one of them: `rate` removed from `rate-limit-overview`.

## Known gaps

- Additive changes between dated versions aren't events (they never get a version).
- GraphQL isn't watched.
