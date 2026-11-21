# Neon (`neon/api`)

## What it watches

The Neon API (`console.neon.tech/api/v2`), the management API for Neon projects, branches, computes, databases and roles: operations Neon marks deprecated, and removals of operations it had marked deprecated, between two versions of the API's OpenAPI spec.

Kind: `openapi` (`plugins/openapi`), with `adapter.Neon`.

## Why this source

- **Source:** `packages/sdk/spec/neon-openapi.json` in [neondatabase/neon-pkgs](https://github.com/neondatabase/neon-pkgs), pinned to the commit that last changed it (`https://raw.githubusercontent.com/neondatabase/neon-pkgs/<sha>/packages/sdk/spec/neon-openapi.json`). The newest such commit comes from the GitHub commits API. Checked on 2026-11-19.
- **Why this one:**
  - It's Neon's own spec, a copy of the one Neon publishes for its API, kept in Neon's own repo. A daily job in the repo ("SDK Spec Drift") pulls `https://neon.com/api_spec/release/v2.json` and fails when it differs from the committed copy; a maintainer then commits the new one.
  - It marks deprecations with the `deprecated` flag, and the description says why or until when.
  - There are 23 versions from 2026-06-24 to 2026-11-17, and a commit can be cited and replayed.
- **Alternatives considered:**
  - **`https://neon.com/api_spec/release/v2.json`.** The same spec served live, but it has no versions to pin, cite or replay.
  - **The npm package `@neondatabase/api-client`,** which ships the spec as YAML in each release. `@neon/sdk` succeeded it (neon-pkgs's README calls it the "successor to `@neondatabase/api-client`"), and there's been no release since 2.7.3 on 2026-07-01.
  - **The [changelog](https://neon.com/docs/changelog)** (RSS at `https://neon.com/docs/changelog/rss.xml`). It's a weekly digest of everything Neon shipped, with Postgres versions, Console features and AI models in the same post as an API endpoint's deprecation. It does name some API changes ("The deprecated `GET /consumption_history/account` endpoint has been removed.", 2026-06-19), but as prose in a digest, not something to read reliably.
  - **The SDKs** (`@neon/sdk`, `@neondatabase/serverless`, the `neon-api` Python package). Their changelogs don't list removed API operations in a form to read.
- **Quote:** "Deprecated: to test a branch's already-saved configuration, use `sendNeonAuthEmailProviderTest`" (`POST /projects/{project_id}/branches/{branch_id}/auth/send_test_email`, deprecated in the spec on 2026-08-23)

## How the provider announces changes

- **Operations are named by the path a caller sends,** `POST /api/v2/projects/…`: the spec's server is `https://console.neon.tech/api/v2` and its paths leave out `/api/v2`, so the adapter moves it into the paths (as Sentry's does with `/api/0`). Quotes cite the spec as published. If Neon ever moves `/api/v2` between the server and the paths, the operations stay the same, rather than every deprecated one reading as removed.
- **No API version beyond `/api/v2`.** `info.version` is `v2` in all 23 commits, so the event's `version` is the date of the spec commit, the closest thing to the API version a change arrived in.
- **The spec changes about once a week**, mostly new operations (144 operations on 2026-06-24, 194 on 2026-11-17).
- **Deprecations use the flag.** In all 23 commits no operation is described as deprecated without it. Two unflagged operations deprecate a parameter in words (`include_v1_metrics` on `GET /consumption_history/projects`, and `name` on `POST …/snapshots/{snapshot_id}/restore`); those are parameter deprecations, not operation ones, so they make no event. Each flagged operation also has an `x-sunset` date, and most descriptions say "Removal scheduled for <date>" or "use X instead".
- **Sunset dates aren't removal dates.** On 2026-11-17, 14 operations say "Removal scheduled for March 1, 2026" (`x-sunset` 2026-03-01) and one "Removal scheduled for July 1, 2026", and all 15 are still in the spec, the live one included (checked 2026-11-19).
- **The changelog announces API deprecations and removals in prose:** `GET /consumption_history/account` was "deprecated, with a planned sunset of June 1, 2026" (2026-04-03) and "has been removed" (2026-06-19).

## What the plugin reads and ignores

- **Reads:** the spec at the newest commit, diffed against the last one. Two events come out of it:
  - **Deprecations:** an operation that gains the `deprecated` flag. The event quotes the whole operation, so Neon's reason and its `x-sunset` date are in the excerpt.
  - **Removals (breaking):** an operation that leaves the spec after the previous spec had marked it deprecated.
- **Matches operations by path template,** so a renamed path parameter isn't a removal. None was renamed in the history.
- **Ignores everything else:**
  - **Removals of operations that were never deprecated.** One operation left the spec in its history, unmarked: `POST /projects/{project_id}/branches/{branch_id}/recover`, on 2026-08-09. neon-pkgs's own PR for it (#408) found "The endpoint was not removed from the API. It was removed from the published spec": it still routes.
  - **`x-sunset` dates and "Removal scheduled" words:** quoted in the deprecation's excerpt, but not events of their own, since the operations outlive them.
  - **Request and response property changes, and new operations.**
- **History:** over the 23 commits this keeps 1 event: the deprecation of `POST /projects/{project_id}/branches/{branch_id}/auth/send_test_email` (beta) on 2026-08-23. The other 15 deprecated operations were already flagged in the first commit.
- **Reads** a historical pair (a version pair, `5c57d009` → `93b93dc9`) and recreates that deprecation (the golden test). The drop test replays `4c6b3be4` to `4497de89`, the `recover` removal.
- **A kept removal:** none happened in neon-pkgs's history, so a test replays the same spec from `@neondatabase/api-client` 2.7.2 to 2.7.3 (a different source, used only for this test): `GET /consumption_history/account`, deprecated and then removed, the removal Neon's changelog announced. That's one breaking event.

## Known gaps

- **Removals Neon didn't mark first aren't events,** even if the endpoint really went away.
- **A sunset date that passes isn't an event.** The 15 operations past their "Removal scheduled" date are reported once, when they were deprecated, and again only if they leave the spec.
- **Lag.** neon-pkgs commits the spec when someone acts on the drift check, days after the live spec changes. On 2026-11-19 the live spec had one operation more than the newest commit.
- **History before 2026-06-24** isn't in neon-pkgs, so deprecations and removals before then (`GET /consumption_history/account`, removed by 2026-06-19) aren't replayable from this source.
- **Only the management API is covered,** not the serverless driver (`@neondatabase/serverless`) or the Data API.
- **Two spec commits on one day would share a `version`.** It hasn't happened in the history.
- **The patch-it client needs `neon` in its list of date-versioned APIs.** The events' `version` is a date (`2026-08-23`), because the API has no version; without the entry, the client reads it as a package version and can't place these events. That list is in the client used by patch-it's hosted ingestion service, not in this repo.
