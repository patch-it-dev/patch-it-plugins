# Cloudflare (`cloudflare/api`)

## What it watches

Cloudflare's REST API (`api.cloudflare.com/client/v4`): every operation Cloudflare names as deprecated on its API deprecations page. It's a `breaking` event, dated by the end of life, when the item says the endpoints stop on a full end-of-life date. Otherwise it's a `deprecation`.

Kind: `notices` (`plugins/notices`), with the feed reader (`plugins/notices/feed`) and its `CloudflareAPI` feed.

## Why this source

- **Source:** [API deprecations](https://developers.cloudflare.com/fundamentals/api/reference/deprecations/), read as its RSS feed `https://developers.cloudflare.com/fundamentals/api/reference/deprecations/index.xml`. Checked on 2026-11-19.
- **Why this one:** it's where Cloudflare announces API deprecations, by its own description: "When this happens, we will communicate when the API will no longer be available and whether there will be a replacement."
  - The feed carries every announcement since 2020, 50 items.
  - Since 2024, most items name what they deprecate under "Deprecated APIs:", usually one `METHOD /path` per line. That list is kept apart from "Affected APIs:", which lists endpoints that only change.
  - Most items since 2025 give an "End of life date:".
  - The deprecated endpoints that are still in Cloudflare's OpenAPI spec are flagged deprecated there too (56 of 58 on 2026-11-19), so the page and the spec agree.
- **Alternatives considered:**
  - **Cloudflare's OpenAPI spec, [cloudflare/api-schemas](https://github.com/cloudflare/api-schemas).** It flags more operations deprecated (321 on 2026-11-19), but only 53 of those carry a date (`x-cfDeprecation` or `x-forge-sunset`).
    - Its removals can't be trusted as shutdowns. Over 39 snapshots from 2024-11 to 2026-11, 477 operations left the spec and only 38 had been deprecated first. 17 came back later, and 141 were path-parameter renames.
    - There's no way to check from outside whether a removed endpoint still answers: without credentials every route returns the same authentication error.
    - It has no tags, and changes about ten times a day.
  - **The [changelog](https://developers.cloudflare.com/changelog/) feed.** It covers every product, mostly announcements; the deprecations page is its API part.
  - **The SDKs** (`cloudflare` on PyPI and npm). They're generated from the spec. The Python SDK's `CHANGELOG.md` lists SDK-level breaking changes (removed methods and types), which is a different subject.
- **Quote:** "End of life date: September 27, 2026" and "After this date, the legacy endpoints will no longer be available." (Legacy Registrar Domain Management API)

## How the provider announces changes

- **No API-wide version.** Everything is under `/client/v4`, with no per-endpoint or date versions.
- **The deprecations page** gets a new item for each announcement, newest first. An item gives the deprecation date, usually the end-of-life date, the deprecated operations and the replacement. It's sometimes updated later, for example when an end of life is extended ("previously September 15, 2026").
- **Notice** runs from three months (Radar's layer 7 attack magnitude parameter, 2025-03-11 to 2025-06-11) to almost two years (Zone Settings Batch API, 2025-04-23 to 2027-03-31, after one extension).

## What the plugin reads and ignores

- **Reads:** each item's "Deprecated APIs:" list, one event per operation. The operation is written as the page writes it, normalized to `{param}` with a leading `/` and no trailing `/`; the oldest items write `GET zones/:zone_identifier/...`.
- **Event type:**
  - **`breaking`** when the item gives a full "End of life date:" and says the endpoints stop ("After this date, the legacy endpoints will no longer be available."). `version` is the end-of-life date. An end-of-life date alone isn't enough: of the 11 items with one on 2026-11-19, only 2 say the endpoints stop.
  - **`deprecation`** otherwise. `version` is the deprecation date, or, when the page gives only a month ("November 2025"), the date the item was published.
- **Drops** (and logs, with why):
  - Items with no "Deprecated APIs:" list: behavior changes, cipher suites, fields and settings ("Affected APIs:", "Modified API"). One 2022 item (Cloudflare Images direct upload v1) names its operation inline instead of in a list, and is dropped with them.
  - Items that tell direct API callers no action is required. The Radar summary and timeseries groups endpoints have an end of life, but "For users calling the API directly (not via the Cloudflare SDK), no action is required": the URLs still fit the new `/summary/{dimension}` endpoints, and only an SDK method moves.
  - List items that aren't one operation: "`GET` and `POST` /…" (the 2026 Workers KV item writes three of its five that way), a DNS-over-HTTPS hostname, "Other similar … endpoints".
  - List items where the text after the operation is a sentence about it, not a name for it, such as `POST /accounts/{account_id}/gateway/rules` "— creating rules with action: "audit_ssh" is no longer accepted". What goes is one value; the operation stays, so naming it would be wrong.
- On the feed saved 2026-11-19 this gives 114 events from 27 items: 5 breaking (Zone Settings Batch API and the Legacy Registrar API) and 109 deprecations. It logs 27 drops: 21 whole items and 6 list items.
- **Cites** a Wayback Machine snapshot of the feed, read back and checked against the live feed (`kit.Cite`).
- **Historical change in the golden test:** the Legacy Registrar Domain Management API, deprecated 2026-04-10 with end of life 2026-09-27. Its three operations left Cloudflare's OpenAPI spec in October 2026.

## Known gaps

- **Field, value and behavior changes aren't events.** The page describes them in prose ("Affected APIs:"), and there's no operation to name as deprecated.
- **Operations the spec deprecates but the page doesn't list aren't events.** That's most of the spec's 321.
- **An extended end of life is a new event with the new date.** The one with the old date isn't withdrawn.
- **An end of life without "the endpoints stop" is only a deprecation.** That includes items where it likely does mean a shutdown (AMP/SXG, Zero Trust Devices), because the item doesn't say so. Whether an endpoint still answers can't be checked from outside: without credentials every route returns the same authentication error.
- **Path parameter names follow the page,** not always the spec (`{zone_identifier}` in old items, `{zone_id}` in the spec).
- **The patch-it client can't place these events yet.** Its version check reads an `api_operation` event's version as a package version, so a date (`2026-09-27`) doesn't parse and the event is dropped. It needs a rule for date-versioned API events, like the one model retirements have.
- **`previous_version` is a read, not a release.** The events are `api_operation` subjects, which the change-event contract requires to name the version they were compared with. A page of notices has no releases, so it's the dated read the page was compared with (the read before; on the first read, that read itself, `2026-11-14T12:00:00Z` in the golden file). `version` stays the notice's own.
