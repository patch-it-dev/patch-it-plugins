# Cloudflare

Every source patch-it reads for Cloudflare, one section each: what it watches, why this source, what it reads and ignores, its known gaps. Each is registered in [`provider.go`](provider.go) and read by its kind ([`kinds/`](../../kinds/)); the saved copies its tests replay, and the golden files, are in [`testdata/`](testdata/README.md).

## Cloudflare API deprecations (`cloudflare/api`)

### What it watches

Cloudflare's REST API (`api.cloudflare.com/client/v4`): every operation Cloudflare names as deprecated on its API deprecations page. It's a `breaking` event, dated by the end of life, when the item says the endpoints stop on a full end-of-life date. Otherwise it's a `deprecation`.

Kind: `notices` (`kinds/notices`), with the feed reader (`kinds/notices/feed`) and its `API` feed ([`api.go`](api.go)).

### Why this source

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
- **Anchors** (checked 2026-11-21, declared in `provider.go`): the feed's own title, "Changelog | API deprecations" (its oldest of 50 entries can roll off). A read without them fails as a change of layout, never as nothing new.

### How the provider announces changes

- **No API-wide version.** Everything is under `/client/v4`, with no per-endpoint or date versions.
- **The deprecations page** gets a new item for each announcement, newest first. An item gives the deprecation date, usually the end-of-life date, the deprecated operations and the replacement. It's sometimes updated later, for example when an end of life is extended ("previously September 15, 2026").
- **Notice** runs from three months (Radar's layer 7 attack magnitude parameter, 2025-03-11 to 2025-06-11) to almost two years (Zone Settings Batch API, 2025-04-23 to 2027-03-31, after one extension).

### What the plugin reads and ignores

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

### Known gaps

- **Field, value and behavior changes aren't events.** The page describes them in prose ("Affected APIs:"), and there's no operation to name as deprecated.
- **Operations the spec deprecates but the page doesn't list aren't events.** That's most of the spec's 321.
- **An extended end of life is a new event with the new date.** The one with the old date isn't withdrawn.
- **An end of life without "the endpoints stop" is only a deprecation.** That includes items where it likely does mean a shutdown (AMP/SXG, Zero Trust Devices), because the item doesn't say so. Whether an endpoint still answers can't be checked from outside: without credentials every route returns the same authentication error.
- **Path parameter names follow the page,** not always the spec (`{zone_identifier}` in old items, `{zone_id}` in the spec).
- **The patch-it client can't place these events yet.** Its version check reads an `api_operation` event's version as a package version, so a date (`2026-09-27`) doesn't parse and the event is dropped. It needs a rule for date-versioned API events, like the one model retirements have.
- **`previous_version` is a read, not a release.** The events are `api_operation` subjects, which the change-event contract requires to name the version they were compared with. A page of notices has no releases, so it's the dated read the page was compared with (the read before; on the first read, that read itself, `2026-11-14T12:00:00Z` in the golden file). `version` stays the notice's own.

## Cloudflare API spec (`cloudflare/api-spec`)

### What it watches

The Cloudflare API (`api.cloudflare.com/client/v4`): operations Cloudflare marks deprecated, and removals of operations it had marked deprecated, day by day in the OpenAPI spec its SDKs and Terraform provider are generated from.

Kind: `openapi` (`kinds/openapi`), with its adapter, `Cloudflare` ([`api_spec.go`](api_spec.go)). (`cloudflare/api` is Cloudflare's deprecations feed, read by the `notices` kind.)

### Why this source

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

### How the provider announces changes

- **One version per day.** The spec is committed several times a day, there are no tags, and `info.version` is a fixed `4.0.0`, so a version is a day: the day's last commit, its date the event's `version`.
- **Operations are named by the path a caller sends** (`GET /client/v4/accounts/…`): the server is `https://api.cloudflare.com/client/v4`, moved into the paths.
- **Deprecations use the flag** (329 operations on 2026-11-20) **or the summary** (11 more, written "List datasets (deprecated)" or "Deprecated: Retrieve legacy embed code HTML"). Other uses of "deprecated" in summaries and descriptions are about a parameter or a model.
- **Path parameters get renamed** (`{zone_identifier}` to `{zone_id}` on 2026-10-30) without the URL changing.
- **Operations leave the spec without notice too:** POST `/accounts/move` and the `/tenants/{tenant_id}` operations on 2026-10-30, never marked.

### What the plugin reads and ignores

- **Reads** every day's spec, diffed against the day before. Two events come out of it:
  - **Deprecations:** an operation that gains the flag or a summary saying so. A flagged one quotes the whole operation (its deprecation message included).
  - **Removals (breaking):** an operation that leaves the spec after the previous day's spec had marked it deprecated.
- **Matches operations by path template,** so a renamed path parameter isn't a removal.
- **Ignores everything else:** removals of operations never marked deprecated, schema and property changes (mostly the generator's), new operations.
- **The golden test** replays 2026-10-29 → 2026-10-30: the three deprecated Registrar domains operations removed, three events. The drop test is the same pair: POST `/accounts/move` and GET `/tenants/{tenant_id}`, gone unmarked, make none.

### Known gaps

- **Deprecated parameters and fields** aren't events: only whole operations are.
- **Removals Cloudflare didn't mark first aren't events.**
- **History:** the GitHub commits API lists at most 1,000 commits, about five months of this spec (back to 2026-06-11 on 2026-11-21); older days aren't listed.
- **Size:** each version is a 30 MB download.
- **The patch-it client needs `cloudflare` in its list of date-versioned APIs** to place the events' date `version`; that list is outside this repo.

## Cloudflare Python SDK (`cloudflare/python-surface`)

### What it watches

The public API of the `cloudflare` package on PyPI, release by release: classes, functions and methods removed or renamed, parameters removed, and parameters that lost their default (callers must now pass them). One `package_release` event per change. Every final release is listed from PyPI, and each consecutive pair is compared. What a release adds (a new module, class, function or method, or a new parameter with a default) is an `added` event.

Kind: `apisurface` (`kinds/apisurface`), with the Python reader (`kinds/apisurface/pysurface`).

### Why this source

- **Source:** the release files on PyPI (<https://pypi.org/project/cloudflare/>), each release's wheel read as published; the code is at <https://github.com/cloudflare/cloudflare-python>. Checked on 2026-11-21.
- **Why this one:** the wheel is exactly what `pip install cloudflare==<version>` installs, so its public names and signatures are what a customer's code calls. A published file never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** GitHub release notes and CHANGELOG.md (commit titles); the spec (the HTTP API).
- **Quote:** `class AsyncOAuthScopesResource(AsyncAPIResource)` (`cloudflare/resources/iam/oauth_scopes/oauth_scopes.py` line 68) at 5.3.0, gone at 5.4.0.

### How the provider announces changes

- The SDK is generated from Cloudflare's API spec (`cloudflare/api-spec`) and every minor removes generated types and resources with the API: hundreds of events in some releases (1,533 at 5.10.0), each a name that no longer imports.
- **Forced on upgrade**, for a repo whose pin reaches the release.

### What the plugin reads and ignores

- Reads the `.py` and `.pyi` files under `cloudflare/` in the release's wheel (or, when a release has none, its sdist) as text, never importing or running them: module-level classes and functions, the methods of module-level classes, their parameters and defaults, `__all__`, an `__init__`'s re-exports from its own modules, and star imports. A `.pyi` stub is read instead of the `.py` beside it.
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
- Releases are large (about 60,000 public names); a nested response type removed with its API reads as one event per module that defines it.

## Cloudflare SDKs and Terraform provider (`cloudflare/typescript`, `cloudflare/python`, `cloudflare/go`, `cloudflare/terraform`)

### What it watches

Methods, sub-resources, types and attributes removed or renamed in each release of Cloudflare's TypeScript, Python and Go SDKs and its Terraform provider, from that release's section of the repo's `CHANGELOG.md` at its tag. One `package_release` event per bullet that names what it removes plainly. Cloudflare's API notices are a separate source (`cloudflare/api`).

Kind: `changelog` (`kinds/changelog`, `sdks.go`).

| Source | Package | Releases listed from | Layer |
|---|---|---|---|
| `cloudflare/typescript` | `npm:cloudflare` | npm | sdk |
| `cloudflare/python` | `pypi:cloudflare` | PyPI | sdk |
| `cloudflare/go` | `go:github.com/cloudflare/cloudflare-go/v7` (events name the module at each release's major) | the repo's `vX.Y.Z` tags | sdk |
| `cloudflare/terraform` | none (Terraform registry `cloudflare/cloudflare`; events carry ecosystem `terraform`) | the repo's `vX.Y.Z` tags | infra |

### Why this source

- **Source:** https://github.com/cloudflare/cloudflare-typescript/blob/main/CHANGELOG.md, https://github.com/cloudflare/cloudflare-python/blob/main/CHANGELOG.md, https://github.com/cloudflare/cloudflare-go/blob/main/CHANGELOG.md, https://github.com/cloudflare/terraform-provider-cloudflare/blob/main/CHANGELOG.md, each read raw at the release's `vX.Y.Z` tag; checked on 2026-11-20.
- **Why this one:** the SDKs are generated and breaking changes ship in minors; release-please puts each under "⚠ BREAKING CHANGES", one bullet per area (`**accounts:** …`), naming what was removed.
- **Alternatives considered:** GitHub release notes (the same text); the per-major upgrade guides (`docs/guides/version-5-upgrade.md` for Terraform: one entry per resource, no plain removal sentences); diffing the published packages (`apisurface`, when it's built).
- **Quote:** "**ssl:** The `Host`, `Status`, and `ValidationMethod` types have been removed from `cloudflare.types.ssl`." (cloudflare-python 5.6.0).

### How the provider announces changes

Frequent minors, several a month; breaking bullets in a minor's "⚠ BREAKING CHANGES" section.

### What the plugin reads and ignores

- Reads the release's section; the area prefix (`**ssl:**`) is read past. Counts `X has been removed (from Y)`, `X and Y types removed`, `Remove X`, `X has been renamed to Y`, and bare names under a removal heading.
- Drops (and logs): changed return types, changed import IDs, simplified responses, renamed enum values, anything that removes a list of names "along with" others in prose.

### Known gaps

- A Terraform attribute removed from a block is named by the attribute only when the bullet doesn't name its resource plainly.
- A release whose section is missing at its tag is skipped as nothing to read.
