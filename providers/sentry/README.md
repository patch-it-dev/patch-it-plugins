# Sentry

Every source patch-it reads for Sentry, one section each: what it watches, why this source, what it reads and ignores, its known gaps. Each is registered in [`provider.go`](provider.go) and read by its kind ([`kinds/`](../../kinds/)); the saved copies its tests replay, and the golden files, are in [`testdata/`](testdata/README.md).

## Sentry (`sentry/api`)

### What it watches

Sentry's public REST API (`https://{region}.sentry.io/api/0/...`): operations Sentry deprecates, between releases of its OpenAPI spec. Most of them it deprecates in the operation's description, not with the `deprecated` flag.

Kind: `openapi` (`kinds/openapi`), with its adapter, `Sentry` ([`api.go`](api.go)).

### Why this source

- **Source:** `openapi-derefed.json`, an asset of each `api@X.Y.Z` release of [getsentry/sentry](https://github.com/getsentry/sentry/releases), e.g. `https://github.com/getsentry/sentry/releases/download/api%402.16.0/openapi-derefed.json`.
  - The plugin reads the release with the highest `api@` version, not the newest by date (GitHub lists releases in an order that isn't version order), leaving out drafts, prereleases and releases without the spec.
  - A release asset is pinned by its release tag: the URL names the tag, and the tag names one release. GitHub does let a maintainer replace an asset in place, which a commit SHA would rule out; there's no SHA-pinned copy of the spec since it moved here.
  - Checked on 2026-11-19: 19 `api@` releases, `api@2.5.0` (2026-11-14) to `api@2.16.0` (2026-11-18).
- **History:** until 2026-11-14 the spec was published in [getsentry/sentry-api-schema](https://github.com/getsentry/sentry-api-schema), committed as `openapi-derefed.json` (1,130 commits that changed it since 2020-08-25) and tagged `0.1.0` to `2.4.0` from 2026-02-10. That repo is archived ("@sentry/api has moved"); its `2.4.0` is byte for byte `api@2.5.0`. The adapter reads a tag or a commit of it too (`https://raw.githubusercontent.com/getsentry/sentry-api-schema/<tag or sha>/openapi-derefed.json`), so that history can be replayed.
- **Why this one:** it's Sentry's own OpenAPI spec of its public REST API, with every `$ref` resolved, published with `@sentry/api` ("The official, auto-generated TypeScript client for Sentry's public REST API"). It's the one machine-readable record of which endpoints Sentry documents and deprecates, and it carries the deprecation notices.
- **Alternatives considered:**
  - **The [changelog](https://sentry.io/changelog/)** (RSS). It's product news. Its items around the alert-rule deprecation ("Monitors & Alerts are in Early Access", 2026-04-20; "Monitors & Alerts — Now Generally Available", 2026-05-07) don't name an endpoint, and the feed carries titles only.
  - **The SDKs' changelogs** (`sentry-python`, `sentry-javascript`). They're about the event-ingestion SDKs, not the REST API.
  - **`src/sentry/api/urls.py` in getsentry/sentry.** It lists the routes, but not which are public, documented or deprecated.
- **Quote:** "## Deprecated 🚧 Use [Fetch an Organization's Monitors](/api/monitors/fetch-an-organizations-monitors) and [Fetch Alerts](/api/monitors/fetch-alerts) instead." (the description of `GET /api/0/organizations/{organization_id_or_slug}/alert-rules/` from sentry-api-schema `1da1f31f`, 2026-01-22; a line break after the heading)

### How the provider announces changes

- **No API versions.** Every path is under `/api/0/`, and `info.version` is `v0`. The spec's own versions (`api@2.16.0`, `0.224.0`) are the `@sentry/api` package's, released several times a day lately, not the API's.
- **The event's `version` is a date:** the spec release's publication date (`2026-11-18` for `api@2.16.0`), or a commit's date when replaying the history before tags. Several releases a day share one. The patch-it client keeps a date-versioned API's events only if it's in its list of date-versioned APIs, so `sentry` needs to be in it.
- **Deprecations are in words.** Over the whole history (1,130 commits, then the 19 `api@` releases) the `deprecated` flag is set on 3 operations, and the description deprecates 13, always no later than the flag:
  - **2022-03-01:** `GET .../stats/`, words and flag together: "This endpoint is deprecated in favor of [Organization Stats V2]".
  - **2024-07-31:** `POST .../user-feedback/`: "*This endpoint is DEPRECATED. We document it here for older SDKs ...". Flagged on 2026-06-16.
  - **2025-07-25:** `GET /api/0/projects/{organization_id_or_slug}/{project_id_or_slug}/issues/`: "**Deprecated**: This endpoint has been replaced with the [Organization Issues endpoint]". Flagged on 2026-06-16.
  - **2026-01-22:** the 10 issue and metric alert-rule operations (`.../alert-rules/`, `.../alert-rules/{alert_rule_id}/`, `.../rules/`, `.../rules/{rule_id}/`): "## Deprecated 🚧 Use [...] instead." Never flagged.
- **Endpoints leave the spec without a word.** See the removals below.

### What the plugin reads and ignores

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
- **Reads** the three description deprecations in the golden test (sentry-api-schema commits `ead7f0cf` → `4650a66a`, `d64149d6` → `c2f638f6`, `47231e61` → `1da1f31f`). The drop test replays `2.0.1` → `2.1.0`, the `/api/0` move. Other tests replay `0.223.0` → `0.224.0` (the flag after the words) and the two commits that added the "deprecation" wording that isn't a deprecation.

### Known gaps

- **No removals.** When Sentry turns an endpoint off, the plugin has already reported its deprecation, if Sentry said so in the spec, and reports nothing else.
- **The same deprecation can be reported twice** if Sentry removes the words from a description and puts them back on a later day: the diff keeps no memory, and each event is dated by its release. It hasn't happened in the history, and the two operations still deprecated today also carry the flag, which keeps them from it.
- **A deprecation said another way isn't seen,** like "has been replaced" with no "deprecated", or a parameter deprecated in prose.
- **Undocumented endpoints aren't watched.** The spec covers what Sentry documents publicly: 199 path items in `api@2.16.0`.
- **Request and response property changes aren't events,** including deprecated parameters and fields.

## Sentry Node.js SDK (`sentry/node-surface`)

### What it watches

The public API of the `@sentry/node` package on npm, release by release, read from its TypeScript declarations: exported names removed, class members removed, parameters removed, and parameters that are no longer optional. One `package_release` event per change. Every final release is listed from the npm registry (pre-releases and deprecated versions left out), and each consecutive pair is compared. What a release adds (a new module, class, function or method, or a new parameter with a default) is an `added` event.

Kind: `apisurface` (`kinds/apisurface`), with the TypeScript declaration reader (`kinds/apisurface/dtsurface`).

### Why this source

- **Source:** the release tarballs on the npm registry (<https://www.npmjs.com/package/@sentry/node>), each version's `dist.tarball`, read as published; the code is at <https://github.com/getsentry/sentry-javascript>. Checked on 2026-11-21.
- **Why this one:** the declarations in the tarball are exactly what TypeScript checks a customer's code against after `npm install @sentry/node@<version>`, and what editors show JavaScript users. A published tarball never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** CHANGELOG.md; MIGRATION.md (majors only).
- **Quote:** `export { systemErrorIntegration } from "@sentry/node-core"` (`build/types/index.d.ts` line 35) at 9.47.2, gone at 10.0.0.

### How the provider announces changes

- Releases every week or two; most of what it exports is re-exported from `@sentry/core` and `@sentry/node-core`, read here by name.
- **Forced on upgrade**, for a repo whose lockfile reaches the release.

### What the plugin reads and ignores

- Reads `package.json` and the `.d.ts`, `.d.mts` and `.d.cts` files of each release's tarball from the npm registry, in memory, as text (`kinds/apisurface/dtsurface`): nothing in the package is run, no install script, no TypeScript compiler. The entry points are `package.json`'s `exports` (each subpath, wildcards matched against the files; for each, its `types` condition, else the `import` one), or its `types`/`typings`.
- For each entry point: every exported name (classes, functions, enums, variables, interfaces, type aliases, namespaces, and names re-exported from another package, by name only), and below a class or namespace what it holds: a class's public methods and properties, and its constructor's parameters; a namespace's classes, functions and namespaces. Symbols are named by the import path code writes: `@sentry/node.init`.
- Parameters compare by position, as TypeScript passes them: a parameter gone off the end (with no rest parameter now) is removed; one optional before and required now is a `changed_default`. A renamed parameter is no change.
- Ignores names marked `@internal`, `private` and `protected` members, `#private` fields, setters, index signatures, interface and object-type members, interfaces and type aliases inside a namespace (an entry point's own are read), entry points under a private path (`_shims`, `internal`, `src`), the `./index` duplicate of the root, and what the default export holds (it's the same declaration as a named one).
- One declaration reachable at several import paths is one event, every path in its `symbols`.
- Drops (and logs) a removal that may not be one: a member of a name now re-exported from another package, now a value, an interface or a type (whose members aren't read); a method gone from a class that extends a class of another package (it may define it); a name an entry point may still get from a package it re-exports wholesale (`export * from "other"`).

### Known gaps

- What a property's type holds isn't followed: `client.chat.completions.create` is read as the `Completions` class's `create`, reached through namespaces and entry points, not through the `chat` property's type.
- Type changes (a parameter's type narrowed, a return type changed, an interface's fields) aren't events.
- A package that re-exports another wholesale (`export * from "@scope/core"`) hides that package's API: it's a source of its own, or not read.
- Classes are indexed by name across files to follow a method to its base class: two classes of the same name (in two namespaces) are read as one, which can only turn a removal into a note, never add one.
- A parameter added without being optional is a note (the change-event contract has no subject for it yet).
- Names re-exported from `@sentry/core` are read by name: their signatures are that package's.

## Sentry Python SDK (`sentry/python-surface`)

### What it watches

The public API of the `sentry-sdk` package on PyPI, release by release: classes, functions and methods removed or renamed, parameters removed, and parameters that lost their default (callers must now pass them). One `package_release` event per change. Every final release is listed from PyPI, and each consecutive pair is compared. What a release adds (a new module, class, function or method, or a new parameter with a default) is an `added` event.

Kind: `apisurface` (`kinds/apisurface`), with the Python reader (`kinds/apisurface/pysurface`).

### Why this source

- **Source:** the release files on PyPI (<https://pypi.org/project/sentry-sdk/>), each release's wheel read as published; the code is at <https://github.com/getsentry/sentry-python>. Checked on 2026-11-21.
- **Why this one:** the wheel is exactly what `pip install sentry-sdk==<version>` installs, so its public names and signatures are what a customer's code calls. A published file never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** CHANGELOG.md; MIGRATION_GUIDE.md (majors only, for the `migrationguide` kind).
- **Quote:** `def to_timestamp(value: "datetime") -> float` (`sentry_sdk/utils.py` line 251) at 2.73.0, gone at 2.74.0.

### How the provider announces changes

- The SDK releases every week or two; helpers in public-looking modules (`sentry_sdk.utils`) are removed in minors.
- **Forced on upgrade**, for a repo whose pin reaches the release.

### What the plugin reads and ignores

- Reads the `.py` and `.pyi` files under `sentry_sdk/` in the release's wheel (or, when a release has none, its sdist) as text, never importing or running them: module-level classes and functions, the methods of module-level classes, their parameters and defaults, `__all__`, an `__init__`'s re-exports from its own modules, and star imports. A `.pyi` stub is read instead of the `.py` beside it.
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
- `sentry_sdk.utils` and similar modules have no leading underscore but are mostly the SDK's own helpers: their removals are events, though few customers call them.

## Sentry SDKs (`sentry/python-changelog`, `sentry/javascript-changelog`, `sentry/go`, `sentry/dotnet`)

### What it watches

Names removed, renamed or deprecated in each release of Sentry's Python, JavaScript, Go and .NET SDKs, from that release's section of the SDK's `CHANGELOG.md` at its tag. One `package_release` event per bullet that names what it changes plainly. Sentry's API is a separate source (`sentry/api`).

Kind: `changelog` (`kinds/changelog`, `sdks.go`).

| Source | Package | Releases listed from | Tag |
|---|---|---|---|
| `sentry/python-changelog` | `pypi:sentry-sdk` | PyPI | `X.Y.Z` |
| `sentry/javascript-changelog` | `npm:@sentry/node`, `npm:@sentry/browser` (released together) | npm (`@sentry/node`) | `X.Y.Z` |
| `sentry/go` | `go:github.com/getsentry/sentry-go` | the repo's `vX.Y.Z` tags | `vX.Y.Z` |
| `sentry/dotnet` | `nuget:Sentry` | the repo's `X.Y.Z` tags | `X.Y.Z` |

### Why this source

- **Source:** https://github.com/getsentry/sentry-python/blob/master/CHANGELOG.md, https://github.com/getsentry/sentry-javascript/blob/develop/CHANGELOG.md, https://github.com/getsentry/sentry-go/blob/master/CHANGELOG.md, https://github.com/getsentry/sentry-dotnet/blob/main/CHANGELOG.md, each read raw at the release's tag; checked on 2026-11-20.
- **Why this one:** Sentry writes every major's removals into the changelog itself, under "Removed" / "Breaking Changes" / "Deprecations" headings or as conventional-commit bullets (`feat(core)!: Remove \`BaseClient\``). The Python 2.0 changelog section carries the same "Removed" and "Deprecated" lists as its `MIGRATION_GUIDE.md`, so the guide isn't a second source (it would repeat the same events under another id).
- **Alternatives considered:** the migration guides (`MIGRATION_GUIDE.md`, `MIGRATION.md`: prose sections per major, mostly behavior and options; the JavaScript one's headings aren't plain enough to read without false positives); GitHub release notes (the same text).
- **Quote:** "Removed `sentry_sdk.tracing.Span.new_span`. Use `sentry_sdk.tracing.Span.start_child` instead." (sentry-python 2.0.0).

### How the provider announces changes

A major a year or so per SDK, announced with a migration guide and a changelog section; deprecations land in minors before. sentry-python has a 3.0.0rc1 tagged: its 3.0.0 section will be read when 3.0.0 is on PyPI.

### What the plugin reads and ignores

- Reads the release's section only; a bullet counts under a breaking, removal or deprecation heading, or marked `!` (conventional commits), when its first sentence names what it removes plainly.
- Drops (and logs): removed options and integrations' settings, dropped runtime and framework versions, behavior changes, and bullets that list names in prose.

### Known gaps

- Options passed to `init()` (`with_locals`, `enableLogs`) aren't symbols: their removal isn't an event.
- A release whose section is missing at its tag is skipped as nothing to read.
