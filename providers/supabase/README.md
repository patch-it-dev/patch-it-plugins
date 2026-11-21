# Supabase

Every source patch-it reads for Supabase, one section each: what it watches, why this source, what it reads and ignores, its known gaps. Each is registered in [`provider.go`](provider.go) and read by its kind ([`kinds/`](../../kinds/)); the saved copies its tests replay, and the golden files, are in [`testdata/`](testdata/README.md).

## Supabase (`supabase/management-api`)

### What it watches

The Supabase Management API (`api.supabase.com/v1`): operations Supabase marks deprecated, and removals of operations it had marked deprecated, between two versions of the API's OpenAPI spec.

Kind: `openapi` (`kinds/openapi`), with its adapter, `Supabase` ([`management_api.go`](management_api.go)).

### Why this source

- **Source:** `apps/docs/spec/api_v1_openapi.json` in [supabase/supabase](https://github.com/supabase/supabase), pinned to the commit that last changed it (`https://raw.githubusercontent.com/supabase/supabase/<sha>/apps/docs/spec/api_v1_openapi.json`). The newest such commit comes from the GitHub commits API. Checked on 2026-11-19.
- **Why this one:**
  - It's Supabase's own description of the API: the spec its API reference is built from.
  - It marks deprecations with the `deprecated` flag, sometimes with the reason ("This endpoint is deprecated - use the deploy endpoint.").
  - There are 90 versions from 2024-05-31 to 2026-11-18, and a commit can be cited and replayed.
- **Alternatives considered:**
  - **`https://api.supabase.com/api/v1-json`.** The same spec served live, but it has no versions to pin, cite or replay.
  - **The [changelog](https://supabase.com/changelog)** (RSS at `https://supabase.com/changelog-rss.xml`, 238 posts since 2022). It has about 30 breaking or deprecation posts. Almost all are about the platform rather than an API operation: Postgres versions, pooler ports, OAuth providers, the CLI, self-hosting. Or they're behavior changes ("OAuth token endpoint will return HTTP 200 instead of 201") with no field or operation to name.
  - **There's no deprecations page** (`/docs/guides/deprecated` is a 404).
  - **The SDKs.** `supabase-js` (2.119) and `supabase-py` (2.34) haven't had a major release in years, and their changelogs have no breaking sections to read.
- **Quote:** "This endpoint is deprecated - use the deploy endpoint." (`POST /v1/projects/{ref}/functions`, deprecated in the spec on 2025-02-19)

### How the provider announces changes

- **No API version beyond `/v1`.** `info.version` is a fixed `1.0.0`, so the event's `version` is the date of the spec commit, the closest thing to the API version a change arrived in.
- **The spec changes a few times a month.** Deprecations are rare: two in two and a half years. Three more operations were added already marked deprecated and later un-deprecated, so they never became deprecated.

### What the plugin reads and ignores

- **Reads:** the spec at the newest commit, diffed against the last one. Two events come out of it:
  - **Deprecations:** an operation that gains the `deprecated` flag. The event quotes the whole operation, so Supabase's reason, when there is one, is in the excerpt.
  - **Removals (breaking):** an operation that leaves the spec after the previous spec had marked it deprecated.
- **Matches operations by path template:** `/v1/branches/{branch_id}/merge` and `/v1/branches/{branch_id_or_ref}/merge` are one operation (renamed 2025-09-11).
- **Ignores everything else:**
  - **Removals of operations that were never deprecated.** 22 operations left the spec in its history, none deprecated first: 14 analytics warehouse operations on 2024-07-15, 7 path-parameter renames, and one that came back 12 days later. Without credentials every route answers `401`, made-up ones included, so whether a removed one still works can't be checked from outside.
  - **Request and response property changes, and new operations.**
- Over the 90 commits this keeps 2 events, both deprecations: `POST /v1/projects/{ref}/functions` (2025-02-19) and `GET /v1/projects/{ref}/analytics/endpoints/logs.all` (2026-06-29).
- **Reads** a historical pair (a version pair, `693bdc38` → `eb21ae25`) and recreates the `logs.all` deprecation. The golden test also reads the `functions` deprecation as a pair (`57d668c6` → `bf04ff66`). The drop tests replay the warehouse removal and the branch path-parameter rename.

### Known gaps

- **A shutdown said only in words isn't an event.** On 2026-09-30 the `logs.all` description became "This endpoint has been removed and always responds with `410 Gone`", but the operation stayed in the spec. The diff sees no structural change.
- **Removals Supabase didn't mark first aren't events,** even if the endpoint really went away.
- **Only the Management API is covered.** The APIs most apps call (the Data API through PostgREST, Auth, Storage, Realtime) and the SDKs are not.
- **Two spec commits on one day are one version** (four days in the history), read at the day's last commit: a change made and undone the same day isn't seen.
- **The patch-it client can't place these events yet.** Their `version` is a date (`2026-06-29`), because the API has no version. The client's version check reads an `api_operation` version as a package version, so it needs the same rule for date-versioned API events that `cloudflare`'s need.

## Supabase JavaScript client (`supabase/js-surface`)

### What it watches

The public API of the `@supabase/supabase-js` package on npm, release by release, read from its TypeScript declarations: exported names removed, class members removed, parameters removed, and parameters that are no longer optional. One `package_release` event per change. Every final release is listed from the npm registry (pre-releases and deprecated versions left out), and each consecutive pair is compared.

Kind: `apisurface` (`kinds/apisurface`), with the TypeScript declaration reader (`kinds/apisurface/dtsurface`).

### Why this source

- **Source:** the release tarballs on the npm registry (<https://www.npmjs.com/package/@supabase/supabase-js>), each version's `dist.tarball`, read as published; the code is at <https://github.com/supabase/supabase-js>. Checked on 2026-11-21.
- **Why this one:** the declarations in the tarball are exactly what TypeScript checks a customer's code against after `npm install @supabase/supabase-js@<version>`, and what editors show JavaScript users. A published tarball never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** CHANGELOG.md; the v2 migration guide.
- **Quote:** `getSubscriptions(): RealtimeSubscription[]` (`SupabaseClient`, `dist/module/SupabaseClient.d.ts` line 100) at 1.35.7, gone at 2.0.0.

### How the provider announces changes

- The client is small and wraps separate packages (`@supabase/auth-js`, `@supabase/postgrest-js`, …) whose APIs it re-exports wholesale.
- **Forced on upgrade**, for a repo whose lockfile reaches the release.

### What the plugin reads and ignores

- Reads `package.json` and the `.d.ts`, `.d.mts` and `.d.cts` files of each release's tarball from the npm registry, in memory, as text (`kinds/apisurface/dtsurface`): nothing in the package is run, no install script, no TypeScript compiler. The entry points are `package.json`'s `exports` (each subpath, wildcards matched against the files; for each, its `types` condition, else the `import` one), or its `types`/`typings`.
- For each entry point: every exported name (classes, functions, enums, variables, interfaces, type aliases, namespaces, and names re-exported from another package, by name only), and below a class or namespace what it holds: a class's public methods and properties, and its constructor's parameters; a namespace's classes, functions and namespaces. Symbols are named by the import path code writes: `@supabase/supabase-js.SupabaseClient.from`.
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
- The packages it wraps are separate packages, not read by this source.

## Supabase Python client (`supabase/python-surface`)

### What it watches

The public API of the `supabase` package on PyPI, release by release: classes, functions and methods removed or renamed, parameters removed, and parameters that lost their default (callers must now pass them). One `package_release` event per change. Every final release is listed from PyPI, and each consecutive pair is compared.

Kind: `apisurface` (`kinds/apisurface`), with the Python reader (`kinds/apisurface/pysurface`).

### Why this source

- **Source:** the release files on PyPI (<https://pypi.org/project/supabase/>), each release's wheel read as published; the code is at <https://github.com/supabase/supabase-py>. Checked on 2026-11-21.
- **Why this one:** the wheel is exactly what `pip install supabase==<version>` installs, so its public names and signatures are what a customer's code calls. A published file never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** CHANGELOG.md (release-please).
- **Quote:** `api: Optional[SyncGoTrueAPI] = None, replace_default_headers: bool = False,` in `SupabaseAuthClient.__init__` (`supabase/lib/auth_client.py`) at 0.7.1, gone at 1.0.0.

### How the provider announces changes

- The client is small (about a hundred public names) and wraps separate packages (postgrest, realtime, storage3, supabase_auth) whose own APIs aren't read here.
- **Forced on upgrade**, for a repo whose pin reaches the release.

### What the plugin reads and ignores

- Reads the `.py` and `.pyi` files under `supabase/` in the release's wheel (or, when a release has none, its sdist) as text, never importing or running them: module-level classes and functions, the methods of module-level classes, their parameters and defaults, `__all__`, an `__init__`'s re-exports from its own modules, and star imports. A `.pyi` stub is read instead of the `.py` beside it.
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
- The packages it wraps (`postgrest`, `supabase_auth`, …) are separate packages, not read by this source.
