# Sentry (`sentry`)

## What it watches

Sentry's public REST API (`https://{region}.sentry.io/api/0/...`): operations Sentry deprecates, between releases of its OpenAPI spec. Most of them it deprecates in the operation's description, not with the `deprecated` flag.

Format plugin: `openapi` (`plugins/openapi`), with `adapter.Sentry`.

## Why this source

- **Source:** `openapi-derefed.json`, an asset of each `api@X.Y.Z` release of [getsentry/sentry](https://github.com/getsentry/sentry/releases), e.g. `https://github.com/getsentry/sentry/releases/download/api%402.16.0/openapi-derefed.json`.
  - The plugin reads the release with the highest `api@` version, not the newest by date (`api@2.9.0` was published after `api@2.12.0`), leaving out drafts, prereleases and releases without the spec.
  - A release asset is pinned by its release tag: the URL names the tag, and the tag names one release. GitHub does let a maintainer replace an asset in place, which a commit SHA would rule out; there's no SHA-pinned copy of the spec since it moved here.
  - Checked on 2026-11-19: 19 `api@` releases, `api@2.5.0` (2026-11-14) to `api@2.16.0` (2026-11-18).
- **History:** until 2026-11-14 the spec was published in [getsentry/sentry-api-schema](https://github.com/getsentry/sentry-api-schema), committed as `openapi-derefed.json` (1,130 commits that changed it since 2020-08-25) and tagged `0.1.0` to `2.4.0` from 2026-02-10. That repo is archived ("@sentry/api has moved"); its `2.4.0` is byte for byte `api@2.5.0`. The adapter reads a tag or a commit of it too (`https://raw.githubusercontent.com/getsentry/sentry-api-schema/<tag or sha>/openapi-derefed.json`), so that history can be replayed.
- **Why this one:** it's Sentry's own OpenAPI spec of its public REST API, with every `$ref` resolved, published with `@sentry/api` ("The official, auto-generated TypeScript client for Sentry's public REST API"). It's the one machine-readable record of which endpoints Sentry documents and deprecates, and it carries the deprecation notices.
- **Alternatives considered:**
  - **The [changelog](https://sentry.io/changelog/)** (RSS). It's product news. Its items around the alert-rule deprecation ("Monitors & Alerts are in Early Access", 2026-04-20; "Monitors & Alerts — Now Generally Available", 2026-05-07) don't name an endpoint, and the feed carries titles only.
  - **The SDKs' changelogs** (`sentry-python`, `sentry-javascript`). They're about the event-ingestion SDKs, not the REST API.
  - **`src/sentry/api/urls.py` in getsentry/sentry.** It lists the routes, but not which are public, documented or deprecated.
- **Quote:** "## Deprecated 🚧 Use [Fetch an Organization's Monitors](/api/monitors/fetch-an-organizations-monitors) and [Fetch Alerts](/api/monitors/fetch-alerts) instead." (the description of `GET /api/0/organizations/{organization_id_or_slug}/alert-rules/` from sentry-api-schema `1da1f31f`, 2026-01-22; a line break after the heading)

## How the provider announces changes

- **No API versions.** Every path is under `/api/0/`, and `info.version` is `v0`. The spec's own versions (`api@2.16.0`, `0.224.0`) are the `@sentry/api` package's, released several times a day lately, not the API's.
- **The event's `version` is a date:** the spec release's publication date (`2026-11-18` for `api@2.16.0`), or a commit's date when replaying the history before tags. Several releases a day share one. The patch-it client keeps a date-versioned API's events only if it's in its list of date-versioned APIs, so `sentry` needs to be in it.
- **Deprecations are in words.** Over the whole history (1,130 commits, then the 19 `api@` releases) the `deprecated` flag is set on 3 operations, and the description deprecates 13, always no later than the flag:
  - **2022-03-01:** `GET .../stats/`, words and flag together: "This endpoint is deprecated in favor of [Organization Stats V2]".
  - **2024-07-31:** `POST .../user-feedback/`: "*This endpoint is DEPRECATED. We document it here for older SDKs ...". Flagged on 2026-06-16.
  - **2025-07-25:** `GET /api/0/projects/{organization_id_or_slug}/{project_id_or_slug}/issues/`: "**Deprecated**: This endpoint has been replaced with the [Organization Issues endpoint]". Flagged on 2026-06-16.
  - **2026-01-22:** the 10 issue and metric alert-rule operations (`.../alert-rules/`, `.../alert-rules/{alert_rule_id}/`, `.../rules/`, `.../rules/{rule_id}/`): "## Deprecated 🚧 Use [...] instead." Never flagged.
- **Endpoints leave the spec without a word.** See the removals below.

## What the plugin reads and ignores

- **Reads:** the spec of the newest `api@` release, diffed against the last one read.
- **One event comes out of it: a deprecation,** when an operation is newly deprecated, by the flag or by its description. Matched by path template.
  - **A description deprecates an operation** when it has a `## Deprecated` heading, a line starting `**Deprecated**:`, or says "This endpoint is deprecated" (any case). Those are the three ways the history says it. The event cites the description.
  - **Not "deprecat" anywhere.** That also matches three descriptions that only mention deprecation: `GET .../events/`'s "**Eventsv2 Deprecation Note**" (about the old `eventsv2` endpoint; 2022-06-16, and again under the 2024-05-10 parameter rename) and `GET .../trace-items/attributes/`'s "examples, and deprecation details" (2026-10-15).
  - **A flagged operation is quoted whole,** `"deprecated":true` first and the description after it.
  - **The flag after the words is no new event:** user-feedback and issues were deprecated when their descriptions said so, not on 2026-06-16.
- **Reads every path as a caller sends it, `/api/0/...`.** Until sentry-api-schema 2.0.1 the server was `https://{region}.sentry.io` and every path started `/api/0/`. From 2.1.0 (2026-11-13) the server is `https://{region}.sentry.io/api/0` and the paths don't. Same URLs, but compared as published all 289 operations would read as removed and added again. So the adapter moves the server URL's path back into every path, and maps each event's `source_pointer` back to the path as published.
- **Ignores everything else:**
  - **Removals, even of deprecated operations.** An operation leaving the spec means Sentry stopped documenting it, and the history doesn't show when, or whether, it stopped working. With `/api/0` in every path, 157 operations left the spec (by path template), 11 of them deprecated first:
    - `GET /api/0/organizations/{organization_slug}/stats/`, gone on 2024-01-22, is still routed in getsentry/sentry's `src/sentry/api/urls.py` (checked 2026-11-19).
    - The 10 alert-rule operations left on 2026-05-08, the day after "Monitors & Alerts — Now Generally Available". Their routes aren't in `urls.py` today, but nothing from Sentry says they stopped answering on 2026-05-08.
    - Compared as published, 446 operations left the spec: the 289 more are the 2026-11-13 move of `/api/0`, `GET .../issues/` and `POST .../user-feedback/` among them, both flagged deprecated. A rule keeping removals of deprecated operations would report those two shut down.
  - **Property changes, and new operations.**
- **History:** 13 events over the whole history, all deprecations: the 13 above, none withdrawn. By the flag alone it would be 3, two of them dated 2026-06-16 instead of 2024 and 2025.
- **Replays** the three description deprecations in the golden test (sentry-api-schema commits `ead7f0cf` → `4650a66a`, `d64149d6` → `c2f638f6`, `47231e61` → `1da1f31f`). The drop test replays `2.0.1` → `2.1.0`, the `/api/0` move. Other tests replay `0.223.0` → `0.224.0` (the flag after the words) and the two commits that added the "deprecation" wording that isn't a deprecation.

## Known gaps

- **No removals.** When Sentry turns an endpoint off, the plugin has already reported its deprecation, if Sentry said so in the spec, and reports nothing else.
- **A deprecation said another way isn't seen,** like "has been replaced" with no "deprecated", or a parameter deprecated in prose.
- **Undocumented endpoints aren't watched.** The spec covers what Sentry documents publicly: 199 path items in `api@2.16.0`.
- **Request and response property changes aren't events,** including deprecated parameters and fields.
