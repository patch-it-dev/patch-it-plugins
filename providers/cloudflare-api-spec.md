# Cloudflare (`cloudflare/api-spec`)

## What it watches

The Cloudflare API (`api.cloudflare.com/client/v4`): operations Cloudflare marks deprecated, and removals of operations it had marked deprecated, day by day in the OpenAPI spec its SDKs and Terraform provider are generated from.

Kind: `openapi` (`plugins/openapi`), with `adapter.Cloudflare`. (`cloudflare/api` is Cloudflare's deprecations feed, read by the `notices` kind.)

## Why this source

- **Source:** `openapi.json` in [cloudflare/api-schemas](https://github.com/cloudflare/api-schemas) (BSD-3-Clause), pinned to a commit (`https://raw.githubusercontent.com/cloudflare/api-schemas/<sha>/openapi.json`). The versions come from the GitHub commits API. Checked on 2026-11-21.
- **Why this one:**
  - It's Cloudflare's own description of the whole API, published by a bot several times a day ("Update schema: openapi.json"), the input to `cloudflare-python`, `cloudflare-typescript` and the Terraform provider.
  - It marks deprecated operations with the `deprecated` flag, and most carry `x-stainless-deprecation-message` with the end-of-life date.
  - Every commit can be cited and replayed.
- **Alternatives considered:**
  - **`openapi.yaml` beside it:** the same spec as YAML.
  - **The [API deprecations page](https://developers.cloudflare.com/fundamentals/api/reference/deprecations/)** and its feed: read by `cloudflare/api` (kind `notices`). It announces end-of-life dates in prose; the spec says which operation, and when it's gone.
  - **The changelog:** product news, not keyed by operation.
- **Quote:** "This operation is deprecated and will reach end of life on September 27, 2026. Use the new Registrar API endpoints (domain-search, domain-check, registrations) instead." (`GET /accounts/{account_id}/registrar/domains`, removed from the spec on 2026-10-30)

## How the provider announces changes

- **One version per day.** The spec is committed several times a day, there are no tags, and `info.version` is a fixed `4.0.0`, so a version is a day: the day's last commit, its date the event's `version`.
- **Operations are named by the path a caller sends** (`GET /client/v4/accounts/…`): the server is `https://api.cloudflare.com/client/v4`, moved into the paths.
- **Deprecations use the flag** (329 operations on 2026-11-20) **or the summary** (11 more, written "List datasets (deprecated)" or "Deprecated: Retrieve legacy embed code HTML"). Other uses of "deprecated" in summaries and descriptions are about a parameter or a model.
- **Path parameters get renamed** (`{zone_identifier}` to `{zone_id}` on 2026-10-30) without the URL changing.
- **Operations leave the spec without notice too:** POST `/accounts/move` and the `/tenants/{tenant_id}` operations on 2026-10-30, never marked.

## What the plugin reads and ignores

- **Reads** every day's spec, diffed against the day before. Two events come out of it:
  - **Deprecations:** an operation that gains the flag or a summary saying so. A flagged one quotes the whole operation (its deprecation message included).
  - **Removals (breaking):** an operation that leaves the spec after the previous day's spec had marked it deprecated.
- **Matches operations by path template,** so a renamed path parameter isn't a removal.
- **Ignores everything else:** removals of operations never marked deprecated, schema and property changes (mostly the generator's), new operations.
- **The golden test** replays 2026-10-29 → 2026-10-30: the three deprecated Registrar domains operations removed, three events. The drop test is the same pair: POST `/accounts/move` and GET `/tenants/{tenant_id}`, gone unmarked, make none.

## Known gaps

- **Deprecated parameters and fields** aren't events: only whole operations are.
- **Removals Cloudflare didn't mark first aren't events.**
- **History:** the GitHub commits API lists at most 1,000 commits, about five months of this spec (back to 2026-06-11 on 2026-11-21); older days aren't listed.
- **Size:** each version is a 30 MB download.
- **The patch-it client needs `cloudflare` in its list of date-versioned APIs** to place the events' date `version`; that list is outside this repo.
