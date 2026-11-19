# Algolia (`algolia-search`, `algolia-ingestion`, `algolia-recommend`)

## What it watches

Three of Algolia's REST APIs, one plugin each: the Search API (`algolia-search`), the Ingestion API (`algolia-ingestion`) and the Recommend API (`algolia-recommend`). Each reports operations Algolia marks deprecated, and removals of operations it had marked deprecated, between two releases of that API's OpenAPI spec.

Format plugin: `openapi` (`plugins/openapi`), with `adapter.Algolia`.

## Why this source

- **Source:** `specs/bundled/<api>.yml` in [algolia/api-clients-automation](https://github.com/algolia/api-clients-automation), pinned to a release tag (`https://raw.githubusercontent.com/algolia/api-clients-automation/<tag>/specs/bundled/search.yml`). The newest tag comes from the GitHub refs API. Checked on 2026-11-19.
- **Why this one:**
  - It's the spec Algolia generates every API client from (Python, JS, Go, …), so it's the API as Algolia's own clients see it.
  - It has 186 release tags from 2023-08-22 to 2026-11-17 (`released-2026-11-17-31c309fe2a`), and a tag never changes.
  - Deprecations use the `deprecated` flag, sometimes with the replacement in the description ("please use `getTasks` instead").
- **Alternatives considered:**
  - **The [changelog](https://changelog.algolia.com),** hosted on AnnounceKit. It's product news, and its feed URLs return the HTML page, not a feed.
  - **There's no deprecations page** (`/doc/guides/getting-started/deprecations/` is a 404).
  - **`specs/major-breaking-changes-rename.json`** maps client method names from the last major version of the clients to the current ones. It's a one-time migration map, not a changelog.
  - **The API clients' own changelogs.** They're generated from the same specs.
- **Quote:** "Retrieves a list of tasks using the v1 endpoint, please use `getTasks` instead." (`GET /1/tasks`, deprecated in the Ingestion API spec in `released-2025-01-22-232affcac`)

## How the provider announces changes

- **No API version beyond `/1/` in the paths.** Each API client has its own version numbers, so the event's `version` is the date of the spec release (`2025-04-01`).
- **The specs get a release a few times a week.** Deprecations are rare: in the history, 9 in the Search API and 9 in the Ingestion API, none in Recommend.

## What the plugin reads and ignores

- **Reads:** each API's spec at the newest release tag, diffed against the last one. Two events come out of it:
  - **Deprecations:** an operation that gains the `deprecated` flag. The event quotes the whole operation, so a replacement Algolia names in the description is in the excerpt.
  - **Removals (breaking):** an operation that leaves the spec after the previous release had marked it deprecated.
- **Finds the newest release** by its date. A day can have several releases (three on 2024-09-19), and the commit in a tag says nothing about their order, so same-day releases are ordered by their commit's date.
- **Ignores entries that describe only an API client, not the REST API:**
  - helpers marked `x-helper: true`: `/saveObjects`, `/waitForTask`, `/replaceAllObjects`, … (16 in the Search API);
  - the custom-request methods for any path (`customGet`, `customPost`, `customPut`, `customDelete` on `/{path}`).
- **Ignores everything else:**
  - **Removals of operations that were never deprecated.** 17 removals in the history, none deprecated first. 13 were the client-only entries above: the custom-request methods `/1{path}` in all three specs, which became `/{path}` on 2024-03-25, and `/saveObjects`. The other 4 were Ingestion API operations: GET and POST `/1/transformations/models`, `/1/transformations/copilot`, and `/1/sources/{sourceID}/discover`.
  - **Property changes, and new operations.**
- From 2023-08-22 to 2026-11-17 this keeps 18 events, all deprecations:
  - **Search API:** the 9 multi-cluster operations under `/1/clusters` (2025-04-01).
  - **Ingestion API:** 8 v1 `/1/tasks` operations (2025-01-22) and `PUT /1/tasks/{taskID}/disable` (2024-07-26).
- **The golden tests** replay the Search and Ingestion deprecations. The drop tests replay the `/1/transformations/models` removal and, for `algolia-recommend`, the custom-request move of 2024-03-25.

## Known gaps

- **Only three APIs are watched.** Analytics, Insights, A/B testing, Query Suggestions, Personalization, Monitoring and others have specs in the same repo, but no deprecations in them yet.
- **Removals Algolia didn't mark first aren't events,** even if the endpoint really went away.
- **A "removed" event means removed from the spec.** If a deprecated operation, such as the v1 `/1/tasks` ones, leaves the API clients' spec while Algolia still serves it, the plugin reports it removed. `vercel` and `supabase` share this.
- **Changes to the API clients themselves aren't covered:** method renames between major versions, the helpers.
- **Two releases on one day share a `version`.**
- **The patch-it client can't place these events yet.** Their `version` is a date, because the API has no version. It needs the same rule for date-versioned API events that `cloudflare`'s and `supabase`'s need.
