# Vercel (`vercel/api`)

## What it watches

Vercel's REST API, release by release: operations Vercel marks deprecated, and deprecated operations it then removes, between two versions of the OpenAPI spec Vercel generates its official SDK (`@vercel/sdk`) from.

Kind: `openapi` (`plugins/openapi`), with `adapter.Vercel`.

## Why this source

- **Source:** `vercel-spec.json` in [vercel/sdk](https://github.com/vercel/sdk), pinned to a release tag (`https://raw.githubusercontent.com/vercel/sdk/<tag>/vercel-spec.json`), with the newest tag taken from the GitHub Releases API. Checked on 2026-11-18.
- **Why this one:** it's Vercel's own machine-readable description of the API, committed with every SDK release (207 releases from 2024-11-25 to 2026-11-17, nearly one a day). A pinned tag can be cited and replayed.
- **Alternatives considered:**
  - [`https://openapi.vercel.sh/`](https://openapi.vercel.sh/) is the same spec served live, at most a release ahead (3 more paths out of 356 on 2026-11-18), but it has no versions to pin, cite or replay.
  - The [changelog](https://vercel.com/changelog), also published as Atom at `https://vercel.com/atom`, mixes product news and blog posts. It has no categories and holds only the latest 100 entries, about three weeks.
  - There's no central deprecations page (`https://vercel.com/docs/deprecations` is a 404). Deprecations are scattered across changelog posts and per-product docs.
  - `@vercel/sdk`'s generated commit notes flag "Breaking ⚠️" on harmless response enum additions.
  - The Python SDK ([vercel/vercel-py](https://github.com/vercel/vercel-py), `vercel` on PyPI, 0.13.0) is hand-written helpers, not a client of this spec.
- **Quote:** "Check the price for a domain (deprecated)" (the summary of `GET /v4/domains/price` from v1.13.5). The same day, Vercel's changelog said "The Legacy Domains API is now deprecated. … Sunset date: 2025-11-09." ([changelog](https://vercel.com/changelog/new-domains-registrar-api-for-domain-search-pricing-purchase-and-management)).

## How the provider announces changes

- There's no API-wide version. Each endpoint carries its own in the path (`/v13/deployments`, `/v6/user/tokens`), and older versions usually keep working beside newer ones. `info.version` is a fixed `0.0.1`, so the event's `version` is the SDK release tag (`1.18.7`), which is also what a repo pins in `package.json`.
- Deprecations are mostly said in words. Of the 22 operations deprecated over two years, 16 had no `deprecated` flag, only a summary ending "(deprecated)" or starting "Deprecated:".
- A sunset is announced in the changelog with a date, roughly a month ahead for the Legacy Domains API. After the date, the endpoint answers with an error ("This endpoint has been sunsetted on November 9, 2025"). The operation left the spec later, between one and two months after the sunset.

## What the plugin reads and ignores

- **Reads:** the spec at the newest tag, diffed against the last one. Two events come out of it:
  - **Deprecations:** an operation that gains the `deprecated` flag, or a summary that starts "Deprecated:" (with the colon) or ends "(deprecated)". A bare leading "Deprecated" doesn't count, because it would match an operation about deprecated things, such as "Deprecated runtimes for a project". The event cites the flag or the summary.
  - **Removals (breaking):** an operation that leaves the spec after the previous spec had marked it deprecated. "Removed" means removed from the spec, which isn't always the endpoint shutting down. All four in the history were real sunsets, but a deprecated operation could also leave the spec for a newer path version while its old URL keeps answering for a while.
- **Matches operations by path template:** `DELETE /v2/sandboxes/drives/{name}` and `.../{nameOrId}` are one operation, because a caller sends the same URL to both (renamed in v1.28.65).
- **Ignores everything else**, because in the spec's history it was mostly not a change to the API:
  - **Unmarked removals.** 103 operations left the spec without being deprecated first. 16 came back in a later release (`/v1/log-drains`, `/v1/integrations/git-namespaces`), and the `/v1/edge-config` ones were the product's [rename to Global Config](https://vercel.com/changelog/edge-config-is-now-global-config).
  - **Response properties.** 67,048 response-property removals in two years, about a thousand a release, from generated schemas reshuffling.
  - **Request properties.** 478 breaking request-property changes, most of them union members compared by position as their order moved. Outside unions there were 142, and 27 of those reverted later.
  - **New operations.** Nobody has to fix code for a new endpoint.
- Over the 207 releases, this keeps 26 events: 22 deprecations, and the removal of the four Legacy Domains operations, all four of which Vercel shut down.
- **Reads** a historical pair (a version pair, `v1.18.6` → `v1.18.7`) and recreates the removal of `GET /v4/domains/price` and `GET /v4/domains/status`. The golden test also runs the deprecation of all four operations (`v1.13.4` to `v1.13.5`). The drop tests replay an unmarked removal (`v1.6.2` to `v1.6.3`) and the path-parameter rename (`v1.28.64` to `v1.28.65`).

## Known gaps

- Request and response property changes aren't events, even the ones that match an announced change. For example, `secret` left the `type` values of `POST /v10/projects/{idOrName}/env` in 2026-04, after Vercel [sunset legacy secrets](https://vercel.com/changelog/legacy-environment-variable-secrets-are-being-sunset), and it isn't reported. Telling those apart from the spec's churn needs a better union comparison first.
- An operation Vercel removes without marking it deprecated first isn't reported.
- A deprecation or sunset announced only in the changelog, with no spec change, isn't an event.
- Runtime deprecations (Node.js 20 on 2026-10-01, the legacy build image) are announced only in the changelog. They'd be `ci_runtime` events from a separate plugin.
- The removal event arrives after the endpoint has stopped working, because the spec drops it late. The deprecation event before it is the timely one.
- **Client:** the events are `api_operation`, the same kind as Stripe's, so the patch-it client's existing handler takes them, with no new fix kind. Two things are still open on the client side:
  - Its version gate has no entry for Vercel, whose SDK (`@vercel/sdk`) is on npm.
  - A sunset breaks every caller of the endpoint, whatever their SDK version, including code that calls the REST API directly.
