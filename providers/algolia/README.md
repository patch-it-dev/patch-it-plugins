# Algolia

Every source patch-it reads for Algolia, one section each: what it watches, why this source, what it reads and ignores, its known gaps. Each is registered in [`provider.go`](provider.go) and read by its kind ([`kinds/`](../../kinds/)); the saved copies its tests replay, and the golden files, are in [`testdata/`](testdata/README.md).

## Algolia (`algolia/search-api`, `algolia/ingestion-api`, `algolia/recommend-api`)

### What it watches

Three of Algolia's REST APIs, one plugin each: the Search API (`algolia-search`), the Ingestion API (`algolia-ingestion`) and the Recommend API (`algolia-recommend`). Each reports operations Algolia marks deprecated, and removals of operations it had marked deprecated, between two releases of that API's OpenAPI spec.

Kind: `openapi` (`kinds/openapi`), with its adapter, `Algolia` ([`specs.go`](specs.go)).

### Why this source

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

### How the provider announces changes

- **No API version beyond `/1/` in the paths.** Each API client has its own version numbers, so the event's `version` is the date of the spec release (`2025-04-01`).
- **The specs get a release a few times a week.** Deprecations are rare: in the history, 9 in the Search API and 9 in the Ingestion API, none in Recommend.

### What the plugin reads and ignores

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

### Known gaps

- **Only three APIs are watched.** Analytics, Insights, A/B testing, Query Suggestions, Personalization, Monitoring and others have specs in the same repo, but no deprecations in them yet.
- **Removals Algolia didn't mark first aren't events,** even if the endpoint really went away.
- **A "removed" event means removed from the spec.** If a deprecated operation, such as the v1 `/1/tasks` ones, leaves the API clients' spec while Algolia still serves it, the plugin reports it removed. `vercel` and `supabase` share this.
- **Changes to the API clients themselves aren't covered:** method renames between major versions, the helpers.
- **Two releases on one day share a `version`.**
- **The patch-it client can't place these events yet.** Their `version` is a date, because the API has no version. It needs the same rule for date-versioned API events that `cloudflare`'s and `supabase`'s need.

## Algolia Python client (`algolia/python-surface`)

### What it watches

The public API of the `algoliasearch` package on PyPI, release by release: classes, functions and methods removed or renamed, parameters removed, and parameters that lost their default (callers must now pass them). One `package_release` event per change. Every final release is listed from PyPI, and each consecutive pair is compared.

Kind: `apisurface` (`kinds/apisurface`), with the Python reader (`kinds/apisurface/pysurface`).

### Why this source

- **Source:** the release files on PyPI (<https://pypi.org/project/algoliasearch/>), each release's wheel read as published; the code is at <https://github.com/algolia/algoliasearch-client-python>. Checked on 2026-11-21.
- **Why this one:** the wheel is exactly what `pip install algoliasearch==<version>` installs, so its public names and signatures are what a customer's code calls. A published file never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** CHANGELOG.md; the specs themselves (the HTTP API).
- **Quote:** `class FacetFilters(BaseModel)` (`algoliasearch/recommend/models/facet_filters.py` line 21) at 4.44.1, gone at 4.44.3 (back at 4.44.4).

### How the provider announces changes

- The client is generated from Algolia's API specs (the `algolia/*-api` sources) and released every week or two.
- **Forced on upgrade**, for a repo whose pin reaches the release.

### What the plugin reads and ignores

- Reads the `.py` and `.pyi` files under `algoliasearch/` in the release's wheel (or, when a release has none, its sdist) as text, never importing or running them: module-level classes and functions, the methods of module-level classes, their parameters and defaults, `__all__`, an `__init__`'s re-exports from its own modules, and star imports. A `.pyi` stub is read instead of the `.py` beside it.
- Ignores private names: a leading underscore, a module under a private one or under one named `internal` or `vendor`, a name left out of `__all__`, and tests. Also classes nested in classes, a property's parameters (and its setter), and `self`/`cls` (callers never pass them).
- Drops (and logs): an overloaded function's parameters; a removed parameter when the function now takes `**kwargs`; a new parameter without a default (the change-event contract has no subject for it yet).
- A removal is a rename only when the same scope loses one and gains one of the same kind with identical parameters, at least one of them besides `self`/`cls`; anything less sure is a removal.
- Drops (and logs) a removal that may not be one: a name its module still binds in the newer release (an alias such as `DeadlineExceededError = GatewayTimeoutError`, an import from a private module, a star import from outside the package, a module-level `__getattr__` that hands out old names); a symbol whose file the newer release has too big to read (`MaxFileBytes`); a method whose class is still there but whose bases changed, which one of its bases now defines, or which the class now defines as an attribute.

### Known gaps

- A positional parameter renamed reads as the old one removed: a call that passes it by keyword breaks, one that passes it by position doesn't.
- A class with no `__init__` of its own (generated pydantic or dataclass-like types) has no parameters to compare, so its rename reads as a removal of the old name, and its fields aren't compared.
- A module that star-imports from outside the package, or defines `__getattr__`, hides every removal from it: those are notes, not events.
- Type changes (an annotation narrowed, a return type changed) aren't events.
- The wheel read is the first one by file name (a `-none-any` wheel when there is one); a release whose platform wheels differ in their Python files is read from that one.
