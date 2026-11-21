# Supabase (`supabase/management-api`)

## What it watches

The Supabase Management API (`api.supabase.com/v1`): operations Supabase marks deprecated, and removals of operations it had marked deprecated, between two versions of the API's OpenAPI spec.

Kind: `openapi` (`plugins/openapi`), with `adapter.Supabase`.

## Why this source

- **Source:** `apps/docs/spec/api_v1_openapi.json` in [supabase/supabase](https://github.com/supabase/supabase), pinned to the commit that last changed it (`https://raw.githubusercontent.com/supabase/supabase/<sha>/apps/docs/spec/api_v1_openapi.json`). The newest such commit comes from the GitHub commits API. Checked on 2026-11-19.
- **Why this one:**
  - It's Supabase's own description of the API: the spec its API reference is built from.
  - It marks deprecations with the `deprecated` flag, sometimes with the reason ("This endpoint is deprecated - use the deploy endpoint.").
  - There are 90 versions from 2024-05-31 to 2026-11-18, and a commit can be cited and replayed.
- **Alternatives considered:**
  - **`https://api.supabase.com/api/v1-json`.** The same spec served live, but it has no versions to pin, cite or replay.
  - **The [changelog](https://supabase.com/changelog)** (RSS at `https://supabase.com/changelog-rss.xml`, 238 posts since 2022). It has about 30 breaking or deprecation posts. Almost all are about the platform rather than an API operation: Postgres versions, pooler ports, OAuth providers, the CLI, self-hosting. Or they're behavior changes ("OAuth token endpoint will return HTTP 200 instead of 201") with no field or operation to name.
  - **There's no deprecations page** (`/docs/guides/deprecated` is a 404).
  - **The SDKs.** `supabase-js` (2.119) and `supabase-py` (2.34) haven't had a major release in years, and their changelogs have no breaking sections to read.
- **Quote:** "This endpoint is deprecated - use the deploy endpoint." (`POST /v1/projects/{ref}/functions`, deprecated in the spec on 2025-02-19)

## How the provider announces changes

- **No API version beyond `/v1`.** `info.version` is a fixed `1.0.0`, so the event's `version` is the date of the spec commit, the closest thing to the API version a change arrived in.
- **The spec changes a few times a month.** Deprecations are rare: two in two and a half years. Three more operations were added already marked deprecated and later un-deprecated, so they never became deprecated.

## What the plugin reads and ignores

- **Reads:** the spec at the newest commit, diffed against the last one. Two events come out of it:
  - **Deprecations:** an operation that gains the `deprecated` flag. The event quotes the whole operation, so Supabase's reason, when there is one, is in the excerpt.
  - **Removals (breaking):** an operation that leaves the spec after the previous spec had marked it deprecated.
- **Matches operations by path template:** `/v1/branches/{branch_id}/merge` and `/v1/branches/{branch_id_or_ref}/merge` are one operation (renamed 2025-09-11).
- **Ignores everything else:**
  - **Removals of operations that were never deprecated.** 22 operations left the spec in its history, none deprecated first: 14 analytics warehouse operations on 2024-07-15, 7 path-parameter renames, and one that came back 12 days later. Without credentials every route answers `401`, made-up ones included, so whether a removed one still works can't be checked from outside.
  - **Request and response property changes, and new operations.**
- Over the 90 commits this keeps 2 events, both deprecations: `POST /v1/projects/{ref}/functions` (2025-02-19) and `GET /v1/projects/{ref}/analytics/endpoints/logs.all` (2026-06-29).
- **Reads** a historical pair (a version pair, `693bdc38` → `eb21ae25`) and recreates the `logs.all` deprecation. The golden test also reads the `functions` deprecation as a pair (`57d668c6` → `bf04ff66`). The drop tests replay the warehouse removal and the branch path-parameter rename.

## Known gaps

- **A shutdown said only in words isn't an event.** On 2026-09-30 the `logs.all` description became "This endpoint has been removed and always responds with `410 Gone`", but the operation stayed in the spec. The diff sees no structural change.
- **Removals Supabase didn't mark first aren't events,** even if the endpoint really went away.
- **Only the Management API is covered.** The APIs most apps call (the Data API through PostgREST, Auth, Storage, Realtime) and the SDKs are not.
- **Two spec commits on one day share a `version`.** It happened on four days in the history. The events stay distinct, because the operation is part of the fingerprint, but the second run's `previous_version` equals its `version`.
- **The patch-it client can't place these events yet.** Their `version` is a date (`2026-06-29`), because the API has no version. The client's version check reads an `api_operation` version as a package version, so it needs the same rule for date-versioned API events that `cloudflare`'s need.
