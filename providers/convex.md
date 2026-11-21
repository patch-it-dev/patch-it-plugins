# Convex (`convex/deployment-api`)

## What it watches

The Convex Deployment API (`{deployment-url}/api/v1`, the platform API for managing one deployment: environment variables, log streams and integrations, usage limits, data sync): operations Convex marks deprecated, and removals of operations it had marked deprecated, between two versions of the API's OpenAPI spec.

Kind: `openapi` (`plugins/openapi`), with `adapter.Convex`.

## Why this source

- **Source:** `npm-packages/@convex-dev/platform/deployment-openapi.json` in [get-convex/convex-backend](https://github.com/get-convex/convex-backend), pinned to the commit that last changed it (`https://raw.githubusercontent.com/get-convex/convex-backend/<sha>/npm-packages/@convex-dev/platform/deployment-openapi.json`). The newest such commit comes from the GitHub commits API. Checked on 2026-11-19.
- **Why a commit, not a release tag:** the repo tags `precompiled-YYYY-MM-DD-<sha>` builds (1,213 of them), but those are daily builds of the self-hosted backend, not releases of the API. There are several on some days, ordered only by commit date, and the spec is the same in most of them. The spec at a tag is the spec at the last commit before it that changed the file, so the commits see every version a tag would, and finding the newest takes one call instead of paging through the tags. A commit never changes, so it can be cited and replayed.
- **Why this one:**
  - It's Convex's own description of the API: its docs build renders the API reference from it.
  - It marks deprecations with the `deprecated` flag and says what to use instead ("Deprecated: use `GET /integrations` instead."). Convex's docs build drops flagged operations, "since they stay served only for existing callers" (`npm-packages/docs/scripts/prepare-openapi-specs.js`).
  - There are 68 versions from 2025-08-19, when the spec was added, to 2026-11-16.
- **Alternatives considered:**
  - **The Management API spec** (`management-openapi.json`, next to it: teams, projects, deployments). Its `info.license` is `LicenseRef-Convex`, which nothing in the repo defines, so the plugin neither reads nor saves it.
  - **The SDK changelogs** (`convex` on npm, `get-convex/convex-js` `CHANGELOG.md`; `convex-py`). One flat list of bullets per version (95 versions in `convex-js`), with no removed or deprecated sections for the `changelog` plugin to read.
  - **[Convex News](https://news.convex.dev/)** (RSS at `https://news.convex.dev/rss/`). Product news and announcements, newest post 2026-09-21; it doesn't name API operations.
  - **The `@convex-dev/platform` npm package** (0.1.11). Its tarball has only `package.json` and the generated client, not the specs.
- **Quote:** "Deprecated: use `GET /integrations` instead." (`GET /api/v1/list_log_streams`, deprecated in the spec on 2026-11-05)

## How the provider announces changes

- **No API version beyond `/api/v1`.** `info.version` is `1.0.0` in every commit, so the event's `version` is the spec commit's committer date (UTC), the closest thing to the API version a change arrived in. The patch-it client keeps a date-versioned API's events only if it's in its list of date-versioned APIs, so `convex` needs to be in it.
- **The spec changes a few times a month,** mostly with new operations and properties as Convex builds out the platform API.
- **Deprecations are rare:** one so far. On 2026-10-30 (`f35879bb`, "Rename log_streaming to integrations") Convex added six `/integrations` operations; on 2026-11-05 (`5e5159cb`) it marked the six log stream operations deprecated, each naming its replacement. The commit says "The routes still work, so existing callers and older dashboards keep working."

## What the plugin reads and ignores

- **Reads:** the spec at the newest commit, diffed against the last one. Two events come out of it:
  - **Deprecations:** an operation that gains the `deprecated` flag. The event quotes the whole operation, so the replacement Convex names is in the excerpt.
  - **Removals (breaking):** an operation that leaves the spec after the previous spec had marked it deprecated. None yet.
- **Operations are named by the path a caller sends,** `GET /api/v1/list_log_streams`: the server URL is `{deployment-url}/api/v1` (since the first commit) and the spec's paths leave out `/api/v1`, so the adapter moves it into the paths (as Sentry's does with `/api/0`). Quotes cite the spec as published. If Convex ever moves `/api/v1` between the server and the paths, the operations stay the same, rather than every deprecated one reading as removed.
- **Matches operations by path template,** as for every spec here. No path parameter has been renamed in this spec yet.
- **Ignores everything else:**
  - **Request and response property changes,** and new operations. Over the history the differ finds 18 property removals, 16 type changes, 10 schema removals and 3 properties no longer required, most of them in the data sync, usage-limit and S3 export APIs. The drop test is one of them: on 2026-07-14 (`3ead9091`, "Remove name from usage limits") the optional `name` left the usage-limit request and response; the commit says it "was never editable in the UI (always `None` on create, only echoed back)", and the request type at that commit doesn't reject unknown fields, so a caller still sending it isn't refused.
  - **Removals of operations that were never deprecated.** None in the history: no operation has left the spec.
- Over the 68 commits this keeps 6 events, all deprecations on 2026-11-05: `GET /api/v1/list_log_streams`, `GET /api/v1/get_log_stream/{id}`, `POST /api/v1/create_log_stream`, `POST /api/v1/update_log_stream/{id}`, `POST /api/v1/delete_log_stream/{id}`, `POST /api/v1/rotate_webhook_secret/{id}`.
- **Reads** a historical pair (a version pair, `f35879bb` → `5e5159cb`) and recreates those six; the golden test also runs them as a daily ingest. The drop test replays `3f6edf7a` to `3ead9091`.

## Known gaps

- **Only the Deployment API is covered.** Not the Management API (see above), the client SDKs (`convex` on npm, `convex-py`), or the functions runtime. SDK removals are written as plain bullets in their changelogs, so nothing reads them yet.
- **Removals Convex doesn't mark first aren't events,** even if the endpoint really went away. None has happened yet.
- **Two spec commits on one day share a `version`.** It happened on 13 days in the history. The events stay distinct, because the operation is part of the fingerprint, but the second run's `previous_version` equals its `version`.
- **The patch-it client can't place these events until `convex` is in its list of date-versioned APIs.**
