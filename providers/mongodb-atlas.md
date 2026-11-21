# MongoDB Atlas (`mongodb/atlas-admin-api`)

## What it watches

The MongoDB Atlas Administration API (`/api/atlas/v2`): an endpoint's API version deprecated (unless MongoDB reactivates it), and an endpoint's API version removed at its sunset date, as MongoDB lists them in its own machine-readable API changelog.

Kind: `changelog` (`plugins/changelog`), with its JSON parser (`json.go`) and the `MongoDBAtlas` log.

## Why this source

- **Source:** `changelog/changelog.json` in [mongodb/openapi](https://github.com/mongodb/openapi), read at the commit that last changed it (`https://raw.githubusercontent.com/mongodb/openapi/<sha>/changelog/changelog.json`). The commit comes from the GitHub commits API. Checked on 2026-11-19.
- **Why this one:** it's MongoDB's own record of what changed, released by MongoDB's CI ("Release API Changelog") a few times a month. It's also what the [API changelog docs page](https://www.mongodb.com/docs/atlas/reference/api-resources-spec/changelog/) shows.
  - It covers 188 days from 2023-02-01 to 2026-11-18. Each day lists every endpoint that changed, per API version, with a change code from oasdiff, a sentence, and whether the change is backward compatible.
  - The life-cycle entries say what happens and when: "Resource version 2023-01-01 deprecated and marked for removal on 2025-06-01", "endpoint with API Version '2023-01-01' was removed as it has reached its sunset date '2026-01-22'".
- **Alternatives considered:**
  - **The per-version OpenAPI specs in the same repo** (`openapi/v2/openapi-<version>.json`). The changelog is MongoDB's own diff of them, already classified, so diffing them again would only re-derive it.
  - **The docs changelog page.** It's built from the same data, as HTML.
  - **The drivers' changelogs** (PyMongo, the Node.js driver). They're a different surface, the database driver, not this API, and a separate plugin if added.
- **Quote:** "endpoint with API Version '2023-01-01' was removed as it has reached its sunset date '2026-01-22'" (the shared-tier backup endpoints, 2026-01-22)

## How the provider announces changes

- **Versioning by date, per resource.** A caller picks the version per request in the `Accept` header (`application/vnd.atlas.2023-01-01+json`). Each API version of an endpoint is deprecated when a newer one ships, and removed at a sunset date, usually a year or more later.
- **The changelog gets a release a few times a month.** It holds 22,273 changes, 481 of them marked not backward compatible.

## What the plugin reads and ignores

- **Reads** the change codes about a whole endpoint at one API version, replaying the days oldest first:
  - `endpoint-deprecated` and `resource-version-deprecated` become **`deprecation`** events, unless a later `endpoint-reactivated` withdraws them. MongoDB has reactivated 13 times. For example, `GET /api/atlas/v2/groups/{groupId}/clusters` at `2024-05-30` was deprecated on 2024-06-25 and reactivated the next day.
  - `endpoint-removed` becomes a **`breaking`** event, unless the same endpoint and version shows up again on a later day (none has): then the removal goes, and so does any deprecation before it. A removal announced twice counts from its first announcement. All 38 in the file say the version "has reached its sunset date".
  - **`version`** is the API version the change is about (`2023-01-01`): the one a caller's `Accept` header names.
  - A deprecation counts from its first announcement since the last reactivation: the changelog repeats some.
- **Cites** the file at its commit, with a JSON pointer to the change and the change itself as the excerpt: every field MongoDB wrote, in its order, whitespace aside.
- **Ignores** every other change code. Most are additions. The other "not backward compatible" ones are property-level, and the flag is broad: a response enum gaining a value counts (68 times). Removed or newly required request properties are probably real breaks, but they aren't read until measured.
- **Not read:** `api-path-removed-without-deprecation`. Its one entry (`GET …/streams/activeVpcPeeringConnections` at `2024-11-13`, 2025-01-10) comes with `endpoint-added` the same day, and the endpoint changes again later, so it wasn't removed. Also not read: `sunset-deleted`, whose one entry keeps the endpoint deprecated.
- On the changelog released 2026-11-18 this gives 81 events: 43 deprecations still standing and 38 removals.

## Known gaps

- **Property-level breaking changes aren't events.** Examples are a request property removed (4), made required (11) or added as required (3), or an enum value no longer accepted (36), even where MongoDB marks them not backward compatible.
- **The changelog doesn't always announce a deprecation before a removal.** None of the 38 removals had an `endpoint-deprecated` or `resource-version-deprecated` entry for the same endpoint and version in the changelog, so some removals arrive without an earlier deprecation event. The removal entries do say the sunset date was reached.
- **An event stored before it's withdrawn stays stored:** a deprecation before its reactivation, or a removal before its version comes back. The plugin doesn't emit it again, but nothing withdraws an event already in the feed.
- **The drivers aren't covered,** and they're what most application code uses.
- **The patch-it client can't place these events yet.** Their `version` is a dated API version (`2023-01-01`), chosen per request in an `Accept` header, a scheme the client's version check doesn't know.
