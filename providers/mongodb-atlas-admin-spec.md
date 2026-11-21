# MongoDB (`mongodb/atlas-admin-spec`)

## What it watches

The MongoDB Atlas Administration API v2 (`cloud.mongodb.com/api/atlas/v2`): operations MongoDB marks deprecated, and removals of operations it had marked deprecated, day by day in the OpenAPI spec MongoDB publishes for it.

Kind: `openapi` (`plugins/openapi`), with `adapter.MongoDBAtlas`. The API's own changelog is a separate source, `mongodb/atlas-admin-api` (kind `changelog`).

## Why this source

- **Source:** `openapi/v2.json` in [mongodb/openapi](https://github.com/mongodb/openapi), pinned to a commit (`https://raw.githubusercontent.com/mongodb/openapi/<sha>/openapi/v2.json`). The versions come from the GitHub commits API. Checked on 2026-11-21.
- **Why this one:**
  - It's MongoDB's own spec of the whole API, committed by its release workflow ("ci(prod): Release OpenAPI Spec") a few times a week: 768 commits from 2024-07-25 to 2026-11-19.
  - It marks deprecated operations with the `deprecated` flag.
  - Every commit can be cited and replayed.
- **Alternatives considered:**
  - **The repo's `v0.0.N` tags:** they mark the repo's tooling releases, not the spec.
  - **The per-resource-version files in `openapi/v2/`:** one file per API resource version; `v2.json` holds every operation with all its versions, which is what a caller of any version sees removed.
  - **`changelog/changelog.json`:** read by `mongodb/atlas-admin-api`; it reports changes per resource version, in prose, not removals of whole operations.
- **Quote:** `"deprecated": true` on `GET /api/atlas/v2/users/byName/{userName}` ("Returns the details for one MongoDB Cloud user account with the specified username."), new on 2025-02-20.

## How the provider announces changes

- **One version per day,** the day's last commit, its date the event's `version`: `info.version` is a fixed `2.0`, and the dated resource versions (`application/vnd.atlas.2025-03-12+json`) are per operation, several in one spec.
- **Deprecations use the flag** (53 operations on 2026-11-19). Fourteen descriptions mention "deprecated" without it, each about the deprecated invitation flow, not the operation.
- **Deprecated operations do leave:** the tenant backup operations were flagged, then removed between 2025-10 and 2026-11.
- **Operations also leave unmarked:** GET `/rest/unauth/version` on 2024-08-07.

## What the plugin reads and ignores

- **Reads** every day's spec, diffed against the day before. Two events come out of it:
  - **Deprecations:** an operation that gains the `deprecated` flag, quoting the whole operation.
  - **Removals (breaking):** an operation that leaves the spec after the previous day's spec had marked it deprecated.
- **Ignores everything else:** removals of operations never marked deprecated, schema and property changes (the changelog source reports those per resource version), new operations.
- **The golden test** replays 2025-02-18 → 2025-02-20, when 14 user-management operations were deprecated at once; three are saved, three events. **The drop test** replays 2024-07-25 → 2024-08-07: GET `/rest/unauth/version` gone unmarked, no event.

## Known gaps

- **Deprecated resource versions** (an operation's `2023-01-01` content type sunset while newer ones stay) aren't events: only whole operations are. The changelog source covers them.
- **Removals MongoDB didn't flag first aren't events.**
- **History starts on 2024-07-25,** when `openapi/v2.json` was first committed.
- **License:** the spec is CC BY-NC-SA 3.0 US; only small excerpts are saved for tests.
