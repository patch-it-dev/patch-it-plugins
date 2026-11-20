# Salesforce (`salesforce`)

## What it watches

Salesforce's REST API version policy: whole numbered API version ranges (`/services/data/vNN.N/`) that Salesforce's own end-of-life page says are already retired (every resource under that version returns `410:GONE`) or scheduled to be.

Format plugin: `eol` (`plugins/eol`), with the `Salesforce` page.

## Why this source

- **Source:** [API End-of-Life Policy](https://developer.salesforce.com/docs/platform/api-rest/guide/api-rest-eol.html), Salesforce's own table of REST API version ranges and their support status. The live page 403s a plain fetch (bot-protected), so it's read through a Wayback Machine snapshot, same as `retirements` and `feed`. Checked 2026-11-20 (page crawled 2026-10-31).
- **Why this one:** it's the only place Salesforce states, in one page, which whole version ranges still work, which are scheduled to stop, and which already have. Salesforce's own policy text above the table: "Salesforce is committed to supporting each API version for a minimum of 3 years from the date of first release... Salesforce notifies customers who use an API version scheduled for deprecation at least 1 year before support for the version ends."
- **Alternatives considered:**
  - **A changelog or RSS/Atom feed.** None found. Salesforce Help's release notes (HTML and PDF, three times a year: Spring, Summer, Winter) cover new features and, inconsistently, "Critical Updates" with their own auto-activation dates, but as prose spread across hundreds of pages per release, not a page or feed keyed by what changed in the API. No official feed of any of this exists.
  - **An OpenAPI spec.** Salesforce publishes no stable, versioned one. There's a beta, on-demand generator (`POST .../async/specifications/oas3`) for the sObjects REST API only, explicitly "subject to change... Don't build production integrations based on this," generated live per org and not pinned to a release, so there's nothing to diff between two points in time.
  - **Official Python and JS/TS SDKs for the REST API.** Checked 2026-11-20: Salesforce publishes none. `simple-salesforce` (PyPI) and `jsforce` (npm) are the community-maintained libraries almost everyone actually uses, but they're not Salesforce's own, so a breaking change in either is about the library, not the API; left for a future, separate client-library plugin if either becomes worth watching on its own. Salesforce does maintain its own SDKs for other surfaces (the `sf` CLI, Lightning Web Components), just not a general REST/SOAP API client.
  - **The retirement-detail sub-pages** the main table links to (e.g. "Salesforce Platform API Versions 31.0 through 40.0 Retirement"), each its own Salesforce Help article. The main page already states the same range, status and season; reading only it keeps one source of truth and one snapshot per run.
- **Quote:** "As of Summer ’25, these versions are retired and unavailable." (versions 21.0 through 30.0, a real, already-happened retirement)

## How the provider announces changes

- **Versioned by path, numbered, incrementing:** `/services/data/vNN.N/...` (currently up to v67.0; a new major number roughly every release, three times a year). `GET /services/data/vNN.N/` ("List Available REST Resources") is the one resource that exists at every version and nothing else, so it's this plugin's one honest, documented `operation` to cite for "this whole version is gone."
- **No exact date, only a named release:** "Summer '27", not a calendar date. Salesforce's exact production rollout for a named release is staggered across weekends per instance (confirmed by cross-checking Salesforce's own admin blog against three others, which disagree on the exact weekend), so there is no single verifiable day to convert "Summer '27" into; the event's `version` is the API version number, and the season name is quoted, not guessed into a date.
- **Three years minimum per version, one year's notice before deprecation,** per the policy text above the table.

## What the plugin reads and ignores

- **Reads** every row of the end-of-life table: a version range, and its status cell. A range already described as retired ("As of Summer 'XX, these versions are retired and unavailable.") becomes a `breaking` event per version number in the range; a range with both a deprecation and a retirement sentence ("Deprecated and unsupported from Summer 'XX. Retired from Summer 'YY.") becomes a `deprecation` event per version number. One event per version number, not one per range, so a repo pinned to one specific version is matched directly.
- **Ignores** a row that only says "Supported." (on the live page, 2026-11-20: versions 41.0 through 67.0) — nothing to report, and the pinned drop test.
- **Cites** the Wayback snapshot, with the row's version-range label and its full status text as the excerpt.
- On the page saved 2026-10-31 this gives 34 events: 10 deprecations (versions 31.0–40.0) and 24 already-breaking (versions 7.0–30.0, both past retirement bands).

## Known gaps

- **No per-operation or per-field changes.** This plugin only sees whole-version retirement; a field or resource removed within a still-supported version (the kind GitHub's or GitLab's `openapi` plugin would catch, if Salesforce had a stable spec to diff) isn't watched.
- **Critical Updates aren't watched.** These are Salesforce's other real mechanism for staged breaking changes (an org-level toggle that eventually activates by default, then becomes mandatory), but there's no single, official, publicly dated list of them across releases to read; only the per-org Setup page and third-party trackers exist.
- **The community SDKs (`simple-salesforce`, `jsforce`) aren't watched,** even though they're what most Python and JS code calling Salesforce actually uses (see Alternatives considered).
- **The client fix is a version bump, not a code search.** Unlike an operation or parameter removal, there's no single call site to point at: the fix is "pin a newer API version" wherever the repo's SDK, base URL or `Accept`/version header names the retiring one.
